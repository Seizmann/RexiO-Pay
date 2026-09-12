package branding

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/common"
	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
	dbpkg "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	db "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
	"github.com/Seizmann/RexiO-Pay/backend/internal/phone"
)

type Handler struct{ queries *db.Queries }

func NewHandler(pool *dbpkg.Pool) *Handler { return &Handler{queries: db.New(pool.SqlDB)} }
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.Get)
	r.Patch("/", h.Update)
	return r
}

type updateRequest struct {
	Name         string `json:"name"`
	LogoURL      string `json:"logo_url"`
	FaviconURL   string `json:"favicon_url"`
	BrandColor   string `json:"brand_color"`
	SupportEmail string `json:"support_email"`
	SupportPhone string `json:"support_phone"`
}

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := common.MerchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	item, err := h.queries.GetMerchant(r.Context(), id)
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	common.WriteJSON(w, http.StatusOK, item)
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
	req.Name = strings.TrimSpace(req.Name)
	req.LogoURL = strings.TrimSpace(req.LogoURL)
	req.FaviconURL = strings.TrimSpace(req.FaviconURL)
	req.BrandColor = strings.TrimSpace(req.BrandColor)
	req.SupportEmail = strings.TrimSpace(req.SupportEmail)
	req.SupportPhone = phone.Canonicalize(req.SupportPhone)
	if req.Name == "" || !hexColor.MatchString(req.BrandColor) {
		apierr.Render(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "name and a six-digit brand_color are required")
		return
	}
	if req.SupportPhone == "" {
		apierr.RenderParam(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "support_phone must be a valid Bangladesh mobile number", "support_phone")
		return
	}
	if err := h.queries.UpdateMerchantBranding(r.Context(), db.UpdateMerchantBrandingParams{ID: id, Name: req.Name, LogoUrl: req.LogoURL, FaviconUrl: req.FaviconURL, BrandColor: req.BrandColor, SupportEmail: req.SupportEmail, SupportPhone: req.SupportPhone}); err != nil {
		apierr.Internal(w, err)
		return
	}
	item, err := h.queries.GetMerchant(r.Context(), id)
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	common.WriteJSON(w, http.StatusOK, item)
}
