// Package tasks provides example admin tasks for the modulith template.
package tasks

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/cmelgarejo/go-modulith-template/internal/admin"
)

// CleanupExpiredSessionsTask removes expired sessions from the database.
// Sessions are deleted if they expired more than 7 days ago.
type CleanupExpiredSessionsTask struct {
	db *sql.DB
}

// NewCleanupExpiredSessionsTask creates a new cleanup expired sessions task.
func NewCleanupExpiredSessionsTask(db *sql.DB) admin.Task {
	return &CleanupExpiredSessionsTask{db: db}
}

// Name returns the task identifier.
func (t *CleanupExpiredSessionsTask) Name() string {
	return "cleanup:sessions"
}

// Description returns a human-readable description of the task.
func (t *CleanupExpiredSessionsTask) Description() string {
	return "Remove expired sessions (older than 7 days past expiration)"
}

// Execute runs the cleanup task.
func (t *CleanupExpiredSessionsTask) Execute(ctx context.Context) error {
	result, err := t.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP - INTERVAL '7 days'")
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	slog.Info("Cleaned up expired sessions", "count", count)

	return nil
}
