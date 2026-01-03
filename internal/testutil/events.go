// Package testutil provides testing utilities including testcontainers setup.
package testutil

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/cmelgarejo/go-modulith-template/internal/events"
)

// EventCollector collects events for testing.
type EventCollector struct {
	mu     sync.RWMutex
	events []events.Event
	ch     chan events.Event
}

// NewEventCollector creates a new event collector.
func NewEventCollector() *EventCollector {
	return &EventCollector{
		events: make([]events.Event, 0),
		ch:     make(chan events.Event, 100),
	}
}

// Subscribe subscribes to events on the bus and collects them.
func (c *EventCollector) Subscribe(bus *events.Bus, eventName string) {
	bus.Subscribe(eventName, func(_ context.Context, e events.Event) error {
		c.mu.Lock()
		c.events = append(c.events, e)
		c.mu.Unlock()

		select {
		case c.ch <- e:
		default:
			// Channel full, drop event
		}

		return nil
	})
}

// WaitForEvent waits for an event with the given timeout.
func (c *EventCollector) WaitForEvent(timeout time.Duration) (events.Event, error) {
	select {
	case event := <-c.ch:
		return event, nil
	case <-time.After(timeout):
		return events.Event{}, errors.New("timeout waiting for event")
	}
}

// AllEvents returns all collected events.
func (c *EventCollector) AllEvents() []events.Event {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]events.Event, len(c.events))
	copy(result, c.events)

	return result
}

// Clear clears all collected events.
func (c *EventCollector) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.events = c.events[:0]
	// Drain any pending events from the channel
	for {
		select {
		case <-c.ch:
		default:
			return
		}
	}
}

// Count returns the number of collected events.
func (c *EventCollector) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.events)
}
