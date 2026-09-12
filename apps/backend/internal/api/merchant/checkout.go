package merchant

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
	dbsqlc "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
	"github.com/Seizmann/RexiO-Pay/backend/internal/phone"
	"github.com/go-chi/chi/v5"
)

type checkoutResponse struct {
	ID                   string    `json:"id"`
	Amount               int64     `json:"amount"`
	Currency             string    `json:"currency"`
	Status               string    `json:"status"`
	NeedsReview          bool      `json:"needs_review"`
	ReviewReason         any       `json:"review_reason,omitempty"`
	ExpiresAt            time.Time `json:"expires_at"`
	CreatedAt            time.Time `json:"created_at"`
	SenderNumberClaim    any       `json:"sender_number_claim,omitempty"`
	TrxIDClaim           any       `json:"trx_id_claim,omitempty"`
	ConfirmedAt          any       `json:"confirmed_at,omitempty"`
	MerchantName         string    `json:"merchant_name"`
	MerchantLogoURL      string    `json:"merchant_logo_url,omitempty"`
	MerchantBrandColor   string    `json:"merchant_brand_color,omitempty"`
	MerchantSupportEmail string    `json:"merchant_support_email,omitempty"`
	MerchantSupportPhone string    `json:"merchant_support_phone,omitempty"`
	PaymentProfileID     string    `json:"payment_profile_id"`
}

func makeCheckoutResponse(row dbsqlc.GetSessionForPublicCheckoutRow) checkoutResponse {
	return checkoutResponse{
		ID:                   row.ID,
		Amount:               row.Amount,
		Currency:             row.Currency,
		Status:               row.Status,
		NeedsReview:          row.NeedsReview,
		ReviewReason:         nullableString(row.ReviewReason),
		ExpiresAt:            row.ExpiresAt,
		CreatedAt:            row.CreatedAt,
		SenderNumberClaim:    nullableString(row.SenderNumberClaim),
		TrxIDClaim:           nullableString(row.TrxIDClaim),
		ConfirmedAt:          nullableTime(row.ConfirmedAt),
		MerchantName:         row.MerchantName,
		MerchantLogoURL:      row.MerchantLogoUrl,
		MerchantBrandColor:   row.MerchantBrandColor,
		MerchantSupportEmail: row.MerchantSupportEmail,
		MerchantSupportPhone: row.MerchantSupportPhone,
		PaymentProfileID:     row.PaymentProfileID,
	}
}

type claimRequest struct {
	SenderNumber string `json:"sender_number"`
	TrxID        string `json:"trx_id"`
	Confirmed    bool   `json:"confirmed"`
}

func (h *Handler) getCheckout(w http.ResponseWriter, r *http.Request) {
	if !h.ensurePool(w) {
		return
	}
	row, err := h.queries().GetSessionForPublicCheckout(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			apierr.NotFound(w)
			return
		}
		logInternal("get checkout", err)
		apierr.Internal(w, err)
		return
	}
	if row.Status == "pending" && time.Now().After(row.ExpiresAt) {
		if err := h.queries().UpdateSessionStatus(r.Context(), dbsqlc.UpdateSessionStatusParams{ID: row.ID, Status: "expired"}); err != nil {
			logInternal("expire checkout", err)
		} else {
			row.Status = "expired"
		}
	}
	writeJSON(w, http.StatusOK, makeCheckoutResponse(row))
}

func (h *Handler) claimCheckout(w http.ResponseWriter, r *http.Request) {
	if !h.ensurePool(w) {
		return
	}
	var req claimRequest
	if _, err := decodeBody(r, &req); err != nil {
		invalid(w, "must be valid JSON", "body")
		return
	}
	sender := phone.Canonicalize(req.SenderNumber)
	if sender == "" {
		invalid(w, "must be a valid Bangladeshi mobile number", "sender_number")
		return
	}
	req.TrxID = strings.TrimSpace(req.TrxID)
	if len(req.TrxID) > 100 {
		invalid(w, "must be at most 100 characters", "trx_id")
		return
	}
	id := chi.URLParam(r, "id")
	q := h.queries()
	row, err := q.GetSessionForPublicCheckout(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			apierr.NotFound(w)
			return
		}
		logInternal("get checkout for claim", err)
		apierr.Internal(w, err)
		return
	}
	if row.Status != "pending" {
		apierr.Render(w, http.StatusConflict, apierr.CodeInvalidRequest, "checkout session is no longer pending")
		return
	}
	if time.Now().After(row.ExpiresAt) {
		_ = q.UpdateSessionStatus(r.Context(), dbsqlc.UpdateSessionStatusParams{ID: id, Status: "expired"})
		apierr.Render(w, http.StatusConflict, apierr.CodeInvalidRequest, "checkout session has expired")
		return
	}
	if err := q.UpdateSessionClaim(r.Context(), dbsqlc.UpdateSessionClaimParams{
		ID:                id,
		SenderNumberClaim: sql.NullString{String: sender, Valid: true},
		TrxIDClaim:        sql.NullString{String: req.TrxID, Valid: req.TrxID != ""},
	}); err != nil {
		logInternal("claim checkout", err)
		apierr.Internal(w, err)
		return
	}
	if req.Confirmed {
		if err := q.UpdateSessionConfirmed(r.Context(), id); err != nil {
			logInternal("confirm checkout", err)
			apierr.Internal(w, err)
			return
		}
	}
	updated, err := q.GetSessionForPublicCheckout(r.Context(), id)
	if err != nil {
		logInternal("reload checkout", err)
		apierr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, makeCheckoutResponse(updated))
}
