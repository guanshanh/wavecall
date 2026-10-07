package signaling

import (
	"encoding/json"
	"log/slog"

	"github.com/gorilla/websocket"
)

// Client represents a connected WebSocket client.
type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	userID string
	roomID string
}

// Send enqueues a message to be sent to this client.
func (c *Client) Send(msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	select {
	case c.send <- data:
	default:
		slog.Warn("client send buffer full, dropping message", "userId", c.userID)
	}
	return nil
}

// writePump pumps messages from the send channel to the WebSocket connection.
func (c *Client) writePump() {
	defer c.conn.Close()
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			slog.Error("websocket write error", "err", err, "userId", c.userID)
			return
		}
	}
}

// Close shuts down the client's send channel.
func (c *Client) Close() {
	close(c.send)
}
