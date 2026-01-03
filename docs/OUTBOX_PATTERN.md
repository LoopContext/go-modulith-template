# Outbox Pattern Guide

This guide explains how to use the transactional outbox pattern for reliable event publishing in your modulith application.

## Table of Contents

1. [Overview](#overview)
2. [When to Use Outbox](#when-to-use-outbox)
3. [How It Works](#how-it-works)
4. [Implementation](#implementation)
5. [Usage Examples](#usage-examples)
6. [Outbox Publisher Worker](#outbox-publisher-worker)
7. [Best Practices](#best-practices)
8. [Limitations](#limitations)

## Overview

The outbox pattern ensures that events are stored in the database as part of the same transaction as business data, then published asynchronously by a background worker. This guarantees that events are never lost, even if the application crashes before events are published.

**Key Benefits:**

- ✅ **Reliable Event Delivery**: Events are stored in the database (ACID guarantees)
- ✅ **No Event Loss**: Events survive application crashes
- ✅ **Multi-Instance Support**: Events can be processed by any instance
- ✅ **Transaction Consistency**: Events are part of the same transaction as business data

See [WHY_OUTBOX_AND_SAGAS.md](./WHY_OUTBOX_AND_SAGAS.md) for more details on when outbox is needed.

## When to Use Outbox

Use the outbox pattern when:

- ✅ **Multiple Instances**: Deploying multiple instances (horizontal scaling)
- ✅ **Critical Events**: Events are critical business events (not just logging)
- ✅ **Event Durability**: Events must survive application restarts
- ✅ **Reliable Delivery**: Event loss is not acceptable

**Do NOT use outbox when:**

- ❌ **Single Instance**: Single instance deployment (direct event bus is fine)
- ❌ **Fire-and-Forget Events**: Events are just notifications/logging
- ❌ **Event Loss OK**: Occasional event loss is acceptable
- ❌ **All In-Process**: All event handlers are in-process and don't need persistence

## How It Works

### Without Outbox (Direct Publishing)

```
┌──────────────┐
│   Service    │
│              │
│  1. Save to DB │
│  2. Publish Event │ ────┐
│              │         │
└──────────────┘         │
                         │
                    ┌────▼─────┐
                    │Event Bus │
                    └──────────┘

Problem: If app crashes between step 1 and 2, event is lost!
```

### With Outbox Pattern

```
┌──────────────┐
│   Service    │
│              │
│  1. Begin Transaction │
│  2. Save to DB        │
│  3. Store Event in    │
│     Outbox Table      │
│  4. Commit Transaction│ ───┐
│              │            │
└──────────────┘            │
                            │
                       ┌────▼─────┐
                       │ Outbox   │
                       │  Table   │
                       └────┬─────┘
                            │
                       ┌────▼─────┐
                       │ Publisher│
                       │  Worker  │
                       └────┬─────┘
                            │
                       ┌────▼─────┐
                       │Event Bus │
                       └──────────┘

Solution: Event is stored in DB transaction, published later by worker!
```

## Implementation

### 1. Database Schema

The outbox table schema is provided in `migrations/shared/000001_outbox_table.up.sql`:

```sql
CREATE TABLE IF NOT EXISTS outbox (
    id VARCHAR(255) PRIMARY KEY,
    event_name VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    published_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_outbox_unpublished
    ON outbox(created_at) WHERE published_at IS NULL;
```

### 2. Outbox Repository

The outbox repository is in `internal/outbox/outbox.go`:

```go
import "github.com/cmelgarejo/go-modulith-template/internal/outbox"

// Create outbox repository
outboxRepo := outbox.NewRepository(db)

// Store event in transaction
tx, _ := db.BeginTx(ctx, nil)
defer tx.Rollback()

// Store event in outbox (part of transaction)
if err := outboxRepo.Store(ctx, tx, "user.created", eventPayload); err != nil {
    return err
}

// Commit transaction (event is now in outbox)
if err := tx.Commit(); err != nil {
    return err
}
```

### 3. Outbox Publisher

The outbox publisher reads events from the outbox table and publishes them:

```go
import "github.com/cmelgarejo/go-modulith-template/internal/outbox"

// Create publisher
publisher := outbox.NewPublisher(outboxRepo, func(ctx context.Context, eventName string, payload interface{}) {
    eventBus.Publish(ctx, events.Event{
        Name:    eventName,
        Payload: payload,
    })
})

// Process events (call periodically)
if err := publisher.Process(ctx); err != nil {
    log.Error("Failed to process outbox events", "error", err)
}
```

## Usage Examples

### Example 1: Storing Event in Transaction

```go
func (s *UserService) CreateUser(ctx context.Context, req *CreateUserRequest) error {
    return s.repo.WithTx(ctx, func(txRepo Repository) error {
        // Create user
        user, err := txRepo.CreateUser(ctx, req.Email)
        if err != nil {
            return err
        }

        // Get transaction from context (helper function)
        tx := getTxFromContext(ctx) // You'll need to implement this

        // Store event in outbox (part of same transaction)
        if err := s.outboxRepo.Store(ctx, tx, "user.created", map[string]interface{}{
            "user_id": user.ID,
            "email":   user.Email,
        }); err != nil {
            return err
        }

        // Transaction commits - user and event both saved
        return nil
    })
}
```

### Example 2: Outbox Publisher Worker

Create a background worker to process outbox events:

```go
// cmd/worker/main.go or similar
func runOutboxPublisher(ctx context.Context, publisher *outbox.Publisher) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := publisher.Process(ctx); err != nil {
                log.Error("Failed to process outbox events", "error", err)
            }
        }
    }
}
```

### Example 3: Integration with Saga Pattern

Use outbox with sagas to ensure events are published reliably:

```go
saga := saga.New()

saga.AddStep("create_order",
    func(ctx context.Context) error {
        tx, _ := db.BeginTx(ctx, nil)
        defer tx.Rollback()

        // Create order
        if err := orderRepo.CreateOrder(ctx, tx, order); err != nil {
            return err
        }

        // Store event in outbox
        if err := outboxRepo.Store(ctx, tx, "order.created", order); err != nil {
            return err
        }

        return tx.Commit()
    },
    func(ctx context.Context) error {
        // Compensation: cancel order and store cancellation event
        tx, _ := db.BeginTx(ctx, nil)
        defer tx.Rollback()

        if err := orderRepo.CancelOrder(ctx, tx, orderID); err != nil {
            return err
        }

        if err := outboxRepo.Store(ctx, tx, "order.cancelled", orderID); err != nil {
            return err
        }

        return tx.Commit()
    },
)
```

## Outbox Publisher Worker

The outbox publisher should run as a background worker that periodically processes unpublished events.

### Worker Implementation

```go
package main

import (
    "context"
    "time"

    "github.com/cmelgarejo/go-modulith-template/internal/events"
    "github.com/cmelgarejo/go-modulith-template/internal/outbox"
)

func startOutboxPublisher(ctx context.Context, outboxRepo *outbox.Repository, eventBus *events.Bus) {
    publisher := outbox.NewPublisher(outboxRepo, func(ctx context.Context, eventName string, payload interface{}) {
        eventBus.Publish(ctx, events.Event{
            Name:    eventName,
            Payload: payload,
        })
    })

    // Process events every 5 seconds
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := publisher.Process(ctx); err != nil {
                log.Error("Failed to process outbox events", "error", err)
            }
        }
    }
}
```

### Running the Worker

The worker can run in:

1. **Same Process**: As a goroutine in the main application
2. **Separate Process**: As a dedicated worker service (recommended for production)
3. **Cron Job**: As a scheduled job that runs periodically

## Best Practices

### 1. Batch Processing

Process events in batches for efficiency:

```go
publisher.SetBatchSize(100) // Process up to 100 events per batch
```

### 2. Error Handling

Handle errors gracefully - failed events can be retried:

```go
// Publisher continues processing even if one event fails
// Failed events remain in outbox for retry
```

### 3. Monitoring

Monitor outbox table size and processing lag:

```sql
-- Check unpublished events
SELECT COUNT(*) FROM outbox WHERE published_at IS NULL;

-- Check processing lag
SELECT MAX(created_at) FROM outbox WHERE published_at IS NULL;
```

### 4. Cleanup

Periodically clean up old published events:

```sql
-- Delete events published more than 7 days ago
DELETE FROM outbox
WHERE published_at IS NOT NULL
  AND published_at < NOW() - INTERVAL '7 days';
```

### 5. Idempotent Handlers

Ensure event handlers are idempotent (safe to process same event multiple times):

```go
func handleUserCreated(ctx context.Context, event events.Event) error {
    userID := event.Payload["user_id"].(string)

    // Check if already processed (idempotent)
    if alreadyProcessed(userID) {
        return nil
    }

    // Process event
    return processUserCreated(userID)
}
```

## Limitations

### Simple Implementation Limitations

The current implementation has these limitations:

- ❌ **No Retries**: Failed events are not automatically retried
- ❌ **No Dead Letter Queue**: Failed events remain in outbox indefinitely
- ❌ **No Event Ordering**: Events are processed in creation order, but not guaranteed across batches
- ❌ **No Deduplication**: Same event can be processed multiple times if publisher crashes

### Production Enhancements

For production, consider:

- **Retry Logic**: Retry failed events with exponential backoff
- **Dead Letter Queue**: Move permanently failed events to DLQ
- **Event Ordering**: Ensure events are processed in order (per entity)
- **Deduplication**: Track processed events to prevent duplicates
- **Monitoring**: Add metrics and alerting for outbox health

## Integration with Existing Event Bus

The outbox pattern works alongside the existing event bus. Events stored in outbox are eventually published to the event bus, where they are handled by existing subscribers.

**No changes needed to existing event handlers** - they continue to work as before!

## Summary

- Use outbox pattern for reliable event delivery in multi-instance deployments
- Store events in outbox table as part of database transactions
- Process events asynchronously with background publisher worker
- Monitor outbox health and cleanup old events
- Ensure event handlers are idempotent

For more information, see:
- [WHY_OUTBOX_AND_SAGAS.md](./WHY_OUTBOX_AND_SAGAS.md) - When and why to use outbox
- [SAGA_PATTERNS.md](./SAGA_PATTERNS.md) - Using outbox with sagas
- `internal/outbox/outbox.go` - Implementation code

