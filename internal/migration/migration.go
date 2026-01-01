// Package migration provides multi-module database migration support.
package migration

import (
	"fmt"
	"log/slog"

	"github.com/cmelgarejo/go-modulith-template/internal/registry"
	"github.com/golang-migrate/migrate/v4"
	// Import postgres driver for migrations
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	// Import file source driver for migrations
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Runner manages database migrations for multiple modules.
type Runner struct {
	dbDSN string
	reg   *registry.Registry
}

// NewRunner creates a new migration runner.
func NewRunner(dbDSN string, reg *registry.Registry) *Runner {
	return &Runner{
		dbDSN: dbDSN,
		reg:   reg,
	}
}

// RunAll runs migrations for all registered modules that implement ModuleMigrations.
func (r *Runner) RunAll() error {
	modules := r.reg.Modules()
	if len(modules) == 0 {
		slog.Info("No modules registered, skipping migrations")
		return nil
	}

	migratedCount := 0

	for _, mod := range modules {
		migMod, ok := mod.(registry.ModuleMigrations)
		if !ok {
			continue
		}

		path := migMod.MigrationPath()
		if path == "" {
			continue
		}

		slog.Info("Running migrations for module", "module", mod.Name(), "path", path)

		if err := r.runModuleMigration(mod.Name(), path); err != nil {
			return fmt.Errorf("failed to run migrations for module %s: %w", mod.Name(), err)
		}

		migratedCount++
	}

	if migratedCount == 0 {
		slog.Info("No modules with migrations found")
	} else {
		slog.Info("All module migrations completed successfully", "count", migratedCount)
	}

	return nil
}

// runModuleMigration runs migrations for a single module.
func (r *Runner) runModuleMigration(moduleName, path string) error {
	m, err := migrate.New(
		fmt.Sprintf("file://%s", path),
		r.dbDSN,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize migration: %w", err)
	}

	defer func() {
		sourceErr, dbErr := m.Close()
		if sourceErr != nil {
			slog.Error("Failed to close migration source", "module", moduleName, "error", sourceErr)
		}

		if dbErr != nil {
			slog.Error("Failed to close migration database connection", "module", moduleName, "error", dbErr)
		}
	}()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migration: %w", err)
	}

	if err == migrate.ErrNoChange {
		slog.Debug("No new migrations to apply", "module", moduleName)
	} else {
		slog.Info("Migrations applied successfully", "module", moduleName)
	}

	return nil
}

// RunForModule runs migrations for a specific module by name.
func (r *Runner) RunForModule(moduleName string) error {
	mod := r.reg.GetModule(moduleName)
	if mod == nil {
		return fmt.Errorf("module %s not found", moduleName)
	}

	migMod, ok := mod.(registry.ModuleMigrations)
	if !ok {
		return fmt.Errorf("module %s does not implement ModuleMigrations", moduleName)
	}

	path := migMod.MigrationPath()
	if path == "" {
		return fmt.Errorf("module %s has no migration path", moduleName)
	}

	return r.runModuleMigration(moduleName, path)
}

