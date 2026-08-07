package handlers

import (
	"encoding/json"
	"net/http"

	"openfire-server/internal/auth"
	"openfire-server/internal/matchmaking"
	"openfire-server/internal/models"
)

// MatchmakingHandler exposes the HTTP entry points for the matchmaking
// queue (mirrors the WS match_queue / match_cancel messages).
type MatchmakingHandler struct {
	queue *matchmaking.Queue
}

// NewMatchmakingHandler builds a MatchmakingHandler.
func NewMatchmakingHandler(queue *matchmaking.Queue) *MatchmakingHandler {
	return &MatchmakingHandler{queue: queue}
}

type queueRequest struct {
	Mode string `json:"mode"`
}

// Queue handles POST /api/match/queue. The match_found push is still
// delivered over the player's lobby WebSocket, so the player must also
// be connected to /api/ws to receive it.
func (h *MatchmakingHandler) Queue(w http.ResponseWriter, r *http.Request) {
	userID, username, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req queueRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if !models.KnownMode(req.Mode) {
		writeJSONError(w, http.StatusBadRequest, "unknown mode")
		return
	}
	if err := h.queue.Enqueue(req.Mode, userID, username); err != nil {
		writeJSONError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"queued": true})
}

// Cancel handles POST /api/match/cancel.
func (h *MatchmakingHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	h.queue.Cancel(userID)
	writeJSON(w, http.StatusOK, map[string]bool{"canceled": true})
}
