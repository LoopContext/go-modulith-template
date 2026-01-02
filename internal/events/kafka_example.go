// Package events provides event bus implementations.
//
// This file contains a complete example implementation of Kafka event bus.
// To use this implementation, uncomment the code and add the dependency:
//
//	go get github.com/segmentio/kafka-go
package events

/*
import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

// KafkaBusExample is a complete Kafka event bus implementation.
type KafkaBusExample struct {
	config   DistributedBusConfig
	handlers map[string][]Handler
	writer   *kafka.Writer
	reader   *kafka.Reader
	mu       sync.RWMutex
}

// NewKafkaBusExample creates a new Kafka event bus with full implementation.
func NewKafkaBusExample(cfg DistributedBusConfig) (*KafkaBusExample, error) {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Brokers...),
		Topic:    cfg.Topic,
		Balancer: &kafka.LeastBytes{},
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		Topic:    cfg.Topic,
		GroupID:  cfg.ConsumerGroup,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	return &KafkaBusExample{
		config:   cfg,
		handlers: make(map[string][]Handler),
		writer:   writer,
		reader:   reader,
	}, nil
}

// Subscribe registers a handler for a specific event name.
func (kb *KafkaBusExample) Subscribe(eventName string, handler Handler) {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	kb.handlers[eventName] = append(kb.handlers[eventName], handler)
}

// Publish sends an event to Kafka.
func (kb *KafkaBusExample) Publish(ctx context.Context, event Event) error {
	// Serialize event payload
	payloadBytes, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	// Create Kafka message
	message := kafka.Message{
		Key:   []byte(event.Name),
		Value: payloadBytes,
		Headers: []kafka.Header{
			{Key: "event-name", Value: []byte(event.Name)},
		},
	}

	// Write to Kafka
	if err := kb.writer.WriteMessages(ctx, message); err != nil {
		slog.ErrorContext(ctx, "Failed to publish event to Kafka",
			"event", event.Name,
			"error", err,
		)
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// Start begins consuming messages from Kafka and dispatching to handlers.
func (kb *KafkaBusExample) Start(ctx context.Context) error {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				m, err := kb.reader.ReadMessage(ctx)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						return
					}
					slog.ErrorContext(ctx, "Failed to read Kafka message", "error", err)
					continue
				}

				kb.dispatch(ctx, m)
			}
		}
	}()

	return nil
}

// dispatch dispatches a Kafka message to registered handlers.
func (kb *KafkaBusExample) dispatch(ctx context.Context, msg kafka.Message) {
	// Extract event name from headers or key
	eventName := string(msg.Key)
	if len(msg.Headers) > 0 {
		for _, h := range msg.Headers {
			if h.Key == "event-name" {
				eventName = string(h.Value)
				break
			}
		}
	}

	// Deserialize payload
	var payload interface{}
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		slog.ErrorContext(ctx, "Failed to unmarshal event payload",
			"event", eventName,
			"error", err,
		)
		return
	}

	event := Event{
		Name:    eventName,
		Payload: payload,
	}

	// Get handlers
	kb.mu.RLock()
	handlers, ok := kb.handlers[eventName]
	kb.mu.RUnlock()

	if !ok {
		return
	}

	// Dispatch to all handlers
	for _, handler := range handlers {
		go func(h Handler) {
			if err := h(ctx, event); err != nil {
				slog.ErrorContext(ctx, "Event handler error",
					"event", eventName,
					"error", err,
				)
			}
		}(handler)
	}
}

// Close closes the Kafka connections.
func (kb *KafkaBusExample) Close() error {
	var errs []error

	if err := kb.writer.Close(); err != nil {
		errs = append(errs, err)
	}

	if err := kb.reader.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
*/

