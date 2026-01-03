// Package examples provides example integration tests showing how to test modules end-to-end.
package examples

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/cmelgarejo/go-modulith-template/internal/config"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/registry"
	"github.com/cmelgarejo/go-modulith-template/internal/testutil"
	"github.com/cmelgarejo/go-modulith-template/modules/auth"
)

// TestExampleModuleCommunication demonstrates testing inter-module communication.
// This example shows:
// - Module A calling Module B via gRPC (in-process)
// - Event publishing between modules
// - Testing event-driven workflows
func TestExampleModuleCommunication(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Step 1: Set up test database
	pgContainer, db := setupTestDatabaseModule(ctx, t)
	defer cleanupTestDatabaseModule(ctx, t, pgContainer, db)

	// Step 2: Create registry with event bus
	cfg := testutil.TestConfig()
	cfg.DBDSN = pgContainer.DSN

	eventBus, eventCollector := setupEventBusModule()

	reg := setupRegistryModule(t, db, cfg, eventBus)

	if err := reg.InitializeAll(); err != nil {
		t.Fatalf("Failed to initialize modules: %v", err)
	}

	if err := testutil.RunMigrationsForTest(ctx, pgContainer.DSN, reg); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Step 3: Test event publishing and verification
	testEventPublishing(ctx, t, eventBus, eventCollector)

	t.Log("Module communication test complete")
}

func setupTestDatabaseModule(ctx context.Context, t *testing.T) (*testutil.PostgresContainer, *sql.DB) {
	pgContainer, err := testutil.NewPostgresContainer(ctx, t)
	if err != nil {
		t.Fatalf("Failed to create postgres container: %v", err)
	}

	db, err := pgContainer.DB(ctx)
	if err != nil {
		_ = pgContainer.Close(ctx)

		t.Fatalf("Failed to connect to database: %v", err)
	}

	return pgContainer, db
}

func cleanupTestDatabaseModule(ctx context.Context, t *testing.T, pgContainer *testutil.PostgresContainer, db *sql.DB) {
	if err := db.Close(); err != nil {
		t.Errorf("Failed to close database: %v", err)
	}

	if err := pgContainer.Close(ctx); err != nil {
		t.Errorf("Failed to close container: %v", err)
	}
}

func setupEventBusModule() (*events.Bus, *testutil.EventCollector) {
	eventBus := events.NewBus()
	eventCollector := testutil.NewEventCollector()

	// Subscribe to events
	eventCollector.Subscribe(eventBus, "user.created")

	return eventBus, eventCollector
}

func setupRegistryModule(_ *testing.T, db *sql.DB, cfg *config.AppConfig, eventBus *events.Bus) *registry.Registry {
	reg := testutil.NewTestRegistryBuilder().
		WithDatabase(db).
		WithConfig(cfg).
		WithEventBus(eventBus).
		WithModules(auth.NewModule()).
		Build()

	return reg
}

func testEventPublishing(ctx context.Context, t *testing.T, eventBus *events.Bus, eventCollector *testutil.EventCollector) {
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

	// Verify event was received
	receivedEvent, err := eventCollector.WaitForEvent(2 * time.Second)
	if err != nil {
		t.Fatalf("Timeout waiting for event: %v", err)
	}

	expectedEventName := "user.created"

	if receivedEvent.Name != expectedEventName {
		t.Errorf("Expected event %s, got %s", expectedEventName, receivedEvent.Name)
	}

	t.Logf("Event received: %s", receivedEvent.Name)
	t.Logf("Event payload: %+v", receivedEvent.Payload)

	// Step 5: Test inter-module gRPC calls
	// In a real scenario, you would:
	// 1. Create a gRPC test server (as shown in grpc_service_test.go)
	// 2. Use one module's service to call another module's service
	// 3. Verify the interaction
}
