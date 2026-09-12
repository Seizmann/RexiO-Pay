package merchant

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
	dbsqlc "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
	"github.com/Seizmann/RexiO-Pay/backend/internal/idempotency"
	"github.com/Seizmann/RexiO-Pay/backend/internal/idgen"
	"github.com/go-chi/chi/v5"
)

type createSessionRequest struct {
	Amount            int64           `json:"amount"`
	PaymentProfileIDs []string        `json:"payment_profile_ids"`
	PaymentProfileID  string          `json:"payment_profile_id"`
	Customer          json.RawMessage `json:"customer"`
	ReturnURL         string          `json:"return_url"`
	CancelURL         string          `json:"cancel_url"`
	Metadata          json.RawMessage `json:"metadata"`
}

type sessionResponse struct {
	ID               string          `json:"id"`
	Amount           int64           `json:"amount"`
	Currency         string          `json:"currency"`
	Status           string          `json:"status"`
	Source           string          `json:"source"`
	PaymentProfileID string          `json:"payment_profile_id"`
	Customer         json.RawMessage `json:"customer"`
	Metadata         json.RawMessage `json:"metadata"`
	NeedsReview      bool            `json:"needs_review"`
	ReviewReason     any             `json:"review_reason,omitempty"`
	ExpiresAt        time.Time       `json:"expires_at"`
	ReturnURL        string          `json:"return_url,omitempty"`
	CancelURL        string          `json:"cancel_url,omitempty"`
	PaymentLinkID    any             `json:"payment_link_id,omitempty"`
	CheckoutURL      string          `json:"checkout_url"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

func makeSessionResponse(session dbsqlc.CheckoutSession, appURL string) sessionResponse {
	customer := session.Customer
	if len(customer) == 0 {
		customer = json.RawMessage(`{}`)
	}
	metadata := session.Metadata
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}
	return sessionResponse{
		ID:               session.ID,
		Amount:           session.Amount,
		Currency:         session.Currency,
		Status:           session.Status,
		Source:           session.Source,
		PaymentProfileID: session.PaymentProfileID,
		Customer:         customer,
		Metadata:         metadata,
		NeedsReview:      session.NeedsReview,
		ReviewReason:     nullableString(session.ReviewReason),
		ExpiresAt:        session.ExpiresAt,
		ReturnURL:        session.ReturnUrl,
		CancelURL:        session.CancelUrl,
		PaymentLinkID:    nullableString(session.PaymentLinkID),
		CheckoutURL:      checkoutURL(appURL, session.ID),
		CreatedAt:        session.CreatedAt,
		UpdatedAt:        session.UpdatedAt,
	}
}

func (h *Handler) createSession(w http.ResponseWriter, r *http.Request) {
	if !h.ensurePool(w) {
		return
	}
	merchant, ok := merchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	key, keyOK := idempotencyKey(r)
	if raw := strings.TrimSpace(r.Header.Get("Idempotency-Key")); raw != "" && !keyOK {
		invalid(w, "must be at most 255 characters", "Idempotency-Key")
		return
	}
	var req createSessionRequest
	body, err := decodeBody(r, &req)
	if err != nil {
		invalid(w, "must be valid JSON", "body")
		return
	}
	if keyOK {
		cached, checkErr := idempotency.Check(r.Context(), h.Pool, key, merchant)
		if checkErr != nil {
			logInternal("idempotency check", checkErr)
			apierr.Internal(w, checkErr)
			return
		}
		if cached != nil {
			// Check carries the original request hash after the small helper
			// extension in internal/idempotency. Old rows without a hash are
			// still safely replayed.
			if cached.RequestHash != "" && cached.RequestHash != requestHash(body) {
				apierr.Render(w, http.StatusConflict, apierr.CodeIdempotencyConflict, "idempotency key was already used with a different request")
				return
			}
			writeCached(w, cached)
			return
		}
	}
	if req.Amount <= 0 {
		invalid(w, "must be greater than zero", "amount")
		return
	}
	if req.Amount > 1000000000 {
		invalid(w, "is too large", "amount")
		return
	}
	profileID := strings.TrimSpace(req.PaymentProfileID)
	if profileID == "" && len(req.PaymentProfileIDs) > 0 {
		profileID = strings.TrimSpace(req.PaymentProfileIDs[0])
	}
	if profileID == "" {
		invalid(w, "at least one payment profile is required", "payment_profile_ids")
		return
	}
	if len(req.PaymentProfileIDs) > 1 {
		invalid(w, "only one payment profile is supported per session", "payment_profile_ids")
		return
	}
	if len(req.Customer) == 0 {
		req.Customer = json.RawMessage(`{}`)
	}
	if !json.Valid(req.Customer) || string(req.Customer) == "null" {
		invalid(w, "must be a JSON object", "customer")
		return
	}
	if len(req.Metadata) == 0 {
		req.Metadata = json.RawMessage(`{}`)
	}
	if !json.Valid(req.Metadata) || string(req.Metadata) == "null" {
		invalid(w, "must be a JSON object", "metadata")
		return
	}
	if !h.validateRequestDomains(r, merchant, req.ReturnURL, req.CancelURL, w) {
		return
	}

	q := h.queries()
	profile, err := q.GetProfile(r.Context(), dbsqlc.GetProfileParams{ID: profileID, MerchantID: merchant})
	if err != nil {
		notFoundOrInternal(w, err)
		return
	}
	if profile.Status != "active" {
		apierr.Render(w, http.StatusConflict, apierr.CodeInvalidRequest, "payment profile is not active")
		return
	}
	if h.Billing == nil {
		// A nil limiter is useful in isolated tests, but production handlers
		// always get one from NewHandler.
	} else if !h.Billing.Check(r.Context(), w, merchant, "sessions_per_month") {
		return
	}

	ttl := h.SessionTTL
	if ttl <= 0 {
		ttl = defaultSessionTTL
	}
	expiresAt := time.Now().UTC().Add(ttl)
	session, err := q.CreateSession(r.Context(), dbsqlc.CreateSessionParams{
		ID:               idgen.New(idgen.PrefixSession),
		MerchantID:       merchant,
		PaymentProfileID: profile.ID,
		Source:           "api",
		Amount:           req.Amount,
		Currency:         "BDT",
		Customer:         req.Customer,
		Metadata:         req.Metadata,
		ReturnUrl:        req.ReturnURL,
		CancelUrl:        req.CancelURL,
		PaymentLinkID:    sql.NullString{},
		ExpiresAt:        expiresAt,
	})
	if err != nil {
		logInternal("create session", err)
		apierr.Internal(w, err)
		return
	}
	if err := q.IncrMerchantSessionCount(r.Context(), merchant); err != nil {
		logInternal("increment session count", err)
		apierr.Internal(w, err)
		return
	}

	response := makeSessionResponse(session, h.AppURL)
	responseBody, err := json.Marshal(response)
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	if keyOK {
		if err := idempotency.Store(r.Context(), h.Pool, key, merchant, requestHash(body), responseBody, http.StatusCreated); err != nil {
			logInternal("idempotency store", err)
			apierr.Internal(w, err)
			return
		}
	}
	writeJSON(w, http.StatusCreated, response)
}

func (h *Handler) getSession(w http.ResponseWriter, r *http.Request) {
	if !h.ensurePool(w) {
		return
	}
	merchant, ok := merchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	session, err := h.queries().GetSession(r.Context(), dbsqlc.GetSessionParams{ID: chi.URLParam(r, "id"), MerchantID: merchant})
	if err != nil {
		notFoundOrInternal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, makeSessionResponse(session, h.AppURL))
}

func (h *Handler) cancelSession(w http.ResponseWriter, r *http.Request) {
	if !h.ensurePool(w) {
		return
	}
	merchant, ok := merchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	q := h.queries()
	id := chi.URLParam(r, "id")
	session, err := q.GetSession(r.Context(), dbsqlc.GetSessionParams{ID: id, MerchantID: merchant})
	if err != nil {
		notFoundOrInternal(w, err)
		return
	}
	if session.Status != "pending" {
		apierr.Render(w, http.StatusConflict, apierr.CodeInvalidRequest, "only pending sessions can be canceled")
		return
	}
	if err := q.UpdateSessionStatus(r.Context(), dbsqlc.UpdateSessionStatusParams{ID: id, Status: "canceled"}); err != nil {
		logInternal("cancel session", err)
		apierr.Internal(w, err)
		return
	}
	session.Status = "canceled"
	writeJSON(w, http.StatusOK, makeSessionResponse(session, h.AppURL))
}
