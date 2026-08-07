// Package lobby manages WebSocket client connections, the online roster and
// lobby_update broadcasts.
package lobby

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Client represents a single authenticated WebSocket connection.
type Client struct {
	UserID   int64
	Username string
	Conn     *websocket.Conn
	Send     chan []byte
	hub      *Hub
}

// Hub tracks all connected clients and broadcasts lobby state.
type Hub struct {
	mu         sync.RWMutex
	clients    map[int64]*Client
	queueCount int
}

// NewHub creates an empty Hub.
func NewHub() *Hub {
	return &Hub{clients: make(map[int64]*Client)}
}

// NewClient constructs a Client registered against the hub.
func (h *Hub) NewClient(userID int64, username string, conn *websocket.Conn) *Client {
	return &Client{
		UserID:   userID,
		Username: username,
		Conn:     conn,
		Send:     make(chan []byte, 64),
		hub:      h,
	}
}

// Register adds a client and broadcasts an updated lobby state.
func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	h.clients[c.UserID] = c
	h.mu.Unlock()
	h.BroadcastLobby()
}

// Unregister removes a client, closes its send channel and broadcasts.
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	if cur, ok := h.clients[c.UserID]; ok && cur == c {
		delete(h.clients, c.UserID)
		close(c.Send)
	}
	h.mu.Unlock()
	h.BroadcastLobby()
}

// OnlineCount returns the number of connected clients.
func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// SetQueueCount updates the in-queue counter and re-broadcasts lobby state.
func (h *Hub) SetQueueCount(n int) {
	h.mu.Lock()
	h.queueCount = n
	h.mu.Unlock()
	h.BroadcastLobby()
}

// QueueCount returns the current in-queue counter.
func (h *Hub) QueueCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.queueCount
}

// SendToUser delivers a JSON message to a single user (non-blocking, dropped if slow).
func (h *Hub) SendToUser(userID int64, msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	c := h.clients[userID]
	if c == nil {
		return
	}
	select {
	case c.Send <- data:
	default:
	}
}

// BroadcastLobby sends a lobby_update to every connected client.
func (h *Hub) BroadcastLobby() {
	h.mu.RLock()
	online := len(h.clients)
	inQueue := h.queueCount
	clients := make([]*Client, 0, len(h.clients))
	for _, c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	data, err := json.Marshal(map[string]interface{}{
		"type":     "lobby_update",
		"online":   online,
		"in_queue": inQueue,
	})
	if err != nil {
		return
	}
	for _, c := range clients {
		select {
		case c.Send <- data:
		default:
		}
	}
}

// WritePump drains the client's Send channel into the WebSocket. It exits when
// the Send channel is closed (by Unregister), then closes the connection.
func (c *Client) WritePump() {
	for msg := range c.Send {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := c.Conn.Write(ctx, websocket.MessageText, msg)
		cancel()
		if err != nil {
			return
		}
	}
	_ = c.Conn.Close(websocket.StatusNormalClosure, "")
}

// TrySend pushes a message to the client without blocking.
func (c *Client) TrySend(data []byte) {
	select {
	case c.Send <- data:
	default:
	}
}
