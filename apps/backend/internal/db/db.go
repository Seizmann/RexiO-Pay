package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

// Pool wraps both the raw pgxpool.Pool (for named types, COPY, etc.)
// and a *sql.DB adapter (for sqlc-generated queries which use database/sql DBTX).
type Pool struct {
	Raw   *pgxpool.Pool
	SqlDB *sql.DB
}

// Open creates a pgxpool and wraps it in the dual Pool type.
// Callers should defer pool.Close().
func Open(ctx context.Context, dsn string) (*Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("db.Open parse config: %w", err)
	}
	raw, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db.Open new pool: %w", err)
	}
	if err := raw.Ping(ctx); err != nil {
		return nil, fmt.Errorf("db.Open ping: %w", err)
	}
	// Wrap for sqlc database/sql compatibility
	sqlDB := stdlib.OpenDBFromPool(raw)
	return &Pool{Raw: raw, SqlDB: sqlDB}, nil
}

// Close closes both the sql.DB adapter and the underlying pgxpool.
func (p *Pool) Close() {
	_ = p.SqlDB.Close()
	p.Raw.Close()
}

// QueryRow is a convenience pass-through for raw one-off queries.
func (p *Pool) QueryRow(ctx context.Context, sql string, args ...any) interface{ Scan(...any) error } {
	return p.Raw.QueryRow(ctx, sql, args...)
}
