// Package otp exposes the merchant onboarding OTP HTTP endpoints.
package otp

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
	"github.com/Seizmann/RexiO-Pay/backend/internal/config"
	dbpool "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	"github.com/Seizmann/RexiO-Pay/backend/internal/middleware"
	otppkg "github.com/Seizmann/RexiO-Pay/backend/internal/otp"
	"github.com/go-chi/chi/v5"
)

// Handler serves merchant-scoped onboarding OTP requests. Authentication and
// platform-device routing are intentionally composed by the caller.
type Handler struct {
	service *otppkg.Service
}

// New creates an OTP HTTP handler backed by the application's database pool.
func New(pool *dbpool.Pool, cfg *config.Config) *Handler {
	return &Handler{service: otppkg.New(pool, cfg)}
}

// NewHandler creates an OTP HTTP handler backed by the supplied service.
func NewHandler(service *otppkg.Service) *Handler { return &Handler{service: service} }

// Routes mounts POST /init and GET /{id}. Mount it at
// /v1/onboarding/otp to expose the requested API paths.
func (h *Handler) Routes(r chi.Router) {
	r.Post("/init", h.Init)
	r.Get("/{id}", h.GetStatus)
}

// Init handles POST /v1/onboarding/otp/init.
func (h *Handler) Init(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.service == nil {
		apierr.Internal(w, errors.New("otp handler is not configured"))
		return
	}
	merchant := middleware.GetMerchant(r.Context())
	if merchant == nil || strings.TrimSpace(merchant.MerchantID) == "" {
		apierr.Unauthorized(w)
		return
	}
	var request initRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		apierr.Render(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "invalid request body")
		return
	}
	result, err := h.service.Init(r.Context(), merchant.MerchantID, request.Provider, request.AccountType, request.MFSNumber)
	if err != nil {
		renderError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// GetStatus handles GET /v1/onboarding/otp/:id.
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.service == nil {
		apierr.Internal(w, errors.New("otp handler is not configured"))
		return
	}
	merchant := middleware.GetMerchant(r.Context())
	if merchant == nil || strings.TrimSpace(merchant.MerchantID) == "" {
		apierr.Unauthorized(w)
		return
	}
	result, err := h.service.GetStatus(r.Context(), merchant.MerchantID, chi.URLParam(r, "id"))
	if err != nil {
		renderError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type initRequest struct {
	Provider    string `json:"provider"`
	AccountType string `json:"account_type"`
	MFSNumber   string `json:"mfs_number"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("otp handler: encode response", "err", err)
	}
}

func renderError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, otppkg.ErrInvalidProvider), errors.Is(err, otppkg.ErrInvalidAccountType),
		errors.Is(err, otppkg.ErrInvalidPhone), errors.Is(err, otppkg.ErrInvalidAmount),
		errors.Is(err, otppkg.ErrInvalidOTPID):
		apierr.Render(w, http.StatusBadRequest, apierr.CodeInvalidRequest, err.Error())
	case errors.Is(err, otppkg.ErrNoPendingOTP):
		apierr.NotFound(w)
	case errors.Is(err, otppkg.ErrPlatformDevice):
		apierr.Forbidden(w)
	default:
		slog.Error("otp handler: service error", "err", err)
		apierr.Internal(w, err)
	}
}
