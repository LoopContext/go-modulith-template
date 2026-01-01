// Package events provides a simple event bus for in-process communication.
package events

import (
	"context"
	"log/slog"
	"sync"
)

// Event represents a simple data carrier
type Event struct {
	Name    string
	Payload interface{}
}

// Handler defines a function that processes an event
type Handler func(ctx context.Context, event Event) error

// ErrorHandler defines a function that handles errors from event handlers
type ErrorHandler func(ctx context.Context, event Event, err error)

// Bus is a simple in-process event distributor
type Bus struct {
	mu           sync.RWMutex
	handlers     map[string][]Handler
	errorHandler ErrorHandler
}

// NewBus creates a new internal event bus
func NewBus() *Bus {
	return &Bus{
		handlers: make(map[string][]Handler),
		errorHandler: func(ctx context.Context, event Event, err error) {
			slog.ErrorContext(ctx, "Event handler error",
				"event", event.Name,
				"error", err,
			)
		},
	}
}

// SetErrorHandler sets a custom error handler for the event bus.
func (b *Bus) SetErrorHandler(handler ErrorHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.errorHandler = handler
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
	errorHandler := b.errorHandler
	b.mu.RUnlock()

	if !ok {
		return
	}

	for _, handler := range handlers {
		// Run in goroutine for non-blocking decoupling
		go func(h Handler) {
			if err := h(ctx, event); err != nil && errorHandler != nil {
				errorHandler(ctx, event, err)
			}
		}(handler)
	}
}
