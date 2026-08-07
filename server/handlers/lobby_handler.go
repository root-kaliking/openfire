package handlers

import (
	"net/http"

	"openfire-server/internal/auth"
	"openfire-server/internal/lobby"
)

// LobbyHandler upgrades authenticated requests to the lobby WebSocket.
type LobbyHandler struct {
	hub *lobby.Hub
}

// NewLobbyHandler builds a LobbyHandler.
func NewLobbyHandler(hub *lobby.Hub) *LobbyHandler {
	return &LobbyHandler{hub: hub}
}

// ServeWS handles GET /api/ws.
func (h *LobbyHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	userID, username, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	h.hub.ServeWS(w, r, userID, username)
}
