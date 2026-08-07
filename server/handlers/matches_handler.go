package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"openfire-server/internal/auth"
	"openfire-server/internal/models"
)

// MatchesHandler serves match history and detail.
type MatchesHandler struct {
	db *sql.DB
}

// NewMatchesHandler builds a MatchesHandler.
func NewMatchesHandler(db *sql.DB) *MatchesHandler {
	return &MatchesHandler{db: db}
}

// matchDetailResponse is the GET /api/matches/{id} and GET /api/matches
// element payload. It mirrors the API contract exactly.
type matchDetailResponse struct {
	ID         string               `json:"id"`
	Mode       string               `json:"mode"`
	Status     string               `json:"status"`
	CreatedAt  time.Time            `json:"created_at"`
	FinishedAt *time.Time           `json:"finished_at"`
	Players    []models.MatchPlayer `json:"players"`
}

// GetMatch handles GET /api/matches/{id}.
func (h *MatchesHandler) GetMatch(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := auth.UserFromContext(r.Context()); !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	matchID := chi.URLParam(r, "id")

	var resp matchDetailResponse
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, mode, status, created_at, finished_at FROM matches WHERE id=$1`, matchID).
		Scan(&resp.ID, &resp.Mode, &resp.Status, &resp.CreatedAt, &resp.FinishedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSONError(w, http.StatusNotFound, "match not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}

	players, err := h.fetchPlayers(r, matchID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}
	resp.Players = players
	writeJSON(w, http.StatusOK, resp)
}

// ListMatches handles GET /api/matches, returning the caller's 20 most
// recent matches.
func (h *MatchesHandler) ListMatches(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	rows, err := h.db.QueryContext(r.Context(),
		`SELECT m.id, m.mode, m.status, m.created_at, m.finished_at
		   FROM matches m
		   JOIN match_players mp ON mp.match_id = m.id
		  WHERE mp.user_id = $1
		  ORDER BY m.created_at DESC
		  LIMIT 20`, userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}
	defer rows.Close()

	out := make([]matchDetailResponse, 0, 20)
	for rows.Next() {
		var resp matchDetailResponse
		if err := rows.Scan(&resp.ID, &resp.Mode, &resp.Status, &resp.CreatedAt, &resp.FinishedAt); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "server error")
			return
		}
		players, err := h.fetchPlayers(r, resp.ID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "server error")
			return
		}
		resp.Players = players
		out = append(out, resp)
	}
	if err := rows.Err(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// fetchPlayers loads a match's player rows, scanning nullable
// placement/team columns into the *int model fields.
func (h *MatchesHandler) fetchPlayers(r *http.Request, matchID string) ([]models.MatchPlayer, error) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT user_id, username, kills, deaths, placement, team
		   FROM match_players
		  WHERE match_id=$1`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	players := make([]models.MatchPlayer, 0)
	for rows.Next() {
		var (
			p                models.MatchPlayer
			placement, team  sql.NullInt64
		)
		if err := rows.Scan(&p.UserID, &p.Username, &p.Kills, &p.Deaths, &placement, &team); err != nil {
			return nil, err
		}
		if placement.Valid {
			v := int(placement.Int64)
			p.Placement = &v
		}
		if team.Valid {
			v := int(team.Int64)
			p.Team = &v
		}
		players = append(players, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return players, nil
}
