// Package events provides event bus implementations.
//
// This file contains a complete example implementation of Valkey Pub/Sub event bus.
// To use this implementation, uncomment the code and add a compatible dependency:
//
//	go get github.com/valkey-io/valkey-go
//	OR
//	go get github.com/redis/go-redis/v9
package events

/*
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/redis/go-redis/v9"
)

// ValkeyBusExample is a complete Valkey Pub/Sub event bus implementation.
type ValkeyBusExample struct {
	config   DistributedBusConfig
	handlers map[string][]Handler
	client   *redis.Client
	pubsub   *redis.PubSub
	mu       sync.RWMutex
}

// NewValkeyBusExample creates a new Valkey Pub/Sub event bus.
func NewValkeyBusExample(cfg DistributedBusConfig) (*ValkeyBusExample, error) {
	if len(cfg.Brokers) == 0 {
		return nil, errors.New("at least one Valkey broker address required")
	}

	client := redis.NewClient(&redis.Options{
		Addr: cfg.Brokers[0], // Use first broker as Valkey address
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Valkey: %w", err)
	}

	// Create pubsub
	pubsub := client.Subscribe(ctx, cfg.Topic)

	return &ValkeyBusExample{
		config:   cfg,
		handlers: make(map[string][]Handler),
		client:   client,
		pubsub:   pubsub,
	}, nil
}

// Subscribe registers a handler for a specific event name.
func (rb *ValkeyBusExample) Subscribe(eventName string, handler Handler) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.handlers[eventName] = append(rb.handlers[eventName], handler)
}

// Publish sends an event to Valkey Pub/Sub.
func (rb *ValkeyBusExample) Publish(ctx context.Context, event Event) error {
	// Serialize event
	eventData := map[string]interface{}{
		"name":    event.Name,
		"payload": event.Payload,
	}

	eventBytes, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Publish to Valkey channel
	if err := rb.client.Publish(ctx, rb.config.Topic, eventBytes).Err(); err != nil {
		slog.ErrorContext(ctx, "Failed to publish event to Valkey",
			"event", event.Name,
			"error", err,
		)
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// Start begins consuming messages from Valkey and dispatching to handlers.
func (rb *ValkeyBusExample) Start(ctx context.Context) error {
	ch := rb.pubsub.Channel()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-ch:
				rb.dispatch(ctx, msg)
			}
		}
	}()

	return nil
}

// dispatch dispatches a Valkey message to registered handlers.
func (rb *ValkeyBusExample) dispatch(ctx context.Context, msg *redis.Message) {
	// Deserialize event
	var eventData struct {
		Name    string      `json:"name"`
		Payload interface{} `json:"payload"`
	}

	if err := json.Unmarshal([]byte(msg.Payload), &eventData); err != nil {
		slog.ErrorContext(ctx, "Failed to unmarshal event",
			"error", err,
		)
		return
	}

	event := Event{
		Name:    eventData.Name,
		Payload: eventData.Payload,
	}

	// Get handlers
	rb.mu.RLock()
	handlers, ok := rb.handlers[event.Name]
	rb.mu.RUnlock()

	if !ok {
		return
	}

	// Dispatch to all handlers
	for _, handler := range handlers {
		go func(h Handler) {
			if err := h(ctx, event); err != nil {
				slog.ErrorContext(ctx, "Event handler error",
					"event", event.Name,
					"error", err,
				)
			}
		}(handler)
	}
}

// Close closes the Valkey connections.
func (rb *ValkeyBusExample) Close() error {
	var errs []error

	if rb.pubsub != nil {
		if err := rb.pubsub.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if rb.client != nil {
		if err := rb.client.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
*/
