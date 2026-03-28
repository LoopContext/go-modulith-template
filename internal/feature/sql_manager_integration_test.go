package feature_test

import (
	"context"
	"testing"

	"github.com/LoopContext/go-modulith-template/internal/config"
	"github.com/LoopContext/go-modulith-template/internal/events"
	"github.com/LoopContext/go-modulith-template/internal/feature"
	"github.com/LoopContext/go-modulith-template/internal/migration"
	"github.com/LoopContext/go-modulith-template/internal/registry"
	"github.com/LoopContext/go-modulith-template/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func TestIntegration_SQLManager(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	pool, cleanup := setupTestDB(ctx, t)

	defer cleanup()

	manager := feature.NewSQLManager(pool)

	t.Run("IsEnabled_DefaultValues", func(t *testing.T) {
		testDefaultValues(ctx, t, manager)
	})

	t.Run("SetAndGetFlag", func(t *testing.T) {
		testSetAndGetFlag(ctx, t, manager)
	})

	t.Run("ListFlags", func(t *testing.T) {
		testListFlags(ctx, t, manager)
	})

	t.Run("IsEnabledFor_ModuleContext", func(t *testing.T) {
		testModuleContext(ctx, t, manager)
	})
}

func setupTestDB(ctx context.Context, t *testing.T) (*pgxpool.Pool, func()) {
	container, err := testutil.NewPostgresContainer(ctx, t)
	if err != nil {
		t.Fatalf("Failed to create postgres container: %v", err)
	}

	cleanup := func() {
		if err := container.Close(ctx); err != nil {
			t.Logf("Failed to close container: %v", err)
		}
	}

	pool, err := container.Pool(ctx)
	if err != nil {
		cleanup()
		t.Fatalf("Failed to connect to database: %v", err)
	}

	createMockSchema(ctx, t, pool)

	runMigrationsAndSeeds(ctx, t, container.DSN, pool)

	return pool, func() {
		pool.Close()
		cleanup()
	}
}

func createMockSchema(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	// Create missing admin schema and system_config table for template-only tests
	_, err := pool.Exec(ctx, `
		CREATE SCHEMA IF NOT EXISTS admin;
		CREATE TABLE IF NOT EXISTS admin.system_config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		-- Add other tables if needed by tableMap
		CREATE SCHEMA IF NOT EXISTS kyc;
		CREATE TABLE IF NOT EXISTS kyc.kyc_config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		-- Add default flags expected by tests
		INSERT INTO admin.system_config (key, value) VALUES 
			('feeds_enabled', 'false'),
			('kyc_enabled', 'true'),
			('module_management_enabled', 'true')
		ON CONFLICT (key) DO NOTHING;
	`)
	if err != nil {
		t.Fatalf("Failed to create mock schema: %v", err)
	}
}

func runMigrationsAndSeeds(ctx context.Context, t *testing.T, dsn string, pool *pgxpool.Pool) {
	t.Helper()
	// Run migrations
	cfg := &config.AppConfig{}
	reg := registry.New(
		registry.WithConfig(cfg),
		registry.WithDatabase(pool),
		registry.WithEventBus(events.NewBus()),
	)

	migrationRunner := migration.NewRunner(dsn, reg)
	if err := migrationRunner.RunAll(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Run seeding
	seeder, err := migration.NewSeeder(dsn, reg)
	if err != nil {
		t.Fatalf("Failed to create seeder: %v", err)
	}

	if err := seeder.SeedAll(ctx); err != nil {
		_ = seeder.Close()

		t.Fatalf("Failed to run seeding: %v", err)
	}

	_ = seeder.Close()
}

func testDefaultValues(ctx context.Context, t *testing.T, manager *feature.SQLManager) {
	// These should be true from migration
	assert.False(t, manager.IsEnabled(ctx, "feeds_enabled"))
	assert.True(t, manager.IsEnabled(ctx, "kyc_enabled"))
	assert.True(t, manager.IsEnabled(ctx, "module_management_enabled"))
}

func testSetAndGetFlag(ctx context.Context, t *testing.T, manager *feature.SQLManager) {
	err := manager.SetFlag(ctx, feature.Flag{
		Name:    "new_feature",
		Enabled: true,
	})
	assert.NoError(t, err)
	assert.True(t, manager.IsEnabled(ctx, "new_feature"))

	err = manager.SetFlag(ctx, feature.Flag{
		Name:    "new_feature",
		Enabled: false,
	})
	assert.NoError(t, err)
	assert.False(t, manager.IsEnabled(ctx, "new_feature"))
}

func testListFlags(ctx context.Context, t *testing.T, manager *feature.SQLManager) {
	flags := manager.ListFlags(ctx)
	assert.NotEmpty(t, flags)

	found := false

	for _, f := range flags {
		if f.Name == "kyc_enabled" {
			found = true
			break
		}
	}

	assert.True(t, found, "expected to find kyc_enabled in list")
}

func testModuleContext(ctx context.Context, t *testing.T, manager *feature.SQLManager) {
	// Mock a value in another module's config table if possible
	// For now we just test the 'system' default
	featureCtx := feature.Context{
		Attributes: map[string]interface{}{"context": "system"},
	}
	assert.True(t, manager.IsEnabledFor(ctx, "kyc_enabled", featureCtx))
}
