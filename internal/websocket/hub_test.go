package websocket

import (
	"context"
	"testing"
	"time"
)

func TestNewHub(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	if hub == nil {
		t.Fatal("Expected hub to be created")
	}

	if hub.clients == nil {
		t.Error("Expected clients map to be initialized")
	}

	if hub.userClients == nil {
		t.Error("Expected userClients map to be initialized")
	}
}

func TestHub_RegisterClient(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	go hub.Run()
	defer hub.Stop()

	mockClient := &Client{
		id:     "client-1",
		userID: "user-1",
		hub:    hub,
		send:   make(chan *Message, 256),
	}

	hub.register <- mockClient

	// Give it time to process
	time.Sleep(50 * time.Millisecond)

	if hub.GetTotalConnections() != 1 {
		t.Errorf("Expected 1 connection, got %d", hub.GetTotalConnections())
	}

	if hub.GetConnectedUsers() != 1 {
		t.Errorf("Expected 1 connected user, got %d", hub.GetConnectedUsers())
	}
}

func TestHub_UnregisterClient(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	go hub.Run()
	defer hub.Stop()

	mockClient := &Client{
		id:     "client-1",
		userID: "user-1",
		hub:    hub,
		send:   make(chan *Message, 256),
	}

	hub.register <- mockClient

	time.Sleep(50 * time.Millisecond)

	hub.unregister <- mockClient

	time.Sleep(50 * time.Millisecond)

	if hub.GetTotalConnections() != 0 {
		t.Errorf("Expected 0 connections, got %d", hub.GetTotalConnections())
	}

	if hub.GetConnectedUsers() != 0 {
		t.Errorf("Expected 0 connected users, got %d", hub.GetConnectedUsers())
	}
}

func TestHub_Broadcast(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	go hub.Run()
	defer hub.Stop()

	// Create mock clients
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

	// Broadcast a message
	testMessage := &Message{
		Type:    "test.event",
		Payload: map[string]string{"data": "test"},
	}

	hub.Broadcast(testMessage)
	time.Sleep(50 * time.Millisecond)

	// Check both clients received the message
	select {
	case msg := <-client1.send:
		if msg.Type != "test.event" {
			t.Errorf("Expected message type 'test.event', got '%s'", msg.Type)
		}
	default:
		t.Error("Expected message in client1 send channel")
	}

	select {
	case msg := <-client2.send:
		if msg.Type != "test.event" {
			t.Errorf("Expected message type 'test.event', got '%s'", msg.Type)
		}
	default:
		t.Error("Expected message in client2 send channel")
	}
}

func TestHub_SendToUser(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	go hub.Run()
	defer hub.Stop()

	client1, client2, client3 := createTestClients(hub)

	hub.register <- client1

	hub.register <- client2

	hub.register <- client3

	time.Sleep(50 * time.Millisecond)

	// Send message to user-1
	testMessage := &Message{
		Type:    "user.notification",
		Payload: map[string]string{"message": "Hello user-1"},
		UserID:  "user-1",
	}

	hub.SendToUser("user-1", testMessage)
	time.Sleep(50 * time.Millisecond)

	assertClientReceivedMessage(t, client1, "client1")
	assertClientReceivedMessage(t, client2, "client2")
	assertClientDidNotReceiveMessage(t, client3, "client3")
}

func createTestClients(hub *Hub) (*Client, *Client, *Client) {
	client1 := &Client{
		id:     "client-1",
		userID: "user-1",
		hub:    hub,
		send:   make(chan *Message, 256),
	}

	client2 := &Client{
		id:     "client-2",
		userID: "user-1",
		hub:    hub,
		send:   make(chan *Message, 256),
	}

	client3 := &Client{
		id:     "client-3",
		userID: "user-2",
		hub:    hub,
		send:   make(chan *Message, 256),
	}

	return client1, client2, client3
}

func assertClientReceivedMessage(t *testing.T, client *Client, name string) {
	t.Helper()

	select {
	case <-client.send:
	default:
		t.Errorf("Expected message in %s send channel", name)
	}
}

func assertClientDidNotReceiveMessage(t *testing.T, client *Client, name string) {
	t.Helper()

	select {
	case <-client.send:
		t.Errorf("%s should not have received the message", name)
	default:
		// Expected: no message
	}
}

func TestHub_MultipleConnectionsSameUser(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	go hub.Run()
	defer hub.Stop()

	// Register multiple connections for same user
	for i := 0; i < 3; i++ {
		client := &Client{
			id:     "client-" + string(rune(i)),
			userID: "user-1",
			hub:    hub,
			send:   make(chan *Message, 256),
		}
		hub.register <- client
	}

	time.Sleep(50 * time.Millisecond)

	if hub.GetTotalConnections() != 3 {
		t.Errorf("Expected 3 connections, got %d", hub.GetTotalConnections())
	}

	if hub.GetConnectedUsers() != 1 {
		t.Errorf("Expected 1 unique user, got %d", hub.GetConnectedUsers())
	}
}

func TestHub_GracefulShutdown(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)

	go hub.Run()

	mockClient := &Client{
		id:     "client-1",
		userID: "user-1",
		hub:    hub,
		send:   make(chan *Message, 256),
	}

	hub.register <- mockClient

	time.Sleep(50 * time.Millisecond)

	hub.Stop()

	time.Sleep(50 * time.Millisecond)

	// After shutdown, connections should be cleared
	if hub.GetTotalConnections() != 0 {
		t.Errorf("Expected 0 connections after shutdown, got %d", hub.GetTotalConnections())
	}
}
