// Package events provides a simple event bus for in-process communication.
package events

import (
	"context"
	"sync"
)

// Event represents a simple data carrier
type Event struct {
	Name    string
	Payload interface{}
}

// Handler defines a function that processes an event
type Handler func(ctx context.Context, event Event) error

// Bus is a simple in-process event distributor
type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

// NewBus creates a new internal event bus
func NewBus() *Bus {
	return &Bus{
		handlers: make(map[string][]Handler),
	}
}

// Subscribe registers a handler for a specific event name
func (b *Bus) Subscribe(eventName string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventName] = append(b.handlers[eventName], handler)
}

// Publish broadcasts an event to all registered handlers
func (b *Bus) Publish(ctx context.Context, event Event) {
	b.mu.RLock()
	handlers, ok := b.handlers[event.Name]
	b.mu.RUnlock()

	if !ok {
		return
	}

	for _, handler := range handlers {
		// Run in goroutine for non-blocking decoupling
		go func(h Handler) {
			// We ignore the error here because it's an async handler.
			// In a more robust system, we would have a dead-letter queue or retry logic.
			_ = h(ctx, event)
		}(handler)
	}
}
