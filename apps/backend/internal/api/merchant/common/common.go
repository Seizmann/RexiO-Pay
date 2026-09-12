package common

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
	"github.com/Seizmann/RexiO-Pay/backend/internal/middleware"
)

// MerchantID returns the merchant authenticated by APIKeyAuth.
func MerchantID(r *http.Request) (string, bool) {
	mc := middleware.GetMerchant(r.Context())
	return func() (string, bool) {
		if mc == nil || strings.TrimSpace(mc.MerchantID) == "" {
			return "", false
		}
		return mc.MerchantID, true
	}()
}

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		apierr.Render(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "invalid JSON request body")
		return false
	}
	return true
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func PathID(r *http.Request, name string) string {
	return strings.TrimSpace(chi.URLParam(r, name))
}
