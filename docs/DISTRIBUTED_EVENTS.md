# Distributed Events Guide

This guide explains how to use distributed events in the modulith template, when to use them, and how to migrate from in-process events to distributed events.

## Table of Contents

1. [When to Use Distributed Events](#when-to-use-distributed-events)
2. [Architecture Overview](#architecture-overview)
3. [Implementation Options](#implementation-options)
4. [Migration Guide](#migration-guide)
5. [Best Practices](#best-practices)

## When to Use Distributed Events

Distributed events are necessary when:

### Horizontal Scaling Requirements

-   **Multiple Instances**: You're running multiple instances of your application
-   **Load Balancing**: Requests are distributed across instances
-   **Event Delivery**: Events need to reach all instances, not just the one that published them

### Multi-Instance Deployments

-   **Kubernetes**: Multiple pods running the same service
-   **Cloud Deployments**: Auto-scaling groups with multiple instances
-   **High Availability**: Need events to work even if one instance fails

### Event Persistence Needs

-   **Durability**: Events must survive instance restarts
-   **Replay**: Need to replay events for recovery or debugging
-   **Audit**: Events must be stored for compliance or auditing

### Cross-Service Communication

-   **Microservices**: When modules are split into separate services
-   **External Systems**: Events need to reach external services
-   **Event Sourcing**: Using events as the source of truth

## Architecture Overview

### In-Process vs Distributed Comparison

#### In-Process Events (Default)

```
┌─────────────────────────────────────┐
│         Single Instance             │
│  ┌──────────┐      ┌──────────┐    │
│  │ Module A │─────▶│ Event Bus│    │
│  └──────────┘      └──────────┘    │
│       │                  │          │
│       │                  ▼          │
│       │            ┌──────────┐    │
│       └───────────▶│ Module B │    │
│                    └──────────┘    │
└─────────────────────────────────────┘
```

**Characteristics:**

-   Events are published and consumed within the same process
-   Fast (no network overhead)
-   Simple to implement
-   Limited to single instance

#### Distributed Events

```
┌──────────────┐         ┌──────────────┐
│  Instance 1  │         │  Instance 2  │
│  ┌────────┐  │         │  ┌────────┐  │
│  │Module A│  │         │  │Module B│  │
│  └───┬────┘  │         │  └───┬────┘  │
│      │       │         │      │       │
└──────┼───────┘         └──────┼───────┘
       │                        │
       │    ┌──────────────┐    │
       └───▶│ Message Broker│◀──┘
            │  (Kafka/Redis) │
            └──────────────┘
```

**Characteristics:**

-   Events travel over the network
-   Works across multiple instances
-   Can persist events
-   Slightly higher latency

### Composite Event Bus Pattern

The template supports a **Composite Event Bus** that publishes to both local and distributed buses:

```go
localBus := events.NewBus()
distributedBus := events.NewKafkaBus(config)
compositeBus := events.NewCompositeEventBus(localBus, distributedBus)

// Events are published to both:
// 1. Local bus (immediate, in-process)
// 2. Distributed bus (persistent, cross-instance)
compositeBus.Publish(ctx, event)
```

**Benefits:**

-   Local handlers run immediately (low latency)
-   Events are also persisted for distributed consumption
-   Gradual migration path

## Implementation Options

### Kafka Implementation

Kafka is ideal for high-throughput, persistent event streaming.

#### Setup

1. Add dependency:

```bash
go get github.com/segmentio/kafka-go
```

2. Configure Kafka bus:

```go
import "github.com/cmelgarejo/go-modulith-template/internal/events"

cfg := events.DefaultDistributedBusConfig()
cfg.Brokers = []string{"kafka:9092"}
cfg.Topic = "modulith-events"
cfg.ConsumerGroup = "modulith-service"

kafkaBus, err := events.NewKafkaBus(cfg)
if err != nil {
    log.Fatal(err)
}
```

3. Start consuming:

```go
ctx := context.Background()
if err := kafkaBus.Start(ctx); err != nil {
    log.Fatal(err)
}
```

#### Complete Example

See `internal/events/kafka_example.go` for a complete implementation example.

### Redis Pub/Sub Implementation

Redis Pub/Sub is simpler and suitable for lower-volume, real-time events.

#### Setup

1. Add dependency:

```bash
go get github.com/redis/go-redis/v9
```

2. Configure Redis bus:

```go
import "github.com/cmelgarejo/go-modulith-template/internal/events"

cfg := events.DefaultDistributedBusConfig()
cfg.Brokers = []string{"redis:6379"}
cfg.Topic = "modulith-events"

redisBus, err := events.NewRedisBus(cfg)
if err != nil {
    log.Fatal(err)
}
```

#### Complete Example

See `internal/events/redis_example.go` for a complete implementation example.

### AWS SNS/SQS Implementation (Optional)

For AWS deployments, you can use SNS for publishing and SQS for consuming.

**Note:** This is not included in the base template but can be implemented following the same pattern.

## Migration Guide

### Step-by-Step Migration

#### Phase 1: Prepare (No Changes)

1. Ensure all events use typed constants from `internal/events/types.go`
2. Verify event payloads are serializable (JSON-compatible)
3. Test current in-process event flow

#### Phase 2: Add Distributed Bus (Non-Breaking)

1. Add distributed bus alongside existing in-process bus:

```go
// In cmd/server/setup/registry.go when creating the registry
localBus := events.NewBus()
distributedBus := events.NewKafkaBus(kafkaConfig) // or Redis
compositeBus := events.NewCompositeEventBus(localBus, distributedBus)

reg := registry.New(
    registry.WithEventBus(compositeBus),
    // ... other options
)
```

2. Start distributed bus consumer:

```go
if err := distributedBus.Start(ctx); err != nil {
    log.Fatal(err)
}
```

3. Events now publish to both buses (backward compatible)

#### Phase 3: Test Distributed Events

1. Deploy multiple instances
2. Publish events from one instance
3. Verify events are received by all instances
4. Monitor for latency and errors

#### Phase 4: Optimize (Optional)

1. Remove local bus if all handlers are distributed
2. Adjust consumer group settings for your workload
3. Implement dead letter queues for failed events

### Testing Distributed Events

Use the test utilities to test distributed events:

```go
func TestDistributedEvents(t *testing.T) {
    // Set up test Kafka/Redis
    // Create distributed bus
    // Publish events
    // Verify delivery across instances
}
```

## Best Practices

### Event Schema Versioning

Events should include version information:

```go
type Event struct {
    Name    string
    Version string  // e.g., "v1"
    Payload interface{}
}
```

**Migration Strategy:**

-   Support multiple versions during transition
-   Document breaking changes
-   Provide migration scripts if needed

### Error Handling and Retries

**Retry Strategy:**

-   Implement exponential backoff
-   Set maximum retry attempts
-   Use dead letter queues for failed events

**Example:**

```go
bus.Subscribe("user.created", func(ctx context.Context, e events.Event) error {
    // Retry logic with exponential backoff
    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
        if err := processEvent(ctx, e); err == nil {
            return nil
        }
        time.Sleep(time.Duration(i+1) * time.Second)
    }
    // Send to dead letter queue
    return sendToDLQ(ctx, e)
})
```

### Dead Letter Queues

For events that fail after all retries:

```go
// Configure DLQ in your bus implementation
dlqBus := events.NewKafkaBus(events.DistributedBusConfig{
    Topic: "modulith-events-dlq",
    // ... other config
})
```

### Event Ordering Guarantees

**Kafka:**

-   Events with the same key are ordered within a partition
-   Use event key (e.g., user_id) for ordering

**Redis Pub/Sub:**

-   No ordering guarantees
-   Use Kafka if ordering is critical

### Monitoring and Observability

**Metrics to Track:**

-   Event publish rate
-   Event consumption rate
-   Processing latency
-   Error rate
-   Dead letter queue size

**Example:**

```go
// Add metrics to event bus
metrics.RecordEventPublished(event.Name)
metrics.RecordEventProcessed(event.Name, duration)
```

### Performance Considerations

**Batching:**

-   Batch events when possible
-   Adjust batch size based on throughput

**Partitioning:**

-   Use appropriate partition keys for Kafka
-   Distribute load evenly

**Connection Pooling:**

-   Reuse connections to message broker
-   Configure appropriate pool sizes

## Troubleshooting

### Events Not Received

1. **Check Consumer Group**: Ensure consumer group is configured correctly
2. **Verify Subscriptions**: Confirm handlers are subscribed to correct event names
3. **Check Network**: Verify connectivity to message broker
4. **Review Logs**: Check for error messages in event handlers

### High Latency

1. **Batch Size**: Increase batch size for higher throughput
2. **Partitioning**: Adjust partition strategy
3. **Connection Pool**: Increase connection pool size
4. **Network**: Check network latency to broker

### Event Loss

1. **Acknowledgment**: Ensure events are acknowledged after processing
2. **Persistence**: Verify broker persistence is enabled
3. **Retries**: Implement retry logic
4. **Monitoring**: Set up alerts for event loss

## References

-   [Kafka Documentation](https://kafka.apache.org/documentation/)
-   [Redis Pub/Sub Documentation](https://redis.io/docs/manual/pubsub/)
-   [Event Sourcing Patterns](https://martinfowler.com/eaaDev/EventSourcing.html)
-   See `internal/events/distributed.go` for implementation details
