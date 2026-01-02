// Package commands provides command-line subcommands for the server.
package commands

import (
	"context"
	"log/slog"
	"os"

	"github.com/cmelgarejo/go-modulith-template/internal/admin"
	adminTasks "github.com/cmelgarejo/go-modulith-template/internal/admin/tasks"
	"github.com/cmelgarejo/go-modulith-template/cmd/server/setup"
)

// RunAdminCommand runs the admin command.
func RunAdminCommand(taskName string) {
	_, db, _ := CommonSetup()

	runner := admin.NewRunner()

	// Register example admin tasks
	adminTasks.RegisterExampleTasks(runner, db)

	// TODO: Modules can register admin tasks here via an interface
	// For now, show available tasks
	if !runner.Has(taskName) {
		slog.Error("Unknown admin task", "task", taskName)

		tasks := runner.List()
		if len(tasks) == 0 {
			slog.Info("No admin tasks registered")
		} else {
			slog.Info("Available admin tasks:")

			for _, t := range tasks {
				slog.Info("  " + t.Name() + " - " + t.Description())
			}
		}

		setup.CloseDB(db)
		os.Exit(1)
	}

	if err := runner.Run(context.Background(), taskName); err != nil {
		slog.Error("Admin task failed", "task", taskName, "error", err)
		setup.CloseDB(db)
		os.Exit(1)
	}

	setup.CloseDB(db)
	slog.Info("✅ Admin task completed successfully", "task", taskName)
}

