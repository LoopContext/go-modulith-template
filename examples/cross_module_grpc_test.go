// Package examples demonstrates cross-module gRPC testing patterns.
//
// This example shows how to test interactions between modules via gRPC,
// including error propagation, authentication/authorization, and context propagation.
package examples

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LoopContext/go-modulith-template/internal/events"
	"github.com/LoopContext/go-modulith-template/internal/testutil"
	"github.com/LoopContext/go-modulith-template/modules/auth"
)

// setupGRPCTestServer sets up a gRPC test server with database and registry.
func setupGRPCTestServer(ctx context.Context, t *testing.T) (*testutil.PostgresContainer, *testutil.GRPCTestServer, interface{}) {
	t.Helper()

	pgContainer, err := testutil.NewPostgresContainer(ctx, t)
	require.NoError(t, err)

	db, err := pgContainer.Pool(ctx)
	require.NoError(t, err)

	cfg := testutil.TestConfig()
	cfg.DBDSN = pgContainer.DSN

	eventBus := events.NewBus()
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

	grpcTestServer, err := testutil.NewGRPCTestServer(cfg, reg)
	require.NoError(t, err)

	err = grpcTestServer.Start()
	require.NoError(t, err)

	conn := grpcTestServer.Client()
	require.NotNil(t, conn)

	return pgContainer, grpcTestServer, conn
}

// TestCrossModuleGRPC_ModuleCommunication demonstrates testing module A calling module B via gRPC.
func TestCrossModuleGRPC_ModuleCommunication(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Setup test database
	pgContainer, err := testutil.NewPostgresContainer(ctx, t)
	require.NoError(t, err)

	defer func() {
		_ = pgContainer.Close(ctx)
	}()

	db, err := pgContainer.Pool(ctx)
	require.NoError(t, err)

	// Setup registry with modules
	cfg := testutil.TestConfig()
	cfg.DBDSN = pgContainer.DSN

	eventBus := events.NewBus()
	eventCollector := testutil.NewEventCollector()

	// Setup registry using testutil builder
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

	// Create gRPC test server
	grpcTestServer, err := testutil.NewGRPCTestServer(cfg, reg)
	require.NoError(t, err)

	err = grpcTestServer.Start()
	require.NoError(t, err)

	defer func() {
		_ = grpcTestServer.Stop()
	}()

	// Get client connection
	conn := grpcTestServer.Client()
	require.NotNil(t, conn)

	// Get client for auth module service
	// Note: In real usage, you would use the generated gRPC client:
	// authClient := authv1.NewAuthServiceClient(conn)
	// This is a simplified example showing the pattern

	// Test: Call module service
	// Example: requestLogin or other operations
	// resp, err := authClient.RequestLogin(ctx, &authv1.RequestLoginRequest{...})
	// require.NoError(t, err)
	// assert.NotNil(t, resp)

	// Verify event was published (if applicable)
	time.Sleep(100 * time.Millisecond) // Give event bus time to process

	events := eventCollector.AllEvents()
	assert.GreaterOrEqual(t, len(events), 0) // Adjust based on actual test
}

// TestCrossModuleGRPC_ErrorPropagation demonstrates testing error propagation in cross-module calls.
func TestCrossModuleGRPC_ErrorPropagation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	pgContainer, grpcTestServer, conn := setupGRPCTestServer(ctx, t)

	defer func() {
		_ = pgContainer.Close(ctx)
	}()

	defer func() {
		_ = grpcTestServer.Stop()
	}()

	_ = conn // Use conn in test

	// Test: Call with invalid input to test error propagation
	// Example: Call with invalid request that should return error
	// _, err := authClient.RequestLogin(ctx, &authv1.RequestLoginRequest{
	//     Email: "", // Invalid: empty email
	// })
	// require.Error(t, err)
	// assert.Contains(t, err.Error(), "invalid")

	// Verify error code is correct (e.g., InvalidArgument)
	// status, ok := status.FromError(err)
	// require.True(t, ok)
	// assert.Equal(t, codes.InvalidArgument, status.Code())
}

// TestCrossModuleGRPC_ContextPropagation demonstrates testing context propagation in cross-module calls.
func TestCrossModuleGRPC_ContextPropagation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Setup (similar to above)
	pgContainer, err := testutil.NewPostgresContainer(ctx, t)
	require.NoError(t, err)

	defer func() {
		_ = pgContainer.Close(ctx)
	}()

	db, err := pgContainer.Pool(ctx)
	require.NoError(t, err)

	cfg := testutil.TestConfig()
	cfg.DBDSN = pgContainer.DSN

	eventBus := events.NewBus()
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

	grpcTestServer, err := testutil.NewGRPCTestServer(cfg, reg)
	require.NoError(t, err)

	err = grpcTestServer.Start()
	require.NoError(t, err)

	defer func() {
		_ = grpcTestServer.Stop()
	}()

	conn := grpcTestServer.Client()
	require.NotNil(t, conn)

	// Test: Add metadata/context values and verify they propagate
	// Example: Add user ID or trace ID to context
	// ctxWithMetadata := metadata.NewOutgoingContext(ctx, metadata.Pairs("user-id", "test-user"))
	// resp, err := authClient.RequestLogin(ctxWithMetadata, &authv1.RequestLoginRequest{...})
	// Verify metadata was propagated and used correctly
}

// Note: These helper functions are examples showing the pattern.
// In actual usage, you would reuse helpers from module_communication_test.go
// or create shared test utilities in internal/testutil.
