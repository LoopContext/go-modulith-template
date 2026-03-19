// Package tasks provides example admin tasks for the modulith template.
package tasks

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/cmelgarejo/go-modulith-template/internal/admin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CleanupExpiredMagicCodesTask removes expired magic codes from the database.
type CleanupExpiredMagicCodesTask struct {
	db *pgxpool.Pool
}

// NewCleanupExpiredMagicCodesTask creates a new cleanup expired magic codes task.
func NewCleanupExpiredMagicCodesTask(db *pgxpool.Pool) admin.Task {
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
	result, err := t.db.Exec(ctx, "DELETE FROM auth.magic_codes WHERE expires_at < CURRENT_TIMESTAMP")
	if err != nil {
		return fmt.Errorf("failed to cleanup expired magic codes: %w", err)
	}

	count := result.RowsAffected()


	slog.Info("Cleaned up expired magic codes", "count", count)

	return nil
}
