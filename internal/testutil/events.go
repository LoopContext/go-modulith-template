// Package testutil provides testing utilities including testcontainers setup.
package testutil

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/LoopContext/go-modulith-template/internal/events"
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

// WaitForEventByName waits until the named event is observed or the timeout expires.
func (c *EventCollector) WaitForEventByName(eventName string, timeout time.Duration) (events.Event, error) {
	if event, ok := c.EventByName(eventName); ok {
		return event, nil
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case event := <-c.ch:
			if event.Name == eventName {
				return event, nil
			}
		case <-timer.C:
			return events.Event{}, fmt.Errorf("timeout waiting for event %s", eventName)
		}
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

// EventByName returns the first collected event that matches the given name.
func (c *EventCollector) EventByName(eventName string) (events.Event, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, event := range c.events {
		if event.Name == eventName {
			return event, true
		}
	}

	return events.Event{}, false
}

// HasEvent reports whether an event with the given name was collected.
func (c *EventCollector) HasEvent(eventName string) bool {
	_, ok := c.EventByName(eventName)
	return ok
}

// Count returns the number of collected events.
func (c *EventCollector) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.events)
}
