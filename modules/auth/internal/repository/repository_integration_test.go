package repository_test

import (
	"context"
	"testing"

	"github.com/cmelgarejo/go-modulith-template/internal/config"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/migration"
	"github.com/cmelgarejo/go-modulith-template/internal/registry"
	"github.com/cmelgarejo/go-modulith-template/internal/testutil"
	"github.com/cmelgarejo/go-modulith-template/modules/auth"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
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
	pool, repo := setupTestDB(ctx, t, container, container.DSN)

	defer pool.Close()

	// Test CreateUser
	testEmail := "test@example.com"
	testPhone := "+1234567890"
	testID := "user_01h455vb4pex5vsknk084sn02q"

	err = repo.CreateUser(ctx, testID, testEmail, testPhone)
	if err != nil {
		t.Errorf("CreateUser() error = %v", err)
	}

	// Verify user was created
	verifyUserCreated(ctx, t, pool, testID, testEmail)
}

func TestIntegration_SQLRepository_GetUserRole_Platform(t *testing.T) {
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
	pool, repo := setupTestDB(ctx, t, container, container.DSN)

	defer pool.Close()

	// 1. Create User
	userID := "user_role_test"
	if err := repo.CreateUser(ctx, userID, "role_test@example.com", ""); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// 2. Create Role
	roleID := "role_platform"
	roleName := "platform"

	_, err = pool.Exec(ctx, "INSERT INTO auth.roles (id, name) VALUES ($1, $2)", roleID, roleName)
	if err != nil {
		t.Fatalf("Failed to create role: %v", err)
	}

	// 3. Assign Role
	_, err = pool.Exec(ctx, "INSERT INTO auth.user_roles (user_id, role_id) VALUES ($1, $2)", userID, roleID)
	if err != nil {
		t.Fatalf("Failed to assign role: %v", err)
	}

	// 4. Test GetUserRole
	role, err := repo.GetUserRole(ctx, userID)
	if err != nil {
		t.Errorf("GetUserRole() error = %v", err)
	}

	if role != roleName {
		t.Errorf("GetUserRole() got = %v, want %v", role, roleName)
	}
}

func TestIntegration_SQLRepository_GetUserRole_Default(t *testing.T) {
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
	pool, repo := setupTestDB(ctx, t, container, container.DSN)

	defer pool.Close()

	// Test Default Role
	userID2 := "user_no_role"
	if err := repo.CreateUser(ctx, userID2, "no_role@example.com", ""); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	role2, err := repo.GetUserRole(ctx, userID2)
	if err != nil {
		t.Errorf("GetUserRole() error = %v", err)
	}

	if role2 != "user" {
		t.Errorf("GetUserRole() for user without role got = %v, want %v", role2, "user")
	}
}

func setupTestDB(ctx context.Context, t *testing.T, container *testutil.PostgresContainer, dsn string) (*pgxpool.Pool, *repository.SQLRepository) {
	t.Helper()

	pool, err := container.Pool(ctx)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	_ = setupIntegrationRegistry(ctx, t, pool, dsn)

	repo := repository.NewSQLRepository(pool)

	return pool, repo
}

