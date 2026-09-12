// Package merchant implements the authenticated merchant REST API and the
// unauthenticated checkout endpoints.
package merchant

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/apikeys"
	"github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/branding"
	"github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/devices"
	"github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/domains"
	"github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/paymentprofiles"
	merchantsettings "github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/settings"
	merchantstorage "github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/storage"
	"github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/teammembers"
	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
	"github.com/Seizmann/RexiO-Pay/backend/internal/billing"
	dbpkg "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	dbsqlc "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
	"github.com/Seizmann/RexiO-Pay/backend/internal/idempotency"
	"github.com/Seizmann/RexiO-Pay/backend/internal/middleware"
	storager2 "github.com/Seizmann/RexiO-Pay/backend/internal/storage"
	"github.com/go-chi/chi/v5"
)

const (
	defaultSessionTTL = 30 * time.Minute
	maxJSONBody       = 1 << 20
)

// Handler contains the dependencies for the merchant API. AppURL is used only
// to build checkout URLs and may be left empty when the handler is mounted
// behind a reverse proxy (the relative URL is then returned).
type Handler struct {
	Pool                *dbpkg.Pool
	AppURL              string
	SessionTTL          time.Duration
	Billing             *billing.Limiter
	Storage             *storager2.R2
	DeviceEncryptionKey []byte
}

// NewHandler constructs a merchant API handler with the default session TTL.
func NewHandler(pool *dbpkg.Pool) *Handler {
	return &Handler{Pool: pool, SessionTTL: defaultSessionTTL, Billing: billing.New(pool)}
}

// NewHandlerWithDependencies constructs a merchant API handler and supplies the
// dependencies needed by the storage and device-pairing routes. NewHandler is
// kept for callers that do not enable those integrations.
func NewHandlerWithDependencies(pool *dbpkg.Pool, store *storager2.R2, deviceEncryptionKey []byte) *Handler {
	h := NewHandler(pool)
	h.Storage = store
	h.DeviceEncryptionKey = deviceEncryptionKey
	return h
}

// New is kept as a short constructor for callers that use package-level
// handlers in tests or in server composition.
func New(pool *dbpkg.Pool) *Handler { return NewHandler(pool) }

// Routes returns all /v1 routes owned by this package. The caller can mount
// the returned handler at the server root without changing main.go.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// Checkout and device pairing routes intentionally precede the API-key
	// middleware: customers and newly paired devices do not have merchant API
	// credentials yet.
	r.Get("/v1/checkout/{id}", h.getCheckout)
	r.Post("/v1/checkout/{id}/claim", h.claimCheckout)
	pair := devices.NewPairHandler(h.Pool, h.DeviceEncryptionKey)
	r.Mount("/v1/device/pair", pair.Routes())

	r.Route("/v1", func(r chi.Router) {
		r.Use(middleware.APIKeyAuth(h.Pool))
		r.Post("/sessions", h.createSession)
		r.Get("/sessions/{id}", h.getSession)
		r.Post("/sessions/{id}/cancel", h.cancelSession)
		r.Get("/payments", h.listPayments)
		r.Get("/payments/{id}", h.getPayment)
		r.Post("/payment-links", h.createPaymentLink)
		r.Get("/payment-links", h.listPaymentLinks)
		r.Get("/payment-links/{id}", h.getPaymentLink)

		// Merchant dashboard resources all require API-key authentication. The
		// subhandlers scope every query to the merchant in the auth context.
		r.Mount("/paymentprofiles", paymentprofiles.NewHandler(h.Pool).Routes())
		r.Mount("/apikeys", apikeys.NewHandler(h.Pool).Routes())
		r.Mount("/devices", devices.NewHandler(h.Pool).Routes())
		r.Mount("/teammembers", teammembers.NewHandler(h.Pool).Routes())
		r.Mount("/branding", branding.NewHandler(h.Pool).Routes())
		r.Mount("/settings", merchantsettings.NewHandler(h.Pool).Routes())
		r.Mount("/domains", domains.NewHandler(h.Pool).Routes())
		if h.Storage != nil {
			r.Mount("/storage", merchantstorage.NewHandler(h.Storage).Routes())
		}
	})
	return r
}

func (h *Handler) queries() *dbsqlc.Queries {
	if h == nil || h.Pool == nil || h.Pool.SqlDB == nil {
		return nil
	}
	return dbsqlc.New(h.Pool.SqlDB)
}

