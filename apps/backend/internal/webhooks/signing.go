// Package webhooks provides webhook event delivery, signing, and retry helpers.
package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultTimestampTolerance is the maximum age of a webhook signature.
	DefaultTimestampTolerance = 5 * time.Minute
	SignatureHeader           = "X-Rexio-Signature"
	EventHeader               = "X-Rexio-Event"
)

var (
	ErrInvalidSignature = errors.New("invalid webhook signature")
	ErrExpiredSignature = errors.New("expired webhook signature")
	ErrMalformedHeader  = errors.New("malformed webhook signature header")
)

// Sign returns the Stripe-style value for a webhook signature header:
// t=<unix timestamp>,v1=<lowercase hex HMAC-SHA256>.
// The signed value is timestamp + "." + the exact request body.
func Sign(secret string, payload []byte, timestamp time.Time) string {
	ts := strconv.FormatInt(timestamp.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(ts))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(payload)
	return "t=" + ts + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}

// SignPayload is an explicit alias for Sign.
func SignPayload(secret string, payload []byte, timestamp time.Time) string {
	return Sign(secret, payload, timestamp)
}

// ParseSignatureHeader extracts the timestamp and all v1 signatures from a
// webhook signature header. Unknown components are ignored so the format can
// be extended without breaking consumers.
func ParseSignatureHeader(header string) (time.Time, []string, error) {
	var (
		timestamp int64
		haveTS    bool
		sigs      []string
	)
	for _, component := range strings.Split(header, ",") {
		parts := strings.SplitN(strings.TrimSpace(component), "=", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return time.Time{}, nil, ErrMalformedHeader
		}
		switch parts[0] {
		case "t":
			if haveTS {
				return time.Time{}, nil, ErrMalformedHeader
			}
			parsed, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return time.Time{}, nil, fmt.Errorf("%w: invalid timestamp", ErrMalformedHeader)
			}
			timestamp, haveTS = parsed, true
		case "v1":
			if _, err := hex.DecodeString(parts[1]); err != nil {
				return time.Time{}, nil, fmt.Errorf("%w: invalid v1 signature", ErrMalformedHeader)
			}
			sigs = append(sigs, parts[1])
		}
	}
	if !haveTS || len(sigs) == 0 {
		return time.Time{}, nil, ErrMalformedHeader
	}
	return time.Unix(timestamp, 0), sigs, nil
}

// Verify checks a Stripe-style webhook signature using a five-minute default
// timestamp tolerance. It compares signatures in constant time.
func Verify(secret string, payload []byte, header string, now time.Time, tolerance time.Duration) error {
	if tolerance <= 0 {
		tolerance = DefaultTimestampTolerance
	}
	timestamp, signatures, err := ParseSignatureHeader(header)
	if err != nil {
		return err
	}
	age := now.Sub(timestamp)
	if age < -tolerance || age > tolerance {
		return ErrExpiredSignature
	}

	expectedHeader := Sign(secret, payload, timestamp)
	_, expected, err := ParseSignatureHeader(expectedHeader)
	if err != nil {
		return err
	}
	expectedBytes, err := hex.DecodeString(expected[0])
	if err != nil {
		return err
	}
	for _, signature := range signatures {
		provided, err := hex.DecodeString(signature)
		if err == nil && hmac.Equal(expectedBytes, provided) {
			return nil
		}
	}
	return ErrInvalidSignature
}

// VerifySignature is Verify with the documented five-minute tolerance.
func VerifySignature(secret string, payload []byte, header string, now time.Time) error {
	return Verify(secret, payload, header, now, DefaultTimestampTolerance)
}
