// Package commands provides command-line subcommands for the server.
package commands

import (
	"log/slog"

	"github.com/cmelgarejo/go-modulith-template/cmd/server/setup"
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

