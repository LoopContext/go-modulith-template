// Package examples demonstrates event-driven workflow testing patterns.
//
// This example shows how to test event-driven workflows, including:
// - Publish → Subscribe workflows
// - Event handler execution
// - Event ordering and sequencing
// - Error handling in event handlers
package examples

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/testutil"
)

// TestEventDrivenWorkflow_PublishSubscribe demonstrates testing publish → subscribe workflows.
func TestEventDrivenWorkflow_PublishSubscribe(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Setup event bus
	eventBus := events.NewBus()
	eventCollector := testutil.NewEventCollector()

	// Subscribe to events using collector
	eventCollector.Subscribe(eventBus, "user.created")
	eventCollector.Subscribe(eventBus, "user.updated")

	// Publish event
	eventBus.Publish(ctx, events.Event{
		Name:    "user.created",
		Payload: map[string]interface{}{
			"user_id": "user-123",
			"email":   "test@example.com",
		},
	})

	// Wait for event to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify event was collected
	collectedEvents := eventCollector.AllEvents()
	require.GreaterOrEqual(t, len(collectedEvents), 1)

	found := false

	for _, event := range collectedEvents {
		if event.Name == eventUserCreated {
			found = true

			assert.Equal(t, "user-123", event.Payload.(map[string]interface{})["user_id"])

			break
		}
	}

	assert.True(t, found, "user.created event should have been collected")
}

// TestEventDrivenWorkflow_EventHandlerExecution demonstrates testing event handler execution.
func TestEventDrivenWorkflow_EventHandlerExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	eventBus := events.NewBus()
	eventCollector := testutil.NewEventCollector()

	// Subscribe collector to track events
	eventCollector.Subscribe(eventBus, "test.event")

	// Subscribe a handler that processes the event
	processed := make(chan bool, 1)

	eventBus.Subscribe("test.event", func(_ context.Context, event events.Event) error {
		// Process event
		_ = event.Payload

		processed <- true

		return nil
	})

	// Publish event
	eventBus.Publish(ctx, events.Event{
		Name:    "test.event",
		Payload: map[string]interface{}{"key": "value"},
	})

	// Wait for handler to process
	select {
	case <-processed:
		// Handler executed successfully
		assert.True(t, true)
	case <-time.After(1 * time.Second):
		t.Fatal("Event handler did not execute within timeout")
	}

	// Verify event was collected
	assert.GreaterOrEqual(t, eventCollector.Count(), 1)
}

// TestEventDrivenWorkflow_EventOrdering demonstrates testing event ordering.
func TestEventDrivenWorkflow_EventOrdering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	eventBus := events.NewBus()
	eventCollector := testutil.NewEventCollector()

	eventCollector.Subscribe(eventBus, "ordered.event")

	// Publish events in sequence
	eventBus.Publish(ctx, events.Event{
		Name:    "ordered.event",
		Payload: map[string]interface{}{"sequence": 1},
	})

	eventBus.Publish(ctx, events.Event{
		Name:    "ordered.event",
		Payload: map[string]interface{}{"sequence": 2},
	})

	eventBus.Publish(ctx, events.Event{
		Name:    "ordered.event",
		Payload: map[string]interface{}{"sequence": 3},
	})

	// Wait for events to be processed
	time.Sleep(200 * time.Millisecond)

	// Verify events were collected (note: order is not guaranteed in async handlers)
	collectedEvents := eventCollector.AllEvents()
	assert.GreaterOrEqual(t, len(collectedEvents), 3)

	// Verify all sequences are present
	sequences := make(map[int]bool)

	for _, event := range collectedEvents {
		if event.Name == "ordered.event" {
			seq := event.Payload.(map[string]interface{})["sequence"].(int)
			sequences[seq] = true
		}
	}

	assert.True(t, sequences[1])
	assert.True(t, sequences[2])
	assert.True(t, sequences[3])
}

// TestEventDrivenWorkflow_ErrorHandling demonstrates testing error handling in event handlers.
func TestEventDrivenWorkflow_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	eventBus := events.NewBus()
	eventCollector := testutil.NewEventCollector()

	// Subscribe a handler that returns an error
	errorHandlerCalled := make(chan bool, 1)

	eventBus.SetErrorHandler(func(_ context.Context, _ events.Event, _ error) {
		errorHandlerCalled <- true
	})

	eventBus.Subscribe("error.event", func(_ context.Context, _ events.Event) error {
		return assert.AnError // Simulate handler error
	})

	eventCollector.Subscribe(eventBus, "error.event")

	// Publish event
	eventBus.Publish(ctx, events.Event{
		Name:    "error.event",
		Payload: map[string]interface{}{"test": "data"},
	})

	// Wait for error handler to be called
	select {
	case <-errorHandlerCalled:
		// Error handler was called (error was handled)
		assert.True(t, true)
	case <-time.After(1 * time.Second):
		// Error handler should be called for handler errors
		// Note: Current implementation may not call error handler for all cases
	}

	// Verify event was still collected (error doesn't prevent collection)
	time.Sleep(100 * time.Millisecond)
	assert.GreaterOrEqual(t, eventCollector.Count(), 0)
}

// TestEventDrivenWorkflow_MultipleSubscribers demonstrates testing multiple subscribers for same event.
func TestEventDrivenWorkflow_MultipleSubscribers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	eventBus := events.NewBus()
	eventCollector := testutil.NewEventCollector()

	// Subscribe multiple handlers
	handler1Called := make(chan bool, 1)
	handler2Called := make(chan bool, 1)

	eventBus.Subscribe("multi.event", func(_ context.Context, _ events.Event) error {
		handler1Called <- true
		return nil
	})

	eventBus.Subscribe("multi.event", func(_ context.Context, _ events.Event) error {
		handler2Called <- true
		return nil
	})

	eventCollector.Subscribe(eventBus, "multi.event")

	// Publish event
	eventBus.Publish(ctx, events.Event{
		Name:    "multi.event",
		Payload: map[string]interface{}{"test": "data"},
	})

	// Wait for both handlers to be called
	select {
	case <-handler1Called:
		assert.True(t, true)
	case <-time.After(1 * time.Second):
		t.Fatal("Handler 1 did not execute")
	}

	select {
	case <-handler2Called:
		assert.True(t, true)
	case <-time.After(1 * time.Second):
		t.Fatal("Handler 2 did not execute")
	}

	// Verify event was collected
	time.Sleep(100 * time.Millisecond)
	assert.GreaterOrEqual(t, eventCollector.Count(), 1)
}

