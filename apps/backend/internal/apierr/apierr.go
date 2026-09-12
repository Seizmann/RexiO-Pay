// Package apierr provides a consistent JSON error response shape and sentinel error codes.
package apierr

import (
	"encoding/json"
	"net/http"
)

// Error codes (sent in the "code" field of error responses).
const (
	CodePlanLimitExceeded    = "plan_limit_exceeded"
	CodeDomainNotWhitelisted = "domain_not_whitelisted"
	CodeIdempotencyConflict  = "idempotency_conflict"
	CodeDeviceDisabled       = "device_disabled"
	CodeUnauthorized         = "unauthorized"
	CodeForbidden            = "forbidden"
	CodeNotFound             = "not_found"
	CodeUserNotFound         = "user_not_found"
	CodeInvalidRequest       = "invalid_request"
	CodeInternalError        = "internal_error"
)

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Param   string `json:"param,omitempty"`
}

// Render writes an application/json error response.
func Render(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorBody{Error: errorDetail{Code: code, Message: message}})
}

// RenderParam writes an error response with an optional field name.
func RenderParam(w http.ResponseWriter, status int, code, message, param string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorBody{Error: errorDetail{Code: code, Message: message, Param: param}})
}

// NotFound writes a 404 not_found response.
func NotFound(w http.ResponseWriter) {
	Render(w, http.StatusNotFound, CodeNotFound, "resource not found")
}

// Unauthorized writes a 401 unauthorized response.
func Unauthorized(w http.ResponseWriter) {
	Render(w, http.StatusUnauthorized, CodeUnauthorized, "authentication required")
}

// Forbidden writes a 403 forbidden response.
func Forbidden(w http.ResponseWriter) {
	Render(w, http.StatusForbidden, CodeForbidden, "insufficient permissions")
}

// Internal writes a 500 internal_error response.
func Internal(w http.ResponseWriter, err error) {
	// Do not leak internal error details to callers.
	Render(w, http.StatusInternalServerError, CodeInternalError, "an internal error occurred")
	_ = err // caller should log separately
}