func setupIntegrationRegistry(_ context.Context, t *testing.T, pool *pgxpool.Pool, dsn string) *registry.Registry {
	t.Helper()
	// Set up registry and run migrations
	cfg := &config.AppConfig{
		Env:      "test",
		LogLevel: "debug",
		Auth: config.AuthConfig{
			JWTPrivateKeyPEM: `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQCgKaue+zKl57/5
QuzzKZIm0nQe5Jopmd10ie/fB8k3nAReUwQ0aiaVws9FmeT1fylKzuLrEN4Xh0wy
ZYrEwV0xTaxBOu708yZikVCMz1bF16mhoODBrm2+cNE0bfpxzwoFt/zyP6AigxWJ
5XHJzJHFoaDw3334oLvaG1lkcDjfFUEKMbIk+CN2hXCbI6BSJCo989y4RPoFkZBH
eNgKiRiHZm5ypsNEdvjItlRGM7hwtAH81v+OtdlTeWp+mlz3SCUyCagEP1Gs3L0Y
aeoYOEA1ylpmapaDhKnobk4oFb9ujF60CGkLt/eOjt63AvQQtmAKJLK4Y0EgS7hi
3kh9ZxDJAgMBAAECggEACx+px3jR2Ggp0wspzGxynD1zCpXWlGIXDLOFB4JebTBp
6A7JYXlbGBaq8T4ST7yjs3B+arrfefBDKgYXmh+5GoxqLmOd86d5kh0rZFEE/IIR
SkqTbmnWpGq1SCtrpzQTRNKcMxgyxbYN+Zq7PIh3oJH5TN49o2/ibCNvId5epmh5
Qyvy2FhYZGhtxg3K+WApQxfeTOq/o+BbNdSUrcQaLDeKe3PS3KFykCj+dno3EiFn
dyEwLQcP63dSoUqW6ObR634DSIRR0CNWqRyeWD0SxRbjNV9bIk/bOjJ4FrPjEuRB
gT/LhMsD1fthTMyAyNpryxDknc2mYCrHd/ix5nEakwKBgQDMdfe0CpSbfNqix9T6
TAasGZaXVSBJ3n4GCwFOn5KaJfPhAB64n9x82YvliWOyl5u16SgxnBj3vKGLskCP
DXSLvQWBheZBFoPxEKsGXp2ddEFXf7zVcjG4nYz8Z0Kn5JGImhjwajcQKIVJBCvR
vTwCWl3/9spKARs0Zue6hBd0owKBgQDIiRzDJlonRL6TCS8bJT/LBdWGIn1Syz/A
zbssfD9Qh89TL5i7zfPcGm4Yzk+Z4zbh1/67D33GvMPr1aKnzcbR4+4+xZiVaZjl
m0tDONGFxrZAyvbdHLJiXZBujoRO96bGsjZtyEZ+hG+MV0s+FCX7fkFWJa1+vpyv
aAkZcrjPowKBgQCK76bRC1eMiT0w3EYXh84I6KJyV4BHcg+FH7lVqg2+/gdJYAGA
R/FWTaZI5iF/XJKM/NE5VO+KeP31pb1E+Em4I0w4hbq/hANIrqDpBSZptnQodz7k
dGLhJv6FDc43tJRIlR5ZUHP2YPKheVolfkfm+W1i4Fr6CuJnq33QOq6NrQKBgFml
Oa9fiLO/PnZah61Z5H+stvxElMObSn+1OHQ1gtRMMflc8Kkb82S0h/0c1WbUtOcW
+K/EyBQ8tFTL5u+exL91Zj63dHNuhkQ2PNnrH3bvEvA6C0tjFbd1XiieGzV17h8q
8bv36NOL/pW9PEyfEy+vDCQnqbxcF40uM8slhsqDAoGBAMGCthWkf2eG0Y4Scksf
r/gNlU+15OnndSq0UQt2xjiy+0XQ5CVHaIyyaLiFiYjsLYdaxfOckMMrvP3RqObE
8b9897yqs3ENFV+lJA7z/gZntQFLmlfzQadbGRuVeZfh+u7NqM4j73SRNMubEBEd
7mlsQJQ+USaHSReSju9xmzH8
-----END PRIVATE KEY-----`,
		},
	}
	reg := registry.New(
		registry.WithConfig(cfg),
		registry.WithDatabase(pool),
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

	return reg
}

func verifyUserCreated(ctx context.Context, t *testing.T, pool *pgxpool.Pool, userID, expectedEmail string) {
	t.Helper()

	var email string

	err := pool.QueryRow(ctx, "SELECT email FROM auth.users WHERE id = $1", userID).Scan(&email)
	if err != nil {
		t.Errorf("Failed to query user: %v", err)
	}

	if email != expectedEmail {
		t.Errorf("Expected email %s, got %s", expectedEmail, email)
	}
}
