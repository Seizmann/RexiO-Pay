package devices

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
	cryptopkg "github.com/Seizmann/RexiO-Pay/backend/internal/crypto"
	dbpkg "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	db "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
	"github.com/Seizmann/RexiO-Pay/backend/internal/idgen"
)

const PairingTTL = 15 * time.Minute

var ErrPairingExpired = errors.New("pairing token expired")

// PairingExpired is kept independent of the database clock so it can be used
// by dashboard code and tested without a live database.
func PairingExpired(expiresAt, now time.Time) bool {
	return expiresAt.IsZero() || !now.Before(expiresAt)
}

// NewPairingToken returns an opaque, URL/QR-safe token.
func NewPairingToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// CreatePairing creates a pending device and returns the QR token and expiry.
// A dashboard should encode the token together with the server URL and
// merchant ID; the token itself contains no tenant information.
func CreatePairing(ctx context.Context, pool *dbpkg.Pool, merchantID, name string, now time.Time) (device db.Device, token string, expiresAt time.Time, err error) {
	if strings.TrimSpace(merchantID) == "" {
		return db.Device{}, "", time.Time{}, errors.New("merchant id is required")
	}
	if now.IsZero() {
		now = time.Now()
	}
	token, err = NewPairingToken()
	if err != nil {
		return db.Device{}, "", time.Time{}, err
	}
	expiresAt = now.Add(PairingTTL)
	device, err = db.New(pool.SqlDB).CreateDevice(ctx, db.CreateDeviceParams{
		ID: idgen.New(idgen.PrefixDevice), MerchantID: merchantID, Name: strings.TrimSpace(name),
		PairingToken: sql.NullString{String: token, Valid: true}, PairingExpiresAt: sql.NullTime{Time: expiresAt, Valid: true},
	})
	return device, token, expiresAt, err
}

type PairHandler struct {
	Pool          *dbpkg.Pool
	EncryptionKey []byte
	Now           func() time.Time
}

func NewPairHandler(pool *dbpkg.Pool, encryptionKey []byte) *PairHandler {
	return &PairHandler{Pool: pool, EncryptionKey: encryptionKey}
}

func (h *PairHandler) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

// ServeHTTP consumes a one-time QR token and returns a generated 32-byte
// secret exactly once. Only encrypted secret material is persisted.
func (h *PairHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Pool == nil || len(h.EncryptionKey) != 32 {
		apierr.Render(w, http.StatusInternalServerError, apierr.CodeInternalError, "pairing is not configured")
		return
	}
	var req PairRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req); err != nil || strings.TrimSpace(req.PairingToken) == "" {
		apierr.Render(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "pairing_token is required")
		return
	}
	q := db.New(h.Pool.SqlDB)
	device, err := q.GetDeviceByPairingToken(r.Context(), sql.NullString{String: strings.TrimSpace(req.PairingToken), Valid: true})
	if err != nil {
		apierr.Render(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "pairing token is invalid or expired")
		return
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		apierr.Internal(w, err)
		return
	}
	ciphertext, nonce, err := cryptopkg.Encrypt(h.EncryptionKey, secret)
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	if err := q.StorePairedDevice(r.Context(), db.StorePairedDeviceParams{ID: device.ID, SecretCiphertext: ciphertext, SecretNonce: nonce}); err != nil {
		apierr.Internal(w, err)
		return
	}
	if err := q.UpdateDeviceMetadata(r.Context(), db.UpdateDeviceMetadataParams{
		ID: device.ID, Name: req.DeviceName, Model: req.Model, AndroidVersion: req.AndroidVersion, AppVersion: req.AppVersion,
	}); err != nil {
		apierr.Internal(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(PairResponse{DeviceID: device.ID, DeviceSecret: base64.RawURLEncoding.EncodeToString(secret), Status: "active"})
}
