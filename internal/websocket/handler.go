package websocket

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(_ *http.Request) bool {
		// TODO: In production, implement proper origin checking
		return true
	},
}

// Handler handles WebSocket upgrade requests.
type Handler struct {
	hub *Hub
}

// NewHandler creates a new WebSocket HTTP handler.
func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

// ServeHTTP handles the WebSocket upgrade request.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context (set by auth middleware)
	userID := getUserIDFromContext(r)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Failed to upgrade connection", "error", err)
		return
	}

	client := NewClient(h.hub, conn, userID)
	h.hub.register <- client

	slog.Info("WebSocket connection established",
		"user_id", userID,
		"remote_addr", r.RemoteAddr)

	// Start goroutines for reading and writing
	go client.WritePump()
	go client.ReadPump()
}

// getUserIDFromContext extracts the user ID from the request context.
// This should be set by your authentication middleware.
func getUserIDFromContext(r *http.Request) string {
	// Try to get from query parameter (for demo/dev purposes)
	userID := r.URL.Query().Get("user_id")
	if userID != "" {
		return userID
	}

	// In production, get from context set by auth middleware:
	// if userID, ok := r.Context().Value("user_id").(string); ok {
	//     return userID
	// }

	return ""
}

