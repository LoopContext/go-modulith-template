// Package testutil provides testing utilities including testcontainers setup.
package testutil

import (
	"context"
	"fmt"

	"github.com/LoopContext/go-modulith-template/internal/migration"
	"github.com/LoopContext/go-modulith-template/internal/registry"
)

// RunMigrationsForTest runs migrations for a test registry.
func RunMigrationsForTest(_ context.Context, dbDSN string, reg *registry.Registry) error {
	runner := migration.NewRunner(dbDSN, reg)
	if err := runner.RunAll(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
