# Why Outbox and Sagas? When Are They Needed?

This document explains when and why you need the Outbox pattern and Saga pattern in your modulith application. Both patterns are **optional** by default, but become **necessary** as your application scales and complexity grows.

## Table of Contents

1. [Quick Decision Guide](#quick-decision-guide)
2. [Outbox Pattern: When and Why](#outbox-pattern-when-and-why)
3. [Saga Pattern: When and Why](#saga-pattern-when-and-why)
4. [Real-World Scenarios](#real-world-scenarios)
5. [Migration Path](#migration-path)
6. [Common Misconceptions](#common-misconceptions)

## Quick Decision Guide

### Do I Need Outbox Pattern?

```
┌─────────────────────────────────────────┐
│ Are you deploying multiple instances?       │
│ (Horizontal scaling, Kubernetes, etc.)      │
└─────────────────┬───────────────────────────┘
                  │
        ┌─────────┴─────────┐
        │                    │
       YES                  NO
        │                    │
        ▼                    ▼
┌───────────────┐   ┌──────────────────────┐
│ Do events     │   │ Are events critical  │
│ need to reach │   │ business events?     │
│ all instances?│   │ (not just logging)   │
└───────┬───────┘   └──────────┬───────────┘
        │                      │
       YES                     YES
        │                      │
        ▼                      ▼
   ┌─────────┐           ┌─────────┐
   │ NEED    │           │ NEED    │
   │ OUTBOX  │           │ OUTBOX  │
   └─────────┘           └─────────┘
```

**Answer: You need Outbox if:**

-   ✅ Deploying multiple instances (horizontal scaling)
-   ✅ Events must be delivered reliably (critical business events)
-   ✅ Events must survive application restarts
-   ✅ Event ordering matters

**Answer: You DON'T need Outbox if:**

-   ❌ Single instance deployment
-   ❌ Events are just notifications/logging (fire-and-forget is OK)
-   ❌ Event loss is acceptable
-   ❌ All event handlers are in-process

### Do I Need Saga Pattern?

```
┌─────────────────────────────────────────┐
│ Does your operation span multiple      │
│ modules? (e.g., order → inventory →    │
│ payment)                                │
└─────────────────┬───────────────────────┘
                  │
        ┌─────────┴─────────┐
        │                    │
       YES                  NO
        │                    │
        ▼                    ▼
┌───────────────┐   ┌──────────────────────┐
│ Do you need   │   │ You DON'T need       │
│ compensation  │   │ saga. Use simple     │
│ if a step     │   │ transactions.        │
│ fails?        │   │                      │
└───────┬───────┘   └──────────────────────┘
        │
       YES
        │
        ▼
   ┌─────────┐
   │ NEED    │
   │ SAGA    │
   └─────────┘
```

**Answer: You need Saga if:**

-   ✅ Operations span multiple modules
-   ✅ Need compensation (rollback) on failure
-   ✅ Need all-or-nothing semantics across modules
-   ✅ Complex multi-step workflows with dependencies

**Answer: You DON'T need Saga if:**

-   ❌ Single-module operations
-   ❌ Simple transactions within one module
-   ❌ Eventual consistency is acceptable
-   ❌ No compensation needed

## Outbox Pattern: When and Why

### What Problem Does Outbox Solve?

The Outbox pattern solves the **"dual-write problem"** in distributed systems:

**Problem:** When you need to:

1. Save data to database (in a transaction)
2. Publish an event (to event bus)

If step 2 fails, you have inconsistent state. If the application crashes between step 1 and 2, the event is lost forever.

### Current Behavior (Without Outbox)

Looking at your current code:

```go
// modules/auth/internal/service/service.go
func (s *AuthService) RequestLogin(ctx context.Context, req *authv1.RequestLoginRequest) {
    // 1. Save to database
    err = s.repo.CreateMagicCode(ctx, code, email, phone, expiresAt)

    // 2. Publish event (fire-and-forget)
    s.bus.Publish(ctx, events.Event{
        Name: notifier.EventMagicCodeRequested,
        Payload: map[string]interface{}{...},
    })
}
```

**Current Limitations:**

-   ❌ If `Publish()` fails, event is lost (but data is saved)
-   ❌ If application crashes between save and publish, event is lost
-   ❌ In multi-instance deployments, events only reach handlers in the same instance
-   ❌ No guarantee of event delivery

**This is OK when:**

-   ✅ Single instance deployment
-   ✅ Events are notifications (not critical)
-   ✅ Event loss is acceptable
-   ✅ Event handlers are in-process

### When Outbox Becomes Necessary

#### Scenario 1: Horizontal Scaling

**Problem:** Multiple instances, events only reach same instance

```
Instance 1: User creates order → publishes "order.created"
Instance 2: Inventory handler (never receives event)
Instance 3: Notification handler (never receives event)
```

**Solution:** Outbox ensures events are published reliably and reach all instances.

#### Scenario 2: Critical Business Events

**Problem:** Event loss breaks business logic

```go
// Order module
func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) {
    // Save order
    order, err := s.repo.CreateOrder(ctx, ...)

    // Publish event (if this fails, inventory never knows!)
    s.bus.Publish(ctx, events.Event{
        Name: "order.created",
        Payload: order,
    })
}

// Inventory module (subscribes to "order.created")
// If event is lost, inventory is never reserved!
```

**Solution:** Outbox guarantees event delivery as part of the transaction.

#### Scenario 3: Event Durability

**Problem:** Application restart loses in-flight events

```
1. Order created (saved to DB)
2. Event queued in memory
3. Application crashes
4. Event lost forever
```

**Solution:** Outbox stores events in database, survives restarts.

### How Outbox Works

```
┌─────────────────────────────────────────────────┐
│ Transaction                                      │
│ ┌─────────────────────────────────────────────┐ │
│ │ 1. Save business data (order, user, etc.)   │ │
│ │ 2. Save event to outbox table (same tx)    │ │
│ │ 3. Commit transaction                      │ │
│ └─────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────────────┐
│ Outbox Table (in database)                      │
│ ┌─────────────────────────────────────────────┐ │
│ │ id | event_name | payload | created_at      │ │
│ │ 1  | order.cr.. | {...}   | 2024-01-01...  │ │
│ └─────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────────────┐
│ Outbox Publisher (background worker)            │
│ ┌─────────────────────────────────────────────┐ │
│ │ 1. Read unpublished events from outbox     │ │
│ │ 2. Publish to event bus                     │ │
│ │ 3. Mark as published                       │ │
│ │ 4. Retry on failure                        │ │
│ └─────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────────────┐
│ Event Bus (reaches all instances)               │
└─────────────────────────────────────────────────┘
```

### When to Enable Outbox

**Enable Outbox when:**

1. **Deploying multiple instances** (Kubernetes, auto-scaling, etc.)
2. **Events are critical** (order created, payment processed, etc.)
3. **Event durability required** (must survive restarts)
4. **Event ordering matters** (events must be processed in sequence)
5. **Audit/compliance** (events must be logged)

**Keep it disabled when:**

1. Single instance deployment
2. Events are just notifications (fire-and-forget OK)
3. Event loss is acceptable
4. All handlers are in-process (no network calls)

## Saga Pattern: When and Why

### What Problem Does Saga Solve?

The Saga pattern solves the **"distributed transaction problem"** in microservices/moduliths:

**Problem:** You can't use database transactions across multiple modules because:

-   Each module has its own database (in microservices)
-   Even in modulith, modules should be isolated
-   Network calls can't be part of a database transaction

### Example: Order Creation Without Saga

**The Problem:**

```go
// Order Module
func CreateOrder(ctx context.Context, req *CreateOrderRequest) {
    // Step 1: Create order
    order, err := orderRepo.CreateOrder(ctx, ...)

    // Step 2: Reserve inventory (different module!)
    _, err = inventoryClient.ReserveStock(ctx, order.Items)

    // Step 3: Charge payment (different module!)
    _, err = paymentClient.Charge(ctx, order.Total)

    // What if step 3 fails?
    // - Order is created ✅
    // - Inventory is reserved ✅
    // - Payment failed ❌
    // - Now what? Order exists but not paid!
}
```

**Without Saga:**

-   ❌ No way to rollback across modules
-   ❌ Inconsistent state (order created, inventory reserved, payment failed)
-   ❌ Manual cleanup required
-   ❌ Complex error handling

### When Saga Becomes Necessary

#### Scenario 1: Multi-Module Operations

**Problem:** Operations that require coordination across modules

```
Order Creation Saga:
1. Create order (order module)
2. Reserve inventory (inventory module)
3. Charge payment (payment module)
4. Send notification (notification module)

If step 3 fails:
- Compensate: Cancel order
- Compensate: Release inventory
- Don't send notification
```

**Solution:** Saga orchestrates steps and compensates on failure.

#### Scenario 2: Compensation Required

**Problem:** Need to undo operations when later steps fail

```go
// Without Saga (manual compensation)
func CreateOrder(ctx context.Context, req *CreateOrderRequest) error {
    order, err := orderRepo.CreateOrder(ctx, ...)
    if err != nil {
        return err
    }

    err = inventoryClient.ReserveStock(ctx, order.Items)
    if err != nil {
        // Manual cleanup!
        orderRepo.CancelOrder(ctx, order.ID)
        return err
    }

    err = paymentClient.Charge(ctx, order.Total)
    if err != nil {
        // More manual cleanup!
        orderRepo.CancelOrder(ctx, order.ID)
        inventoryClient.ReleaseStock(ctx, order.Items)
        return err
    }

    return nil
}
```

**Solution:** Saga automatically handles compensation.

#### Scenario 3: Complex Workflows

**Problem:** Multi-step processes with dependencies

```
User Registration Saga:
1. Create user account (auth module)
2. Create user profile (profile module)
3. Send welcome email (notification module)
4. Initialize user preferences (preferences module)

If step 2 fails:
- Compensate: Delete user account
- Don't send email
- Don't initialize preferences
```

**Solution:** Saga manages the workflow and compensation.

### How Saga Works

```
┌─────────────────────────────────────────────────┐
│ Saga Orchestration                              │
│ ┌─────────────────────────────────────────────┐ │
│ │ Step 1: Create Order                        │ │
│ │   Execute: orderRepo.CreateOrder()         │ │
│ │   Compensate: orderRepo.CancelOrder()      │ │
│ └─────────────────────────────────────────────┘ │
│                    │                            │
│                    ▼                            │
│ ┌─────────────────────────────────────────────┐ │
│ │ Step 2: Reserve Inventory                   │ │
│ │   Execute: inventoryClient.ReserveStock()  │ │
│ │   Compensate: inventoryClient.ReleaseStock()│ │
│ └─────────────────────────────────────────────┘ │
│                    │                            │
│                    ▼                            │
│ ┌─────────────────────────────────────────────┐ │
│ │ Step 3: Charge Payment                      │ │
│ │   Execute: paymentClient.Charge()           │ │
│ │   Compensate: paymentClient.Refund()       │ │
│ └─────────────────────────────────────────────┘ │
│                    │                            │
│                    ▼                            │
│         ┌──────────┴──────────┐                 │
│         │                     │                  │
│      SUCCESS               FAILURE               │
│         │                     │                  │
│         │                     ▼                  │
│         │         ┌──────────────────────┐      │
│         │         │ Run Compensations    │      │
│         │         │ (in reverse order)   │      │
│         │         └──────────────────────┘      │
│         │                                        │
│         ▼                                        │
│    ┌─────────┐                                   │
│    │ SUCCESS │                                   │
│    └─────────┘                                   │
└─────────────────────────────────────────────────┘
```

### When to Use Saga

**Use Saga when:**

1. **Multi-module operations** (order → inventory → payment)
2. **Compensation needed** (rollback on failure)
3. **All-or-nothing semantics** (strong consistency)
4. **Complex workflows** (multi-step with dependencies)

**Don't use Saga when:**

1. **Single-module operations** (use simple transactions)
2. **Eventual consistency OK** (order created, inventory reserved eventually)
3. **No compensation needed** (operations are idempotent)
4. **Simple operations** (create user, update profile)

## Real-World Scenarios

### Scenario 1: E-Commerce Order Processing

**Operation:** Customer places order

**Modules Involved:**

-   Order module (create order)
-   Inventory module (reserve stock)
-   Payment module (charge card)
-   Notification module (send confirmation)

**Without Saga:**

```go
// Manual, error-prone
order := createOrder()
inventory := reserveStock()
payment := chargeCard()  // Fails!
// Now what? Order exists, inventory reserved, but not paid
// Manual cleanup required
```

**With Saga:**

```go
saga := saga.New("order-creation").
    AddStep("create-order", createOrder, cancelOrder).
    AddStep("reserve-inventory", reserveStock, releaseStock).
    AddStep("charge-payment", chargeCard, refundCard).
    AddStep("send-notification", sendEmail, nil)  // No compensation needed

err := saga.Execute(ctx)
// Automatic compensation if any step fails
```

**Decision:** ✅ **Need Saga** - Multi-module operation with compensation

### Scenario 2: User Registration

**Operation:** New user signs up

**Modules Involved:**

-   Auth module (create account)
-   Profile module (create profile)
-   Notification module (send welcome email)

**Without Saga:**

```go
// Simple case - eventual consistency OK
user := authService.CreateUser()
profileService.CreateProfile(user.ID)  // Can fail, but that's OK
notificationService.SendWelcomeEmail(user.ID)  // Can fail, that's OK
```

**With Saga (if needed):**

```go
// Only if you need all-or-nothing
saga := saga.New("user-registration").
    AddStep("create-user", createUser, deleteUser).
    AddStep("create-profile", createProfile, deleteProfile).
    AddStep("send-email", sendEmail, nil)

err := saga.Execute(ctx)
```

**Decision:** ❌ **Don't need Saga** - Eventual consistency is acceptable, or use simple transactions if needed

### Scenario 3: Magic Code Notification

**Operation:** Send magic code for login

**Current Implementation:**

```go
// Save magic code to database
err = s.repo.CreateMagicCode(ctx, code, email, phone, expiresAt)

// Publish event (fire-and-forget)
s.bus.Publish(ctx, events.Event{
    Name: notifier.EventMagicCodeRequested,
    Payload: {...},
})
```

**Without Outbox:**

-   Event might be lost if publish fails
-   Event might be lost if app crashes
-   In multi-instance, only one instance gets event

**With Outbox:**

-   Event guaranteed to be published
-   Event survives app restarts
-   Event reaches all instances

**Decision:**

-   **Single instance:** ❌ **Don't need Outbox** - Event loss is acceptable for notifications
-   **Multiple instances:** ✅ **Need Outbox** - Event must reach notification handler
-   **Critical events:** ✅ **Need Outbox** - If magic code notification is critical

### Scenario 4: Payment Processing

**Operation:** Process payment for order

**Modules Involved:**

-   Payment module (charge card)
-   Order module (update order status)
-   Notification module (send receipt)

**Without Outbox:**

```go
// Payment processed
payment := paymentService.ProcessPayment(orderID, amount)

// Update order status
orderService.UpdateStatus(orderID, "paid")  // If this fails, order still shows "pending"

// Send receipt
notificationService.SendReceipt(orderID)  // If this fails, customer doesn't get receipt
```

**With Outbox:**

```go
// In transaction
tx.Begin()
paymentService.ProcessPayment(orderID, amount)
outbox.Store(ctx, events.Event{
    Name: "payment.processed",
    Payload: payment,
})
tx.Commit()

// Outbox publisher ensures event reaches:
// - Order module (updates status)
// - Notification module (sends receipt)
```

**Decision:** ✅ **Need Outbox** - Critical business event, must reach all handlers

## Migration Path

### Phase 1: Start Simple (Current State)

**Outbox:** Disabled

-   Single instance deployment
-   Fire-and-forget events
-   Event loss acceptable

**Saga:** Not used

-   Single-module operations
-   Simple transactions
-   Eventual consistency OK

### Phase 2: Scale Horizontally

**Outbox:** Enable when deploying multiple instances

```go
// Enable outbox
outboxRepo := outbox.NewRepository(db)
outboxPub := outbox.NewPublisher(eventBus)

reg := registry.New(
    registry.WithEventBus(events.NewBus()),
    registry.WithOutbox(outboxRepo, outboxPub),  // Enable
)
```

**Saga:** Still not needed (unless building multi-module workflows)

### Phase 3: Complex Workflows

**Saga:** Enable when building multi-module operations

```go
// Use saga for order creation
saga := saga.New("order-creation").
    AddStep("create-order", createOrder, cancelOrder).
    AddStep("reserve-inventory", reserveStock, releaseStock).
    AddStep("charge-payment", chargeCard, refundCard)

err := saga.Execute(ctx)
```

### Phase 4: Production Scale

**Saga:** Consider Temporal for production

-   Durable execution
-   Built-in retries
-   Better observability
-   Production-grade reliability

## Common Misconceptions

### ❌ "I need Outbox for all events"

**Reality:** Only needed for:

-   Critical business events
-   Multi-instance deployments
-   Events that must be durable

**Example:** Logging events don't need outbox.

### ❌ "I need Saga for all multi-module operations"

**Reality:** Only needed when:

-   Compensation is required
-   All-or-nothing semantics needed
-   Eventual consistency is not acceptable

**Example:** User registration can use eventual consistency (create user, create profile eventually).

### ❌ "Outbox and Saga are the same thing"

**Reality:**

-   **Outbox:** Ensures reliable event publishing
-   **Saga:** Orchestrates multi-step operations with compensation

**They solve different problems and can be used together.**

### ❌ "I must use these patterns from day one"

**Reality:** Start simple, add patterns as needed:

1. Start with simple events and transactions
2. Add Outbox when scaling horizontally
3. Add Saga when building complex workflows
4. Consider Temporal for production-scale sagas

## Summary

### Outbox Pattern

**Optional when:**

-   Single instance
-   Fire-and-forget events
-   Event loss acceptable

**Necessary when:**

-   Multiple instances
-   Critical business events
-   Event durability required

### Saga Pattern

**Optional when:**

-   Single-module operations
-   Eventual consistency OK
-   No compensation needed

**Necessary when:**

-   Multi-module operations
-   Compensation required
-   All-or-nothing semantics

### Decision Framework

1. **Start simple** - Don't add complexity until needed
2. **Enable Outbox** - When scaling horizontally or events become critical
3. **Use Saga** - When building multi-module workflows with compensation
4. **Consider Temporal** - For production-scale saga orchestration

Both patterns are **optional by default** but become **necessary** as your application grows in complexity and scale.
