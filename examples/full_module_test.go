// Package examples provides example integration tests showing how to test modules end-to-end.
package examples

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/cmelgarejo/go-modulith-template/internal/config"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/registry"
	"github.com/cmelgarejo/go-modulith-template/internal/testutil"
	"github.com/cmelgarejo/go-modulith-template/modules/auth"
)

// TestExampleFullModule demonstrates a complete end-to-end test for a module.
// This example shows:
// - Database setup
// - Migration execution
// - Service initialization
// - gRPC endpoint testing
// - Event verification
// - Cleanup
func TestExampleFullModule(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Step 1: Set up test database using testcontainers
	pgContainer, db := setupTestDatabaseFull(ctx, t)
	defer cleanupTestDatabaseFull(ctx, t, pgContainer)

	// Step 2: Create configuration
	cfg := testutil.TestConfig()
	cfg.DBDSN = pgContainer.DSN

	// Step 3: Set up event bus and collector
	eventBus, eventCollector := setupEventBusFull()

	// Step 4: Create registry with all dependencies
	reg := setupRegistryFull(t, db, cfg, eventBus)

	// Step 5: Initialize modules
	if err := reg.InitializeAll(); err != nil {
		t.Fatalf("Failed to initialize modules: %v", err)
	}

	// Step 6: Run migrations
	if err := testutil.RunMigrationsForTest(ctx, pgContainer.DSN, reg); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Step 7: Create gRPC test server
	grpcServer := setupGRPCServerFull(t, cfg, reg)
	defer cleanupGRPCServerFull(t, grpcServer)

	// Step 8: Test gRPC endpoints
	// Example: Test RequestLogin endpoint
	// authClient := authv1.NewAuthServiceClient(grpcServer.Client())
	// resp, err := authClient.RequestLogin(ctx, &authv1.RequestLoginRequest{
	//     Email: "test@example.com",
	// })
	// if err != nil {
	//     t.Fatalf("RequestLogin failed: %v", err)
	// }
	// t.Logf("Login response: %+v", resp)

	// Step 9: Verify events were published
	// Wait a bit for async event processing
	time.Sleep(500 * time.Millisecond)

	allEvents := eventCollector.AllEvents()
	t.Logf("Total events received: %d", len(allEvents))

	for _, event := range allEvents {
		t.Logf("Event: %s, Payload: %+v", event.Name, event.Payload)
	}

	// Step 10: Verify database state
	// Example: Check that data was persisted
	// var count int
	// err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE email = $1", "test@example.com").Scan(&count)
	// if err != nil {
	//     t.Fatalf("Failed to query users: %v", err)
	// }
	// if count != 1 {
	//     t.Errorf("Expected 1 user, got %d", count)
	// }

	// Step 11: Test module lifecycle hooks
	if err := reg.OnStartAll(ctx); err != nil {
		t.Errorf("OnStartAll failed: %v", err)
	}

	// Cleanup: OnStopAll will be called by defer if needed
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := reg.OnStopAll(shutdownCtx); err != nil {
		t.Errorf("OnStopAll failed: %v", err)
	}

	t.Log("Full module integration test complete")
}

func setupTestDatabaseFull(ctx context.Context, t *testing.T) (*testutil.PostgresContainer, *pgxpool.Pool) {
	pgContainer, err := testutil.NewPostgresContainer(ctx, t)
	if err != nil {
		t.Fatalf("Failed to create postgres container: %v", err)
	}

	db, err := pgContainer.Pool(ctx)
	if err != nil {
		_ = pgContainer.Close(ctx)

		t.Fatalf("Failed to connect to database: %v", err)
	}

	return pgContainer, db
}

func cleanupTestDatabaseFull(ctx context.Context, t *testing.T, pgContainer *testutil.PostgresContainer) {
	if err := pgContainer.Close(ctx); err != nil {
		t.Errorf("Failed to close container: %v", err)
	}
}

func setupEventBusFull() (*events.Bus, *testutil.EventCollector) {
	eventBus := events.NewBus()
	eventCollector := testutil.NewEventCollector()

	// Subscribe to relevant events
	eventCollector.Subscribe(eventBus, "user.created")
	eventCollector.Subscribe(eventBus, "auth.magic_code_requested")

	return eventBus, eventCollector
}

func setupRegistryFull(_ *testing.T, db *pgxpool.Pool, cfg *config.AppConfig, eventBus *events.Bus) *registry.Registry {
	reg := testutil.NewTestRegistryBuilder().
		WithDatabase(db).
		WithConfig(cfg).
		WithEventBus(eventBus).
		WithModules(auth.NewModule()).
		Build()

	return reg
}


func setupGRPCServerFull(t *testing.T, cfg *config.AppConfig, reg *registry.Registry) *testutil.GRPCTestServer {
	grpcServer, err := testutil.NewGRPCTestServer(cfg, reg)
	if err != nil {
		t.Fatalf("Failed to create gRPC test server: %v", err)
	}

	if err := grpcServer.Start(); err != nil {
		_ = grpcServer.Stop()

		t.Fatalf("Failed to start gRPC server: %v", err)
	}

	return grpcServer
}

func cleanupGRPCServerFull(t *testing.T, grpcServer *testutil.GRPCTestServer) {
	if err := grpcServer.Stop(); err != nil {
		t.Errorf("Failed to stop gRPC server: %v", err)
	}
}
