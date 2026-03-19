// Package examples provides example integration tests showing how to test modules end-to-end.
//
// This file demonstrates:
// - Setting up testcontainers for database testing
// - Running migrations in tests
// - Testing gRPC service methods
// - Testing event bus integration
// - Testing repository layer with real database
//
// To run integration tests:
//
//	go test -v -run Integration ./examples/...
//
// Or use the Makefile:
//
//	make test-integration
package examples

import (
	"context"
	"testing"
	"time"

	"github.com/cmelgarejo/go-modulith-template/internal/config"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/migration"
	"github.com/cmelgarejo/go-modulith-template/internal/registry"
	"github.com/cmelgarejo/go-modulith-template/internal/testutil"
	"github.com/cmelgarejo/go-modulith-template/modules/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ExampleIntegrationTest demonstrates a complete integration test for a module.
// This test:
// 1. Sets up a real PostgreSQL database using testcontainers
// 2. Runs migrations to create the schema
// 3. Initializes the module with real dependencies
// 4. Tests the repository layer through SQL queries
// 5. Verifies event bus integration
func ExampleIntegrationTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Step 1: Set up test database using testcontainers
	pgContainer, err := testutil.NewPostgresContainer(ctx, t)
	if err != nil {
		t.Fatalf("Failed to create postgres container: %v", err)
	}

	defer func() {
		if err := pgContainer.Close(ctx); err != nil {
			t.Errorf("Failed to close container: %v", err)
		}
	}()

	// Step 2: Get database connection pool
	pool, err := pgContainer.Pool(ctx)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	defer pool.Close()

	// Step 3: Run migrations
	reg := setupRegistry(t, pool)

	// Run migrations
	migrationRunner := migration.NewRunner(pgContainer.DSN, reg)
	if err := migrationRunner.RunAll(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Step 4: Get event bus from registry for testing
	eventBus := reg.EventBus()

	// Step 5: Test through database queries
	testDatabaseOperations(ctx, t, pool)

	// Step 6: Test event bus integration
	testEventBusIntegration(ctx, t, eventBus)
}

func setupRegistry(t *testing.T, db *pgxpool.Pool) *registry.Registry {
	t.Helper()

	// Create a minimal registry for migration discovery
	cfg := &config.AppConfig{
		Env:      "test",
		LogLevel: "debug",
		Auth: config.AuthConfig{
			JWTPrivateKeyPEM: "test-auth-private-key",
		},
	}

	reg := registry.New(
		registry.WithConfig(cfg),
		registry.WithDatabase(db),
		registry.WithEventBus(events.NewBus()),
	)

	// Register the auth module
	reg.Register(auth.NewModule())

	// Initialize modules (required before migrations)
	if err := reg.InitializeAll(); err != nil {
		t.Fatalf("Failed to initialize modules: %v", err)
	}

	return reg
}

func testDatabaseOperations(ctx context.Context, t *testing.T, db *pgxpool.Pool) {
	t.Helper()

	// Test that migrations created the expected tables
	var tableExists bool

	err := db.QueryRow(ctx,
		"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'auth' AND table_name = 'magic_codes')",
	).Scan(&tableExists)
	if err != nil {
		t.Fatalf("Failed to check table existence: %v", err)
	}

	if !tableExists {
		t.Error("Expected auth.magic_codes table to exist after migrations")
	}

	// Test inserting and querying data
	email := "test@example.com"
	code := "123456"

	_, err = db.Exec(ctx,
		"INSERT INTO auth.magic_codes (email, code, expires_at) VALUES ($1, $2, NOW() + INTERVAL '10 minutes')",
		email, code,
	)
	if err != nil {
		t.Fatalf("Failed to insert magic code: %v", err)
	}

	var storedCode string

	err = db.QueryRow(ctx,
		"SELECT code FROM auth.magic_codes WHERE email = $1 ORDER BY created_at DESC LIMIT 1",
		email,
	).Scan(&storedCode)
	if err != nil {
		t.Fatalf("Failed to query magic code: %v", err)
	}

	if storedCode != code {
		t.Errorf("Expected magic code %s, got %s", code, storedCode)
	}
}

func testEventBusIntegration(ctx context.Context, t *testing.T, eventBus *events.Bus) {
	t.Helper()

	// Create a channel to receive events
	eventReceived := make(chan events.Event, 1)

	// Subscribe to a specific event
	eventBus.Subscribe("test.event", func(_ context.Context, e events.Event) error {
		select {
		case eventReceived <- e:
		default:
		}

		return nil
	})

	// Publish a test event directly to verify event bus works
	testEvent := events.Event{
		Name: "test.event",
		Payload: map[string]interface{}{
			"test": "data",
		},
	}
	eventBus.Publish(ctx, testEvent)

	// Wait for event (with timeout)
	select {
	case event := <-eventReceived:
		if event.Name != "test.event" {
			t.Errorf("Expected event 'test.event', got %s", event.Name)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for event")
	}
}

// ExampleRepositoryIntegrationTest demonstrates testing database operations directly.
func ExampleRepositoryIntegrationTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Set up test database
	pgContainer, err := testutil.NewPostgresContainer(ctx, t)
	if err != nil {
		t.Fatalf("Failed to create postgres container: %v", err)
	}

	defer func() {
		if err := pgContainer.Close(ctx); err != nil {
			t.Errorf("Failed to close container: %v", err)
		}
	}()

	pool, err := pgContainer.Pool(ctx)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	defer pool.Close()

	// For this example, we'll create a simple table
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Example: Test database operations directly
	t.Run("DatabaseOperations", func(t *testing.T) {
		testUserOperations(ctx, t, pool)
	})
}

func testUserOperations(ctx context.Context, t *testing.T, db *pgxpool.Pool) {
	t.Helper()

	userID := "test-user-123"
	email := "test@example.com"

	// Insert
	_, err := db.Exec(ctx,
		"INSERT INTO test_users (id, email) VALUES ($1, $2)",
		userID, email,
	)
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	// Query
	var storedEmail string

	err = db.QueryRow(ctx,
		"SELECT email FROM test_users WHERE id = $1",
		userID,
	).Scan(&storedEmail)
	if err != nil {
		t.Fatalf("Failed to query user: %v", err)
	}

	if storedEmail != email {
		t.Errorf("Expected email %s, got %s", email, storedEmail)
	}
}

// ExampleGRPCIntegrationTest demonstrates testing gRPC endpoints.
func ExampleGRPCIntegrationTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Log("gRPC integration test template - implement based on your needs")
}

