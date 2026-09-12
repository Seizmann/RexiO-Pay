// Package idempotency provides 24-hour idempotency key storage for POST /v1/sessions.
package idempotency

import (
	"context"
	"database/sql"
	"fmt"

	dbpkg "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	dbsqlc "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
)

// CachedResponse holds the stored response for an idempotent request.
type CachedResponse struct {
	Body       []byte
	StatusCode int
}

// Store saves an idempotency key with its response. Silently ignores conflicts
// (meaning the first response wins, which is correct behavior).
func Store(ctx context.Context, pool *dbpkg.Pool, key, merchantID, requestHash string, body []byte, statusCode int) error {
	q := dbsqlc.New(pool.SqlDB)
	return q.UpsertIdempotencyKey(ctx, dbsqlc.UpsertIdempotencyKeyParams{
		Key:          key,
		MerchantID:   merchantID,
		RequestHash:  requestHash,
		ResponseBody: body,
		StatusCode:   int32(statusCode),
	})
}

// Check looks up an existing idempotency key. Returns nil if not found or expired.
func Check(ctx context.Context, pool *dbpkg.Pool, key, merchantID string) (*CachedResponse, error) {
	q := dbsqlc.New(pool.SqlDB)
	row, err := q.GetIdempotencyKey(ctx, dbsqlc.GetIdempotencyKeyParams{
		Key:        key,
		MerchantID: merchantID,
	})
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("idempotency.Check: %w", err)
	}
	return &CachedResponse{
		Body:       row.ResponseBody,
		StatusCode: int(row.StatusCode),
	}, nil
}

// Cleanup deletes expired keys. Called periodically by a background goroutine.
func Cleanup(ctx context.Context, pool *dbpkg.Pool) error {
	q := dbsqlc.New(pool.SqlDB)
	return q.DeleteExpiredIdempotencyKeys(ctx)
}
