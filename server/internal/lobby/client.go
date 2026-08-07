package lobby

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

// ClientMessage is any client→server WebSocket message.
//
//	{"type":"match_queue","mode":"deathmatch"}
//	{"type":"match_cancel"}
type ClientMessage struct {
	Type string `json:"type"`
	Mode string `json:"mode,omitempty"`
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
		var msg ClientMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			c.sendError("malformed message")
			continue
		}
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		c.hub.mu.RLock()
		handler := c.hub.handler
		c.hub.mu.RUnlock()
		if handler != nil {
			handler.HandleInbound(c.userID, c.username, msg)
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

// sendError pushes an {"type":"error","message":...} frame to the client.
func (c *Client) sendError(message string) {
	data, err := json.Marshal(map[string]string{"type": "error", "message": message})
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
	}
}
