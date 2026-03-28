// Package examples provides example integration tests showing how to test modules end-to-end.
package examples

import (
	"context"
	"testing"

	"github.com/LoopContext/go-modulith-template/internal/config"
	"github.com/LoopContext/go-modulith-template/internal/registry"
	"github.com/LoopContext/go-modulith-template/internal/testutil"
	"github.com/LoopContext/go-modulith-template/modules/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestExampleGRPCService demonstrates testing gRPC services end-to-end.
// This example shows:
// - Setting up a test gRPC server
// - Creating a gRPC client
// - Testing authenticated endpoints
// - Testing error handling
// - Verifying responses
func TestExampleGRPCService(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Step 1: Set up test database
	pgContainer, db := setupTestDatabaseGRPC(ctx, t)
	defer cleanupTestDatabaseGRPC(ctx, t, pgContainer)

	// Step 2: Create registry and run migrations
	cfg := testutil.TestConfig()
	cfg.DBDSN = pgContainer.DSN

	reg := setupRegistryGRPC(t, db, cfg)

	if err := reg.InitializeAll(); err != nil {
		t.Fatalf("Failed to initialize modules: %v", err)
	}

	if err := testutil.RunMigrationsForTest(ctx, pgContainer.DSN, reg); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Step 3: Create gRPC test server
	grpcServer := setupGRPCServerGRPC(t, cfg, reg)
	defer cleanupGRPCServerGRPC(t, grpcServer)

	// Step 4: Test gRPC client connection
	testGRPCClient(t, grpcServer)
}

func setupTestDatabaseGRPC(ctx context.Context, t *testing.T) (*testutil.PostgresContainer, *pgxpool.Pool) {
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

func cleanupTestDatabaseGRPC(ctx context.Context, t *testing.T, pgContainer *testutil.PostgresContainer) {
	if err := pgContainer.Close(ctx); err != nil {
		t.Errorf("Failed to close container: %v", err)
	}
}

func setupRegistryGRPC(_ *testing.T, db *pgxpool.Pool, cfg *config.AppConfig) *registry.Registry {
	reg := testutil.NewTestRegistryBuilder().
		WithDatabase(db).
		WithConfig(cfg).
		WithModules(auth.NewModule()).
		Build()

	return reg
}

func setupGRPCServerGRPC(t *testing.T, cfg *config.AppConfig, reg *registry.Registry) *testutil.GRPCTestServer {
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

func cleanupGRPCServerGRPC(t *testing.T, grpcServer *testutil.GRPCTestServer) {
	if err := grpcServer.Stop(); err != nil {
		t.Errorf("Failed to stop gRPC server: %v", err)
	}
}

func testGRPCClient(t *testing.T, grpcServer *testutil.GRPCTestServer) {
	client := grpcServer.Client()
	if client == nil {
		t.Fatal("gRPC client is nil")
	}

	// At this point, you would use the client to just actual gRPC calls
	// Example:
	// authClient := authv1.NewAuthServiceClient(client)
	// resp, err := authClient.RequestLogin(ctx, &authv1.RequestLoginRequest{
	//     Email: "test@example.com",
	// })
	// if err != nil {
	//     t.Fatalf("RequestLogin failed: %v", err)
	// }
	// t.Logf("Login response: %+v", resp)

	t.Log("gRPC test server is running and client is connected")
	t.Logf("Server address: %s", grpcServer.Address())
}

// TestExampleGRPCErrorHandling demonstrates testing error handling in gRPC services.
func TestExampleGRPCErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Set up test environment (similar to above)
	pgContainer, db := setupTestDatabaseGRPC(ctx, t)
	defer cleanupTestDatabaseGRPC(ctx, t, pgContainer)

	cfg := testutil.TestConfig()
	cfg.DBDSN = pgContainer.DSN

	reg := setupRegistryGRPC(t, db, cfg)

	if err := reg.InitializeAll(); err != nil {
		t.Fatalf("Failed to initialize modules: %v", err)
	}

	if err := testutil.RunMigrationsForTest(ctx, pgContainer.DSN, reg); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	grpcServer := setupGRPCServerGRPC(t, cfg, reg)
	defer cleanupGRPCServerGRPC(t, grpcServer)

	// Example: Test invalid request
	// This would test error handling:
	// authClient := authv1.NewAuthServiceClient(grpcServer.Client())
	// _, err := authClient.RequestLogin(ctx, &authv1.RequestLoginRequest{
	//     Email: "invalid-email", // Invalid email format
	// })
	// if err == nil {
	//     t.Fatal("Expected error for invalid email")
	// }
	// status, ok := status.FromError(err)
	// if !ok {
	//     t.Fatal("Expected gRPC status error")
	// }
	// if status.Code() != codes.InvalidArgument {
	//     t.Fatalf("Expected InvalidArgument, got %v", status.Code())
	// }

	t.Log("Error handling test setup complete")
}
