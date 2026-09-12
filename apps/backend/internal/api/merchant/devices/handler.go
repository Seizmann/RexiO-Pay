package devices

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/common"
	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
	"github.com/Seizmann/RexiO-Pay/backend/internal/billing"
	dbpkg "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	db "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
	devpkg "github.com/Seizmann/RexiO-Pay/backend/internal/devices"
	"github.com/Seizmann/RexiO-Pay/backend/internal/idgen"
)

type Handler struct {
	pool    *dbpkg.Pool
	queries *db.Queries
	limits  *billing.Limiter
}

func NewHandler(pool *dbpkg.Pool) *Handler {
	return &Handler{pool: pool, queries: db.New(pool.SqlDB), limits: billing.New(pool)}
}
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Post("/{id}/disable", h.Disable)
	r.Post("/{id}/repair", h.Repair)
	return r
}

type createRequest struct {
	Name string `json:"name"`
}
type deviceResponse struct {
	ID               string      `json:"id"`
	Name             string      `json:"name"`
	Model            string      `json:"model"`
	AndroidVersion   string      `json:"android_version"`
	AppVersion       string      `json:"app_version"`
	Status           string      `json:"status"`
	FailedSigCount   int32       `json:"failed_sig_count"`
	LastHeartbeatAt  interface{} `json:"last_heartbeat_at,omitempty"`
	LastSMSSyncedAt  interface{} `json:"last_sms_synced_at,omitempty"`
	PairingToken     string      `json:"pairing_token,omitempty"`
	PairingExpiresAt interface{} `json:"pairing_expires_at,omitempty"`
	CreatedAt        interface{} `json:"created_at"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := common.MerchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	items, err := h.queries.ListDevicesForMerchant(r.Context(), merchantID)
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	out := make([]deviceResponse, 0, len(items))
	for _, item := range items {
		out = append(out, format(item, false))
	}
	common.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := common.MerchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	if !h.limits.Check(r.Context(), w, merchantID, billing.ResourceDevices) {
		return
	}
	var req createRequest
	if !common.DecodeJSON(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		apierr.RenderParam(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "name is required", "name")
		return
	}
	// Pairing tokens are returned only at creation/repair and expire quickly.
	item, _, _, err := devpkg.CreatePairing(r.Context(), h.pool, merchantID, req.Name, time.Now())
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	common.WriteJSON(w, http.StatusCreated, format(item, true))
}
func (h *Handler) Disable(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := common.MerchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	id := common.PathID(r, "id")
	item, err := h.queries.GetDevice(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		apierr.NotFound(w)
		return
	}
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	if item.MerchantID != merchantID {
		apierr.NotFound(w)
		return
	}
	if err := h.queries.DisableDevice(r.Context(), id); err != nil {
		apierr.Internal(w, err)
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]any{"id": id, "status": "disabled"})
}
func (h *Handler) Repair(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := common.MerchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	id := common.PathID(r, "id")
	item, err := h.queries.GetDevice(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		apierr.NotFound(w)
		return
	}
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	if item.MerchantID != merchantID {
		apierr.NotFound(w)
		return
	}
	token := idgen.New("pair_")
	expires := time.Now().Add(15 * time.Minute)
	if err := h.queries.RegeneratePairingToken(r.Context(), db.RegeneratePairingTokenParams{ID: id, PairingToken: sql.NullString{String: token, Valid: true}, PairingExpiresAt: sql.NullTime{Time: expires, Valid: true}}); err != nil {
		apierr.Internal(w, err)
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]any{"id": id, "pairing_token": token, "pairing_expires_at": expires})
}
func format(d db.Device, includePairing bool) deviceResponse {
	out := deviceResponse{ID: d.ID, Name: d.Name, Model: d.Model, AndroidVersion: d.AndroidVersion, AppVersion: d.AppVersion, Status: d.Status, FailedSigCount: d.FailedSigCount, CreatedAt: d.CreatedAt}
	if d.LastHeartbeatAt.Valid {
		out.LastHeartbeatAt = d.LastHeartbeatAt.Time
	}
	if d.LastSmsSyncedAt.Valid {
		out.LastSMSSyncedAt = d.LastSmsSyncedAt.Time
	}
	if includePairing && d.PairingToken.Valid {
		out.PairingToken = d.PairingToken.String
	}
	if includePairing && d.PairingExpiresAt.Valid {
		out.PairingExpiresAt = d.PairingExpiresAt.Time
	}
	return out
}
