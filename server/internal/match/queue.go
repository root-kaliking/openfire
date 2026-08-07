// Package match implements the matchmaking queue, match formation, the
// dedicated-server lifecycle and result aggregation.
package match

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"openfire-server/internal/config"
	"openfire-server/internal/gameserver"
	"openfire-server/internal/lobby"
	"openfire-server/internal/store"
)

// Sentinel errors.
var (
	ErrAlreadyInQueue = errors.New("already in queue")
	ErrNotFound       = errors.New("not found")
)

// PlayerInfo is returned to the GS in the /started response.
type PlayerInfo struct {
	UserID       int64  `json:"user_id"`
	Username     string `json:"username"`
	SessionToken string `json:"session_token"`
}

// PlayerResult is part of the /end request body.
type PlayerResult struct {
	UserID    int64 `json:"user_id"`
	Placement int   `json:"placement"`
	Kills     int   `json:"kills"`
	Deaths    int   `json:"deaths"`
}

// Ticket is a queued player.
type Ticket struct {
	UserID   int64
	Username string
	Mode     string
	JoinedAt time.Time
}

// ActiveMatch is a match that has been formed and (likely) has a GS running.
type ActiveMatch struct {
	MatchID      string
	Mode         string
	Port         int
	UserIDs      []int64
	LaunchedAt   time.Time
	StartedAt    time.Time
	GSNotified   bool
	readyCh      chan struct{}
	readyClosed  bool
}

// Service ties together the queue, DB, hub and gameserver manager.
type Service struct {
	cfg     config.Config
	store   *store.Store
	pool    *pgxpool.Pool
	hub     *lobby.Hub
	gs      *gameserver.Manager
	log     *slog.Logger
	mu      sync.Mutex
	tickets map[int64]*Ticket
	active  map[string]*ActiveMatch
}

// NewService constructs a match Service.
func NewService(cfg config.Config, st *store.Store, hub *lobby.Hub, gs *gameserver.Manager, log *slog.Logger) *Service {
	return &Service{
		cfg:     cfg,
		store:   st,
		pool:    st.Pool,
		hub:     hub,
		gs:      gs,
		log:     log,
		tickets: make(map[int64]*Ticket),
		active:  make(map[string]*ActiveMatch),
	}
}

// Start launches the matcher and the duration-sweep goroutines.
func (s *Service) Start() {
	go s.runMatcher()
	go s.runSweeps()
}

// Enqueue adds a player to the queue for a mode. Returns a ticket id.
func (s *Service) Enqueue(userID int64, username, mode string) (string, error) {
	s.mu.Lock()
	if _, ok := s.tickets[userID]; ok {
		s.mu.Unlock()
		return "", ErrAlreadyInQueue
	}
	ticket := "t-" + genID(8)
	s.tickets[userID] = &Ticket{
		UserID:   userID,
		Username: username,
		Mode:     mode,
		JoinedAt: time.Now(),
	}
	n := len(s.tickets)
	s.mu.Unlock()
	s.hub.SetQueueCount(n)
	return ticket, nil
}

// Cancel removes a player from the queue. Returns whether a ticket was removed.
func (s *Service) Cancel(userID int64) bool {
	s.mu.Lock()
	_, ok := s.tickets[userID]
	delete(s.tickets, userID)
	n := len(s.tickets)
	s.mu.Unlock()
	if ok {
		s.hub.SetQueueCount(n)
	}
	return ok
}

// runMatcher periodically tries to form full matches from the queue.
func (s *Service) runMatcher() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		s.tryMatch()
	}
}

func (s *Service) tryMatch() {
	maxPlayers := s.cfg.GameserverMaxPlayers

	s.mu.Lock()
	byMode := make(map[string][]*Ticket)
	for _, t := range s.tickets {
		byMode[t.Mode] = append(byMode[t.Mode], t)
	}
	var chosen []*Ticket
	var chosenMode string
	for mode, list := range byMode {
		if len(list) >= maxPlayers {
			sort.Slice(list, func(i, j int) bool { return list[i].JoinedAt.Before(list[j].JoinedAt) })
			chosen = list[:maxPlayers]
			chosenMode = mode
			break
		}
	}
	if chosen != nil {
		for _, t := range chosen {
			delete(s.tickets, t.UserID)
		}
	}
	s.mu.Unlock()

	if chosen == nil {
		return
	}
	s.hub.SetQueueCount(s.queueLen())
	s.formMatch(chosenMode, chosen)
}

func (s *Service) queueLen() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.tickets)
}

