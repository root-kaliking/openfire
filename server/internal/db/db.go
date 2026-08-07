// Package db wraps the pgx connection pool and runs the embedded
// schema.sql on startup. All statements in schema.sql are idempotent
// (CREATE TABLE/INDEX IF NOT EXISTS) so running it on every boot is safe.
package db

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schemaSQL string

// DB holds the Postgres connection pool.
type DB struct {
	Pool *pgxpool.Pool
}

// New creates a connection pool against the given DATABASE_URL and pings it.
func New(ctx context.Context, databaseURL string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse db url: %w", err)
	}
	cfg.MaxConns = 20
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 15 * time.Minute

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(pingCtx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return &DB{Pool: pool}, nil
}

// Migrate executes schema.sql (idempotent). Safe to call on every boot.
func (d *DB) Migrate(ctx context.Context) error {
	if _, err := d.Pool.Exec(ctx, schemaSQL); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}

// Close releases the pool resources.
func (d *DB) Close() {
	if d.Pool != nil {
		d.Pool.Close()
	}
}
