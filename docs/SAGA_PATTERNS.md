# Saga Pattern Guide

This guide explains how to use the saga pattern for orchestrating multi-step operations with compensation support in your modulith application.

## Table of Contents

1. [Overview](#overview)
2. [When to Use Sagas](#when-to-use-sagas)
3. [Simple Saga Implementation](#simple-saga-implementation)
4. [Production Recommendations](#production-recommendations)
5. [Examples](#examples)
6. [Best Practices](#best-practices)
7. [Limitations](#limitations)

## Overview

The saga pattern is used to manage multi-step operations that span multiple modules or services. Unlike traditional database transactions (ACID), sagas provide **eventual consistency** through compensation (rollback) actions.

**Key Concepts:**

- **Saga**: A sequence of steps that must all complete successfully, or be compensated (rolled back)
- **Step**: An individual operation (e.g., create order, reserve inventory, process payment)
- **Compensation**: A rollback action that undoes a completed step (e.g., cancel order, release inventory, refund payment)

## When to Use Sagas

Use the saga pattern when:

- ✅ Operations span multiple modules/services
- ✅ You need to maintain data consistency across module boundaries
- ✅ You need compensation (rollback) for failed operations
- ✅ Operations are long-running or distributed

**Do NOT use sagas when:**

- ❌ Operations fit within a single database transaction (use `WithTx` instead)
- ❌ Operations are simple and don't require compensation
- ❌ You need strict ACID guarantees (consider 2PC or distributed transactions)

See [WHY_OUTBOX_AND_SAGAS.md](./WHY_OUTBOX_AND_SAGAS.md) for more details on when sagas are needed.

## Simple Saga Implementation

The template provides a simple, in-memory saga orchestrator in `internal/saga` for development and prototyping.

### Basic Usage

```go
import (
    "context"
    "github.com/cmelgarejo/go-modulith-template/internal/saga"
)

// Create a new saga
saga := saga.New()

// Add steps with execution and compensation functions
saga.AddStep("create_order",
    func(ctx context.Context) error {
        // Execute: Create order
        return orderService.CreateOrder(ctx, orderID)
    },
    func(ctx context.Context) error {
        // Compensate: Cancel order if later steps fail
        return orderService.CancelOrder(ctx, orderID)
    },
)

saga.AddStep("reserve_inventory",
    func(ctx context.Context) error {
        return inventoryService.ReserveItem(ctx, itemID, quantity)
    },
    func(ctx context.Context) error {
        return inventoryService.ReleaseReservation(ctx, itemID, quantity)
    },
)

saga.AddStep("process_payment",
    func(ctx context.Context) error {
        return paymentService.ProcessPayment(ctx, orderID, amount)
    },
    func(ctx context.Context) error {
        return paymentService.RefundPayment(ctx, orderID)
    },
)

// Execute the saga
if err := saga.Execute(ctx); err != nil {
    // Compensation has been executed automatically
    log.Error("Saga failed", "error", err)
    return err
}

// All steps completed successfully
log.Info("Order created successfully")
```

### How It Works

1. **Execution**: Steps are executed in order (step1 → step2 → step3)
2. **Failure Handling**: If any step fails, compensation is executed for all **completed** steps in reverse order (step2 compensate → step1 compensate)
3. **Compensation**: Compensation functions are optional - if `nil`, the step cannot be compensated

### Example: Order Creation Saga

See `examples/saga_order_creation_test.go` for a complete example showing:

- Multi-step saga (order → inventory → payment)
- Compensation on failure
- Error handling

## Production Recommendations

**⚠️ Important**: The simple saga implementation in `internal/saga` is suitable for:

- Development and prototyping
- Single-instance deployments
- Short-running operations
- Simple compensation logic

**For production use, consider:**

### Temporal (Recommended)

[Temporal](https://temporal.io/) provides:

- ✅ Durable saga orchestration (survives crashes)
- ✅ Distributed execution
- ✅ Retries and timeouts
- ✅ State management
- ✅ Workflow history and debugging
- ✅ Built-in compensation patterns

**Migration Path:**

1. Start with `internal/saga` for development
2. Migrate to Temporal when you need:
   - Multi-instance deployments
   - Long-running workflows
   - Complex retry logic
   - Workflow visibility and debugging

### Other Options

- **AWS Step Functions**: For AWS deployments
- **Zeebe**: Open-source workflow engine
- **Custom implementation**: For specific requirements

## Examples

### Example 1: Simple Saga (No Compensation)

```go
saga := saga.New()

saga.AddStep("step1", func(ctx context.Context) error {
    return service1.DoSomething(ctx)
}, nil) // No compensation needed

saga.AddStep("step2", func(ctx context.Context) error {
    return service2.DoSomething(ctx)
}, nil)

err := saga.Execute(ctx)
```

### Example 2: Saga with Compensation

```go
saga := saga.New()

saga.AddStep("create_resource",
    func(ctx context.Context) error {
        return createResource(ctx, resourceID)
    },
    func(ctx context.Context) error {
        return deleteResource(ctx, resourceID)
    },
)

saga.AddStep("send_notification",
    func(ctx context.Context) error {
        return sendNotification(ctx, resourceID)
    },
    func(ctx context.Context) error {
        // Notifications can't be "undone", but we can log the cancellation
        return logCancellation(ctx, resourceID)
    },
)

err := saga.Execute(ctx)
```

### Example 3: Saga with Outbox Pattern

When using sagas with the outbox pattern, store events in the outbox as part of saga steps:

```go
saga := saga.New()

saga.AddStep("create_order",
    func(ctx context.Context) error {
        tx, _ := db.BeginTx(ctx, nil)
        defer tx.Rollback()

        // Create order in transaction
        if err := orderRepo.CreateOrder(ctx, tx, order); err != nil {
            return err
        }

        // Store event in outbox (part of transaction)
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

## Best Practices

### 1. Idempotent Compensation

Make compensation functions idempotent - they should be safe to call multiple times:

```go
func compensateOrder(ctx context.Context, orderID string) error {
    order, err := orderRepo.GetOrder(ctx, orderID)
    if err != nil {
        return err
    }

    // Already cancelled, return success (idempotent)
    if order.Status == "cancelled" {
        return nil
    }

    return orderRepo.CancelOrder(ctx, orderID)
}
```

### 2. Compensation Order

Compensation is executed in **reverse order** of execution:

- Execution: step1 → step2 → step3
- Compensation: step3 compensate ← step2 compensate ← step1 compensate

Ensure compensation functions account for this order.

### 3. Error Handling

- **Execution errors**: Saga stops and executes compensation
- **Compensation errors**: Logged but don't stop compensation of other steps
- **Final error**: Returns error from failed step, with compensation errors in message if any

### 4. Context Propagation

Always propagate context through saga steps for:

- Request tracing
- Timeouts and cancellation
- Logging correlation IDs

### 5. Step Naming

Use descriptive step names for logging and debugging:

```go
// Good
saga.AddStep("create_order", ...)
saga.AddStep("reserve_inventory", ...)

// Bad
saga.AddStep("step1", ...)
saga.AddStep("step2", ...)
```

## Limitations

### Simple Implementation Limitations

The simple saga implementation (`internal/saga`) has these limitations:

- ❌ **In-memory only**: Saga state is not persisted (lost on crash)
- ❌ **Single instance**: Not suitable for multi-instance deployments
- ❌ **No retries**: Failed steps are not retried
- ❌ **No timeouts**: Steps run until completion or failure
- ❌ **No visibility**: No way to inspect saga state
- ❌ **No long-running workflows**: Not suitable for workflows that span hours/days

### When to Upgrade

Upgrade to Temporal (or similar) when you need:

- Multi-instance deployments
- Durable saga state
- Retry logic
- Timeouts
- Long-running workflows
- Workflow visibility and debugging
- Complex error handling

## Integration with Other Patterns

### Outbox Pattern

Sagas work well with the outbox pattern for reliable event publishing:

1. Execute saga steps within database transactions
2. Store events in outbox table as part of transactions
3. Events are published asynchronously by outbox publisher

See [OUTBOX_PATTERN.md](./OUTBOX_PATTERN.md) for details.

### Event-Driven Architecture

Sagas can trigger events for each step:

1. Execute step
2. Publish event (via outbox if in transaction)
3. Other modules/subscribers react to events

This allows for reactive, event-driven workflows.

## Summary

- Use sagas for multi-step operations spanning multiple modules
- Use the simple implementation (`internal/saga`) for development/prototyping
- Consider Temporal for production deployments
- Make compensation functions idempotent
- Always propagate context through steps
- Integrate with outbox pattern for reliable event publishing

For more information, see:
- [WHY_OUTBOX_AND_SAGAS.md](./WHY_OUTBOX_AND_SAGAS.md) - When and why to use sagas
- [examples/saga_order_creation_test.go](../examples/saga_order_creation_test.go) - Complete example