// formMatch creates the DB rows, starts the GS and records the active match.
// On any failure the players are returned to the queue and resources released.
func (s *Service) formMatch(mode string, players []*Ticket) {
	ctx := context.Background()
	matchID := genID(8) // 16 hex chars

	port, err := s.gs.AllocatePort()
	if err != nil {
		s.log.Warn("cannot allocate port, returning players to queue", "err", err)
		s.returnToQueue(players)
		return
	}

	tokens := make(map[int64]string, len(players))
	for _, p := range players {
		tokens[p.UserID] = genID(16) // 32 hex chars
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.gs.ReleasePort(port)
		s.returnToQueue(players)
		s.log.Error("begin tx failed", "err", err)
		return
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		"INSERT INTO matches(id, mode, status, gs_port) VALUES($1,$2,'queued',$3)",
		matchID, mode, port); err != nil {
		s.gs.ReleasePort(port)
		s.returnToQueue(players)
		s.log.Error("insert match failed", "err", err)
		return
	}
	for _, p := range players {
		if _, err := tx.Exec(ctx,
			"INSERT INTO match_players(match_id, user_id, session_token, connected) VALUES($1,$2,$3,false)",
			matchID, p.UserID, tokens[p.UserID]); err != nil {
			s.gs.ReleasePort(port)
			s.returnToQueue(players)
			s.log.Error("insert match_players failed", "err", err)
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		s.gs.ReleasePort(port)
		s.returnToQueue(players)
		s.log.Error("commit match failed", "err", err)
		return
	}

	userIDs := make([]int64, 0, len(players))
	for _, p := range players {
		userIDs = append(userIDs, p.UserID)
	}
	active := &ActiveMatch{
		MatchID:    matchID,
		Mode:       mode,
		Port:       port,
		UserIDs:    userIDs,
		LaunchedAt: time.Now(),
		readyCh:    make(chan struct{}),
	}

	if err := s.gs.Start(matchID, mode, port, s.onProcessExit); err != nil {
		// Dev mode: binary missing, or process failed to start. Roll back.
		s.log.Warn("gameserver start failed, rolling back match", "match_id", matchID, "err", err)
		_, _ = s.pool.Exec(ctx, "DELETE FROM matches WHERE id=$1", matchID)
		s.gs.ReleasePort(port)
		s.returnToQueue(players)
		return
	}

	s.mu.Lock()
	s.active[matchID] = active
	s.mu.Unlock()

	go s.watchStart(matchID, active)
}

// returnToQueue re-adds players (e.g. after a failed formation).
func (s *Service) returnToQueue(players []*Ticket) {
	s.mu.Lock()
	for _, p := range players {
		s.tickets[p.UserID] = p
	}
	n := len(s.tickets)
	s.mu.Unlock()
	s.hub.SetQueueCount(n)
}

// watchStart kills the GS if it does not signal /started within StartTimeout.
func (s *Service) watchStart(matchID string, a *ActiveMatch) {
	timer := time.NewTimer(s.cfg.GameserverStartTimeout)
	defer timer.Stop()
	select {
	case <-timer.C:
		s.mu.Lock()
		notified := a.GSNotified
		s.mu.Unlock()
		if !notified {
			s.log.Warn("gameserver did not signal started in time, killing",
				"match_id", matchID, "timeout", s.cfg.GameserverStartTimeout)
			s.gs.Stop(matchID) // triggers onProcessExit -> handleUnexpectedExit
		}
	case <-a.readyCh:
	}
}

// OnGSStarted is called by the internal HTTP endpoint. It marks the match
// running, pushes match_found to each player and returns the player roster.
func (s *Service) OnGSStarted(matchID, gsIP string, gsPort int) ([]PlayerInfo, string, error) {
	ctx := context.Background()

	s.mu.Lock()
	a := s.active[matchID]
	if a == nil {
		s.mu.Unlock()
		return nil, "", ErrNotFound
	}
	mode := a.Mode
	s.mu.Unlock()

	if _, err := s.pool.Exec(ctx,
		"UPDATE matches SET status='running', started_at=NOW(), gs_ip=$2 WHERE id=$1",
		matchID, gsIP); err != nil {
		return nil, "", err
	}

	rows, err := s.pool.Query(ctx,
		"SELECT mp.user_id, u.username, mp.session_token FROM match_players mp JOIN users u ON u.id=mp.user_id WHERE mp.match_id=$1",
		matchID)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	var players []PlayerInfo
	for rows.Next() {
		var pi PlayerInfo
		if err := rows.Scan(&pi.UserID, &pi.Username, &pi.SessionToken); err != nil {
			return nil, "", err
		}
		players = append(players, pi)
	}

	s.mu.Lock()
	a.GSNotified = true
	a.StartedAt = time.Now()
	a.markReady()
	s.mu.Unlock()

	for _, p := range players {
		s.hub.SendToUser(p.UserID, map[string]interface{}{
			"type":          "match_found",
			"match_id":      matchID,
			"gs_ip":         gsIP,
			"gs_port":       gsPort,
			"session_token": p.SessionToken,
			"mode":          mode,
		})
	}
	return players, mode, nil
}

// SetPlayerConnected flips the connected flag for a player in a match.
func (s *Service) SetPlayerConnected(matchID string, userID int64, connected bool) error {
	ctx := context.Background()
	tag, err := s.pool.Exec(ctx,
		"UPDATE match_players SET connected=$3 WHERE match_id=$1 AND user_id=$2",
		matchID, userID, connected)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// OnGSEnd finalises a match: writes results + stats, notifies players, kills GS.
func (s *Service) OnGSEnd(matchID string, results []PlayerResult) error {
	ctx := context.Background()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, r := range results {
		if _, err := tx.Exec(ctx,
			"UPDATE match_players SET placement=$3, kills=$4, deaths=$5 WHERE match_id=$1 AND user_id=$2",
			matchID, r.UserID, r.Placement, r.Kills, r.Deaths); err != nil {
			return err
		}
		win, loss := 0, 0
		if r.Placement == 1 {
			win = 1
		} else {
			loss = 1
		}
		if _, err := tx.Exec(ctx,
			"UPDATE user_stats SET wins=wins+$3, losses=losses+$4, kills=kills+$5, deaths=deaths+$6, matches=matches+1 WHERE user_id=$1",
			r.UserID, 0, win, loss, r.Kills, r.Deaths); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx,
		"UPDATE matches SET status='ended', ended_at=NOW() WHERE id=$1", matchID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	a := s.active[matchID]
	if a != nil {
		delete(s.active, matchID)
		a.markReady()
	}
	s.mu.Unlock()

	// Stop the GS (idempotent). The watch goroutine releases the port.
	s.gs.Stop(matchID)

	if a != nil {
		resMap := make(map[int64]PlayerResult, len(results))
		for _, r := range results {
			resMap[r.UserID] = r
		}
		for _, uid := range a.UserIDs {
			r := resMap[uid]
			s.hub.SendToUser(uid, map[string]interface{}{
				"type":     "match_end",
				"match_id": matchID,
				"result": map[string]interface{}{
					"placement": r.Placement,
					"kills":     r.Kills,
					"deaths":    r.Deaths,
				},
			})
		}
	}
	return nil
}

// onProcessExit is the callback from the gameserver manager. If the match is
// still active the GS exited unexpectedly (crash), so clean it up.
func (s *Service) onProcessExit(matchID string, exitCode int) {
	s.mu.Lock()
	a := s.active[matchID]
	if a == nil {
		s.mu.Unlock()
		return
	}
	delete(s.active, matchID)
	a.markReady()
	userIDs := a.UserIDs
	s.mu.Unlock()

	ctx := context.Background()
	_, _ = s.pool.Exec(ctx,
		"UPDATE matches SET status='ended', ended_at=NOW() WHERE id=$1 AND status<>'ended'", matchID)
	s.log.Warn("gameserver exited unexpectedly; cleaning up match", "match_id", matchID, "exit_code", exitCode)
	for _, uid := range userIDs {
		s.hub.SendToUser(uid, map[string]interface{}{
			"type":   "match_cancelled",
			"reason": "game server ended unexpectedly",
		})
	}
}

// runSweeps force-ends matches that exceed the maximum duration.
func (s *Service) runSweeps() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		var stale []string
		for id, a := range s.active {
			if a.GSNotified && now.Sub(a.StartedAt) > s.cfg.GameserverMaxDuration {
				stale = append(stale, id)
			}
		}
		s.mu.Unlock()
		for _, id := range stale {
			s.log.Warn("match exceeded max duration, force-ending", "match_id", id)
			s.forceEnd(id, "match timed out")
		}
	}
}

// forceEnd ends a match out-of-band (timeout): marks ended, notifies, kills GS.
func (s *Service) forceEnd(matchID, reason string) {
	ctx := context.Background()
	s.mu.Lock()
	a := s.active[matchID]
	if a == nil {
		s.mu.Unlock()
		return
	}
	delete(s.active, matchID)
	a.markReady()
	userIDs := a.UserIDs
	s.mu.Unlock()

	_, _ = s.pool.Exec(ctx,
		"UPDATE matches SET status='ended', ended_at=NOW() WHERE id=$1 AND status<>'ended'", matchID)
	s.gs.Stop(matchID) // watch goroutine releases the port + onProcessExit is a no-op
	for _, uid := range userIDs {
		s.hub.SendToUser(uid, map[string]interface{}{
			"type":   "match_cancelled",
			"reason": reason,
		})
	}
}

// markReady closes the ready channel once.
func (a *ActiveMatch) markReady() {
	if !a.readyClosed {
		a.readyClosed = true
		close(a.readyCh)
	}
}

// genID returns n random bytes as a hex string (2*n chars).
func genID(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// Extremely unlikely; fall back to time-based value to avoid panics.
		return fmt.Sprintf("%016x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
