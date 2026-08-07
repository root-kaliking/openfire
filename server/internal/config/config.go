// Package config centralizes all runtime configuration sourced from
// environment variables with developer-friendly defaults so the server
// can be started with `go run .` without any extra setup.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config is the resolved configuration for the central server.
type Config struct {
	HTTPAddr string

	DatabaseURL    string
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
	RunMigrations  bool

	JWTSecret      string
	JWTTTLMinutes  int

	InternalToken  string

	GodotBinaryPath string
	GodotProjectPath string
	GSListenHost    string
	GSPortMin       int
	GSPortMax       int

	MatchIntervalMS int
	DMMinPlayers    int
	DMMaxPlayers    int

	LobbyBroadcastMS int
	GSHeartbeatTimeoutSec int
}

// Load reads configuration from environment variables, applying safe
// development defaults when values are missing.
func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:           getenv("OPENFIRE_HTTP_ADDR", ":8080"),
		DatabaseURL:        getenv("OPENFIRE_DATABASE_URL", "postgres://openfire:openfire@localhost:5432/openfire?sslmode=disable"),
		RedisAddr:          getenv("OPENFIRE_REDIS_ADDR", "localhost:6379"),
		RedisPassword:      getenv("OPENFIRE_REDIS_PASSWORD", ""),
		RedisDB:            getenvInt("OPENFIRE_REDIS_DB", 0),
		RunMigrations:      getenvBool("OPENFIRE_RUN_MIGRATIONS", true),
		JWTSecret:          getenv("OPENFIRE_JWT_SECRET", "dev-jwt-secret-change-me"),
		JWTTTLMinutes:      getenvInt("OPENFIRE_JWT_TTL_MINUTES", 1440),
		InternalToken:      getenv("OPENFIRE_INTERNAL_TOKEN", "dev-internal-token"),
		GodotBinaryPath:    getenv("OPENFIRE_GODOT_BINARY", "godot"),
		GodotProjectPath:   getenv("OPENFIRE_GODOT_PROJECT_PATH", "/workspace"),
		GSListenHost:       getenv("OPENFIRE_GS_HOST", "127.0.0.1"),
		GSPortMin:          getenvInt("OPENFIRE_GS_PORT_MIN", 27100),
		GSPortMax:          getenvInt("OPENFIRE_GS_PORT_MAX", 27200),
		MatchIntervalMS:    getenvInt("OPENFIRE_MATCH_INTERVAL_MS", 1000),
		DMMinPlayers:       getenvInt("OPENFIRE_DM_MIN_PLAYERS", 4),
		DMMaxPlayers:       getenvInt("OPENFIRE_DM_MAX_PLAYERS", 8),
		LobbyBroadcastMS:   getenvInt("OPENFIRE_LOBBY_BROADCAST_MS", 2000),
		GSHeartbeatTimeoutSec: getenvInt("OPENFIRE_GS_HEARTBEAT_TIMEOUT_SEC", 15),
	}

	if cfg.GSPortMax < cfg.GSPortMin {
		return nil, fmt.Errorf("OPENFIRE_GS_PORT_MAX (%d) < PORT_MIN (%d)", cfg.GSPortMax, cfg.GSPortMin)
	}
	if cfg.DMMinPlayers < 2 {
		return nil, fmt.Errorf("OPENFIRE_DM_MIN_PLAYERS must be >= 2")
	}
	if cfg.DMMaxPlayers < cfg.DMMinPlayers {
		return nil, fmt.Errorf("OPENFIRE_DM_MAX_PLAYERS (%d) < MIN (%d)", cfg.DMMaxPlayers, cfg.DMMinPlayers)
	}
	if strings.TrimSpace(cfg.JWTSecret) == "" {
		return nil, fmt.Errorf("OPENFIRE_JWT_SECRET must not be empty")
	}
	if strings.TrimSpace(cfg.InternalToken) == "" {
		return nil, fmt.Errorf("OPENFIRE_INTERNAL_TOKEN must not be empty")
	}
	return cfg, nil
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

func getenvBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
