package websocket

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

// Client represents a single WebSocket connection.
type Client struct {
	// id is the unique identifier for this client connection
	id string

	// userID is the authenticated user ID (empty for unauthenticated)
	userID string

	// hub is the parent hub managing this client
	hub *Hub

	// conn is the WebSocket connection
	conn *websocket.Conn

	// send is the channel for outbound messages
	send chan *Message
}

// NewClient creates a new WebSocket client.
func NewClient(hub *Hub, conn *websocket.Conn, userID string) *Client {
	return &Client{
		id:     uuid.New().String(),
		userID: userID,
		hub:    hub,
		conn:   conn,
		send:   make(chan *Message, 256),
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c

		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("WebSocket read error", "error", err, "client_id", c.id)
			}

			break
		}

		// Currently we only handle pings from clients, but this is where you'd
		// process incoming messages if needed (e.g., subscribe to specific topics)
		slog.Debug("Received message from client", "client_id", c.id, "message", string(message))
	}
}

// WritePump pumps messages from the hub to the WebSocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()

		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if !ok {
				// Hub closed the channel
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.writeJSON(message); err != nil {
				slog.Error("Failed to write message", "error", err, "client_id", c.id)
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// GetID returns the client's unique identifier.
func (c *Client) GetID() string {
	return c.id
}

// GetUserID returns the authenticated user ID.
func (c *Client) GetUserID() string {
	return c.userID
}

func (c *Client) writeJSON(message *Message) error {
	w, err := c.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return fmt.Errorf("failed to get next writer: %w", err)
	}

	if err := json.NewEncoder(w).Encode(message); err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return nil
}
