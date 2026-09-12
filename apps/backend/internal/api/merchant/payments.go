package merchant

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
	dbsqlc "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
)

type paymentResponse struct {
	ID                 string    `json:"id"`
	SessionID          string    `json:"session_id"`
	SMSID              any       `json:"sms_id,omitempty"`
	Provider           string    `json:"provider"`
	AccountType        string    `json:"account_type"`
	SenderNumber       string    `json:"sender_number"`
	TrxID              string    `json:"trx_id"`
	Amount             int64     `json:"amount"`
	BalanceAfter       any       `json:"balance_after,omitempty"`
	VerifiedAt         time.Time `json:"verified_at"`
	VerificationSource string    `json:"verification_source"`
	IsRefundFlagged    bool      `json:"is_refund_flagged"`
	RefundNote         any       `json:"refund_note,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

func makePaymentResponse(payment dbsqlc.Payment) paymentResponse {
	return paymentResponse{
		ID:                 payment.ID,
		SessionID:          payment.SessionID,
		SMSID:              nullableString(payment.SmsID),
		Provider:           payment.Provider,
		AccountType:        payment.AccountType,
		SenderNumber:       payment.SenderNumber,
		TrxID:              payment.TrxID,
		Amount:             payment.Amount,
		BalanceAfter:       nullableInt(payment.BalanceAfter),
		VerifiedAt:         payment.VerifiedAt,
		VerificationSource: payment.VerificationSource,
		IsRefundFlagged:    payment.IsRefundFlagged,
		RefundNote:         nullableString(payment.RefundNote),
		CreatedAt:          payment.CreatedAt,
	}
}

func (h *Handler) listPayments(w http.ResponseWriter, r *http.Request) {
	if !h.ensurePool(w) {
		return
	}
	merchant, ok := merchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	limit, offset, ok := parsePage(r)
	if !ok {
		invalid(w, "limit must be 1-100 and offset must be non-negative", "pagination")
		return
	}
	from, ok := parseTimeParam(strings.TrimSpace(r.URL.Query().Get("from")))
	if !ok {
		invalid(w, "must be an RFC3339 timestamp", "from")
		return
	}
	to, ok := parseTimeParam(strings.TrimSpace(r.URL.Query().Get("to")))
	if !ok {
		invalid(w, "must be an RFC3339 timestamp", "to")
		return
	}
	if !from.IsZero() && !to.IsZero() && from.After(to) {
		invalid(w, "must be before to", "from")
		return
	}
	provider := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("provider")))
	if provider != "" && provider != "bkash" && provider != "nagad" {
		invalid(w, "must be bkash or nagad", "provider")
		return
	}
	q := h.queries()
	items, err := q.ListPayments(r.Context(), dbsqlc.ListPaymentsParams{
		MerchantID: merchant,
		Column2:    from,
		Column3:    to,
		Column4:    provider,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		logInternal("list payments", err)
		apierr.Internal(w, err)
		return
	}
	count, err := q.CountPayments(r.Context(), dbsqlc.CountPaymentsParams{MerchantID: merchant, Column2: from, Column3: to, Column4: provider})
	if err != nil {
		logInternal("count payments", err)
		apierr.Internal(w, err)
		return
	}
	responses := make([]paymentResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, makePaymentResponse(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": responses, "has_more": int64(offset)+int64(len(items)) < count})
}

func (h *Handler) getPayment(w http.ResponseWriter, r *http.Request) {
	if !h.ensurePool(w) {
		return
	}
	merchant, ok := merchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	payment, err := h.queries().GetPayment(r.Context(), dbsqlc.GetPaymentParams{ID: chi.URLParam(r, "id"), MerchantID: merchant})
	if err != nil {
		if err == sql.ErrNoRows {
			apierr.NotFound(w)
			return
		}
		logInternal("get payment", err)
		apierr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, makePaymentResponse(payment))
}
