//go:generate mockgen -source=repository.go -destination=mocks/repository_mock.go -package=mocks

// Package repository provides the data access layer for the authentication module.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/db/store"
)

// Repository defines the data access methods for the authentication module.
type Repository interface {
	WithTx(ctx context.Context, fn func(Repository) error) error
	CreateUser(ctx context.Context, id, email, phone string) error
	GetUserByEmail(ctx context.Context, email string) (*store.User, error)
	GetUserByPhone(ctx context.Context, phone string) (*store.User, error)
	CreateMagicCode(ctx context.Context, code, email, phone string, expiresAt time.Time) error
	GetValidMagicCodeByEmail(ctx context.Context, email, code string) (*store.MagicCode, error)
	GetValidMagicCodeByPhone(ctx context.Context, phone, code string) (*store.MagicCode, error)
	InvalidateMagicCodes(ctx context.Context, email, phone string) error
}

// SQLRepository implements the Repository interface using a SQL database.
type SQLRepository struct {
	q  *store.Queries
	db *sql.DB
}

// NewSQLRepository creates a new instance of SQLRepository.
func NewSQLRepository(db *sql.DB) *SQLRepository {
	return &SQLRepository{
		q:  store.New(db),
		db: db,
	}
}

// WithTx executes the given function within a database transaction.
func (r *SQLRepository) WithTx(ctx context.Context, fn func(Repository) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	txRepo := &SQLRepository{
		q:  r.q.WithTx(tx),
		db: r.db,
	}

	if err := fn(txRepo); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// CreateUser persists a new user record in the database.
func (r *SQLRepository) CreateUser(ctx context.Context, id, email, phone string) error {
	if err := r.q.CreateUser(ctx, store.CreateUserParams{
		ID:    id,
		Email: sql.NullString{String: email, Valid: email != ""},
		Phone: sql.NullString{String: phone, Valid: phone != ""},
	}); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUserByEmail retrieves a user record by their email address.
func (r *SQLRepository) GetUserByEmail(ctx context.Context, email string) (*store.User, error) {
	u, err := r.q.GetUserByEmail(ctx, sql.NullString{String: email, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &u, nil
}

// GetUserByPhone retrieves a user record by their phone number.
func (r *SQLRepository) GetUserByPhone(ctx context.Context, phone string) (*store.User, error) {
	u, err := r.q.GetUserByPhone(ctx, sql.NullString{String: phone, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get user by phone: %w", err)
	}

	return &u, nil
}

// CreateMagicCode persists a new magic code for a user.
func (r *SQLRepository) CreateMagicCode(ctx context.Context, code, email, phone string, expiresAt time.Time) error {
	if err := r.q.CreateMagicCode(ctx, store.CreateMagicCodeParams{
		Code:      code,
		UserEmail: sql.NullString{String: email, Valid: email != ""},
		UserPhone: sql.NullString{String: phone, Valid: phone != ""},
		ExpiresAt: expiresAt,
	}); err != nil {
		return fmt.Errorf("failed to create magic code: %w", err)
	}

	return nil
}

// GetValidMagicCodeByEmail retrieves a valid magic code by user email and code value.
func (r *SQLRepository) GetValidMagicCodeByEmail(ctx context.Context, email, code string) (*store.MagicCode, error) {
	mc, err := r.q.GetValidMagicCodeByEmail(ctx, store.GetValidMagicCodeByEmailParams{
		UserEmail: sql.NullString{String: email, Valid: true},
		Code:      code,
		ExpiresAt: time.Now(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get valid magic code by email: %w", err)
	}

	return &mc, nil
}

// GetValidMagicCodeByPhone retrieves a valid magic code by user phone and code value.
func (r *SQLRepository) GetValidMagicCodeByPhone(ctx context.Context, phone, code string) (*store.MagicCode, error) {
	mc, err := r.q.GetValidMagicCodeByPhone(ctx, store.GetValidMagicCodeByPhoneParams{
		UserPhone: sql.NullString{String: phone, Valid: true},
		Code:      code,
		ExpiresAt: time.Now(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get valid magic code by phone: %w", err)
	}

	return &mc, nil
}

// InvalidateMagicCodes deletes all magic codes associated with a user.
func (r *SQLRepository) InvalidateMagicCodes(ctx context.Context, email, phone string) error {
	if email != "" {
		if err := r.q.DeleteMagicCodesByEmail(ctx, sql.NullString{String: email, Valid: true}); err != nil {
			return fmt.Errorf("failed to delete magic codes by email: %w", err)
		}

		return nil
	}

	if phone != "" {
		if err := r.q.DeleteMagicCodesByPhone(ctx, sql.NullString{String: phone, Valid: true}); err != nil {
			return fmt.Errorf("failed to delete magic codes by phone: %w", err)
		}

		return nil
	}

	return nil
}
