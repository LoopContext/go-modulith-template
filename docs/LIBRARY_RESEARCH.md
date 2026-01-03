# Library Research for Production-Ready Improvements

This document analyzes proven Go libraries for implementing Saga/Outbox patterns, cross-module testing, and observability enhancements.

## Summary

Instead of building from scratch, we can leverage battle-tested libraries:

1. **Outbox Pattern**: Use `github.com/pkritiotis/go-outbox` or `github.com/oagudo/outbox`
2. **Saga Pattern**: Use `github.com/rilder-almeida/sagas` or `github.com/tiagomelo/go-saga` (or Temporal for production)
3. **Contract Testing**: Use `github.com/pact-foundation/pact-go`
4. **Observability**: Already using OpenTelemetry/Prometheus - just need templates

## 1. Outbox Pattern Libraries

### Option 1: `github.com/pkritiotis/go-outbox` (Recommended)

**Pros:**
- Simple, clean API
- Supports PostgreSQL with notification triggers
- Kafka integration
- Works within `sql.Tx` transactions
- Extensible interfaces for custom brokers/stores
- Active maintenance

**Cons:**
- PostgreSQL-only (but we use PostgreSQL)
- Requires additional dependencies

**GitHub**: https://github.com/pkritiotis/go-outbox

**Usage Pattern:**
```go
// Within a transaction
tx, _ := db.BeginTx(ctx, nil)
outbox := outbox.NewPostgresOutbox(tx)

// Store event in outbox (part of transaction)
err := outbox.Store(ctx, event)

// Commit transaction (includes outbox entry)
tx.Commit()

// Separate publisher process reads and publishes events
publisher := outbox.NewKafkaPublisher(kafkaProducer)
outbox.ProcessOutbox(ctx, publisher)
```

### Option 2: `github.com/oagudo/outbox`

**Pros:**
- Database and broker agnostic
- Examples for PostgreSQL/Kafka, Oracle/NATS, MySQL/RabbitMQ
- Lightweight

**Cons:**
- Less active maintenance
- More setup required

**GitHub**: https://github.com/oagudo/outbox

### Option 3: `github.com/nrfta/go-outbox`

**Pros:**
- Generic API
- PostgreSQL store with notification triggers
- Kafka and NATS integrations

**Cons:**
- Less documentation
- Smaller community

## Recommendation: `github.com/pkritiotis/go-outbox`

Best fit because:
- PostgreSQL-focused (matches our stack)
- Clean API
- Good documentation
- Active maintenance
- Works with our existing transaction patterns

## 2. Saga Pattern Libraries

### Option 1: `github.com/rilder-almeida/sagas` (Recommended for Simple Cases)

**Pros:**
- Clean API for orchestration
- Compensation support
- Lightweight
- Good for in-process sagas

**Cons:**
- In-memory only (no persistence)
- No retry logic built-in
- Smaller community

**GitHub**: https://pkg.go.dev/github.com/rilder-almeida/sagas

**Usage Pattern:**
```go
saga := sagas.NewSaga("order-creation").
    AddStep("create-order", createOrderStep, compensateOrder).
    AddStep("reserve-inventory", reserveInventory, compensateInventory).
    AddStep("charge-payment", chargePayment, compensatePayment)

result := saga.Execute(ctx)
```

### Option 2: `github.com/tiagomelo/go-saga`

**Pros:**
- Similar to rilder-almeida/sagas
- Ensures all-or-nothing semantics
- In-memory state management

**Cons:**
- Also in-memory only
- Less documentation

**GitHub**: https://pkg.go.dev/github.com/tiagomelo/go-saga

### Option 3: Temporal (Recommended for Production)

**Pros:**
- Production-grade workflow orchestration
- Durable execution
- Built-in retries, timeouts, compensation
- Excellent observability
- Used by many companies

**Cons:**
- Requires Temporal server (infrastructure)
- More complex setup
- Might be overkill for simple sagas

**Website**: https://temporal.io/

