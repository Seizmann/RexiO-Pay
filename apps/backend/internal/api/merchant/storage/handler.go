package storage

import (
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/common"
	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
	"github.com/Seizmann/RexiO-Pay/backend/internal/storage"
)

type Handler struct{ store *storage.R2 }

func NewHandler(store *storage.R2) *Handler { return &Handler{store: store} }
func (h *Handler) Routes() chi.Router       { r := chi.NewRouter(); r.Post("/presign", h.Presign); return r }

type presignRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Kind        string `json:"kind"`
}

func (h *Handler) Presign(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := common.MerchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	var req presignRequest
	if !common.DecodeJSON(w, r, &req) {
		return
	}
	req.Filename = strings.TrimSpace(req.Filename)
	req.ContentType = strings.TrimSpace(req.ContentType)
	req.Kind = strings.Trim(strings.ToLower(strings.TrimSpace(req.Kind)), "/")
	if req.Filename == "" || req.ContentType == "" {
		apierr.Render(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "filename and content_type are required")
		return
	}
	if req.Kind != "logo" && req.Kind != "favicon" {
		apierr.Render(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "kind must be logo or favicon")
		return
	}
	filename := path.Base(req.Filename)
	key := "merchants/" + merchantID + "/" + req.Kind + "/" + filename
	url, err := h.store.PresignPut(r.Context(), key, req.ContentType, 15*time.Minute)
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]any{"key": key, "upload_url": url, "public_url": h.store.PublicURL(key), "expires_in": 900})
}
