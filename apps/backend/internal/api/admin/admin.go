// Package admin implements the platform-admin HTTP API.
package admin

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
	dbpkg "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	db "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
	"github.com/Seizmann/RexiO-Pay/backend/internal/idgen"
	"github.com/Seizmann/RexiO-Pay/backend/internal/middleware"
)

const (
	defaultPageSize = 50
	maxPageSize     = 100
)

type Handler struct {
	pool *dbpkg.Pool
	q    *db.Queries
}

func NewHandler(pool *dbpkg.Pool) *Handler {
	if pool == nil || pool.SqlDB == nil {
		return &Handler{pool: pool}
	}
	return &Handler{pool: pool, q: db.New(pool.SqlDB)}
}

// New is a short constructor retained for server composition and tests.
func New(pool *dbpkg.Pool) *Handler { return NewHandler(pool) }

// Routes returns the admin routes. The caller should mount this under /rexio-admin
// and compose APIKeyAuth before AdminOnly.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	if h == nil || h.pool == nil || h.pool.SqlDB == nil {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				apierr.Unauthorized(w)
			})
		})
	} else {
		r.Use(middleware.AdminOnly(h.pool))
	}
	r.Get("/merchants", h.listMerchants)
	r.Get("/merchants/{merchantID}", h.getMerchant)
	r.Post("/merchants/{merchantID}/plan", h.overridePlan)
	r.Post("/merchants/{merchantID}/disable", h.disableMerchant)
	r.Get("/devices", h.listDevices)
	r.Get("/sms/stats", h.smsStats)
	r.Get("/audit-logs", h.listAuditLogs)
	r.Get("/announcements", h.listAnnouncements)
	r.Post("/announcements", h.createAnnouncement)
	r.Patch("/announcements/{announcementID}", h.updateAnnouncement)
	r.Delete("/announcements/{announcementID}", h.deleteAnnouncement)
	return r
}

func (h *Handler) listMerchants(w http.ResponseWriter, r *http.Request) {
	page := pageParams(r)
	rows, err := h.q.ListMerchantsAdmin(r.Context(), db.ListMerchantsAdminParams{Limit: page.limit, Offset: page.offset})
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": rows, "limit": page.limit, "offset": page.offset})
}

func (h *Handler) getMerchant(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "merchantID")
	row, err := h.q.GetMerchantAdmin(r.Context(), merchantID)
	if errors.Is(err, sql.ErrNoRows) {
		apierr.NotFound(w)
		return
	}
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

type planOverrideRequest struct {
	PlanID     string     `json:"plan_id"`
	PlanStatus string     `json:"plan_status"`
	RenewsAt   *time.Time `json:"plan_renews_at"`
}

func (h *Handler) overridePlan(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "merchantID")
	var req planOverrideRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.PlanID = strings.TrimSpace(req.PlanID)
	req.PlanStatus = strings.TrimSpace(req.PlanStatus)
	if req.PlanID == "" {
		apierr.RenderParam(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "plan_id is required", "plan_id")
		return
	}
	if req.PlanStatus == "" {
		req.PlanStatus = "active"
	}
	if _, err := h.q.GetPlan(r.Context(), req.PlanID); errors.Is(err, sql.ErrNoRows) {
		apierr.RenderParam(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "unknown plan", "plan_id")
		return
	} else if err != nil {
		apierr.Internal(w, err)
		return
	}
	if err := h.q.UpdateMerchantPlan(r.Context(), db.UpdateMerchantPlanParams{
		ID: merchantID, PlanID: req.PlanID, PlanStatus: req.PlanStatus,
		PlanRenewsAt: nullTime(req.RenewsAt),
	}); err != nil {
		apierr.Internal(w, err)
		return
	}
	h.audit(r, merchantID, "admin.plan_override", "merchant", merchantID, map[string]any{"plan_id": req.PlanID, "plan_status": req.PlanStatus})
	writeJSON(w, http.StatusOK, map[string]any{"id": merchantID, "plan_id": req.PlanID, "plan_status": req.PlanStatus, "plan_renews_at": req.RenewsAt})
}

func (h *Handler) disableMerchant(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "merchantID")
	if _, err := h.q.GetMerchant(r.Context(), merchantID); errors.Is(err, sql.ErrNoRows) {
		apierr.NotFound(w)
		return
	} else if err != nil {
		apierr.Internal(w, err)
		return
	}
	if err := h.q.DisableMerchant(r.Context(), merchantID); err != nil {
		apierr.Internal(w, err)
		return
	}
	h.audit(r, merchantID, "admin.merchant_disable", "merchant", merchantID, nil)
	writeJSON(w, http.StatusOK, map[string]any{"id": merchantID, "plan_status": "disabled"})
}

func (h *Handler) listDevices(w http.ResponseWriter, r *http.Request) {
	page := pageParams(r)
	rows, err := h.q.ListAllDevicesForAdmin(r.Context(), db.ListAllDevicesForAdminParams{Limit: page.limit, Offset: page.offset})
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	// Never expose device secrets or pairing tokens in an admin API response.
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		items = append(items, map[string]any{
			"id": row.ID, "merchant_id": row.MerchantID, "merchant_name": row.MerchantName,
			"name": row.Name, "model": row.Model, "android_version": row.AndroidVersion,
			"app_version": row.AppVersion, "status": row.Status, "failed_sig_count": row.FailedSigCount,
			"last_heartbeat_at": nullableTime(row.LastHeartbeatAt), "last_sms_synced_at": nullableTime(row.LastSmsSyncedAt),
			"created_at": row.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items, "limit": page.limit, "offset": page.offset})
}

