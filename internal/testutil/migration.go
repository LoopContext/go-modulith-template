// Package testutil provides testing utilities including testcontainers setup.
package testutil

import (
	"context"

	"github.com/cmelgarejo/go-modulith-template/internal/migration"
	"github.com/cmelgarejo/go-modulith-template/internal/registry"
)

// RunMigrationsForTest runs migrations for a test registry.
func RunMigrationsForTest(ctx context.Context, dbDSN string, reg *registry.Registry) error {
	runner := migration.NewRunner(dbDSN, reg)
	return runner.RunAll()
}

