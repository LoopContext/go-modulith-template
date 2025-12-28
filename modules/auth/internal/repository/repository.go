package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/db/store"
)

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

type SQLRepository struct {
	q  *store.Queries
	db *sql.DB
}

func NewSQLRepository(db *sql.DB) *SQLRepository {
	return &SQLRepository{
		q:  store.New(db),
		db: db,
	}
}

func (r *SQLRepository) WithTx(ctx context.Context, fn func(Repository) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	txRepo := &SQLRepository{
		q:  r.q.WithTx(tx),
		db: r.db,
	}

	if err := fn(txRepo); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *SQLRepository) CreateUser(ctx context.Context, id, email, phone string) error {
	return r.q.CreateUser(ctx, store.CreateUserParams{
		ID:    id,
		Email: sql.NullString{String: email, Valid: email != ""},
		Phone: sql.NullString{String: phone, Valid: phone != ""},
	})
}

func (r *SQLRepository) GetUserByEmail(ctx context.Context, email string) (*store.User, error) {
	u, err := r.q.GetUserByEmail(ctx, sql.NullString{String: email, Valid: true})
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *SQLRepository) GetUserByPhone(ctx context.Context, phone string) (*store.User, error) {
	u, err := r.q.GetUserByPhone(ctx, sql.NullString{String: phone, Valid: true})
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *SQLRepository) CreateMagicCode(ctx context.Context, code, email, phone string, expiresAt time.Time) error {
	return r.q.CreateMagicCode(ctx, store.CreateMagicCodeParams{
		Code:      code,
		UserEmail: sql.NullString{String: email, Valid: email != ""},
		UserPhone: sql.NullString{String: phone, Valid: phone != ""},
		ExpiresAt: expiresAt,
	})
}

func (r *SQLRepository) GetValidMagicCodeByEmail(ctx context.Context, email, code string) (*store.MagicCode, error) {
	mc, err := r.q.GetValidMagicCodeByEmail(ctx, store.GetValidMagicCodeByEmailParams{
		UserEmail: sql.NullString{String: email, Valid: true},
		Code:      code,
	})
	if err != nil {
		return nil, err
	}
	return &mc, nil
}

func (r *SQLRepository) GetValidMagicCodeByPhone(ctx context.Context, phone, code string) (*store.MagicCode, error) {
	mc, err := r.q.GetValidMagicCodeByPhone(ctx, store.GetValidMagicCodeByPhoneParams{
		UserPhone: sql.NullString{String: phone, Valid: true},
		Code:      code,
	})
	if err != nil {
		return nil, err
	}
	return &mc, nil
}

func (r *SQLRepository) InvalidateMagicCodes(ctx context.Context, email, phone string) error {
	if email != "" {
		return r.q.DeleteMagicCodesByEmail(ctx, sql.NullString{String: email, Valid: true})
	}
	if phone != "" {
		return r.q.DeleteMagicCodesByPhone(ctx, sql.NullString{String: phone, Valid: true})
	}
	return nil
}
