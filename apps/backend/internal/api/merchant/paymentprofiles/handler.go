package paymentprofiles

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/common"
	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
	"github.com/Seizmann/RexiO-Pay/backend/internal/billing"
	dbpkg "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	db "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
	"github.com/Seizmann/RexiO-Pay/backend/internal/idgen"
	"github.com/Seizmann/RexiO-Pay/backend/internal/phone"
	"github.com/go-chi/chi/v5"
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
	r.Get("/{id}", h.Get)
	r.Patch("/{id}", h.Update)
	return r
}

type createRequest struct {
	Provider            string `json:"provider"`
	AccountType         string `json:"account_type"`
	MFSNumber           string `json:"mfs_number"`
	DisplayName         string `json:"display_name"`
	BalanceVerification *bool  `json:"balance_verification"`
}

type updateRequest struct {
	DisplayName *string `json:"display_name"`
	SimSlot     *int32  `json:"sim_slot"`
	Status      *string `json:"status"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := common.MerchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	rows, err := h.queries.ListProfilesForMerchant(r.Context(), merchantID)
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]any{"data": rows})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := common.MerchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	id := common.PathID(r, "id")
	profile, err := h.queries.GetProfile(r.Context(), db.GetProfileParams{ID: id, MerchantID: merchantID})
	if errors.Is(err, sql.ErrNoRows) {
		apierr.NotFound(w)
		return
	}
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	common.WriteJSON(w, http.StatusOK, profile)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := common.MerchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	if !h.limits.Check(r.Context(), w, merchantID, billing.ResourcePaymentProfiles) {
		return
	}
	var req createRequest
	if !common.DecodeJSON(w, r, &req) {
		return
	}
	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	req.AccountType = strings.ToLower(strings.TrimSpace(req.AccountType))
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	req.MFSNumber = phone.Canonicalize(req.MFSNumber)
	if req.Provider == "" || req.AccountType == "" || req.MFSNumber == "" {
		apierr.Render(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "provider, account_type, and a valid mfs_number are required")
		return
	}
	balanceVerification := true
	if req.BalanceVerification != nil {
		balanceVerification = *req.BalanceVerification
	}
	profile, err := h.queries.CreateProfile(r.Context(), db.CreateProfileParams{
		ID: idgen.New(idgen.PrefixProfile), MerchantID: merchantID, Provider: req.Provider,
		AccountType: req.AccountType, MfsNumber: req.MFSNumber, DisplayName: req.DisplayName,
		BalanceVerification: balanceVerification, IsOtpVerified: false,
	})
	if err != nil {
		if isConstraint(err) {
			apierr.Render(w, http.StatusConflict, apierr.CodeInvalidRequest, "a payment profile for this number already exists")
		} else {
			apierr.Internal(w, err)
		}
		return
	}
	common.WriteJSON(w, http.StatusCreated, profile)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := common.MerchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	id := common.PathID(r, "id")
	var req updateRequest
	if !common.DecodeJSON(w, r, &req) {
		return
	}
	profile, err := h.queries.GetProfile(r.Context(), db.GetProfileParams{ID: id, MerchantID: merchantID})
	if errors.Is(err, sql.ErrNoRows) {
		apierr.NotFound(w)
		return
	}
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	name := profile.DisplayName
	if req.DisplayName != nil {
		name = strings.TrimSpace(*req.DisplayName)
	}
	slot := profile.SimSlot
	if req.SimSlot != nil {
		if *req.SimSlot < 0 || *req.SimSlot > 3 {
			apierr.RenderParam(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "sim_slot must be between 0 and 3", "sim_slot")
			return
		}
		slot = sql.NullInt32{Int32: *req.SimSlot, Valid: true}
	}
	if err := h.queries.UpdateProfile(r.Context(), db.UpdateProfileParams{ID: id, MerchantID: merchantID, DisplayName: name, SimSlot: slot}); err != nil {
		apierr.Internal(w, err)
		return
	}
	if req.Status != nil {
		status := strings.ToLower(strings.TrimSpace(*req.Status))
		if status != "active" && status != "disabled" && status != "pending" {
			apierr.RenderParam(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "invalid profile status", "status")
			return
		}
		if err := h.queries.UpdateProfileStatus(r.Context(), db.UpdateProfileStatusParams{ID: id, MerchantID: merchantID, Status: status}); err != nil {
			apierr.Internal(w, err)
			return
		}
	}
	updated, err := h.queries.GetProfile(r.Context(), db.GetProfileParams{ID: id, MerchantID: merchantID})
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	common.WriteJSON(w, http.StatusOK, updated)
}

func isConstraint(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "unique")
}
