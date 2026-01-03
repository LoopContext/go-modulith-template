// Package tasks provides example admin tasks for the modulith template.
package tasks

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/cmelgarejo/go-modulith-template/internal/admin"
)

// CleanupExpiredMagicCodesTask removes expired magic codes from the database.
type CleanupExpiredMagicCodesTask struct {
	db *sql.DB
}

// NewCleanupExpiredMagicCodesTask creates a new cleanup expired magic codes task.
func NewCleanupExpiredMagicCodesTask(db *sql.DB) admin.Task {
	return &CleanupExpiredMagicCodesTask{db: db}
}

// Name returns the task identifier.
func (t *CleanupExpiredMagicCodesTask) Name() string {
	return "cleanup:magic-codes"
}

// Description returns a human-readable description of the task.
func (t *CleanupExpiredMagicCodesTask) Description() string {
	return "Remove expired magic codes"
}

// Execute runs the cleanup task.
func (t *CleanupExpiredMagicCodesTask) Execute(ctx context.Context) error {
	result, err := t.db.ExecContext(ctx, "DELETE FROM magic_codes WHERE expires_at < CURRENT_TIMESTAMP")
	if err != nil {
		return fmt.Errorf("failed to cleanup expired magic codes: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	slog.Info("Cleaned up expired magic codes", "count", count)

	return nil
}
