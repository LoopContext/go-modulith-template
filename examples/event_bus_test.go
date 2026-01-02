// Package examples provides example integration tests showing how to test modules end-to-end.
package examples

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/testutil"
)

// ExampleEventBusTest demonstrates comprehensive event bus testing patterns.
// This example shows:
// - Subscribing to events
// - Publishing events
// - Testing event handlers
// - Testing error handling in event handlers
// - Testing event ordering
func ExampleEventBusTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Step 1: Create event bus and collector
	bus := events.NewBus()
	collector := testutil.NewEventCollector()

	// Step 2: Subscribe to multiple events
	collector.Subscribe(bus, "user.created")
	collector.Subscribe(bus, "session.created")

	// Step 3: Test event publishing and receiving
	t.Run("PublishAndReceive", func(t *testing.T) {
		testEvent := events.Event{
			Name: "user.created",
			Payload: map[string]interface{}{
				"user_id": "test-123",
				"email":   "test@example.com",
			},
		}

		bus.Publish(ctx, testEvent)

		receivedEvent, err := collector.WaitForEvent(2 * time.Second)
		if err != nil {
			t.Fatalf("Timeout waiting for event: %v", err)
		}

		if receivedEvent.Name != "user.created" {
			t.Errorf("Expected event user.created, got %s", receivedEvent.Name)
		}

		payload, ok := receivedEvent.Payload.(map[string]interface{})
		if !ok {
			t.Fatal("Expected map payload")
		}

		if payload["user_id"] != "test-123" {
			t.Errorf("Expected user_id test-123, got %v", payload["user_id"])
		}
	})

	// Step 4: Test multiple events
	t.Run("MultipleEvents", func(t *testing.T) {
		eventsToPublish := []events.Event{
			{
				Name:    "user.created",
				Payload: map[string]interface{}{"user_id": "user-1"},
			},
			{
				Name:    "session.created",
				Payload: map[string]interface{}{"session_id": "session-1"},
			},
			{
				Name:    "user.created",
				Payload: map[string]interface{}{"user_id": "user-2"},
			},
		}

		for _, e := range eventsToPublish {
			bus.Publish(ctx, e)
		}

		// Wait for all events
		time.Sleep(500 * time.Millisecond)

		allEvents := collector.AllEvents()
		if len(allEvents) < 3 {
			t.Errorf("Expected at least 3 events, got %d", len(allEvents))
		}
	})

	// Step 5: Test error handling in event handlers
	t.Run("ErrorHandling", func(t *testing.T) {
		errorHandlerCalled := false

		bus.SetErrorHandler(func(ctx context.Context, event events.Event, err error) {
			errorHandlerCalled = true
		})

		// Subscribe a handler that returns an error
		bus.Subscribe("test.error", func(ctx context.Context, event events.Event) error {
			return errors.New("handler error")
		})

		bus.Publish(ctx, events.Event{
			Name:    "test.error",
			Payload: map[string]interface{}{},
		})

		// Wait for error handler to be called
		time.Sleep(100 * time.Millisecond)

		if !errorHandlerCalled {
			t.Error("Expected error handler to be called")
		}
	})

	// Step 6: Test event collector functionality
	t.Run("EventCollector", func(t *testing.T) {
		collector.Clear()

		bus.Publish(ctx, events.Event{
			Name:    "user.created",
			Payload: map[string]interface{}{"user_id": "collector-test"},
		})

		_, err := collector.WaitForEvent(1 * time.Second)
		if err != nil {
			t.Fatalf("Failed to receive event: %v", err)
		}

		if collector.Count() != 1 {
			t.Errorf("Expected 1 event, got %d", collector.Count())
		}

		allEvents := collector.AllEvents()
		if len(allEvents) != 1 {
			t.Errorf("Expected 1 event in AllEvents, got %d", len(allEvents))
		}
	})

	t.Log("Event bus tests complete")
}

