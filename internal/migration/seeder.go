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
)

// ModuleSeeder defines the interface for modules that provide seed data.
type ModuleSeeder interface {
	// Name returns the module identifier.
	Name() string

	// SeedPath returns the path to the module's seed directory.
	// Return empty string if the module has no seed data.
	SeedPath() string
}

// ModuleRegistry defines the basic interface for accessing registered modules.
type ModuleRegistry interface {
	Modules() []interface{}
}

// ModuleProvider defines the interface for accessing registered modules.
type ModuleProvider interface {
	GetModules() []ModuleSeeder
}

// Seeder manages seed data execution for modules.
type Seeder struct {
	db       *sql.DB
	provider ModuleProvider
}

// registryAdapter adapts a ModuleRegistry to ModuleProvider.
type registryAdapter struct {
	registry ModuleRegistry
}

func (r *registryAdapter) GetModules() []ModuleSeeder {
	modules := r.registry.Modules()
	seeders := make([]ModuleSeeder, 0)

	for _, mod := range modules {
		if seeder, ok := mod.(ModuleSeeder); ok {
			seeders = append(seeders, seeder)
		}
	}

	return seeders
}

// NewSeeder creates a new seed data runner.
func NewSeeder(dbDSN string, registry ModuleRegistry) (*Seeder, error) {
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
		provider: &registryAdapter{registry: registry},
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

// SeedAll runs seed data for all modules that implement ModuleSeeder.
func (s *Seeder) SeedAll(ctx context.Context) error {
	modules := s.provider.GetModules()
	if len(modules) == 0 {
		slog.Info("No modules with seed data registered")
		return nil
	}

	for _, seeder := range modules {
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

		slog.Info("Running seed data", "module", moduleName, "path", seedPath)

		if err := s.runSeedFiles(ctx, seedPath, moduleName); err != nil {
			return fmt.Errorf("failed to seed module %s: %w", moduleName, err)
		}

		slog.Info("Seed data completed", "module", moduleName)
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
