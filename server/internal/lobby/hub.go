// Package lobby implements the lobby WebSocket layer: connection
// upgrade, per-client read/write pumps, and user-targeted message
// delivery. Inbound messages are forwarded to an InboundHandler
// (implemented by the matchmaking queue) to avoid an import cycle.
package lobby

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// InboundHandler processes a client→server message. Implemented by the
// matchmaking queue.
type InboundHandler interface {
	HandleInbound(userID, username string, msg ClientMessage)
}

// Hub tracks all connected clients keyed by user id and fans out
// messages to individual users.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]*Client // userID -> client (one live conn per user)
	handler InboundHandler
}

// NewHub builds an empty hub. The inbound handler is wired later via
// SetHandler so the handler can itself hold a reference to the hub.
func NewHub(handler InboundHandler) *Hub {
	return &Hub{clients: make(map[string]*Client), handler: handler}
}

// SetHandler attaches the inbound handler after construction.
func (h *Hub) SetHandler(handler InboundHandler) {
	h.mu.Lock()
	h.handler = handler
	h.mu.Unlock()
}

// OnlineCount returns the number of connected users.
func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// SendToUser marshals payload and pushes it to the user's connection.
// It returns false if the user is not currently connected or the send
// queue is full (backpressure).
func (h *Hub) SendToUser(userID string, payload interface{}) bool {
	h.mu.RLock()
	c := h.clients[userID]
	h.mu.RUnlock()
	if c == nil {
		return false
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	select {
	case c.send <- data:
		return true
	default:
		return false
	}
}

// register replaces any existing connection for the user, closing the
// previous one (a user may only have one live lobby connection).
func (h *Hub) register(c *Client) {
	h.mu.Lock()
	old := h.clients[c.userID]
	h.clients[c.userID] = c
	h.mu.Unlock()
	if old != nil {
		_ = old.conn.Close()
	}
}

func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	if cur := h.clients[c.userID]; cur == c {
		delete(h.clients, c.userID)
	}
	h.mu.Unlock()
}

// ServeWS upgrades an HTTP request to a WebSocket. userID/username must
// already be authenticated and stored in the request context by the
// auth middleware.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, userID, username string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return // upgrade already wrote an http error
	}
	conn.SetReadLimit(8192)
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	c := &Client{
		hub:      h,
		conn:     conn,
		send:     make(chan []byte, 64),
		userID:   userID,
		username: username,
	}
	h.register(c)
	go c.writePump()
	go c.readPump()
}