func (h *Handler) smsStats(w http.ResponseWriter, r *http.Request) {
	rows, err := h.q.ListSMSStatsByProvider(r.Context())
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": rows})
}

func (h *Handler) listAuditLogs(w http.ResponseWriter, r *http.Request) {
	page := pageParams(r)
	arg := db.ListAuditLogsParams{
		Column1: nullableQuery(r, "merchant_id"), Column2: nullableQuery(r, "action"), Column3: nullableQuery(r, "entity"),
		Limit: page.limit, Offset: page.offset,
	}
	if value := strings.TrimSpace(r.URL.Query().Get("from")); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			apierr.RenderParam(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "from must be RFC3339", "from")
			return
		}
		arg.Column4 = parsed
	}
	if value := strings.TrimSpace(r.URL.Query().Get("to")); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			apierr.RenderParam(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "to must be RFC3339", "to")
			return
		}
		arg.Column5 = parsed
	}
	rows, err := h.q.ListAuditLogs(r.Context(), arg)
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": rows, "limit": page.limit, "offset": page.offset})
}

func (h *Handler) listAnnouncements(w http.ResponseWriter, r *http.Request) {
	page := pageParams(r)
	rows, err := h.q.ListAnnouncementsAdmin(r.Context(), db.ListAnnouncementsAdminParams{Limit: page.limit, Offset: page.offset})
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": rows, "limit": page.limit, "offset": page.offset})
}

type announcementRequest struct {
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Severity  string     `json:"severity"`
	ExpiresAt *time.Time `json:"expires_at"`
}

func (h *Handler) createAnnouncement(w http.ResponseWriter, r *http.Request) {
	var req announcementRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !validAnnouncement(w, &req) {
		return
	}
	announcement, err := h.q.CreateAnnouncement(r.Context(), db.CreateAnnouncementParams{
		ID: idgen.New(idgen.PrefixAnnounce), Title: strings.TrimSpace(req.Title), Body: req.Body,
		Severity: req.Severity, ExpiresAt: nullTime(req.ExpiresAt),
	})
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	h.audit(r, "", "admin.announcement_create", "announcement", announcement.ID, map[string]any{"title": announcement.Title})
	writeJSON(w, http.StatusCreated, announcement)
}

func (h *Handler) updateAnnouncement(w http.ResponseWriter, r *http.Request) {
	var req announcementRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !validAnnouncement(w, &req) {
		return
	}
	id := chi.URLParam(r, "announcementID")
	announcement, err := h.q.UpdateAnnouncement(r.Context(), db.UpdateAnnouncementParams{
		ID: id, Title: strings.TrimSpace(req.Title), Body: req.Body, Severity: req.Severity, ExpiresAt: nullTime(req.ExpiresAt),
	})
	if errors.Is(err, sql.ErrNoRows) {
		apierr.NotFound(w)
		return
	}
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	h.audit(r, "", "admin.announcement_update", "announcement", id, nil)
	writeJSON(w, http.StatusOK, announcement)
}

func (h *Handler) deleteAnnouncement(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "announcementID")
	if err := h.q.DeleteAnnouncement(r.Context(), id); err != nil {
		apierr.Internal(w, err)
		return
	}
	h.audit(r, "", "admin.announcement_delete", "announcement", id, nil)
	w.WriteHeader(http.StatusNoContent)
}

func validAnnouncement(w http.ResponseWriter, req *announcementRequest) bool {
	req.Title = strings.TrimSpace(req.Title)
	req.Severity = strings.TrimSpace(req.Severity)
	if req.Title == "" {
		apierr.RenderParam(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "title is required", "title")
		return false
	}
	if strings.TrimSpace(req.Body) == "" {
		apierr.RenderParam(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "body is required", "body")
		return false
	}
	if req.Severity == "" {
		req.Severity = "info"
	}
	switch req.Severity {
	case "info", "success", "warning", "error":
		return true
	default:
		apierr.RenderParam(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "severity must be info, success, warning, or error", "severity")
		return false
	}
}

func (h *Handler) audit(r *http.Request, merchantID, action, entity, entityID string, details map[string]any) {
	payload := json.RawMessage(`{}`)
	if details != nil {
		if encoded, err := json.Marshal(details); err == nil {
			payload = encoded
		}
	}
	actor := sql.NullString{}
	_ = h.q.InsertAuditLog(r.Context(), db.InsertAuditLogParams{
		ID: idgen.New(idgen.PrefixAudit), MerchantID: nullableString(merchantID), ActorUserID: actor,
		Action: action, Entity: entity, EntityID: entityID, Details: payload,
	})
}

type pagination struct{ limit, offset int32 }

func pageParams(r *http.Request) pagination {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}
	if offset < 0 {
		offset = 0
	}
	return pagination{limit: int32(limit), offset: int32(offset)}
}

func nullableQuery(r *http.Request, name string) string {
	return strings.TrimSpace(r.URL.Query().Get(name))
}

func nullableString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func nullTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: value.UTC(), Valid: true}
}

func nullableTime(value sql.NullTime) any {
	if !value.Valid {
		return nil
	}
	return value.Time
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		apierr.Render(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "invalid JSON request body")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
