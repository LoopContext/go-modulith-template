// Package websocket provides real-time communication infrastructure for modules.
// It integrates with the event bus to broadcast events to connected clients.
package websocket

import (
	"context"
	"log/slog"
	"sync"
)

// Hub maintains active WebSocket connections and broadcasts messages.
type Hub struct {
	// clients holds all registered clients by their connection ID
	clients map[string]*Client

	// userClients maps user IDs to their client connection IDs for targeted messaging
	userClients map[string]map[string]bool

	// broadcast channel for messages to all clients
	broadcast chan *Message

	// register channel for new clients
	register chan *Client

	// unregister channel for disconnecting clients
	unregister chan *Client

	// mu protects concurrent access to clients and userClients maps
	mu sync.RWMutex

	// ctx is the root context for the hub
	ctx context.Context

	// cancel is the function to cancel the hub context
	cancel context.CancelFunc
}

// Message represents a WebSocket message to be sent to clients.
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	UserID  string      `json:"-"` // Used for targeting, not sent to client
}

// NewHub creates a new WebSocket hub.
func NewHub(ctx context.Context) *Hub {
	hubCtx, cancel := context.WithCancel(ctx)

	return &Hub{
		clients:     make(map[string]*Client),
		userClients: make(map[string]map[string]bool),
		broadcast:   make(chan *Message, 256),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		ctx:         hubCtx,
		cancel:      cancel,
	}
}

// Run starts the hub's main loop for managing clients and broadcasting messages.
func (h *Hub) Run() {
	slog.Info("WebSocket hub started")

	for {
		select {
		case <-h.ctx.Done():
			h.shutdown()
			return

		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// Stop gracefully shuts down the hub.
func (h *Hub) Stop() {
	h.cancel()
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(message *Message) {
	select {
	case h.broadcast <- message:
	case <-h.ctx.Done():
		slog.Warn("Hub is shutting down, message not sent")
	}
}

// SendToUser sends a message to a specific user (all their connections).
func (h *Hub) SendToUser(userID string, message *Message) {
	h.mu.RLock()
	clientIDs, ok := h.userClients[userID]
	h.mu.RUnlock()

	if !ok {
		slog.Debug("No connected clients for user", "user_id", userID)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for clientID := range clientIDs {
		if client, exists := h.clients[clientID]; exists {
			select {
			case client.send <- message:
			default:
				slog.Warn("Client send buffer full, dropping message", "client_id", clientID)
			}
		}
	}
}

// GetConnectedUsers returns the number of unique users connected.
func (h *Hub) GetConnectedUsers() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.userClients)
}

// GetTotalConnections returns the total number of active connections.
func (h *Hub) GetTotalConnections() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.clients)
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client.id] = client

	if client.userID != "" {
		if h.userClients[client.userID] == nil {
			h.userClients[client.userID] = make(map[string]bool)
		}

		h.userClients[client.userID][client.id] = true
	}

	slog.Info("Client registered",
		"client_id", client.id,
		"user_id", client.userID,
		"total_connections", len(h.clients))
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.id]; !ok {
		return
	}

	delete(h.clients, client.id)

	if client.userID != "" {
		if userClients, exists := h.userClients[client.userID]; exists {
			delete(userClients, client.id)

			if len(userClients) == 0 {
				delete(h.userClients, client.userID)
			}
		}
	}

	close(client.send)

	slog.Info("Client unregistered",
		"client_id", client.id,
		"user_id", client.userID,
		"total_connections", len(h.clients))
}

func (h *Hub) broadcastMessage(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		select {
		case client.send <- message:
		default:
			slog.Warn("Client send buffer full, dropping broadcast message", "client_id", client.id)
		}
	}
}

func (h *Hub) shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()

	slog.Info("Shutting down WebSocket hub", "active_connections", len(h.clients))

	for _, client := range h.clients {
		close(client.send)
	}

	h.clients = make(map[string]*Client)
	h.userClients = make(map[string]map[string]bool)

	slog.Info("WebSocket hub stopped")
}

