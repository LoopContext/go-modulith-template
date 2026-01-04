package repository_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver

	"github.com/cmelgarejo/go-modulith-template/internal/config"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/migration"
	"github.com/cmelgarejo/go-modulith-template/internal/registry"
	"github.com/cmelgarejo/go-modulith-template/internal/testutil"
	"github.com/cmelgarejo/go-modulith-template/modules/auth"
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
	db, repo := setupTestDB(ctx, t, container, container.DSN)

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

func setupTestDB(ctx context.Context, t *testing.T, container *testutil.PostgresContainer, dsn string) (*sql.DB, *repository.SQLRepository) {
	t.Helper()

	db, err := container.DB(ctx)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Set up registry and run migrations
	cfg := &config.AppConfig{
		Env:      "test",
		LogLevel: "debug",
		Auth: config.AuthConfig{
			JWTSecret: "test-secret-key-that-is-at-least-32-bytes-long-for-testing",
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

	// Run migrations
	migrationRunner := migration.NewRunner(dsn, reg)
	if err := migrationRunner.RunAll(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	repo := repository.NewSQLRepository(db)

	return db, repo
}

func verifyUserCreated(ctx context.Context, t *testing.T, db *sql.DB, userID, expectedEmail string) {
	t.Helper()

	var email string

	err := db.QueryRowContext(ctx, "SELECT email FROM auth.users WHERE id = $1", userID).Scan(&email)
	if err != nil {
		t.Errorf("Failed to query user: %v", err)
	}

	if email != expectedEmail {
		t.Errorf("Expected email %s, got %s", expectedEmail, email)
	}
}
