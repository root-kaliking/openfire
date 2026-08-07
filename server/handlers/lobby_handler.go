package handlers

import (
	"net/http"

	"openfire-server/internal/auth"
	"openfire-server/internal/lobby"
)

// LobbyHandler upgrades authenticated requests to the lobby WebSocket.
type LobbyHandler struct {
	hub       *lobby.Hub
	jwtSecret string
}

// NewLobbyHandler builds a LobbyHandler. jwtSecret is used to verify the
// token passed in the ?token=<jwt> query parameter.
func NewLobbyHandler(hub *lobby.Hub, jwtSecret string) *LobbyHandler {
	return &LobbyHandler{hub: hub, jwtSecret: jwtSecret}
}

// ServeWS handles GET /api/ws. The JWT is read from the "token" query
// parameter (Godot's WebSocketPeer cannot set request headers during the
// upgrade handshake). On success the verified user_id/username are passed
// to the hub along with the request.
func (h *LobbyHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		writeJSONError(w, http.StatusUnauthorized, "missing token")
		return
	}
	claims, err := auth.Verify(h.jwtSecret, tokenStr)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}
	h.hub.ServeWS(w, r, claims.UserID, claims.Username)
}
