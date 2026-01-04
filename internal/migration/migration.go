// Package migration provides multi-module database migration support.
package migration

import (
	"fmt"
	"log/slog"
	"net/url"
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
	return r.runModuleMigrationWithDirection(moduleName, path, "up")
}

// runModuleMigrationWithDirection runs migrations for a single module in the specified direction.
func (r *Runner) runModuleMigrationWithDirection(moduleName, path, direction string) error {
	absPath, err := r.resolveMigrationPath(path)
	if err != nil {
		return fmt.Errorf("failed to resolve migration path: %w", err)
	}

	// Build module-specific DSN with per-module migrations table tracking
	moduleDSN, err := r.buildModuleDSN(moduleName)
	if err != nil {
		return fmt.Errorf("failed to build module DSN: %w", err)
	}

	m, err := migrate.New(
		fmt.Sprintf("file://%s", absPath),
		moduleDSN,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize migration: %w", err)
	}

	defer r.closeMigration(m, moduleName)

	switch direction {
	case "down":
		return r.rollbackMigrations(m, moduleName)
	case "down-all":
		return r.rollbackAllMigrations(m, moduleName)
	case "up":
		return r.applyMigrations(m, moduleName)
	default:
		return fmt.Errorf("invalid direction: %s (must be 'up', 'down', or 'down-all')", direction)
	}
}

// buildModuleDSN builds a module-specific DSN with per-module migrations table tracking.
// This ensures each module tracks its migrations independently using x-migrations-table parameter.
func (r *Runner) buildModuleDSN(moduleName string) (string, error) {
	// Parse the existing DSN
	parsedDSN, err := url.Parse(r.dbDSN)
	if err != nil {
		return "", fmt.Errorf("failed to parse DSN: %w", err)
	}

	// Get existing query parameters
	query := parsedDSN.Query()

	// Set module-specific migrations table: <module_name>_schema_migrations
	// This ensures each module has its own migration tracking table
	migrationsTable := fmt.Sprintf("%s_schema_migrations", moduleName)
	query.Set("x-migrations-table", migrationsTable)

	// Rebuild the DSN with the new query parameters
	parsedDSN.RawQuery = query.Encode()

	return parsedDSN.String(), nil
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

// rollbackMigrations rolls back the last migration and logs the result.
func (r *Runner) rollbackMigrations(m *migrate.Migrate, moduleName string) error {
	// Check current version first to handle version 0 case
	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	// If no migrations are applied (version 0 or ErrNilVersion), there's nothing to rollback
	if err == migrate.ErrNilVersion || version == 0 {
		slog.Debug("No migrations to rollback (database is at version 0)", "module", moduleName)
		return nil
	}

	// If database is dirty, we can't rollback
	if dirty {
		return fmt.Errorf("database is in dirty state at version %d, use migrate-force to fix", version)
	}

	// Attempt to rollback
	err = m.Steps(-1)
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	if err == migrate.ErrNoChange {
		slog.Debug("No migrations to rollback", "module", moduleName)
	} else {
		slog.Info("Migration rolled back successfully", "module", moduleName)
	}

	return nil
}

// rollbackAllMigrations rolls back all migrations to version 0 and logs the result.
func (r *Runner) rollbackAllMigrations(m *migrate.Migrate, moduleName string) error {
	// Check current version first to handle version 0 case
	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	// If no migrations are applied (version 0 or ErrNilVersion), there's nothing to rollback
	if err == migrate.ErrNilVersion || version == 0 {
		slog.Debug("No migrations to rollback (database is at version 0)", "module", moduleName)
		return nil
	}

	// If database is dirty, we can't rollback
	if dirty {
		return fmt.Errorf("database is in dirty state at version %d, use migrate-force to fix", version)
	}

	// Rollback all migrations to version 0
	err = m.Down()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to rollback all migrations: %w", err)
	}

	if err == migrate.ErrNoChange {
		slog.Debug("No migrations to rollback", "module", moduleName)
	} else {
		slog.Info("All migrations rolled back successfully", "module", moduleName, "from_version", version)
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

// DownAll rolls back ALL migrations for all registered modules that implement ModuleMigrations.
// Each module's migrations are tracked independently using per-module migrations tables.
// This will rollback all migrations to version 0, dropping all tables.
func (r *Runner) DownAll() error {
	modules := r.reg.Modules()
	if len(modules) == 0 {
		slog.Info("No modules registered, skipping rollback")
		return nil
	}

	modulesWithMigrations := 0
	rolledBackCount := 0

	var lastError error

	for _, mod := range modules {
		migMod, ok := mod.(registry.ModuleMigrations)
		if !ok {
			continue
		}

		path := migMod.MigrationPath()
		if path == "" {
			continue
		}

		modulesWithMigrations++

		slog.Info("Rolling back all migrations for module", "module", mod.Name(), "path", path)

		if err := r.runModuleMigrationWithDirection(mod.Name(), path, "down-all"); err != nil {
			// Log error but continue with other modules
			slog.Error("Failed to rollback all migrations for module", "module", mod.Name(), "error", err)
			lastError = err
			// Don't return error immediately - try to rollback other modules
			continue
		}

		rolledBackCount++
	}

	if modulesWithMigrations == 0 {
		slog.Info("No modules with migrations found to rollback")
		return nil
	}

	if rolledBackCount == 0 {
		slog.Error("Failed to rollback migrations for any module")

		if lastError != nil {
			return fmt.Errorf("all module rollbacks failed, last error: %w", lastError)
		}

		return fmt.Errorf("failed to rollback migrations for any module")
	}

	slog.Info("All module migrations rolled back successfully", "count", rolledBackCount, "total", modulesWithMigrations)

	return nil
}

// DownForModule rolls back the last migration for a specific module by name.
func (r *Runner) DownForModule(moduleName string) error {
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

	slog.Info("Rolling back last migration for module", "module", moduleName, "path", path)

	return r.runModuleMigrationWithDirection(moduleName, path, "down")
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
