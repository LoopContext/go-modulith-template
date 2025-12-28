package events

import (
	"context"
	"testing"
	"time"
)

func TestEventBus(t *testing.T) {
	bus := NewBus()
	ctx := context.Background()
	eventName := "test.event"
	payload := "hello world"

	received := make(chan bool)

	bus.Subscribe(eventName, func(_ context.Context, event Event) error {
		if event.Payload != payload {
			t.Errorf("expected payload %s, got %s", payload, event.Payload)
		}

		received <- true

		return nil
	})

	bus.Publish(ctx, Event{Name: eventName, Payload: payload})

	select {
	case <-received:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for event handler")
	}
}
