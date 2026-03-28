# Testing Guide

This comprehensive guide covers all testing patterns and best practices for the modulith template.

## Table of Contents

1. [Overview](#overview)
2. [Unit Testing](#unit-testing)
3. [Integration Testing](#integration-testing)
4. [Cross-Module gRPC Testing](#cross-module-grpc-testing)
5. [Event Bus Testing](#event-bus-testing)
6. [Contract Testing](#contract-testing)
7. [End-to-End Testing](#end-to-end-testing)
8. [Testing Best Practices](#testing-best-practices)

## Overview

The modulith template supports multiple testing strategies:

- **Unit Tests**: Test individual components in isolation
- **Integration Tests**: Test module interactions with database
- **Cross-Module Tests**: Test gRPC communication between modules
- **Event-Driven Tests**: Test event publishing and subscription
- **Contract Tests**: Test API contracts between modules (Pact)
- **E2E Tests**: Test complete workflows end-to-end

## Unit Testing

### Testing Services

Test service logic with mocked dependencies:

```go
func TestAuthService_RequestLogin(t *testing.T) {
    // Create mocks
    mockRepo := mocks.NewMockRepository(t)
    mockTokenSvc := mocks.NewMockTokenService(t)
    eventBus := events.NewBus()

    // Create service
    svc := service.NewAuthService(mockRepo, mockTokenSvc, eventBus)

    // Setup expectations
    mockRepo.EXPECT().
        CreateMagicCode(gomock.Any(), gomock.Any()).
        Return(nil)

    // Execute
    ctx := context.Background()
    resp, err := svc.RequestLogin(ctx, &authv1.RequestLoginRequest{
        Email: "test@example.com",
    })

    // Assert
    require.NoError(t, err)
    assert.NotNil(t, resp)
}
```

### Testing Repositories

Test repository logic with test database:

```go
func TestRepository_CreateUser(t *testing.T) {
    ctx := context.Background()

    pgContainer, err := testutil.NewPostgresContainer(ctx, t)
    require.NoError(t, err)
    defer pgContainer.Close(ctx)

    db, err := pgContainer.DB(ctx)
    require.NoError(t, err)
    defer db.Close()

    // Run migrations
    // ... setup ...

    repo := repository.NewSQLRepository(db)

    // Test
    err = repo.CreateUser(ctx, "user-123", "test@example.com", "")
    require.NoError(t, err)
}
```

### Mock Generation

Generate mocks using `gomock`:

```bash
# Generate mocks for interfaces
mockgen -source=modules/auth/internal/repository/repository.go \
        -destination=modules/auth/internal/repository/mocks/repository_mock.go \
        -package=mocks
```

See `modules/auth/internal/service/service_mock_test.go` for examples.

### SQLC Type Names in Tests

**Important:** When writing tests that use SQLC-generated types, always use the correct type names with schema prefixes:

```go
// ✅ Correct - Use schema-prefixed types
mockRepo.EXPECT().
    GetUserByEmail(gomock.Any(), email).
    Return(&store.AuthUser{
        ID:    "user-123",
        Email: sql.NullString{String: email, Valid: true},
    }, nil)

mockRepo.EXPECT().
    GetValidMagicCodeByEmail(gomock.Any(), email, code).
    Return(&store.AuthMagicCode{
        Code:      code,
        UserEmail: sql.NullString{String: email, Valid: true},
    }, nil)

// ❌ Wrong - Missing schema prefix
mockRepo.EXPECT().
    GetUserByEmail(gomock.Any(), email).
    Return(&store.User{...}, nil)  // This will cause "undefined: store.User" error
```

SQLC generates types using the pattern `{Schema}{TableName}`:
- `auth.users` → `store.AuthUser`
- `auth.magic_codes` → `store.AuthMagicCode`
- `auth.sessions` → `store.AuthSession`

Always check `modules/<mod>/internal/db/store/models.go` after running `just sqlc` to see the exact generated type names.

## Integration Testing

### Testing Module Initialization

Test module setup and initialization:

```go
func TestModule_Initialize(t *testing.T) {
    ctx := context.Background()

    pgContainer, err := testutil.NewPostgresContainer(ctx, t)
    require.NoError(t, err)
    defer pgContainer.Close(ctx)

    db, err := pgContainer.DB(ctx)
    require.NoError(t, err)
    defer db.Close()

    cfg := testutil.TestConfig()
    cfg.DBDSN = pgContainer.DSN

    reg := registry.New(
        registry.WithConfig(cfg),
        registry.WithDatabase(db),
        registry.WithEventBus(events.NewBus()),
    )

    mod := auth.NewModule()
    reg.Register(mod)

    err = reg.InitializeAll()
    require.NoError(t, err)
}
```

### Testing with Testcontainers

Use testcontainers for integration tests:

```go
import "github.com/LoopContext/go-modulith-template/internal/testutil"

func TestWithDatabase(t *testing.T) {
    ctx := context.Background()

    pgContainer, err := testutil.NewPostgresContainer(ctx, t)
    require.NoError(t, err)
    defer pgContainer.Close(ctx)

    db, err := pgContainer.DB(ctx)
    require.NoError(t, err)
    defer db.Close()

    // Use database in tests
}
```

See `examples/integration_test_example.go` for complete examples.

## Cross-Module gRPC Testing

Test interactions between modules via gRPC.

### Basic Cross-Module Test

```go
func TestCrossModuleGRPC(t *testing.T) {
    ctx := context.Background()

    // Setup test database
    pgContainer, err := testutil.NewPostgresContainer(ctx, t)
    require.NoError(t, err)
    defer pgContainer.Close(ctx)

    db, err := pgContainer.DB(ctx)
    require.NoError(t, err)
    defer db.Close()

    // Setup registry with modules
    cfg := testutil.TestConfig()
    cfg.DBDSN = pgContainer.DSN

    eventBus := events.NewBus()
    reg := setupRegistry(t, db, cfg, eventBus)

    reg.Register(auth.NewModule())
    // reg.Register(order.NewModule()) // Add more modules

    if err := reg.InitializeAll(); err != nil {
        t.Fatalf("Failed to initialize: %v", err)
    }

    // Run migrations
    if err := testutil.RunMigrationsForTest(ctx, pgContainer.DSN, reg); err != nil {
        t.Fatalf("Failed to run migrations: %v", err)
    }

    // Create gRPC server
    grpcServer := grpc.NewServer()
    reg.RegisterGRPCAll(grpcServer)

    // Start server
    listener, err := testutil.NewTestListener()
    require.NoError(t, err)

    go grpcServer.Serve(listener)
    defer grpcServer.Stop()

    // Create client
    conn, err := grpc.NewClient(
        listener.Addr().String(),
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    require.NoError(t, err)
    defer conn.Close()

    // Get client for module service
    // authClient := authv1.NewAuthServiceClient(conn)

    // Test: Call module service
    // resp, err := authClient.RequestLogin(ctx, &authv1.RequestLoginRequest{...})
    // require.NoError(t, err)
}
```

### Testing Error Propagation

Test error handling across module boundaries:

```go
func TestCrossModuleGRPC_ErrorPropagation(t *testing.T) {
    // Setup (similar to above)

    // Call with invalid input
    // _, err := authClient.RequestLogin(ctx, &authv1.RequestLoginRequest{
    //     Email: "", // Invalid
    // })

    // Verify error
    // require.Error(t, err)
    // status, ok := status.FromError(err)
    // require.True(t, ok)
    // assert.Equal(t, codes.InvalidArgument, status.Code())
}
```

### Testing Context Propagation

Test context values (trace IDs, user IDs) propagation:

```go
func TestCrossModuleGRPC_ContextPropagation(t *testing.T) {
    // Setup (similar to above)

    // Add metadata to context
    // ctxWithMetadata := metadata.NewOutgoingContext(ctx,
    //     metadata.Pairs("trace-id", "trace-123"))

    // Call service
    // resp, err := authClient.RequestLogin(ctxWithMetadata, ...)

    // Verify metadata was propagated and used
}
```

See `examples/cross_module_grpc_test.go` for complete examples.

## Event Bus Testing

### Testing Event Publishing

Test event publishing and collection:

```go
func TestEventPublishing(t *testing.T) {
    ctx := context.Background()

    eventBus := events.NewBus()
    eventCollector := testutil.NewEventCollector()

    // Subscribe collector to events
    eventCollector.Subscribe(eventBus, "user.created")

    // Publish event
    eventBus.Publish(ctx, events.Event{
        Name:    "user.created",
        Payload: map[string]interface{}{
            "user_id": "user-123",
            "email":   "test@example.com",
        },
    })

    // Wait for event processing
    time.Sleep(100 * time.Millisecond)

    // Verify event was collected
    collectedEvents := eventCollector.AllEvents()
    require.GreaterOrEqual(t, len(collectedEvents), 1)

    found := false
    for _, event := range collectedEvents {
        if event.Name == "user.created" {
            found = true
            assert.Equal(t, "user-123",
                event.Payload.(map[string]interface{})["user_id"])
            break
        }
    }
    assert.True(t, found)
}
```

### Testing Event Handlers

Test event handler execution:

```go
func TestEventHandler(t *testing.T) {
    ctx := context.Background()

    eventBus := events.NewBus()
    handlerCalled := make(chan bool, 1)

    eventBus.Subscribe("test.event", func(ctx context.Context, event events.Event) error {
        handlerCalled <- true
        return nil
    })

    eventBus.Publish(ctx, events.Event{
        Name:    "test.event",
        Payload: map[string]interface{}{"key": "value"},
    })

    select {
    case <-handlerCalled:
        // Handler executed
    case <-time.After(1 * time.Second):
        t.Fatal("Handler did not execute")
    }
}
```

### Testing Event Ordering

Test event sequencing (note: order not guaranteed in async handlers):

```go
func TestEventOrdering(t *testing.T) {
    ctx := context.Background()

    eventBus := events.NewBus()
    eventCollector := testutil.NewEventCollector()

    eventCollector.Subscribe(eventBus, "ordered.event")

    // Publish events in sequence
    for i := 1; i <= 3; i++ {
        eventBus.Publish(ctx, events.Event{
            Name:    "ordered.event",
            Payload: map[string]interface{}{"sequence": i},
        })
    }

    time.Sleep(200 * time.Millisecond)

    // Verify all events were collected
    collectedEvents := eventCollector.AllEvents()
    assert.GreaterOrEqual(t, len(collectedEvents), 3)
}
```

See `examples/event_driven_workflow_test.go` for complete examples.

## Contract Testing

Contract testing ensures API contracts between modules remain stable.

### Pact Go Setup

Install Pact Go (test-only dependency):

```bash
go get github.com/pact-foundation/pact-go/v2
```

### Consumer Contract Test

Define expected interactions from consumer side:

```go
// examples/contract_testing_consumer_test.go
func TestAuthService_Contract_Consumer(t *testing.T) {
    // Setup Pact
    pact := pactgo.NewPact()
    defer pact.Cleanup()

    // Define expected interaction
    pact.
        AddInteraction().
        Given("user exists").
        UponReceiving("a request to login").
        WithRequest("POST", "/auth.v1.AuthService/RequestLogin").
        WillRespondWith(200, map[string]interface{}{
            "code_sent": true,
        })

    // Test against mock provider
    // ...
}
```

### Provider Contract Verification

Verify provider matches contract:

```go
// examples/contract_testing_provider_test.go
func TestAuthService_Contract_Provider(t *testing.T) {
    // Setup provider (actual service)
    // ...

    // Verify against Pact broker or file
    // pact.VerifyProvider(t, providerURL)
}
```

See `docs/CONTRACT_TESTING.md` for detailed contract testing guide (to be created).

## End-to-End Testing

Test complete workflows from API to database:

```go
func TestE2E_UserRegistration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E test in short mode")
    }

    ctx := context.Background()

    // Setup complete environment
    pgContainer, err := testutil.NewPostgresContainer(ctx, t)
    require.NoError(t, err)
    defer pgContainer.Close(ctx)

    // Setup registry, modules, gRPC server, HTTP gateway
    // ...

    // Execute complete workflow
    // 1. Create user via gRPC
    // 2. Verify user in database
    // 3. Verify events were published
    // 4. Verify notifications were sent
}
```

See `examples/full_module_test.go` for complete E2E examples.

## Testing Best Practices

### 1. Use Testcontainers for Integration Tests

Always use testcontainers for database tests:

```go
pgContainer, err := testutil.NewPostgresContainer(ctx, t)
require.NoError(t, err)
defer pgContainer.Close(ctx)
```

### 2. Clean Up Resources

Always clean up test resources:

```go
defer pgContainer.Close(ctx)
defer db.Close()
defer grpcServer.Stop()
```

### 3. Use Table-Driven Tests

Use table-driven tests for multiple scenarios:

```go
func TestService_MultipleScenarios(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid input", "valid@example.com", false},
        {"invalid email", "invalid", true},
        {"empty input", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test with tt.input
        })
    }
}
```

### 4. Skip Long Tests in Short Mode

Skip integration/E2E tests in short mode:

```go
func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    // ... test code
}
```

Run tests:
```bash
go test -short ./...  # Skip long tests
go test ./...          # Run all tests
```

### 5. Use Context for Timeouts

Use context for timeouts and cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// Use ctx in operations
```

### 6. Verify Events with Collectors

Use EventCollector to verify events:

```go
eventCollector := testutil.NewEventCollector()
eventCollector.Subscribe(eventBus, "user.created")

// ... perform action that should publish event

time.Sleep(100 * time.Millisecond) // Give event bus time
events := eventCollector.AllEvents()
assert.Contains(t, events, expectedEvent)
```

### 7. Test Error Cases

Always test error cases:

```go
func TestService_ErrorCases(t *testing.T) {
    // Test with invalid input
    // Test with missing dependencies
    // Test with database errors
    // Test with network errors
}
```

### 8. Use Mocks for External Dependencies

Mock external dependencies in unit tests:

```go
mockRepo := mocks.NewMockRepository(t)
mockRepo.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(errors.New("db error"))
```

### 9. Test Transaction Rollback

Test transaction rollback scenarios:

```go
func TestRepository_TransactionRollback(t *testing.T) {
    // Setup transaction
    // Perform operations
    // Force error
    // Verify rollback occurred
}
```

### 10. Document Test Patterns

Add comments explaining test patterns:

```go
// This test demonstrates cross-module gRPC communication testing.
// It sets up a full registry with multiple modules and tests
// interactions between them via gRPC.
func TestCrossModuleGRPC(t *testing.T) {
    // ...
}
```

## Running Tests

### Run All Tests

```bash
just test-unit    # Unit tests only
just test         # All tests
go test ./...     # All tests (alternative)
```

### Run Specific Tests

```bash
go test ./modules/auth/internal/service -v
go test -run TestAuthService_RequestLogin ./modules/auth/...
```

### Run Tests with Coverage

```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Run Tests in Parallel

```bash
go test -parallel 4 ./...
```

## Summary

- Use unit tests for isolated components
- Use integration tests with testcontainers for database tests
- Use cross-module tests for gRPC communication
- Use event collectors for event testing
- Use contract tests for API stability
- Always clean up resources
- Test error cases
- Use table-driven tests for multiple scenarios
- Skip long tests in short mode

For examples, see:
- `examples/module_communication_test.go` - Module communication patterns
- `examples/cross_module_grpc_test.go` - Cross-module gRPC testing
- `examples/event_driven_workflow_test.go` - Event-driven testing
- `examples/full_module_test.go` - Complete E2E examples

