package devices

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"
)

const DefaultClockSkew = 5 * time.Minute

var (
	ErrMissingDeviceID  = errors.New("missing device id")
	ErrMissingTimestamp = errors.New("missing timestamp")
	ErrMissingSignature = errors.New("missing signature")
	ErrInvalidTimestamp = errors.New("invalid timestamp")
	ErrTimestampSkew    = errors.New("timestamp skew too large")
	ErrInvalidSignature = errors.New("invalid signature")
)

func Signature(secret []byte, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func VerifySignature(secret []byte, timestamp, signature string, body []byte, now time.Time, tolerance time.Duration) error {
	if timestamp == "" {
		return ErrMissingTimestamp
	}
	if signature == "" {
		return ErrMissingSignature
	}
	unix, err := strconv.ParseInt(strings.TrimSpace(timestamp), 10, 64)
	if err != nil {
		return ErrInvalidTimestamp
	}
	if tolerance <= 0 {
		tolerance = DefaultClockSkew
	}
	delta := now.Unix() - unix
	if delta > int64(tolerance/time.Second) || delta < -int64(tolerance/time.Second) {
		return ErrTimestampSkew
	}
	provided, err := hex.DecodeString(strings.TrimSpace(signature))
	if err != nil {
		return ErrInvalidSignature
	}
	expected := []byte(Signature(secret, timestamp, body))
	if !hmac.Equal(expected, []byte(hex.EncodeToString(provided))) {
		return ErrInvalidSignature
	}
	return nil
}
