package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"
)

// ctxKeyRawBody stores raw request body bytes in context for HMAC verification.
type ctxKeyRawBody struct{}

// ReadBody buffers the raw request body into context so DeviceHMAC can read it
// after chi/middleware has already consumed it.
func ReadBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body []byte
		if r.Body != nil {
			body, _ = io.ReadAll(r.Body)
			r.Body.Close()
		}
		// Restore body for downstream handlers, attach bytes to context
		r.Body = io.NopCloser(bytes.NewReader(body))
		ctx := context.WithValue(r.Context(), ctxKeyRawBody{}, body)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
