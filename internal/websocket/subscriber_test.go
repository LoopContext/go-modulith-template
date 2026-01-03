package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/cmelgarejo/go-modulith-template/internal/events"
)

func TestNewSubscriber(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)
	eventBus := events.NewBus()

	subscriber := NewSubscriber(hub, eventBus)

	if subscriber == nil {
		t.Fatal("Expected subscriber to be created")
	}

	if subscriber.hub != hub {
		t.Error("Expected subscriber hub to match")
	}

	if subscriber.eventBus != eventBus {
		t.Error("Expected subscriber eventBus to match")
	}
}

func TestSubscriber_HandleEventBroadcast(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)
	eventBus := events.NewBus()

	go hub.Run()
	defer hub.Stop()

	subscriber := NewSubscriber(hub, eventBus)
	subscriber.Subscribe()

	// Create a mock client
	mockClient := &Client{
		id:     "client-1",
		userID: "user-1",
		hub:    hub,
		send:   make(chan *Message, 256),
	}

	hub.register <- mockClient

	time.Sleep(50 * time.Millisecond)

	// Publish an event that should be broadcasted
	testEvent := events.Event{
		Name: "order.created",
		Payload: map[string]interface{}{
			"order_id": "123",
			"amount":   100.50,
		},
	}

	eventBus.Publish(ctx, testEvent)
	time.Sleep(100 * time.Millisecond)

	// Check if client received the message
	select {
	case msg := <-mockClient.send:
		if msg.Type != "order.created" {
			t.Errorf("Expected message type 'order.created', got '%s'", msg.Type)
		}

		payload, ok := msg.Payload.(map[string]interface{})
		if !ok {
			t.Fatal("Expected payload to be map")
		}

		if payload["order_id"] != "123" {
			t.Errorf("Expected order_id '123', got '%v'", payload["order_id"])
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Timeout waiting for message")
	}
}

func TestSubscriber_HandleEventWithUserID(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)
	eventBus := events.NewBus()

	go hub.Run()
	defer hub.Stop()

	subscriber := NewSubscriber(hub, eventBus)
	subscriber.Subscribe()

	// Create clients for different users
	client1 := &Client{
		id:     "client-1",
		userID: "user-1",
		hub:    hub,
		send:   make(chan *Message, 256),
	}

	client2 := &Client{
		id:     "client-2",
		userID: "user-2",
		hub:    hub,
		send:   make(chan *Message, 256),
	}

	hub.register <- client1

	hub.register <- client2

	time.Sleep(50 * time.Millisecond)

	// Publish event targeted to user-1
	testEvent := events.Event{
		Name: "notification.sent",
		Payload: map[string]interface{}{
			"user_id": "user-1",
			"message": "You have a new notification",
		},
	}

	eventBus.Publish(ctx, testEvent)
	time.Sleep(100 * time.Millisecond)

	// Check client1 received the message
	select {
	case msg := <-client1.send:
		if msg.Type != "notification.sent" {
			t.Errorf("Expected message type 'notification.sent', got '%s'", msg.Type)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Timeout waiting for message in client1")
	}

	// Check client2 did NOT receive the message
	select {
	case <-client2.send:
		t.Error("Client2 should not have received targeted message")
	case <-time.After(100 * time.Millisecond):
		// Expected: no message
	}
}

func TestSubscriber_SubscribeToEvent(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)
	eventBus := events.NewBus()

	go hub.Run()
	defer hub.Stop()

	subscriber := NewSubscriber(hub, eventBus)

	// Subscribe to custom event
	subscriber.SubscribeToEvent("custom.event")

	mockClient := &Client{
		id:     "client-1",
		userID: "user-1",
		hub:    hub,
		send:   make(chan *Message, 256),
	}

	hub.register <- mockClient

	time.Sleep(50 * time.Millisecond)

	// Publish custom event
	testEvent := events.Event{
		Name:    "custom.event",
		Payload: map[string]string{"data": "test"},
	}

	eventBus.Publish(ctx, testEvent)
	time.Sleep(100 * time.Millisecond)

	select {
	case msg := <-mockClient.send:
		if msg.Type != "custom.event" {
			t.Errorf("Expected message type 'custom.event', got '%s'", msg.Type)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Timeout waiting for custom event")
	}
}

func TestExtractUserID(t *testing.T) {
	tests := []struct {
		name     string
		payload  interface{}
		expected string
	}{
		{
			name:     "nil payload",
			payload:  nil,
			expected: "",
		},
		{
			name: "user_id field",
			payload: map[string]interface{}{
				"user_id": "user-123",
				"data":    "test",
			},
			expected: "user-123",
		},
		{
			name: "userId field",
			payload: map[string]interface{}{
				"userId": "user-456",
				"data":   "test",
			},
			expected: "user-456",
		},
		{
			name: "UserID field",
			payload: map[string]interface{}{
				"UserID": "user-789",
				"data":   "test",
			},
			expected: "user-789",
		},
		{
			name: "no user field",
			payload: map[string]interface{}{
				"data": "test",
			},
			expected: "",
		},
		{
			name:     "non-map payload",
			payload:  "string payload",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractUserID(tt.payload)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
