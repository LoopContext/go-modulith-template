// Package commands provides command-line subcommands for the server.
package commands

import (
	"log/slog"

	"github.com/cmelgarejo/go-modulith-template/cmd/server/setup"
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
