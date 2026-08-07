// Package models defines the data structures shared across the central
// server: database rows, API request/response payloads, WebSocket
// messages and internal GS protocol payloads.
package models

import "time"

// User is the database row plus the canonical "current user" payload.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	CreatedAt    time.Time `json:"created_at"`
	PasswordHash string    `json:"-"`
}

// AuthResponse is returned by register/login.
type AuthResponse struct {
	Token string   `json:"token"`
	User  UserInfo `json:"user"`
}

// UserInfo is the public subset of a user returned in auth/me/profile.
type UserInfo struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

// MatchStatus enumerates the lifecycle of a match.
const (
	MatchStatusCreated    = "created"
	MatchStatusLaunching  = "launching"
	MatchStatusRunning   = "running"
	MatchStatusFinished  = "finished"
	MatchStatusAbandoned = "abandoned"
)

// Match is the database row for a match.
type Match struct {
	ID          string     `json:"id"`
	Mode        string     `json:"mode"`
	Map         string     `json:"map"`
	Status      string     `json:"status"`
	GSHost      string     `json:"gs_host"`
	GSPort      int        `json:"gs_port"`
	WinnerTeam  *int       `json:"winner_team"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at"`
}

// MatchPlayer is the per-match-per-player record.
type MatchPlayer struct {
	MatchID   string    `json:"-"`
	PlayerID  string    `json:"player_id"`
	Username  string    `json:"username"`
	Kills     int       `json:"kills"`
	Deaths    int       `json:"deaths"`
	Score     int       `json:"score"`
	Team      int       `json:"team"`
	JoinToken string    `json:"-"`
}

// MatchDetail is the GET /api/matches/:id payload.
type MatchDetail struct {
	Match   Match         `json:"match"`
	Players []MatchPlayer `json:"players"`
}

// PlayerStats aggregates a user's historical totals.
type PlayerStats struct {
	Matches int `json:"matches"`
	Wins    int `json:"wins"`
	Kills   int `json:"kills"`
	Deaths  int `json:"deaths"`
}

// RecentMatch is a user's most recent match summary row.
type RecentMatch struct {
	MatchID    string     `json:"match_id"`
	Kills      int        `json:"kills"`
	Deaths     int        `json:"deaths"`
	Score      int        `json:"score"`
	FinishedAt *time.Time `json:"finished_at"`
}

// ProfileResponse is the GET /api/profile/:username payload.
type ProfileResponse struct {
	User   UserInfo      `json:"user"`
	Stats  PlayerStats   `json:"stats"`
	Recent []RecentMatch `json:"recent"`
}

// QueueRequest is the body of POST /api/match/queue.
type QueueRequest struct {
	Mode string `json:"mode"`
}

// QueueResponse is returned by match/queue endpoints.
type QueueResponse struct {
	Queued bool `json:"queued"`
}

// ExpectedPlayer is what GS receives on gs_register.
type ExpectedPlayer struct {
	PlayerID  string `json:"player_id"`
	Username  string `json:"username"`
	JoinToken string `json:"join_token"`
}

// GSConfig is the match config handed to GS on register.
type GSConfig struct {
	Mode       string `json:"mode"`
	Map        string `json:"map"`
	MaxPlayers int    `json:"max_players"`
}

// GSRegisterRequest is the body of POST /internal/gs_register.
type GSRegisterRequest struct {
	MatchID    string `json:"match_id"`
	ListenPort int    `json:"listen_port"`
}

// GSRegisterResponse is returned by gs_register.
type GSRegisterResponse struct {
	MatchID        string          `json:"match_id"`
	ListenPort     int             `json:"listen_port"`
	ExpectedPlayers []ExpectedPlayer `json:"expected_players"`
	Config         GSConfig        `json:"config"`
}

// GSReportResult is one player's result reported by GS.
type GSReportResult struct {
	PlayerID string `json:"player_id"`
	Kills    int    `json:"kills"`
	Deaths   int    `json:"deaths"`
	Score    int    `json:"score"`
	Team     int    `json:"team"`
}

// GSReportRequest is the body of POST /internal/gs_report.
type GSReportRequest struct {
	MatchID   string           `json:"match_id"`
	Results   []GSReportResult `json:"results"`
	WinnerTeam int             `json:"winner_team"`
}

// GSHeartbeatRequest is the body of POST /internal/gs_heartbeat.
type GSHeartbeatRequest struct {
	MatchID    string `json:"match_id"`
	PlayerCount int   `json:"player_count"`
}

// OKResponse is a generic ack.
type OKResponse struct {
	OK bool `json:"ok"`
}

// WSInbound is any client→server WebSocket message.
type WSInbound struct {
	Type string `json:"type"`
	Mode string `json:"mode,omitempty"`
}

// WSMatchFound is the match_found push.
type WSMatchFound struct {
	Type      string `json:"type"`
	GSHost    string `json:"gs_host"`
	GSPort    int    `json:"gs_port"`
	MatchID   string `json:"match_id"`
	JoinToken string `json:"join_token"`
}

// WSLobbyState is the periodic lobby_state push.
type WSLobbyState struct {
	Type   string `json:"type"`
	Online int    `json:"online"`
	Queued int    `json:"queued"`
}

// WSError is the error push.
type WSError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
