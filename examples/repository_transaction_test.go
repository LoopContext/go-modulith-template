// Package examples provides example integration tests showing how to test modules end-to-end.
package examples

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/cmelgarejo/go-modulith-template/internal/testutil"
)

// TestExampleRepositoryTransaction demonstrates testing repository transactions.
// This example shows:
// - Testing WithTx() pattern
// - Testing rollback scenarios
// - Testing concurrent transactions
func TestExampleRepositoryTransaction(t *testing.T) {
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

	// Step 2: Create test table
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS test_transactions (
			id TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Step 3: Test successful transaction
	t.Run("SuccessfulTransaction", func(t *testing.T) {
		testSuccessfulTransaction(ctx, t, db)
	})

	// Step 4: Test rollback scenario
	t.Run("RollbackTransaction", func(t *testing.T) {
		testRollbackTransaction(ctx, t, db)
	})

	// Step 5: Test concurrent transactions
	t.Run("ConcurrentTransactions", func(t *testing.T) {
		testConcurrentTransactions(ctx, t, db)
	})

	t.Log("Repository transaction tests complete")
}

func testSuccessfulTransaction(ctx context.Context, t *testing.T, db *sql.DB) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO test_transactions (id, value) VALUES ($1, $2)", "tx-1", "value-1")
	if err != nil {
		_ = tx.Rollback()

		t.Fatalf("Failed to insert: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Verify data was committed
	var value string

	err = db.QueryRowContext(ctx, "SELECT value FROM test_transactions WHERE id = $1", "tx-1").Scan(&value)
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	if value != "value-1" {
		t.Errorf("Expected value-1, got %s", value)
	}
}

func testRollbackTransaction(ctx context.Context, t *testing.T, db *sql.DB) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO test_transactions (id, value) VALUES ($1, $2)", "tx-2", "value-2")
	if err != nil {
		_ = tx.Rollback()

		t.Fatalf("Failed to insert: %v", err)
	}

	// Rollback instead of commit
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}

	// Verify data was NOT committed
	var value string

	err = db.QueryRowContext(ctx, "SELECT value FROM test_transactions WHERE id = $1", "tx-2").Scan(&value)
	if err != sql.ErrNoRows {
		if err == nil {
			t.Error("Expected no rows, but found data after rollback")
		} else {
			t.Fatalf("Unexpected error: %v", err)
		}
	}
}

func testConcurrentTransactions(ctx context.Context, t *testing.T, db *sql.DB) {
	// This would test concurrent access patterns
	// In a real scenario, you would:
	// 1. Start multiple transactions concurrently
	// 2. Perform operations in each
	// 3. Verify isolation and consistency
	tx1, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction 1: %v", err)
	}

	tx2, err := db.BeginTx(ctx, nil)
	if err != nil {
		_ = tx1.Rollback()

		t.Fatalf("Failed to begin transaction 2: %v", err)
	}

	_, err = tx1.ExecContext(ctx, "INSERT INTO test_transactions (id, value) VALUES ($1, $2)", "tx-3", "value-3")
	if err != nil {
		_ = tx1.Rollback()
		_ = tx2.Rollback()

		t.Fatalf("Failed to insert in tx1: %v", err)
	}

	_, err = tx2.ExecContext(ctx, "INSERT INTO test_transactions (id, value) VALUES ($1, $2)", "tx-4", "value-4")
	if err != nil {
		_ = tx1.Rollback()
		_ = tx2.Rollback()

		t.Fatalf("Failed to insert in tx2: %v", err)
	}

	// Commit both
	if err := tx1.Commit(); err != nil {
		_ = tx2.Rollback()

		t.Fatalf("Failed to commit tx1: %v", err)
	}

	if err := tx2.Commit(); err != nil {
		t.Fatalf("Failed to commit tx2: %v", err)
	}

	// Verify both were committed
	var count int

	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM test_transactions WHERE id IN ('tx-3', 'tx-4')").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 rows, got %d", count)
	}
}
