// Package examples provides example integration tests showing how to test modules end-to-end.
package examples

import (
	"context"
	"testing"

	"github.com/cmelgarejo/go-modulith-template/internal/testutil"
	"github.com/cmelgarejo/go-modulith-template/modules/auth"
)

// ExampleGRPCServiceTest demonstrates testing gRPC services end-to-end.
// This example shows:
// - Setting up a test gRPC server
// - Creating a gRPC client
// - Testing authenticated endpoints
// - Testing error handling
// - Verifying responses
func ExampleGRPCServiceTest(t *testing.T) {
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

	// Step 2: Create registry and run migrations
	cfg := testutil.TestConfig()
	cfg.DBDSN = pgContainer.DSN

	reg := testutil.NewTestRegistryBuilder().
		WithDatabase(db).
		WithConfig(cfg).
		WithModules(auth.NewModule()).
		Build()

	if err := reg.InitializeAll(); err != nil {
		t.Fatalf("Failed to initialize modules: %v", err)
	}

	if err := testutil.RunMigrationsForTest(ctx, pgContainer.DSN, reg); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Step 3: Create gRPC test server
	grpcServer, err := testutil.NewGRPCTestServer(cfg, reg)
	if err != nil {
		t.Fatalf("Failed to create gRPC test server: %v", err)
	}

	defer func() {
		if err := grpcServer.Stop(); err != nil {
			t.Errorf("Failed to stop gRPC server: %v", err)
		}
	}()

	if err := grpcServer.Start(); err != nil {
		t.Fatalf("Failed to start gRPC server: %v", err)
	}

	// Step 4: Test gRPC client connection
	client := grpcServer.Client()
	if client == nil {
		t.Fatal("gRPC client is nil")
	}

	// At this point, you would use the client to make actual gRPC calls
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

// ExampleGRPCErrorHandling demonstrates testing error handling in gRPC services.
func ExampleGRPCErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Set up test environment (similar to above)
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

	cfg := testutil.TestConfig()
	cfg.DBDSN = pgContainer.DSN

	reg := testutil.NewTestRegistryBuilder().
		WithDatabase(db).
		WithConfig(cfg).
		WithModules(auth.NewModule()).
		Build()

	if err := reg.InitializeAll(); err != nil {
		t.Fatalf("Failed to initialize modules: %v", err)
	}

	if err := testutil.RunMigrationsForTest(ctx, pgContainer.DSN, reg); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	grpcServer, err := testutil.NewGRPCTestServer(cfg, reg)
	if err != nil {
		t.Fatalf("Failed to create gRPC test server: %v", err)
	}

	defer func() {
		if err := grpcServer.Stop(); err != nil {
			t.Errorf("Failed to stop gRPC server: %v", err)
		}
	}()

	if err := grpcServer.Start(); err != nil {
		t.Fatalf("Failed to start gRPC server: %v", err)
	}

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

