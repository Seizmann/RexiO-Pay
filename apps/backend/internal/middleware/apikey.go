// Package middleware provides HTTP middleware for the RexiO Pay backend.
package middleware

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
	dbpkg "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	dbsqlc "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
)

type ctxKeyMerchant struct{}

// MerchantCtx holds merchant info attached to context by APIKeyAuth.
type MerchantCtx struct {
	MerchantID string
	KeyID      string
}

// GetMerchant retrieves merchant context from the request context.
func GetMerchant(ctx context.Context) *MerchantCtx {
	v, _ := ctx.Value(ctxKeyMerchant{}).(*MerchantCtx)
	return v
}

// APIKeyAuth authenticates requests using a Bearer API key (rk_...).
// On success, attaches MerchantCtx to the request context.
// Also updates last_used_at asynchronously.
func APIKeyAuth(pool *dbpkg.Pool) func(http.Handler) http.Handler {
	q := dbsqlc.New(pool.SqlDB)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				apierr.Unauthorized(w)
				return
			}
			rawKey := strings.TrimPrefix(auth, "Bearer ")
			if len(rawKey) < 10 {
				apierr.Unauthorized(w)
				return
			}

			sum := sha256.Sum256([]byte(rawKey))
			hash := hex.EncodeToString(sum[:])

			apiKey, err := q.GetAPIKeyByHash(r.Context(), hash)
			if err != nil {
				apierr.Unauthorized(w)
				return
			}

			go func() {
				if err := q.UpdateAPIKeyLastUsed(context.Background(), apiKey.ID); err != nil {
					slog.Error("apikey: update last_used_at", "err", err)
				}
			}()

			ctx := context.WithValue(r.Context(), ctxKeyMerchant{}, &MerchantCtx{
				MerchantID: apiKey.MerchantID,
				KeyID:      apiKey.ID,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdminOnly rejects requests where the authenticated merchant's owner is not a platform admin.
// Must be composed after APIKeyAuth.
func AdminOnly(pool *dbpkg.Pool) func(http.Handler) http.Handler {
	q := dbsqlc.New(pool.SqlDB)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mc := GetMerchant(r.Context())
			if mc == nil {
				apierr.Unauthorized(w)
				return
			}
			merchant, err := q.GetMerchant(r.Context(), mc.MerchantID)
			if err != nil {
				apierr.Unauthorized(w)
				return
			}
			// Check if the merchant owner has platform admin flag
			var isAdmin bool
			row := pool.SqlDB.QueryRowContext(r.Context(),
				"SELECT is_platform_admin FROM public.users WHERE id = $1", merchant.OwnerUserID)
			if err := row.Scan(&isAdmin); err != nil {
				if err == sql.ErrNoRows {
					apierr.Forbidden(w)
					return
				}
				apierr.Internal(w, err)
				return
			}
			if !isAdmin {
				apierr.Forbidden(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
