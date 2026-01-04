// Package commands provides command-line subcommands for the server.
package commands

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/cmelgarejo/go-modulith-template/cmd/server/setup"
	"github.com/cmelgarejo/go-modulith-template/internal/migration"
	"github.com/cmelgarejo/go-modulith-template/internal/registry"
)

// RunMigrateCommand runs the migrate command.
func RunMigrateCommand() {
	cfg, db, reg := CommonSetup()
	defer setup.CloseDB(db)

	if err := RunMigrations(cfg.DBDSN, reg); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		return
	}

	slog.Info("✅ Migrations completed successfully")
}

// RunMigrateDownCommand runs the migrate-down command.
func RunMigrateDownCommand() {
	cfg, db, reg := CommonSetup()
	defer setup.CloseDB(db)

	if err := RunDownMigrations(cfg.DBDSN, reg); err != nil {
		slog.Error("Failed to rollback migrations", "error", err)
		os.Exit(1)
	}

	slog.Info("✅ Migrations rolled back successfully")
}

// RunDownMigrations runs down migrations for all modules.
func RunDownMigrations(dbDSN string, reg *registry.Registry) error {
	runner := migration.NewRunner(dbDSN, reg)
	if err := runner.DownAll(); err != nil {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}

	return nil
}
