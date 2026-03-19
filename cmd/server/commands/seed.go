// Package commands provides command-line subcommands for the server.
package commands

import (
	"context"
	"log/slog"

	"github.com/cmelgarejo/go-modulith-template/cmd/server/setup"
	"github.com/cmelgarejo/go-modulith-template/internal/migration"
)

// RunSeedCommand runs the seed command.
func RunSeedCommand() {
	cfg, db, reg := CommonSetup()
	defer setup.CloseDB(db)

	if err := RunSeedData(cfg.DBDSN, reg); err != nil {
		slog.Error("Failed to run seed data", "error", err)
		return
	}

	slog.Info("✅ Seed data completed successfully")
}

// RunSeedModuleCommand runs seed data for a single module.
func RunSeedModuleCommand(moduleName string) {
	cfg, db, reg := CommonSetup()
	defer setup.CloseDB(db)

	seeder, err := migration.NewSeeder(cfg.DBDSN, reg)
	if err != nil {
		slog.Error("Failed to create seeder", "error", err)
		return
	}

	defer func() {
		if err := seeder.Close(); err != nil {
			slog.Error("Failed to close seeder", "error", err)
		}
	}()

	if err := seeder.SeedModule(context.Background(), moduleName); err != nil {
		slog.Error("Failed to seed module", "module", moduleName, "error", err)
		return
	}

	slog.Info("✅ Seed data for module completed successfully", "module", moduleName)
}
