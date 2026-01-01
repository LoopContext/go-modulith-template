package repository_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver

	"github.com/cmelgarejo/go-modulith-template/internal/testutil"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/repository"
)

func TestIntegration_SQLRepository_CreateUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	// Start testcontainer
	container, err := testutil.NewPostgresContainer(ctx, t)
	if err != nil {
		t.Fatalf("Failed to create postgres container: %v", err)
	}

	defer func() {
		if err := container.Close(ctx); err != nil {
			t.Errorf("Failed to close container: %v", err)
		}
	}()

	// Get database connection and setup schema
	db, repo := setupTestDB(ctx, t, container)

	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close database: %v", err)
		}
	}()

	// Test CreateUser
	testEmail := "test@example.com"
	testPhone := "+1234567890"
	testID := "user_01h455vb4pex5vsknk084sn02q"

	err = repo.CreateUser(ctx, testID, testEmail, testPhone)
	if err != nil {
		t.Errorf("CreateUser() error = %v", err)
	}

	// Verify user was created
	verifyUserCreated(ctx, t, db, testID, testEmail)
}

func setupTestDB(ctx context.Context, t *testing.T, container *testutil.PostgresContainer) (*sql.DB, *repository.SQLRepository) {
	t.Helper()

	db, err := container.DB(ctx)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Create schema (simplified for test)
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(64) PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			phone VARCHAR(20),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	repo := repository.NewSQLRepository(db)

	return db, repo
}

func verifyUserCreated(ctx context.Context, t *testing.T, db *sql.DB, userID, expectedEmail string) {
	t.Helper()

	var email string

	err := db.QueryRowContext(ctx, "SELECT email FROM users WHERE id = $1", userID).Scan(&email)
	if err != nil {
		t.Errorf("Failed to query user: %v", err)
	}

	if email != expectedEmail {
		t.Errorf("Expected email %s, got %s", expectedEmail, email)
	}
}

