// Package commands provides command-line subcommands for the server.
package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/cmelgarejo/go-modulith-template/cmd/server/observability"
	"github.com/cmelgarejo/go-modulith-template/cmd/server/setup"
	"github.com/cmelgarejo/go-modulith-template/internal/config"
	"github.com/cmelgarejo/go-modulith-template/internal/migration"
	"github.com/cmelgarejo/go-modulith-template/internal/registry"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CommonSetup loads configuration, initializes database, and creates registry.
func CommonSetup() (*config.AppConfig, *pgxpool.Pool, *registry.Registry) {
	observability.InitLoggerEarly()

	systemEnvVars := setup.CaptureSystemEnvVars()
	_ = setup.LoadDotenv()

	cfg, err := config.Load("configs/server.yaml", systemEnvVars)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	observability.InitLogger(cfg.Env, cfg.LogLevel)

	db, err := setup.InitDB(cfg)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	reg := setup.CreateRegistry(cfg, db)
	setup.RegisterModules(reg)

	if err := reg.InitializeAll(); err != nil {
		setup.CloseDB(db)
		slog.Error("Failed to initialize modules", "error", err)
		os.Exit(1)
	}

	return cfg, db, reg
}

// RunMigrations runs migrations for all modules.
func RunMigrations(dbDSN string, reg *registry.Registry) error {
	runner := migration.NewRunner(dbDSN, reg)
	if err := runner.RunAll(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// RunSeedData runs seed data for all modules.
func RunSeedData(dbDSN string, reg *registry.Registry) error {
	seeder, err := migration.NewSeeder(dbDSN, reg)
	if err != nil {
		return fmt.Errorf("failed to create seeder: %w", err)
	}

	defer func() {
		if err := seeder.Close(); err != nil {
			slog.Error("Failed to close seeder connection", "error", err)
		}
	}()

	if err := seeder.SeedAll(context.Background()); err != nil {
		return fmt.Errorf("failed to seed data: %w", err)
	}

	return nil
}
