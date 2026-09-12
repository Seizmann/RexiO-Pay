package devices

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/sqlc-dev/pqtype"

	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
	cryptopkg "github.com/Seizmann/RexiO-Pay/backend/internal/crypto"
	dbpkg "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	db "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
	"github.com/Seizmann/RexiO-Pay/backend/internal/idgen"
	"github.com/Seizmann/RexiO-Pay/backend/internal/middleware"
	"github.com/Seizmann/RexiO-Pay/backend/internal/sms"
	"github.com/Seizmann/RexiO-Pay/backend/internal/telegram"
)

// AuthMiddleware authenticates all device endpoints using the same headers and
// canonical signing string as REQUIREMENT §12.2. Wrap it with middleware.ReadBody.
type AuthMiddleware struct {
	Pool          *dbpkg.Pool
	EncryptionKey []byte
	Telegram      *telegram.Alerter
	Now           func() time.Time
	ClockSkew     time.Duration
}

func NewAuthMiddleware(pool *dbpkg.Pool, encryptionKey []byte, tg *telegram.Alerter) *AuthMiddleware {
	return &AuthMiddleware{Pool: pool, EncryptionKey: encryptionKey, Telegram: tg}
}

func (a *AuthMiddleware) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

func (a *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a == nil || a.Pool == nil || len(a.EncryptionKey) != 32 {
			apierr.Render(w, http.StatusInternalServerError, apierr.CodeInternalError, "device authentication is not configured")
			return
		}
		deviceID := strings.TrimSpace(r.Header.Get("X-Rexio-Device-Id"))
		timestamp := strings.TrimSpace(r.Header.Get("X-Rexio-Timestamp"))
		signature := strings.TrimSpace(r.Header.Get("X-Rexio-Signature"))
		if deviceID == "" {
			apierr.Render(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "missing device auth headers")
			return
		}
		if timestamp == "" {
			apierr.Render(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "missing device auth headers")
			return
		}
		if signature == "" {
			apierr.Render(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "missing device auth headers")
			return
		}
		q := db.New(a.Pool.SqlDB)
		device, err := q.GetDevice(r.Context(), deviceID)
		if err != nil {
			apierr.Unauthorized(w)
			return
		}
		if device.Status == "disabled" {
			apierr.Render(w, http.StatusUnauthorized, apierr.CodeDeviceDisabled, "device is disabled")
			return
		}
		secret, err := cryptopkg.Decrypt(a.EncryptionKey, device.SecretNonce, device.SecretCiphertext)
		if err != nil {
			slog.Error("device auth secret decrypt failed", "device_id", deviceID, "err", err)
			apierr.Internal(w, err)
			return
		}
		tolerance := a.ClockSkew
		if tolerance <= 0 {
			tolerance = DefaultClockSkew
		}
		if err := VerifySignature(secret, timestamp, signature, middleware.RawBody(r), a.now(), tolerance); err != nil {
			count, countErr := q.IncrFailedSigCount(r.Context(), deviceID)
			if countErr != nil {
				slog.Error("device auth failed count update", "device_id", deviceID, "err", countErr)
			}
			if count >= 3 {
				if disableErr := q.DisableDevice(r.Context(), deviceID); disableErr != nil {
					slog.Error("device auth disable failed", "device_id", deviceID, "err", disableErr)
				}
				if a.Telegram != nil {
					a.Telegram.Send("RexiO Pay: device auto-disabled after 3 failed signatures — device_id: " + deviceID)
				}
				apierr.Render(w, http.StatusUnauthorized, apierr.CodeDeviceDisabled, "device disabled after 3 consecutive signature failures")
				return
			}
			apierr.Render(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "invalid device signature")
			return
		}
		if err := q.ResetFailedSigCount(r.Context(), deviceID); err != nil {
			slog.Error("device auth reset failed count", "device_id", deviceID, "err", err)
		}
		ctx := context.WithValue(r.Context(), deviceIDKey{}, deviceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type deviceIDKey struct{}

func DeviceID(ctx context.Context) string {
	id, _ := ctx.Value(deviceIDKey{}).(string)
	return id
}

// HeartbeatHandler updates liveness and app version. Battery and queue depth
// are accepted for forward compatibility but are not persisted by the MVP
// schema.
type HeartbeatHandler struct{ Pool *dbpkg.Pool }

func (h *HeartbeatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req HeartbeatRequest
	if err := decodeJSON(w, r, &req, 16<<10); err != nil {
		return
	}
	deviceID := DeviceID(r.Context())
	if deviceID == "" {
		deviceID = middleware.GetDeviceID(r.Context())
	}
	if deviceID == "" {
		apierr.Unauthorized(w)
		return
	}
	if h == nil || h.Pool == nil {
		apierr.Internal(w, errors.New("heartbeat is not configured"))
		return
	}
	if err := db.New(h.Pool.SqlDB).UpdateHeartbeat(r.Context(), db.UpdateHeartbeatParams{ID: deviceID, AppVersion: strings.TrimSpace(req.AppVersion)}); err != nil {
		apierr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, HeartbeatResponse{Status: "ok"})
}

// ConfigHandler returns app-update and parser/sender configuration. It does
// not disclose any merchant or device secret.
type ConfigHandler struct{ Config ConfigResponse }

func (h *ConfigHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cfg := ConfigResponse{SenderIDs: map[string][]string{}}
	if h != nil {
		cfg = h.Config
		if cfg.SenderIDs == nil {
			cfg.SenderIDs = map[string][]string{}
		}
	}
	writeJSON(w, http.StatusOK, cfg)
}

// Parser is deliberately injectable. The adapter below integrates with the
// existing provider parser while retaining a stub-friendly seam for future
// parser implementations.
type Parser interface {
	Parse(raw string, provider, accountType string) (json.RawMessage, error)
}

var ErrParserUnavailable = errors.New("sms parser unavailable")

type ParserFunc func(string, string, string) (json.RawMessage, error)

func (f ParserFunc) Parse(raw, provider, accountType string) (json.RawMessage, error) {
	return f(raw, provider, accountType)
}

type SMSParserAdapter struct{ Parser *sms.Parser }

func (p SMSParserAdapter) Parse(raw, provider, accountType string) (json.RawMessage, error) {
	if p.Parser == nil {
		return nil, ErrParserUnavailable
	}
	parsed, err := p.Parser.Parse(provider, raw)
	if err != nil {
		return nil, err
	}
	if accountType != "" && parsed.AccountType != "" && parsed.AccountType != accountType {
		return nil, errors.New("sms account type does not match device profile")
	}
	return json.Marshal(parsed)
}

// NewSMSHandler constructs an ingest handler. Parser may be nil while the
// parser package is being developed; raw SMS rows are still retained with a
// failed parse status.
func NewSMSHandler(pool *dbpkg.Pool, parser Parser) *SMSHandler {
	return &SMSHandler{Pool: pool, Parser: parser}
}

type SMSHandler struct {
	Pool   *dbpkg.Pool
	Parser Parser
	Now    func() time.Time
}

func (h *SMSHandler) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

func (h *SMSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Pool == nil {
		apierr.Internal(w, errors.New("sms ingest is not configured"))
		return
	}
	var req SMSRequest
	if err := decodeJSON(w, r, &req, 1<<20); err != nil {
		return
	}
	items := req.Messages
	if len(items) == 0 && req.SMS != nil {
		items = []SMSItem{*req.SMS}
	}
	if len(items) == 0 || len(items) > 100 {
		apierr.Render(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "messages must contain between 1 and 100 SMS records")
		return
	}
	deviceID := DeviceID(r.Context())
	if deviceID == "" {
		deviceID = middleware.GetDeviceID(r.Context())
	}
	if deviceID == "" {
		apierr.Unauthorized(w)
		return
	}
	device, err := db.New(h.Pool.SqlDB).GetDevice(r.Context(), deviceID)
	if err != nil {
		apierr.Unauthorized(w)
		return
	}
	q := db.New(h.Pool.SqlDB)
	acks := make([]SMSAck, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Body) == "" || strings.TrimSpace(item.ClientID) == "" {
			apierr.Render(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "each SMS requires client_id and body")
			return
		}
		receivedAt := item.SMSAt
		if receivedAt.IsZero() {
			receivedAt = h.now()
		}
		row, insertErr := q.InsertSMSMessage(r.Context(), db.InsertSMSMessageParams{
			ID: idgen.New(idgen.PrefixSMS), MerchantID: device.MerchantID, DeviceID: sql.NullString{String: deviceID, Valid: true},
			Provider: item.Provider, AccountType: item.AccountType, RawText: item.Body, ParseStatus: "pending", MatchStatus: "unmatched", Source: "app", SimSlot: nullableInt(item.SimSlot), ReceivedAt: receivedAt,
		})
		if insertErr != nil {
			// A unique TrxID conflict is not available before parsing; clients can
			// safely retry and receive an error rather than duplicate raw records.
			apierr.Internal(w, insertErr)
			return
		}
		status := "failed"
		if h.Parser != nil {
			parsed, parseErr := h.Parser.Parse(item.Body, item.Provider, item.AccountType)
			if parseErr == nil && json.Valid(parsed) {
				if err := q.UpdateSMSParsed(r.Context(), db.UpdateSMSParsedParams{ID: row.ID, Parsed: pqtype.NullRawMessage{RawMessage: parsed, Valid: true}, Provider: item.Provider, AccountType: item.AccountType}); err == nil {
					status = "parsed"
				} else {
					slog.Error("sms parse update failed", "sms_id", row.ID, "err", err)
				}
			}
		}
		acks = append(acks, SMSAck{ClientID: item.ClientID, SMSID: row.ID, Status: status})
	}
	if err := q.UpdateLastSMSSynced(r.Context(), deviceID); err != nil {
		slog.Error("update last SMS sync failed", "device_id", deviceID, "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]any{"acks": acks})
}

func nullableInt(v *int32) sql.NullInt32 {
	if v == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: *v, Valid: true}
}
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any, limit int64) error {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit)).Decode(dst); err != nil {
		apierr.Render(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "invalid JSON request")
		return err
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
