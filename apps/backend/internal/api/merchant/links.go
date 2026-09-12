package merchant

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
	dbsqlc "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
	"github.com/Seizmann/RexiO-Pay/backend/internal/idgen"
)

type createPaymentLinkRequest struct {
	PaymentProfileID string     `json:"payment_profile_id"`
	Title            string     `json:"title"`
	Amount           *int64     `json:"amount"`
	Reusable         bool       `json:"reusable"`
	ExpiresAt        *time.Time `json:"expires_at"`
}

type paymentLinkResponse struct {
	ID               string    `json:"id"`
	PaymentProfileID string    `json:"payment_profile_id"`
	Title            string    `json:"title"`
	Amount           any       `json:"amount"`
	Reusable         bool      `json:"reusable"`
	Active           bool      `json:"active"`
	ExpiresAt        any       `json:"expires_at,omitempty"`
	URL              string    `json:"url"`
	CreatedAt        time.Time `json:"created_at"`
}

func makePaymentLinkResponse(link dbsqlc.PaymentLink, appURL string) paymentLinkResponse {
	return paymentLinkResponse{
		ID:               link.ID,
		PaymentProfileID: link.PaymentProfileID,
		Title:            link.Title,
		Amount:           nullableInt(link.Amount),
		Reusable:         link.Reusable,
		Active:           link.Active,
		ExpiresAt:        nullableTime(link.ExpiresAt),
		URL:              checkoutURL(appURL, "link/"+link.ID),
		CreatedAt:        link.CreatedAt,
	}
}

func (h *Handler) createPaymentLink(w http.ResponseWriter, r *http.Request) {
	if !h.ensurePool(w) {
		return
	}
	merchant, ok := merchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	var req createPaymentLinkRequest
	if _, err := decodeBody(r, &req); err != nil {
		invalid(w, "must be valid JSON", "body")
		return
	}
	req.PaymentProfileID = strings.TrimSpace(req.PaymentProfileID)
	if req.PaymentProfileID == "" {
		invalid(w, "is required", "payment_profile_id")
		return
	}
	if len(req.Title) > 200 {
		invalid(w, "must be at most 200 characters", "title")
		return
	}
	if req.Amount != nil && *req.Amount <= 0 {
		invalid(w, "must be greater than zero", "amount")
		return
	}
	if req.ExpiresAt != nil && !req.ExpiresAt.After(time.Now()) {
		invalid(w, "must be in the future", "expires_at")
		return
	}
	q := h.queries()
	profile, err := q.GetProfile(r.Context(), dbsqlc.GetProfileParams{ID: req.PaymentProfileID, MerchantID: merchant})
	if err != nil {
		notFoundOrInternal(w, err)
		return
	}
	if profile.Status != "active" {
		apierr.Render(w, http.StatusConflict, apierr.CodeInvalidRequest, "payment profile is not active")
		return
	}
	amount := sql.NullInt64{}
	if req.Amount != nil {
		amount = sql.NullInt64{Int64: *req.Amount, Valid: true}
	}
	expiresAt := sql.NullTime{}
	if req.ExpiresAt != nil {
		expiresAt = sql.NullTime{Time: req.ExpiresAt.UTC(), Valid: true}
	}
	link, err := q.CreatePaymentLink(r.Context(), dbsqlc.CreatePaymentLinkParams{
		ID:               idgen.New(idgen.PrefixLink),
		MerchantID:       merchant,
		PaymentProfileID: req.PaymentProfileID,
		Title:            strings.TrimSpace(req.Title),
		Amount:           amount,
		Reusable:         req.Reusable,
		ExpiresAt:        expiresAt,
	})
	if err != nil {
		logInternal("create payment link", err)
		apierr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, makePaymentLinkResponse(link, h.AppURL))
}

func (h *Handler) listPaymentLinks(w http.ResponseWriter, r *http.Request) {
	if !h.ensurePool(w) {
		return
	}
	merchant, ok := merchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	links, err := h.queries().ListPaymentLinks(r.Context(), merchant)
	if err != nil {
		logInternal("list payment links", err)
		apierr.Internal(w, err)
		return
	}
	responses := make([]paymentLinkResponse, 0, len(links))
	for _, link := range links {
		responses = append(responses, makePaymentLinkResponse(link, h.AppURL))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": responses})
}

func (h *Handler) getPaymentLink(w http.ResponseWriter, r *http.Request) {
	if !h.ensurePool(w) {
		return
	}
	merchant, ok := merchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	link, err := h.queries().GetPaymentLink(r.Context(), dbsqlc.GetPaymentLinkParams{ID: chi.URLParam(r, "id"), MerchantID: merchant})
	if err != nil {
		notFoundOrInternal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, makePaymentLinkResponse(link, h.AppURL))
}
