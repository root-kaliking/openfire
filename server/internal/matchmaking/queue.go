// Package matchmaking implements a simple per-mode FIFO queue. When a
// mode accumulates enough players for a full room it dequeues them,
// creates a match row, launches a Godot dedicated-server process, and
// pushes a match_found message to every matched player over the lobby
// WebSocket.
//
// The queue is in-memory (single central-server instance). The Redis
// client owned by the db package is available for future distributed
// presence/queue work; for the 8-16 players/room target a local FIFO is
// sufficient and keeps match_found delivery on the same instance that
// owns the players' WebSocket connections.
package matchmaking

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	"openfire-server/internal/gs"
	"openfire-server/internal/lobby"
	"openfire-server/internal/models"
)

// ErrAlreadyQueued is returned by Enqueue when the user is already in
// any mode's queue.
var ErrAlreadyQueued = errors.New("already in queue")

// ErrUnknownMode is returned by Enqueue for an unsupported mode.
var ErrUnknownMode = errors.New("unknown mode")

type entry struct {
	userID     string
	username   string
	enqueuedAt time.Time
}

// Queue is the matchmaking FIFO.
type Queue struct {
	mu      sync.Mutex
	queues  map[string][]entry // mode -> entries (FIFO)
	byUser  map[string]string  // userID -> mode (quick lookup / dedup)
	hub     *lobby.Hub
	db      *sql.DB
	gsMgr   *gs.Manager
	gsHost  string
	timeout time.Duration
}

// NewQueue builds a queue and starts a background sweeper that evicts
// players waiting longer than timeout.
func NewQueue(hub *lobby.Hub, db *sql.DB, gsMgr *gs.Manager, gsHost string, timeout time.Duration) *Queue {
	q := &Queue{
		queues:  make(map[string][]entry),
		byUser:  make(map[string]string),
		hub:     hub,
		db:      db,
		gsMgr:   gsMgr,
		gsHost:  gsHost,
		timeout: timeout,
	}
	go q.timeoutSweeper()
	return q
}

// Enqueue adds the user to the mode's queue. If the queue reaches the
// mode's room size a match is created immediately.
func (q *Queue) Enqueue(mode, userID, username string) error {
	if !models.KnownMode(mode) {
		return ErrUnknownMode
	}
	q.mu.Lock()
	if _, ok := q.byUser[userID]; ok {
		q.mu.Unlock()
		return ErrAlreadyQueued
	}
	q.queues[mode] = append(q.queues[mode], entry{userID: userID, username: username, enqueuedAt: time.Now()})
	q.byUser[userID] = mode
	need := models.ModePlayers(mode)
	have := len(q.queues[mode])
	q.mu.Unlock()

	if have >= need {
		q.tryMatch(mode, need)
	}
	return nil
}

// Cancel removes the user from whatever queue it is in. It returns
// false if the user was not queued.
func (q *Queue) Cancel(userID string) bool {
	q.mu.Lock()
	mode, ok := q.byUser[userID]
	if !ok {
		q.mu.Unlock()
		return false
	}
	delete(q.byUser, userID)
	q.queues[mode] = removeEntry(q.queues[mode], userID)
	q.mu.Unlock()
	return true
}

// IsQueued reports whether the user is currently in any queue.
func (q *Queue) IsQueued(userID string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	_, ok := q.byUser[userID]
	return ok
}

// QueuedCount returns the total number of players waiting across modes.
func (q *Queue) QueuedCount() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	n := 0
	for _, list := range q.queues {
		n += len(list)
	}
	return n
}

// HandleInbound implements lobby.InboundHandler, translating WS
// match_queue / match_cancel messages into queue operations.
func (q *Queue) HandleInbound(userID, username string, msg lobby.ClientMessage) {
	switch msg.Type {
	case "match_queue":
		if !models.KnownMode(msg.Mode) {
			q.hub.SendToUser(userID, errorMsg("unknown mode"))
			return
		}
		if err := q.Enqueue(msg.Mode, userID, username); err != nil {
			q.hub.SendToUser(userID, errorMsg(err.Error()))
			return
		}
		q.hub.SendToUser(userID, map[string]string{"type": "queued"})
	case "match_cancel":
		q.Cancel(userID)
		q.hub.SendToUser(userID, map[string]string{"type": "match_canceled"})
	default:
		q.hub.SendToUser(userID, errorMsg("unknown message type"))
	}
}

// tryMatch dequeues need players for mode and creates a match. It is
// called after each successful Enqueue that may have filled a room.
func (q *Queue) tryMatch(mode string, need int) {
	q.mu.Lock()
	if len(q.queues[mode]) < need {
		q.mu.Unlock()
		return
	}
	picked := q.queues[mode][:need]
	q.queues[mode] = q.queues[mode][need:]
	for _, e := range picked {
		delete(q.byUser, e.userID)
	}
	q.mu.Unlock()

	q.createMatch(mode, picked)
}