**Recommendation:**
- For template/documentation: Use `github.com/rilder-almeida/sagas` (simple, no infrastructure)
- For production guidance: Document Temporal as the recommended approach for production systems

## 3. Contract Testing

### Option: `github.com/pact-foundation/pact-go` (Industry Standard)

**Pros:**
- Industry-standard contract testing
- Consumer-driven contracts
- Supports HTTP and async messaging
- Excellent documentation
- Active maintenance
- Integrates with CI/CD

**Cons:**
- Requires Pact Broker for production (optional)
- Some learning curve

**GitHub**: https://github.com/pact-foundation/pact-go

**Usage Pattern:**
```go
// Consumer test
pact := createPact()
pact.AddInteraction().
    Given("user exists").
    UponReceiving("a request to get user").
    WithRequest(...).
    WillRespondWith(...)

// Provider verification
pact.VerifyProvider(types.VerifyRequest{
    ProviderBaseURL: "http://localhost:8080",
    ...
})
```

**Recommendation:** Use Pact Go - it's the industry standard.

## 4. Observability

### Current Stack (Already Excellent)
- OpenTelemetry (metrics, tracing) ✅
- Prometheus client ✅
- Structured logging (log/slog) ✅

### What's Missing (Just Templates)
- Grafana dashboard JSON files
- Prometheus alert rules YAML
- Logging standards documentation

**Recommendation:** No additional libraries needed - just create templates and documentation.

## 5. gRPC Testing

### Current Stack
- `google.golang.org/grpc` ✅
- `go.uber.org/mock` (gomock) ✅
- `github.com/stretchr/testify` ✅
- Custom test utilities in `internal/testutil` ✅

### Potential Enhancement: `github.com/grpc-ecosystem/go-grpc-middleware/testing`

**Pros:**
- Additional testing utilities
- Mock server helpers

**Cons:**
- Our custom `internal/testutil/grpc.go` already provides what we need
- Additional dependency may not be necessary

**Recommendation:** Continue using our existing test utilities - they're sufficient.

## Implementation Recommendations

### Phase 1: Outbox Pattern
**Library:** `github.com/pkritiotis/go-outbox`
- Integrates with existing transaction patterns
- Minimal code changes required
- Works with existing event bus

### Phase 2: Saga Pattern
**Library:** `github.com/rilder-almeida/sagas` for examples/documentation
**Alternative:** Document Temporal for production systems
- Simple library for template examples
- Document Temporal as production recommendation

### Phase 3: Contract Testing
**Library:** `github.com/pact-foundation/pact-go`
- Industry standard
- Well-documented
- Integrates with existing gRPC infrastructure

### Phase 4: Observability Templates
**Libraries:** None needed
- Create Grafana dashboard JSON files
- Create Prometheus alert rules YAML
- Write logging standards documentation

## Dependency Impact

### New Dependencies to Add:
```go
require (
    github.com/pkritiotis/go-outbox v0.1.0  // Outbox pattern
    github.com/rilder-almeida/sagas v0.1.0  // Saga pattern (optional, for examples)
    github.com/pact-foundation/pact-go/v2 v2.0.0  // Contract testing
)
```

### Optional (Production):
- Temporal SDK (if recommending Temporal for production sagas)
- Kafka client (for outbox publisher, if not using event bus)

## Migration Strategy

1. **Outbox**: Wrap existing event publishing with outbox pattern
2. **Saga**: Add saga examples alongside existing transaction examples
3. **Contract Testing**: Add as optional testing strategy
4. **Observability**: Pure documentation/templates, no code changes

## References

- [go-outbox by pkritiotis](https://github.com/pkritiotis/go-outbox)
- [sagas by rilder-almeida](https://pkg.go.dev/github.com/rilder-almeida/sagas)
- [Pact Go](https://github.com/pact-foundation/pact-go)
- [Temporal Go SDK](https://docs.temporal.io/dev-guide/go)
- [OpenTelemetry Go](https://opentelemetry.io/docs/instrumentation/go/)

