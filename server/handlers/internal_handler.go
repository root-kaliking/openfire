package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// InternalHandler receives match results reported by Godot dedicated
// servers. All routes are guarded by the internal-token middleware.
type InternalHandler struct {
	db *sql.DB
}

// NewInternalHandler builds an InternalHandler.
func NewInternalHandler(db *sql.DB) *InternalHandler {
	return &InternalHandler{db: db}
}

type resultPlayer struct {
	UserID    string `json:"user_id"`
	Kills     int    `json:"kills"`
	Deaths    int    `json:"deaths"`
	Placement *int   `json:"placement"`
	Team      *int   `json:"team"`
}

type resultRequest struct {
	Status  string         `json:"status"`
	Players []resultPlayer `json:"players"`
}

// ReportResult handles POST /internal/matches/{id}/result.
func (h *InternalHandler) ReportResult(w http.ResponseWriter, r *http.Request) {
	matchID := chi.URLParam(r, "id")

	var req resultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Status != "finished" && req.Status != "abandoned" {
		writeJSONError(w, http.StatusBadRequest, "invalid status")
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}

	for _, p := range req.Players {
		if _, err := tx.ExecContext(r.Context(),
			`UPDATE match_players
			    SET kills=$1, deaths=$2, placement=$3, team=$4
			  WHERE match_id=$5 AND user_id=$6`,
			p.Kills, p.Deaths, p.Placement, p.Team, matchID, p.UserID); err != nil {
			_ = tx.Rollback()
			writeJSONError(w, http.StatusInternalServerError, "server error")
			return
		}
	}

	res, err := tx.ExecContext(r.Context(),
		`UPDATE matches SET status=$1, finished_at=NOW() WHERE id=$2`,
		req.Status, matchID)
	if err != nil {
		_ = tx.Rollback()
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		_ = tx.Rollback()
		writeJSONError(w, http.StatusNotFound, "match not found")
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
