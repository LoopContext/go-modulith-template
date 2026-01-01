package websocket

import (
	"context"
	"log/slog"

	"github.com/cmelgarejo/go-modulith-template/internal/events"
)

// Subscriber listens to the event bus and forwards events to WebSocket clients.
type Subscriber struct {
	hub      *Hub
	eventBus *events.Bus
}

// NewSubscriber creates a new WebSocket event subscriber.
func NewSubscriber(hub *Hub, eventBus *events.Bus) *Subscriber {
	return &Subscriber{
		hub:      hub,
		eventBus: eventBus,
	}
}

// Subscribe registers the subscriber to listen for specific event patterns.
// By default, it subscribes to common event patterns that should be broadcasted.
func (s *Subscriber) Subscribe() {
	// Subscribe to common event patterns
	// Modules can emit events like: "user.created", "order.created", etc.

	// Wildcard approach: subscribe to events that should be broadcasted
	s.subscribeToPattern("*.created")
	s.subscribeToPattern("*.updated")
	s.subscribeToPattern("*.deleted")
	s.subscribeToPattern("notification.*")
	s.subscribeToPattern("alert.*")

	slog.Info("WebSocket subscriber registered for event patterns")
}

// SubscribeToEvent subscribes to a specific event name.
func (s *Subscriber) SubscribeToEvent(eventName string) {
	s.eventBus.Subscribe(eventName, s.handleEvent)
	slog.Info("WebSocket subscriber registered for event", "event", eventName)
}

func (s *Subscriber) subscribeToPattern(pattern string) {
	// Since the current event bus doesn't support wildcards,
	// we subscribe to specific events. In a production system,
	// you might want to enhance the event bus to support patterns.

	// For now, we'll subscribe to known events based on the pattern
	switch pattern {
	case "*.created":
		s.eventBus.Subscribe("user.created", s.handleEvent)
		s.eventBus.Subscribe("order.created", s.handleEvent)
		s.eventBus.Subscribe("payment.created", s.handleEvent)
	case "*.updated":
		s.eventBus.Subscribe("user.updated", s.handleEvent)
		s.eventBus.Subscribe("order.updated", s.handleEvent)
		s.eventBus.Subscribe("payment.updated", s.handleEvent)
	case "*.deleted":
		s.eventBus.Subscribe("user.deleted", s.handleEvent)
		s.eventBus.Subscribe("order.deleted", s.handleEvent)
	case "notification.*":
		s.eventBus.Subscribe("notification.sent", s.handleEvent)
		s.eventBus.Subscribe("notification.read", s.handleEvent)
	case "alert.*":
		s.eventBus.Subscribe("alert.created", s.handleEvent)
		s.eventBus.Subscribe("alert.resolved", s.handleEvent)
	}
}

func (s *Subscriber) handleEvent(_ context.Context, event events.Event) error {
	// Extract user ID from payload if available for targeted messaging
	userID := extractUserID(event.Payload)

	message := &Message{
		Type:    event.Name,
		Payload: event.Payload,
		UserID:  userID,
	}

	// If user ID is present, send to specific user, otherwise broadcast
	if userID != "" {
		s.hub.SendToUser(userID, message)
		slog.Debug("Event sent to user",
			"event", event.Name,
			"user_id", userID)
	} else {
		s.hub.Broadcast(message)
		slog.Debug("Event broadcasted", "event", event.Name)
	}

	return nil
}

// extractUserID attempts to extract user ID from various payload formats.
func extractUserID(payload interface{}) string {
	if payload == nil {
		return ""
	}

	// Try to extract from map
	if m, ok := payload.(map[string]interface{}); ok {
		// Try common field names
		if userID, ok := m["user_id"].(string); ok {
			return userID
		}

		if userID, ok := m["userId"].(string); ok {
			return userID
		}

		if userID, ok := m["UserID"].(string); ok {
			return userID
		}
	}

	// Try to extract from struct with reflection could be added here
	// For now, modules should include user_id in their payload maps

	return ""
}

