package testutil_test

import (
	"context"
	"testing"

	"github.com/LoopContext/go-modulith-template/internal/testutil"
)

func TestPostgresContainer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.NewPostgresContainer(ctx, t)
	if err != nil {
		t.Fatalf("Failed to create postgres container: %v", err)
	}

	defer func() {
		if err := container.Close(ctx); err != nil {
			t.Errorf("Failed to close container: %v", err)
		}
	}()

	// Test database connection
	pool, err := container.Pool(ctx)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	defer pool.Close()

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		t.Errorf("Failed to ping database: %v", err)
	}

	// Test a simple query
	var result int

	err = pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		t.Errorf("Failed to execute query: %v", err)
	}

	if result != 1 {
		t.Errorf("Expected result 1, got %d", result)
	}
}
