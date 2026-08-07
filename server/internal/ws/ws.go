// Package ws implements the lobby WebSocket layer: connection upgrade,
// per-client read/write pumps, user-targeted message delivery and a
// periodic lobby-state broadcast.
package ws

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"openfire-server/internal/models"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// InboundHandler processes a client→server message. Implemented by the
// match manager (decoupled to avoid an import cycle).
type InboundHandler interface {
	HandleInbound(userID, username string, msg models.WSInbound)
	QueuedCount() int
}

// Hub tracks all connected clients keyed by user id and fans out messages.
type Hub struct {
	mu       sync.RWMutex
	clients  map[string]*Client // userID -> client (one live conn per user)
	handler  InboundHandler
}

// NewHub builds an empty hub.
func NewHub(handler InboundHandler) *Hub {
	return &Hub{clients: make(map[string]*Client), handler: handler}
}

// OnlineCount returns the number of connected users.
func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// SendToUser marshals payload and pushes it to the user's connection.
// Returns false if the user is not currently connected.
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
		// backpressure: drop and let writePump recover / connection die.
		return false
	}
}

// register replaces any existing connection for the user (closing the old).
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
// already be authenticated and stored in the request context by the API
// layer.
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

// BroadcastLobby pushes a lobby_state message to every connected client.
// Called by the periodic broadcaster.
func (h *Hub) BroadcastLobby() {
	online := h.OnlineCount()
	queued := 0
	if h.handler != nil {
		queued = h.handler.QueuedCount()
	}
	msg := models.WSLobbyState{Type: "lobby_state", Online: online, Queued: queued}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.RLock()
	targets := make([]*Client, 0, len(h.clients))
	for _, c := range h.clients {
		targets = append(targets, c)
	}
	h.mu.RUnlock()
	for _, c := range targets {
		select {
		case c.send <- data:
		default:
		}
	}
}

// StartBroadcaster periodically pushes lobby_state until ctx is cancelled.
func (h *Hub) StartBroadcaster(ctx context.Context, intervalMS int) {
	interval := time.Duration(intervalMS) * time.Millisecond
	if interval <= 0 {
		interval = 2 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.BroadcastLobby()
		}
	}
}

// Client is one user's WebSocket session.
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	userID   string
	username string
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister(c)
		_ = c.conn.Close()
	}()
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var msg models.WSInbound
		if err := json.Unmarshal(raw, &msg); err != nil {
			c.sendError("malformed message")
			continue
		}
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		if c.hub.handler != nil {
			c.hub.handler.HandleInbound(c.userID, c.username, msg)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case data, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) sendError(message string) {
	data, err := json.Marshal(models.WSError{Type: "error", Message: message})
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
	}
}

// ErrNoUserInContext is returned by helpers when the JWT middleware did
// not populate the context.
var ErrNoUserInContext = errors.New("no user in context")