// createMatch persists the match + players, launches the GS, and pushes
// match_found to all players. On any failure the players are notified
// and the match is marked abandoned.
func (q *Queue) createMatch(mode string, players []entry) {
	matchID := uuid.NewString()
	ctx := context.Background()

	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("matchmaking: begin tx: %v", err)
		q.failPlayers(players, "server error")
		return
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO matches (id, mode, status, gs_host, gs_port) VALUES ($1,$2,'pending',$3,0)`,
		matchID, mode, q.gsHost); err != nil {
		_ = tx.Rollback()
		log.Printf("matchmaking: insert match: %v", err)
		q.failPlayers(players, "server error")
		return
	}
	for _, p := range players {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO match_players (match_id, user_id, username) VALUES ($1,$2,$3)`,
			matchID, p.userID, p.username); err != nil {
			_ = tx.Rollback()
			log.Printf("matchmaking: insert match_player: %v", err)
			q.failPlayers(players, "server error")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		log.Printf("matchmaking: commit: %v", err)
		q.failPlayers(players, "server error")
		return
	}

	port, err := q.gsMgr.StartMatch(matchID)
	if err != nil {
		log.Printf("matchmaking: start gs for %s: %v", matchID, err)
		q.markAbandoned(matchID)
		q.failPlayers(players, "failed to start game server")
		return
	}

	if _, err := q.db.ExecContext(ctx,
		`UPDATE matches SET gs_port=$1, status='in_progress' WHERE id=$2`,
		port, matchID); err != nil {
		log.Printf("matchmaking: update match %s: %v", matchID, err)
	}

	playerList := make([]matchFoundPlayer, 0, len(players))
	for _, p := range players {
		playerList = append(playerList, matchFoundPlayer{UserID: p.userID, Username: p.username})
	}
	msg := matchFoundMessage{
		Type:    "match_found",
		MatchID: matchID,
		GSHost:  q.gsHost,
		GSPort:  port,
		Players: playerList,
		Mode:    mode,
	}
	for _, p := range players {
		q.hub.SendToUser(p.userID, msg)
	}
}

func (q *Queue) markAbandoned(matchID string) {
	if _, err := q.db.Exec(
		`UPDATE matches SET status='abandoned', finished_at=NOW() WHERE id=$1`,
		matchID); err != nil {
		log.Printf("matchmaking: mark abandoned %s: %v", matchID, err)
	}
}

func (q *Queue) failPlayers(players []entry, message string) {
	for _, p := range players {
		q.hub.SendToUser(p.userID, errorMsg(message))
	}
}

// timeoutSweeper periodically evicts players waiting longer than the
// configured MATCHMAKING_TIMEOUT, notifying them.
func (q *Queue) timeoutSweeper() {
	interval := 5 * time.Second
	if q.timeout > 0 {
		if probe := q.timeout / 4; probe < interval && probe >= time.Second {
			interval = probe
		}
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		q.evictStale()
	}
}

func (q *Queue) evictStale() {
	if q.timeout <= 0 {
		return
	}
	now := time.Now()
	var evicted []entry
	q.mu.Lock()
	for mode, list := range q.queues {
		kept := make([]entry, 0, len(list))
		for _, e := range list {
			if now.Sub(e.enqueuedAt) > q.timeout {
				delete(q.byUser, e.userID)
				evicted = append(evicted, e)
			} else {
				kept = append(kept, e)
			}
		}
		q.queues[mode] = kept
	}
	q.mu.Unlock()

	for _, e := range evicted {
		q.hub.SendToUser(e.userID, errorMsg("matchmaking timed out"))
		q.hub.SendToUser(e.userID, map[string]string{"type": "match_canceled"})
	}
}

func removeEntry(list []entry, userID string) []entry {
	for i, e := range list {
		if e.userID == userID {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func errorMsg(message string) map[string]string {
	return map[string]string{"type": "error", "message": message}
}

// matchFoundPlayer is a player entry in the match_found WS push.
type matchFoundPlayer struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

// matchFoundMessage is the server→client match_found WS push.
//
//	{"type":"match_found","match_id":"...","gs_host":"127.0.0.1",
//	 "gs_port":27020,"players":[{"user_id":"...","username":"..."}],
//	 "mode":"deathmatch"}
type matchFoundMessage struct {
	Type    string            `json:"type"`
	MatchID string            `json:"match_id"`
	GSHost  string            `json:"gs_host"`
	GSPort  int               `json:"gs_port"`
	Players []matchFoundPlayer `json:"players"`
	Mode    string            `json:"mode"`
}
