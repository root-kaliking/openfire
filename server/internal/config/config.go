// Package config loads all central-server settings from environment variables.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for the central server.
type Config struct {
	DatabaseURL            string
	JWTSecret              string
	JWTTTL                 time.Duration
	ListenAddr             string
	InternalToken          string
	GameserverBinary       string
	GameserverPortMin      int
	GameserverPortMax      int
	GameserverMaxPlayers   int
	GameserverStartTimeout time.Duration
	GameserverMaxDuration  time.Duration
	PublicGSIP             string
	CentralURL             string
}

// Load reads configuration from environment variables, applying MVP defaults.
func Load() Config {
	c := Config{
		DatabaseURL:            getenv("DATABASE_URL", "postgres://openfire:openfire@localhost:5432/openfire?sslmode=disable"),
		JWTSecret:              getenv("JWT_SECRET", "change-me"),
		ListenAddr:             getenv("LISTEN_ADDR", ":8080"),
		InternalToken:          getenv("INTERNAL_TOKEN", "dev-internal-token-xxx"),
		GameserverBinary:       getenv("GAMESERVER_BINARY", "/workspace/build/openfire.x86_64"),
		GameserverPortMin:      getenvInt("GAMESERVER_PORT_MIN", 27015),
		GameserverPortMax:      getenvInt("GAMESERVER_PORT_MAX", 27040),
		GameserverMaxPlayers:   getenvInt("GAMESERVER_MAX_PLAYERS", 8),
		GameserverStartTimeout: time.Duration(getenvInt("GAMESERVER_START_TIMEOUT", 30)) * time.Second,
		GameserverMaxDuration:  time.Duration(getenvInt("GAMESERVER_MAX_DURATION", 30)) * time.Minute,
		PublicGSIP:             getenv("PUBLIC_GS_IP", "127.0.0.1"),
		CentralURL:             getenv("CENTRAL_URL", "http://127.0.0.1:8080"),
	}
	c.JWTTTL = time.Duration(getenvInt("JWT_TTL_HOURS", 168)) * time.Hour
	return c
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
