package models

// MatchPlayer is the match_players table row. Placement and Team are
// nullable (NULL until a match finishes and the GS reports results).
type MatchPlayer struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Kills     int    `json:"kills"`
	Deaths    int    `json:"deaths"`
	Placement *int   `json:"placement"`
	Team      *int   `json:"team"`
}
