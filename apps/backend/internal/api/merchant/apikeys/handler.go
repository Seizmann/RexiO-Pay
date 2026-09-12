package apikeys

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/common"
	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
	dbpkg "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	db "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
	"github.com/Seizmann/RexiO-Pay/backend/internal/idgen"
)

type Handler struct{ queries *db.Queries }

func NewHandler(pool *dbpkg.Pool) *Handler { return &Handler{queries: db.New(pool.SqlDB)} }

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Delete("/{id}", h.Revoke)
	return r
}

type createRequest struct {
	Name   string `json:"name"`
	Scopes string `json:"scopes"`
}

type keyResponse struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	KeyPrefix  string      `json:"key_prefix"`
	Scopes     string      `json:"scopes"`
	LastUsedAt interface{} `json:"last_used_at,omitempty"`
	CreatedAt  interface{} `json:"created_at"`
	RevokedAt  interface{} `json:"revoked_at,omitempty"`
	Key        string      `json:"key,omitempty"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := common.MerchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	items, err := h.queries.ListAPIKeys(r.Context(), merchantID)
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	out := make([]keyResponse, 0, len(items))
	for _, item := range items {
		out = append(out, keyResponse{ID: item.ID, Name: item.Name, KeyPrefix: item.KeyPrefix, Scopes: item.Scopes, LastUsedAt: nullableTime(item.LastUsedAt), CreatedAt: item.CreatedAt, RevokedAt: nullableTime(item.RevokedAt)})
	}
	common.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := common.MerchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	var req createRequest
	if !common.DecodeJSON(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Scopes = strings.TrimSpace(req.Scopes)
	if req.Name == "" {
		apierr.RenderParam(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "name is required", "name")
		return
	}
	if req.Scopes == "" {
		req.Scopes = "full"
	}
	if len(req.Name) > 100 || len(req.Scopes) > 500 {
		apierr.Render(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "name or scopes are too long")
		return
	}
	rawKey, err := randomKey()
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	prefix := rawKey[:12]
	sum := sha256.Sum256([]byte(rawKey))
	row, err := h.queries.CreateAPIKey(r.Context(), db.CreateAPIKeyParams{ID: idgen.New(idgen.PrefixAPIKey), MerchantID: merchantID, Name: req.Name, KeyPrefix: prefix, KeyHash: hex.EncodeToString(sum[:]), Scopes: req.Scopes})
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	common.WriteJSON(w, http.StatusCreated, keyResponse{ID: row.ID, Name: row.Name, KeyPrefix: row.KeyPrefix, Scopes: row.Scopes, CreatedAt: row.CreatedAt, Key: rawKey})
}

func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := common.MerchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	id := common.PathID(r, "id")
	if id == "" {
		apierr.NotFound(w)
		return
	}
	if err := h.queries.RevokeAPIKey(r.Context(), db.RevokeAPIKeyParams{ID: id, MerchantID: merchantID}); err != nil {
		apierr.Internal(w, err)
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]any{"id": id, "revoked": true})
}

func randomKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate api key: %w", err)
	}
	return "rk_live_" + hex.EncodeToString(buf), nil
}

func nullableTime(v sql.NullTime) interface{} {
	if !v.Valid {
		return nil
	}
	return v.Time
}
