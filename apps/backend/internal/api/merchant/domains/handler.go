package domains

import (
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/common"
	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
	dbpkg "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	db "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
	"github.com/Seizmann/RexiO-Pay/backend/internal/idgen"
	"github.com/go-chi/chi/v5"
)

type Handler struct{ queries *db.Queries }

func NewHandler(pool *dbpkg.Pool) *Handler { return &Handler{queries: db.New(pool.SqlDB)} }
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Add)
	r.Delete("/{id}", h.Remove)
	return r
}

type addRequest struct {
	Domain string `json:"domain"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	id, ok := common.MerchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	items, err := h.queries.ListDomains(r.Context(), id)
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}
func (h *Handler) Add(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := common.MerchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	var req addRequest
	if !common.DecodeJSON(w, r, &req) {
		return
	}
	domain, err := normalize(req.Domain)
	if err != nil {
		apierr.RenderParam(w, http.StatusBadRequest, apierr.CodeInvalidRequest, err.Error(), "domain")
		return
	}
	item, err := h.queries.AddDomain(r.Context(), db.AddDomainParams{ID: idgen.New(idgen.PrefixDomain), MerchantID: merchantID, Domain: domain})
	if errors.Is(err, sql.ErrNoRows) {
		apierr.Render(w, http.StatusConflict, apierr.CodeInvalidRequest, "domain is already whitelisted")
		return
	}
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	common.WriteJSON(w, http.StatusCreated, item)
}
func (h *Handler) Remove(w http.ResponseWriter, r *http.Request) {
	id, ok := common.MerchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	if err := h.queries.RemoveDomain(r.Context(), db.RemoveDomainParams{ID: common.PathID(r, "id"), MerchantID: id}); err != nil {
		apierr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func normalize(raw string) (string, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return "", errors.New("domain is required")
	}
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil || u.Hostname() == "" {
			return "", errors.New("domain must be a hostname")
		}
		raw = u.Hostname()
	}
	raw = strings.TrimSuffix(raw, ".")
	if strings.ContainsAny(raw, "/\\ @") || !strings.Contains(raw, ".") {
		return "", errors.New("domain must be a hostname")
	}
	return raw, nil
}
