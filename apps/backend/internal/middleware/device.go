package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
	cryptopkg "github.com/Seizmann/RexiO-Pay/backend/internal/crypto"
	dbpkg "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	dbsqlc "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
	"github.com/Seizmann/RexiO-Pay/backend/internal/telegram"
)

type ctxKeyDeviceID struct{}

// GetDeviceID retrieves the authenticated device ID from context.
func GetDeviceID(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyDeviceID{}).(string)
	return v
}

// DeviceHMAC authenticates device API requests using HMAC-SHA256.
//
// Device secrets are stored AES-256-GCM encrypted in the DB (never as hashed values).
// On each request, the ciphertext is decrypted in-memory to verify the HMAC.
// See plan §8 and internal/crypto/aesgcm.go for the full rationale.
//
// Signature: HMAC-SHA256(raw_device_secret, timestamp + "." + raw_body)
// Headers:   X-Rexio-Device-Id, X-Rexio-Timestamp (Unix seconds), X-Rexio-Signature (hex)
func DeviceHMAC(pool *dbpkg.Pool, secretKey []byte, tg *telegram.Alerter) func(http.Handler) http.Handler {
	q := dbsqlc.New(pool.SqlDB)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			deviceID := r.Header.Get("X-Rexio-Device-Id")
			tsStr := r.Header.Get("X-Rexio-Timestamp")
			sigHex := r.Header.Get("X-Rexio-Signature")

			if deviceID == "" || tsStr == "" || sigHex == "" {
				apierr.Render(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "missing device auth headers")
				return
			}

			// Anti-replay: reject if timestamp is more than 5 minutes off
			ts, err := strconv.ParseInt(tsStr, 10, 64)
			if err != nil {
				apierr.Render(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "invalid timestamp")
				return
			}
			skew := math.Abs(float64(time.Now().Unix() - ts))
			if skew > 300 {
				apierr.Render(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "timestamp skew too large (>5 min)")
				return
			}

			device, err := q.GetDevice(r.Context(), deviceID)
			if err != nil {
				apierr.Unauthorized(w)
				return
			}
			if device.Status == "disabled" {
				apierr.Render(w, http.StatusUnauthorized, apierr.CodeDeviceDisabled, "device is disabled")
				return
			}
			if len(device.SecretCiphertext) == 0 || len(device.SecretNonce) == 0 {
				apierr.Render(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "device not paired")
				return
			}

			// Decrypt the device secret in-memory — never use the ciphertext as the HMAC key
			rawSecret, err := cryptopkg.Decrypt(secretKey, device.SecretNonce, device.SecretCiphertext)
			if err != nil {
				slog.Error("device HMAC: decrypt secret", "device_id", deviceID, "err", err)
				apierr.Internal(w, err)
				return
			}

			// Get buffered body from context (set by ReadBody middleware)
			body, _ := r.Context().Value(ctxKeyRawBody{}).([]byte)

			// Compute expected HMAC
			mac := hmac.New(sha256.New, rawSecret)
			mac.Write([]byte(tsStr + "." + string(body)))
			expected := hex.EncodeToString(mac.Sum(nil))

			// Constant-time comparison
			sigBytes, err := hex.DecodeString(strings.TrimSpace(sigHex))
			if err != nil || !hmac.Equal([]byte(expected), sigBytes) {
				count, err2 := q.IncrFailedSigCount(r.Context(), deviceID)
				if err2 != nil {
					slog.Error("device HMAC: incr failed sig count", "err", err2)
				}
				if count >= 3 {
					if err3 := q.DisableDevice(r.Context(), deviceID); err3 != nil {
						slog.Error("device HMAC: disable device", "err", err3)
					}
					if tg != nil {
						tg.Send("RexiO Pay: device auto-disabled after 3 failed signatures — device_id: " + deviceID)
					}
					apierr.Render(w, http.StatusUnauthorized, apierr.CodeDeviceDisabled,
						"device disabled after 3 consecutive signature failures")
					return
				}
				apierr.Render(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "invalid signature")
				return
			}

			// Good signature — reset counter async
			if err := q.ResetFailedSigCount(r.Context(), deviceID); err != nil {
				slog.Error("device HMAC: reset failed sig count", "err", err)
			}

			ctx := context.WithValue(r.Context(), ctxKeyDeviceID{}, deviceID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
