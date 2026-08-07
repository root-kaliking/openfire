// Command server is the single entry point for the OpenFire central
// server: HTTP + WebSocket API, matchmaking, and Godot dedicated-server
// process management.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"

	"openfire-server/internal/config"
	"openfire-server/internal/db"
	"openfire-server/internal/gs"
	"openfire-server/internal/lobby"
	"openfire-server/internal/matchmaking"
	"openfire-server/handlers"
	"openfire-server/middleware"
)

func main() {
	// Load .env if present; missing file is fine (e.g. production).
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	database, err := db.New(cfg.DatabaseURL, cfg.RedisURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer database.Close()

	if err := database.Migrate(cfg.MigrationsPath); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	// Hub and queue have a circular dependency (the hub forwards inbound
	// messages to the queue; the queue pushes messages through the hub),
	// so construct the hub first with a nil handler and wire it after.
	hub := lobby.NewHub(nil)
	gsMgr := gs.New(database.PG, cfg.GodotBin, "/workspace", cfg.CentralURL, cfg.InternalToken, cfg.GSHost, cfg.GSPortMin, cfg.GSPortMax, "logs")
	queue := matchmaking.NewQueue(hub, database.PG, gsMgr, cfg.GSHost, cfg.MatchmakingTimeout)
	hub.SetHandler(queue)

	r := chi.NewRouter()
	r.Get("/healthz", handlers.Health)

	authH := handlers.NewAuthHandler(database.PG, cfg.JWTSecret, cfg.JWTTTL)
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", authH.Register)
		r.Post("/login", authH.Login)
	})

	lobbyH := handlers.NewLobbyHandler(hub, cfg.JWTSecret)
	mmH := handlers.NewMatchmakingHandler(queue)
	matchesH := handlers.NewMatchesHandler(database.PG)

	// WS lobby authenticates via ?token=<jwt> query param (browser/Godot
	// WebSocket clients cannot set Authorization headers during upgrade).
	r.Get("/api/ws", lobbyH.ServeWS)

	// Authenticated player routes (header-based JWT).
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(cfg.JWTSecret))
		r.Post("/api/match/queue", mmH.Queue)
		r.Post("/api/match/cancel", mmH.Cancel)
		r.Get("/api/matches", matchesH.ListMatches)
		r.Get("/api/matches/{id}", matchesH.GetMatch)
	})

	// Internal GS-only routes.
	internalH := handlers.NewInternalHandler(database.PG)
	r.Group(func(r chi.Router) {
		r.Use(middleware.Internal(cfg.InternalToken))
		r.Post("/internal/matches/{id}/result", internalH.ReportResult)
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("openfire central server listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
