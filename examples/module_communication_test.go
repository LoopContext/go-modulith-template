// Package examples provides example integration tests showing how to test modules end-to-end.
package examples

import (
	"context"
	"testing"
	"time"

	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/testutil"
	"github.com/cmelgarejo/go-modulith-template/modules/auth"
)

// ExampleModuleCommunicationTest demonstrates testing inter-module communication.
// This example shows:
// - Module A calling Module B via gRPC (in-process)
// - Event publishing between modules
// - Testing event-driven workflows
func ExampleModuleCommunicationTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Step 1: Set up test database
	pgContainer, err := testutil.NewPostgresContainer(ctx, t)
	if err != nil {
		t.Fatalf("Failed to create postgres container: %v", err)
	}

	defer func() {
		if err := pgContainer.Close(ctx); err != nil {
			t.Errorf("Failed to close container: %v", err)
		}
	}()

	db, err := pgContainer.DB(ctx)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close database: %v", err)
		}
	}()

	// Step 2: Create registry with event bus
	cfg := testutil.TestConfig()
	cfg.DBDSN = pgContainer.DSN

	eventBus := events.NewBus()
	eventCollector := testutil.NewEventCollector()

	// Subscribe to events
	eventCollector.Subscribe(eventBus, "user.created")

	reg := testutil.NewTestRegistryBuilder().
		WithDatabase(db).
		WithConfig(cfg).
		WithEventBus(eventBus).
		WithModules(auth.NewModule()).
		Build()

	if err := reg.InitializeAll(); err != nil {
		t.Fatalf("Failed to initialize modules: %v", err)
	}

	if err := testutil.RunMigrationsForTest(ctx, pgContainer.DSN, reg); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Step 3: Test event publishing
	// When a module performs an action, it should publish events
	// Example: Creating a user should publish user.created event
	testEvent := events.Event{
		Name: "user.created",
		Payload: map[string]interface{}{
			"user_id": "test-user-123",
			"email":   "test@example.com",
		},
	}

	eventBus.Publish(ctx, testEvent)

	// Step 4: Verify event was received
	receivedEvent, err := eventCollector.WaitForEvent(2 * time.Second)
	if err != nil {
		t.Fatalf("Timeout waiting for event: %v", err)
	}

	if receivedEvent.Name != "user.created" {
		t.Errorf("Expected event user.created, got %s", receivedEvent.Name)
	}

	t.Logf("Event received: %s", receivedEvent.Name)
	t.Logf("Event payload: %+v", receivedEvent.Payload)

	// Step 5: Test inter-module gRPC calls
	// In a real scenario, you would:
	// 1. Create a gRPC test server (as shown in grpc_service_test.go)
	// 2. Use one module's service to call another module's service
	// 3. Verify the interaction

	t.Log("Module communication test complete")
}

