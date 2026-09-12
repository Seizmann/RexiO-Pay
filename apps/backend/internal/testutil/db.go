// Package testutil provides test helpers shared across integration tests.
package testutil

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestDB returns a pgxpool connected to TEST_DATABASE_URL.
// The test is skipped if TEST_DATABASE_URL is not set (runs in -short mode).
// Each call uses a fresh connection; callers should not share pools across subtests.
func TestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("testutil.TestDB: connect: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("testutil.TestDB: ping: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
