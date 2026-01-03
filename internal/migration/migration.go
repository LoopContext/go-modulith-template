// Package migration provides multi-module database migration support.
package migration

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

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
	absPath, err := r.resolveMigrationPath(path)
	if err != nil {
		return fmt.Errorf("failed to resolve migration path: %w", err)
	}

	m, err := migrate.New(
		fmt.Sprintf("file://%s", absPath),
		r.dbDSN,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize migration: %w", err)
	}

	defer r.closeMigration(m, moduleName)

	return r.applyMigrations(m, moduleName)
}

// resolveMigrationPath resolves a migration path to an absolute path.
func (r *Runner) resolveMigrationPath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	// Verify the path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		// Try to find project root and resolve from there
		projectRoot := findProjectRoot()
		if projectRoot != "" {
			absPath, err = filepath.Abs(filepath.Join(projectRoot, path))
			if err != nil {
				return "", fmt.Errorf("failed to resolve path from project root: %w", err)
			}
		}
	}

	return absPath, nil
}

// closeMigration closes the migration instance and logs any errors.
func (r *Runner) closeMigration(m *migrate.Migrate, moduleName string) {
	sourceErr, dbErr := m.Close()
	if sourceErr != nil {
		slog.Error("Failed to close migration source", "module", moduleName, "error", sourceErr)
	}

	if dbErr != nil {
		slog.Error("Failed to close migration database connection", "module", moduleName, "error", dbErr)
	}
}

// applyMigrations applies migrations and logs the result.
func (r *Runner) applyMigrations(m *migrate.Migrate, moduleName string) error {
	err := m.Up()
	if err != nil && err != migrate.ErrNoChange {
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

// findProjectRoot finds the project root by looking for go.mod file.
func findProjectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		dir = parent
	}

	return ""
}

