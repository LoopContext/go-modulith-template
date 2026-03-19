// Package tasks provides example admin tasks for the modulith template.
package tasks

import (
	"github.com/cmelgarejo/go-modulith-template/internal/admin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegisterExampleTasks registers example admin tasks with the runner.
// This demonstrates how to register cleanup and maintenance tasks.
func RegisterExampleTasks(runner *admin.Runner, db *pgxpool.Pool) {
	runner.Register(NewCleanupExpiredSessionsTask(db))
	runner.Register(NewCleanupExpiredMagicCodesTask(db))
}
