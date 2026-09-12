package settings

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/common"
	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
	dbpkg "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	db "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
	"github.com/Seizmann/RexiO-Pay/backend/internal/phone"
	"github.com/go-chi/chi/v5"
)

type Handler struct{ queries *db.Queries }

func NewHandler(pool *dbpkg.Pool) *Handler { return &Handler{queries: db.New(pool.SqlDB)} }
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.Get)
	r.Patch("/", h.Update)
	return r
}

type updateRequest map[string]any

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := common.MerchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	merchant, err := h.queries.GetMerchant(r.Context(), id)
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	var settings any
	if len(merchant.Settings) > 0 {
		if err := json.Unmarshal(merchant.Settings, &settings); err != nil {
			settings = map[string]any{}
		}
	} else {
		settings = map[string]any{}
	}
	common.WriteJSON(w, http.StatusOK, map[string]any{"settings": settings})
}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := common.MerchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	var req updateRequest
	if !common.DecodeJSON(w, r, &req) {
		return
	}
	if len(req) == 0 {
		apierr.Render(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "settings cannot be empty")
		return
	}
	for key := range req {
		if strings.TrimSpace(key) == "" {
			apierr.Render(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "invalid setting name")
			return
		}
	}
	current, err := h.queries.GetMerchant(r.Context(), id)
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	merged := map[string]any{}
	if len(current.Settings) > 0 {
		_ = json.Unmarshal(current.Settings, &merged)
	}
	for k, v := range req {
		merged[k] = v
	}
	raw, _ := json.Marshal(merged)
	if err := h.queries.UpdateMerchantSettings(r.Context(), db.UpdateMerchantSettingsParams{ID: id, Settings: raw}); err != nil {
		apierr.Internal(w, err)
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]any{"settings": merged})
}

// NormalizeSupportPhone is shared by branding callers that validate merchant contact fields.
func NormalizeSupportPhone(v string) string { return phone.Canonicalize(v) }
