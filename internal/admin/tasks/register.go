// Package tasks provides example admin tasks for the modulith template.
package tasks

import (
	"database/sql"

	"github.com/cmelgarejo/go-modulith-template/internal/admin"
)

// RegisterExampleTasks registers example admin tasks with the runner.
// This demonstrates how to register cleanup and maintenance tasks.
func RegisterExampleTasks(runner *admin.Runner, db *sql.DB) {
	runner.Register(NewCleanupExpiredSessionsTask(db))
	runner.Register(NewCleanupExpiredMagicCodesTask(db))
}
