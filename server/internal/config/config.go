// Package config loads all runtime configuration from environment
// variables with developer-friendly defaults so the central server can
// be started with `go run ./cmd/server` without any extra setup.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the resolved configuration for the central server.
type Config struct {
	HTTPAddr          string
	DatabaseURL       string
	RedisURL          string
	JWTSecret         string
	JWTTTL            time.Duration
	InternalToken     string
	GSHost            string
	GSPortMin         int
	GSPortMax         int
	GodotBin          string
	CentralURL        string
	MatchmakingTimeout time.Duration
	MigrationsPath    string
}

// Load reads configuration from environment variables, applying the
// defaults defined in the project README when values are missing.
func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:           getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:        getenv("DB_URL", "postgres://openfire:openfire@localhost:5432/openfire?sslmode=disable"),
		RedisURL:           getenv("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:          getenv("JWT_SECRET", "dev-secret-change-me"),
		JWTTTL:             7 * 24 * time.Hour,
		InternalToken:      getenv("INTERNAL_TOKEN", "dev-internal-token"),
		GSHost:             getenv("GS_HOST", "127.0.0.1"),
		GSPortMin:          getenvInt("GS_PORT_MIN", 27020),
		GSPortMax:          getenvInt("GS_PORT_MAX", 27099),
		GodotBin:           getenvGodotBin("godot"),
		CentralURL:         getenv("OPENFIRE_CENTRAL_URL", "http://127.0.0.1:8080"),
		MatchmakingTimeout: getenvDuration("MATCHMAKING_TIMEOUT", 120*time.Second),
		MigrationsPath:     getenv("MIGRATIONS_PATH", "migrations/001_init.sql"),
	}

	if cfg.GSPortMax <= cfg.GSPortMin {
		return nil, fmt.Errorf("GS_PORT_MAX (%d) must be greater than GS_PORT_MIN (%d)", cfg.GSPortMax, cfg.GSPortMin)
	}
	if strings.TrimSpace(cfg.JWTSecret) == "" {
		return nil, fmt.Errorf("JWT_SECRET must not be empty")
	}
	if strings.TrimSpace(cfg.InternalToken) == "" {
		return nil, fmt.Errorf("INTERNAL_TOKEN must not be empty")
	}
	return cfg, nil
}

// getenvGodotBin resolves the Godot binary path. OPENFIRE_GODOT_BIN is
// preferred (it is the variable the GS launcher documents), with a
// fallback to GODOT_BIN and finally the default.
func getenvGodotBin(def string) string {
	if v := os.Getenv("OPENFIRE_GODOT_BIN"); v != "" {
		return v
	}
	if v := os.Getenv("GODOT_BIN"); v != "" {
		return v
	}
	return def
}

func getenv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getenvDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
