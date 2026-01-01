// Package events provides event bus implementations.
package events

import (
	"context"
	"errors"
	"log/slog"
	"sync"
)

// DistributedBusConfig holds configuration for distributed event buses.
type DistributedBusConfig struct {
	// Brokers is a list of broker addresses (e.g., ["kafka:9092"]).
	Brokers []string
	// Topic is the topic/queue name for events.
	Topic string
	// ConsumerGroup is the consumer group ID.
	ConsumerGroup string
	// EnableTLS enables TLS for connections.
	EnableTLS bool
	// TLSCertFile is the path to the TLS certificate.
	TLSCertFile string
	// TLSKeyFile is the path to the TLS key.
	TLSKeyFile string
}

// DefaultDistributedBusConfig returns default configuration.
func DefaultDistributedBusConfig() DistributedBusConfig {
	return DistributedBusConfig{
		Brokers:       []string{"localhost:9092"},
		Topic:         "events",
		ConsumerGroup: "modulith",
		EnableTLS:     false,
	}
}

// KafkaBus is a Kafka-backed event bus implementation.
// NOTE: This is a stub implementation. To use Kafka, add the segmentio/kafka-go dependency:
//
//	go get github.com/segmentio/kafka-go
//
// Then implement the methods using the Kafka client.
type KafkaBus struct {
	config   DistributedBusConfig
	handlers map[string][]Handler
	mu       sync.RWMutex
	// writer *kafka.Writer // Uncomment when adding kafka-go dependency
	// reader *kafka.Reader // Uncomment when adding kafka-go dependency
}

// NewKafkaBus creates a new Kafka event bus.
//
// Example usage with kafka-go:
//
//	import "github.com/segmentio/kafka-go"
//
//	cfg := events.DefaultDistributedBusConfig()
//	cfg.Brokers = []string{"kafka:9092"}
//	bus, err := events.NewKafkaBus(cfg)
func NewKafkaBus(cfg DistributedBusConfig) (*KafkaBus, error) {
	// TODO: Implement with segmentio/kafka-go
	// writer := &kafka.Writer{
	// 	Addr:     kafka.TCP(cfg.Brokers...),
	// 	Topic:    cfg.Topic,
	// 	Balancer: &kafka.LeastBytes{},
	// }
	//
	// reader := kafka.NewReader(kafka.ReaderConfig{
	// 	Brokers: cfg.Brokers,
	// 	Topic:   cfg.Topic,
	// 	GroupID: cfg.ConsumerGroup,
	// })
	return &KafkaBus{
		config:   cfg,
		handlers: make(map[string][]Handler),
		// writer: writer,
		// reader: reader,
	}, nil
}

// Subscribe registers a handler for a specific event name.
func (kb *KafkaBus) Subscribe(eventName string, handler Handler) {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	kb.handlers[eventName] = append(kb.handlers[eventName], handler)
}

// Publish sends an event to Kafka.
func (kb *KafkaBus) Publish(_ context.Context, _ Event) {
	// TODO: Implement with segmentio/kafka-go
	// err := kb.writer.WriteMessages(ctx, kafka.Message{
	// 	Key:   []byte(event.Name),
	// 	Value: // serialize event.Payload,
	// })
	// if err != nil {
	// 	slog.ErrorContext(ctx, "Failed to publish event to Kafka", "event", event.Name, "error", err)
	// }
	slog.Warn("KafkaBus.Publish not implemented: add github.com/segmentio/kafka-go dependency")
}

// Close closes the Kafka connections.
func (kb *KafkaBus) Close() error {
	// TODO: Implement with segmentio/kafka-go
	// if err := kb.writer.Close(); err != nil {
	// 	return err
	// }
	// return kb.reader.Close()
	return nil
}

// Start begins consuming messages from Kafka and dispatching to handlers.
func (kb *KafkaBus) Start(_ context.Context) error {
	// TODO: Implement with segmentio/kafka-go
	// go func() {
	// 	for {
	// 		m, err := kb.reader.ReadMessage(ctx)
	// 		if err != nil {
	// 			if errors.Is(err, context.Canceled) {
	// 				return
	// 			}
	// 			slog.ErrorContext(ctx, "Failed to read Kafka message", "error", err)
	// 			continue
	// 		}
	// 		kb.dispatch(ctx, m)
	// 	}
	// }()
	return errors.New("KafkaBus.Start not implemented: add github.com/segmentio/kafka-go dependency")
}

// CompositeEventBus wraps both local and distributed event buses.
// Events are published to both buses, allowing local handlers to run immediately
// while also persisting events for distributed consumption.
type CompositeEventBus struct {
	local       *Bus
	distributed EventBus
}

// NewCompositeEventBus creates a composite event bus that publishes to both
// a local in-memory bus and a distributed bus.
func NewCompositeEventBus(local *Bus, distributed EventBus) *CompositeEventBus {
	return &CompositeEventBus{
		local:       local,
		distributed: distributed,
	}
}

// Subscribe registers a handler on the local bus.
// For distributed handlers, use the distributed bus directly.
func (cb *CompositeEventBus) Subscribe(eventName string, handler Handler) {
	cb.local.Subscribe(eventName, handler)
}

// Publish sends the event to both local and distributed buses.
func (cb *CompositeEventBus) Publish(ctx context.Context, event Event) {
	// Publish locally first for immediate handling
	cb.local.Publish(ctx, event)

	// Then publish to distributed bus if available
	if cb.distributed != nil {
		cb.distributed.Publish(ctx, event)
	}
}

// Close closes both buses.
func (cb *CompositeEventBus) Close() error {
	var errs []error

	if err := cb.local.Close(); err != nil {
		errs = append(errs, err)
	}

	if cb.distributed != nil {
		if err := cb.distributed.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// LocalBus returns the local in-memory bus.
func (cb *CompositeEventBus) LocalBus() *Bus {
	return cb.local
}

