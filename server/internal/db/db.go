// Package db wraps the PostgreSQL connection pool (via lib/pq) and the
// Redis client used for session/presence state.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq" // postgres driver registration
	"github.com/redis/go-redis/v9"
)

// DB holds the Postgres pool and Redis client.
type DB struct {
	PG    *sql.DB
	Redis *redis.Client
}

// New opens the Postgres pool and Redis client, pinging Postgres to
// fail fast on misconfiguration.
func New(databaseURL, redisURL string) (*DB, error) {
	pg, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	pg.SetMaxOpenConns(20)
	pg.SetMaxIdleConns(5)
	pg.SetConnMaxLifetime(time.Hour)
	if err := pg.Ping(); err != nil {
		pg.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		pg.Close()
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	rdb := redis.NewClient(opt)

	return &DB{PG: pg, Redis: rdb}, nil
}

// Migrate reads the SQL file at path and executes it. Statements are
// idempotent (CREATE TABLE/INDEX IF NOT EXISTS) so it is safe to run on
// every boot.
func (d *DB) Migrate(path string) error {
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", path, err)
	}
	if _, err := d.PG.Exec(string(sqlBytes)); err != nil {
		return fmt.Errorf("apply migration: %w", err)
	}
	return nil
}

// Close releases the Postgres pool and Redis client.
func (d *DB) Close() {
	if d.PG != nil {
		d.PG.Close()
	}
	if d.Redis != nil {
		_ = d.Redis.Close()
	}
}
