package migration

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LoopContext/go-modulith-template/internal/registry"
)

// ModuleSeeder defines the interface for modules that provide seed data.
type ModuleSeeder interface {
	// Name returns the module identifier.
	Name() string

	// SeedPath returns the path to the module's seed directory.
	// Return empty string if the module has no seed data.
	SeedPath() string
}

// ModuleProgrammaticSeeder defines the interface for modules that provide seed data programmatically.
type ModuleProgrammaticSeeder interface {
	// Seed runs programmatic seed data using the application's registry/dependencies.
	Seed(ctx context.Context, r interface{}) error
}

// ModuleRegistry defines the basic interface for accessing registered modules.
type ModuleRegistry interface {
	Modules() []registry.Module
}

// ModuleProvider defines the interface for accessing registered modules.
type ModuleProvider interface {
	GetSQLSeeders() []ModuleSeeder
	GetProgrammaticSeeders() []ModuleProgrammaticSeeder
}

// Seeder manages seed data execution for modules.
type Seeder struct {
	db       *sql.DB
	provider ModuleProvider
	registry ModuleRegistry
}

// registryAdapter adapts a ModuleRegistry to ModuleProvider.
type registryAdapter struct {
	registry ModuleRegistry
}

func (r *registryAdapter) GetSQLSeeders() []ModuleSeeder {
	modules := r.registry.Modules()
	seeders := make([]ModuleSeeder, 0)

	for _, mod := range modules {
		if seeder, ok := mod.(ModuleSeeder); ok {
			seeders = append(seeders, seeder)
		}
	}

	return seeders
}

func (r *registryAdapter) GetProgrammaticSeeders() []ModuleProgrammaticSeeder {
	modules := r.registry.Modules()
	seeders := make([]ModuleProgrammaticSeeder, 0)

	for _, mod := range modules {
		if seeder, ok := mod.(ModuleProgrammaticSeeder); ok {
			seeders = append(seeders, seeder)
		}
	}

	return seeders
}

// NewSeeder creates a new seed data runner.
func NewSeeder(dbDSN string, r ModuleRegistry) (*Seeder, error) {
	db, err := sql.Open("pgx", dbDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Seeder{
		db:       db,
		provider: &registryAdapter{registry: r},
		registry: r,
	}, nil
}

// Close closes the database connection.
func (s *Seeder) Close() error {
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
	}

	return nil
}

// SeedAll runs seed data for all modules that implement ModuleSeeder or ModuleProgrammaticSeeder.
func (s *Seeder) SeedAll(ctx context.Context) error {
	sqlSeeders := s.provider.GetSQLSeeders()
	progSeeders := s.provider.GetProgrammaticSeeders()

	if len(sqlSeeders) == 0 && len(progSeeders) == 0 {
		slog.Info("No modules with seed data registered")
		return nil
	}

	// 1. Run SQL Seeds
	if err := s.runSQLSeeds(ctx, sqlSeeders); err != nil {
		return err
	}

	// 2. Run Programmatic Seeds
	if err := s.runProgrammaticSeeds(ctx, progSeeders); err != nil {
		return err
	}

	return nil
}

// SeedModule runs seed data for a single module.
// SeedModule runs seed data for a single module.
func (s *Seeder) SeedModule(ctx context.Context, moduleName string) error {
	ranSQL, err := s.seedSQLModule(ctx, moduleName)
	if err != nil {
		return err
	}

	ranProg, err := s.seedProgrammaticModule(ctx, moduleName)
	if err != nil {
		return err
	}

	if !ranSQL && !ranProg {
		return fmt.Errorf("module %s not found or has no seed data", moduleName)
	}

	return nil
}

func (s *Seeder) seedSQLModule(ctx context.Context, moduleName string) (bool, error) {
	sqlSeeders := s.provider.GetSQLSeeders()

	for _, seeder := range sqlSeeders {
		if seeder.Name() == moduleName {
			if err := s.runSQLSeeds(ctx, []ModuleSeeder{seeder}); err != nil {
				return true, err
			}

			return true, nil
		}
	}

	return false, nil
}

func (s *Seeder) seedProgrammaticModule(ctx context.Context, moduleName string) (bool, error) {
	progSeeders := s.provider.GetProgrammaticSeeders()

	for _, seeder := range progSeeders {
		if m, ok := seeder.(interface{ Name() string }); ok && m.Name() == moduleName {
			if err := s.runProgrammaticSeeds(ctx, []ModuleProgrammaticSeeder{seeder}); err != nil {
				return true, err
			}

			return true, nil
		}
	}

	return false, nil
}

func (s *Seeder) runSQLSeeds(ctx context.Context, seeders []ModuleSeeder) error {
	for _, seeder := range seeders {
		seedPath := seeder.SeedPath()
		if seedPath == "" {
			continue
		}

		moduleName := seeder.Name()

		// Check if seed directory exists
		if _, err := os.Stat(seedPath); os.IsNotExist(err) {
			slog.Debug("Seed directory does not exist, skipping", "module", moduleName, "path", seedPath)
			continue
		}

		slog.Info("Running SQL seed data", "module", moduleName, "path", seedPath)

		if err := s.runSeedFiles(ctx, seedPath, moduleName); err != nil {
			return fmt.Errorf("failed to SQL seed module %s: %w", moduleName, err)
		}

		slog.Info("SQL seed data completed", "module", moduleName)
	}

	return nil
}

func (s *Seeder) runProgrammaticSeeds(ctx context.Context, seeders []ModuleProgrammaticSeeder) error {
	for _, seeder := range seeders {
		// We need the module name for logging, we can type assert to registry.Module if needed
		moduleName := "unknown"
		if m, ok := seeder.(interface{ Name() string }); ok {
			moduleName = m.Name()
		}

		slog.Info("Running programmatic seed data", "module", moduleName)

		if err := seeder.Seed(ctx, s.registry); err != nil {
			return fmt.Errorf("failed to programmatic seed module %s: %w", moduleName, err)
		}

		slog.Info("Programmatic seed data completed", "module", moduleName)
	}

	return nil
}

// runSeedFiles executes all .sql files in the seed directory in alphabetical order.
func (s *Seeder) runSeedFiles(ctx context.Context, seedPath, moduleName string) error {
	files, err := os.ReadDir(seedPath)
	if err != nil {
		return fmt.Errorf("failed to read seed directory: %w", err)
	}

	// Filter and sort SQL files
	sqlFiles := make([]string, 0, len(files))

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		sqlFiles = append(sqlFiles, file.Name())
	}

	sort.Strings(sqlFiles)

	if len(sqlFiles) == 0 {
		slog.Debug("No seed files found", "module", moduleName, "path", seedPath)
		return nil
	}

	for _, fileName := range sqlFiles {
		filePath := filepath.Join(seedPath, fileName)

		slog.Debug("Executing seed file", "module", moduleName, "file", fileName)

		if err := s.executeSQLFile(ctx, filePath); err != nil {
			return fmt.Errorf("failed to execute seed file %s: %w", fileName, err)
		}
	}

	return nil
}

// executeSQLFile reads and executes a SQL file.
func (s *Seeder) executeSQLFile(ctx context.Context, filePath string) error {
	content, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Execute the SQL content
	if _, err := s.db.ExecContext(ctx, string(content)); err != nil {
		return fmt.Errorf("failed to execute SQL: %w", err)
	}

	return nil
}
