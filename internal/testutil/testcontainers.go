// Package testutil provides testing utilities including testcontainers setup.
package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer wraps a testcontainers postgres instance.
type PostgresContainer struct {
	container *postgres.PostgresContainer
	DSN       string
}

// NewPostgresContainer creates and starts a PostgreSQL testcontainer.
func NewPostgresContainer(ctx context.Context, t *testing.T) (*PostgresContainer, error) {
	t.Helper()

	pgContainer, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	return &PostgresContainer{
		container: pgContainer,
		DSN:       connStr,
	}, nil
}

// Close terminates the container.
func (c *PostgresContainer) Close(ctx context.Context) error {
	if c.container != nil {
		if err := c.container.Terminate(ctx); err != nil {
			return fmt.Errorf("failed to terminate container: %w", err)
		}
	}

	return nil
}

// DB returns a database connection to the test container.
func (c *PostgresContainer) DB(ctx context.Context) (*sql.DB, error) {
	db, err := sql.Open("pgx", c.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