func merchantID(r *http.Request) (string, bool) {
	mc := middleware.GetMerchant(r.Context())
	if mc == nil || strings.TrimSpace(mc.MerchantID) == "" {
		return "", false
	}
	return mc.MerchantID, true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
	_, _ = w.Write([]byte("\n"))
}

func writeCached(w http.ResponseWriter, cached *idempotency.CachedResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(cached.StatusCode)
	_, _ = w.Write(cached.Body)
}

func decodeBody(r *http.Request, dst any) ([]byte, error) {
	if r.Body == nil {
		return nil, errors.New("request body is required")
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxJSONBody+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxJSONBody {
		return nil, errors.New("request body is too large")
	}
	dec := json.NewDecoder(strings.NewReader(string(body)))
	if err := dec.Decode(dst); err != nil {
		return body, err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return body, errors.New("request body must contain one JSON value")
	}
	return body, nil
}

func requestHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func idempotencyKey(r *http.Request) (string, bool) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	return key, key != "" && len(key) <= 255
}

func parsePage(r *http.Request) (limit, offset int32, ok bool) {
	limit = 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			return 0, 0, false
		}
		limit = int32(n)
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return 0, 0, false
		}
		offset = int32(n)
	}
	return limit, offset, true
}

func parseTimeParam(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, true
	}
	t, err := time.Parse(time.RFC3339, value)
	return t, err == nil
}

func notFoundOrInternal(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		apierr.NotFound(w)
		return
	}
	apierr.Internal(w, err)
}

func checkoutURL(base, id string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return "/pay/" + id
	}
	return base + "/pay/" + id
}

func nullableString(value sql.NullString) any {
	if value.Valid {
		return value.String
	}
	return nil
}

func nullableInt(value sql.NullInt64) any {
	if value.Valid {
		return value.Int64
	}
	return nil
}

func nullableTime(value sql.NullTime) any {
	if value.Valid {
		return value.Time
	}
	return nil
}

// allowURL verifies the URL shape and the merchant domain whitelist. A blank
// URL is allowed because return_url/cancel_url are optional in the API.
func (h *Handler) allowURL(r *http.Request, w http.ResponseWriter, merchant, raw, param string) bool {
	if strings.TrimSpace(raw) == "" {
		return true
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		apierr.RenderParam(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "must be a valid HTTP(S) URL", param)
		return false
	}
	if !h.domainAllowed(r, merchant, u.Hostname()) {
		apierr.Render(w, http.StatusForbidden, apierr.CodeDomainNotWhitelisted, "URL domain is not whitelisted")
		return false
	}
	return true
}

func (h *Handler) domainAllowed(r *http.Request, merchant, hostname string) bool {
	hostname = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(hostname), "."))
	if hostname == "" {
		return false
	}
	_, err := h.queries().CheckDomainAllowed(r.Context(), dbsqlc.CheckDomainAllowedParams{MerchantID: merchant, Domain: hostname})
	return err == nil
}

func (h *Handler) validateRequestDomains(r *http.Request, merchant string, returnURL, cancelURL string, w http.ResponseWriter) bool {
	if !h.allowURL(r, w, merchant, returnURL, "return_url") || !h.allowURL(r, w, merchant, cancelURL, "cancel_url") {
		return false
	}
	for _, header := range []string{"Origin", "Referer"} {
		raw := strings.TrimSpace(r.Header.Get(header))
		if raw == "" || raw == "null" {
			continue
		}
		u, err := url.Parse(raw)
		if err != nil || u.Hostname() == "" || !h.domainAllowed(r, merchant, u.Hostname()) {
			apierr.Render(w, http.StatusForbidden, apierr.CodeDomainNotWhitelisted, "request domain is not whitelisted")
			return false
		}
	}
	return true
}

func logInternal(operation string, err error) {
	slog.Error("merchant api request failed", "operation", operation, "err", err)
}

func invalid(w http.ResponseWriter, message, param string) {
	apierr.RenderParam(w, http.StatusBadRequest, apierr.CodeInvalidRequest, message, param)
}

func (h *Handler) ensurePool(w http.ResponseWriter) bool {
	if h == nil || h.Pool == nil || h.Pool.SqlDB == nil {
		apierr.Internal(w, fmt.Errorf("merchant handler has no database pool"))
		return false
	}
	return true
}
