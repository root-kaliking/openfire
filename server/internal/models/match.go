package models

import "time"

// Match lifecycle statuses (matches the CHECK constraint in
// migrations/001_init.sql).
const (
	MatchStatusPending    = "pending"
	MatchStatusInProgress = "in_progress"
	MatchStatusFinished   = "finished"
	MatchStatusAbandoned  = "abandoned"
)

// Match is the matches table row.
type Match struct {
	ID         string     `json:"id"`
	Mode       string     `json:"mode"`
	Status     string     `json:"status"`
	GSHost     string     `json:"gs_host"`
	GSPort     int        `json:"gs_port"`
	CreatedAt  time.Time  `json:"created_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

// KnownMode reports whether mode is a supported matchmaking mode.
func KnownMode(mode string) bool {
	switch mode {
	case "deathmatch", "team_dm", "domination", "battle_royale", "coop", "adventure":
		return true
	}
	return false
}

// ModePlayers returns the room size (players needed to open a match)
// for the given mode. Unknown modes default to 8.
func ModePlayers(mode string) int {
	switch mode {
	case "deathmatch", "team_dm", "domination":
		return 8
	case "battle_royale":
		return 16
	case "coop", "adventure":
		return 4
	default:
		return 8
	}
}
