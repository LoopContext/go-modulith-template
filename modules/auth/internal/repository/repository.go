//go:generate mockgen -source=repository.go -destination=mocks/repository_mock.go -package=mocks

// Package repository provides the data access layer for the authentication module.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/cmelgarejo/go-modulith-template/internal/outbox"
	"github.com/cmelgarejo/go-modulith-template/internal/telemetry"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/db/store"
)

// Repository defines the data access methods for the authentication module.
type Repository interface {
	// Transaction support
	WithTx(ctx context.Context, fn func(Repository) error) error

	// User management
	CreateUser(ctx context.Context, id, email, phone string) error
	GetUserByID(ctx context.Context, id string) (*store.AuthUser, error)
	GetUserByEmail(ctx context.Context, email string) (*store.AuthUser, error)
	GetUserByPhone(ctx context.Context, phone string) (*store.AuthUser, error)
	GetUserRole(ctx context.Context, id string) (string, error)
	AssignRole(ctx context.Context, userID, roleName string) error
	RemoveUserRoles(ctx context.Context, userID string) error
	UpdateUserProfile(ctx context.Context, id, displayName, avatarURL, timezone string) error

	// Magic code (passwordless auth)
	CreateMagicCode(ctx context.Context, code, email, phone string, expiresAt time.Time) error
	GetValidMagicCodeByEmail(ctx context.Context, email, code string) (*store.AuthMagicCode, error)
	GetValidMagicCodeByPhone(ctx context.Context, phone, code string) (*store.AuthMagicCode, error)
	InvalidateMagicCodes(ctx context.Context, email, phone string) error
	CleanupExpiredMagicCodes(ctx context.Context) (int, error)

	// Session management
	CreateSession(ctx context.Context, session *Session) error
	GetSessionByID(ctx context.Context, id string) (*Session, error)
	GetSessionByRefreshTokenHash(ctx context.Context, hash string) (*Session, error)
	GetSessionsByUserID(ctx context.Context, userID string) ([]*Session, error)
	UpdateSessionActivity(ctx context.Context, id string) error
	RevokeSession(ctx context.Context, id string) error
	RevokeAllUserSessions(ctx context.Context, userID string, exceptSessionID string) (int, error)
	CleanupExpiredSessions(ctx context.Context) (int, error)

	// Token blacklist
	BlacklistToken(ctx context.Context, tokenHash, userID, reason string, expiresAt time.Time) error
	IsTokenBlacklisted(ctx context.Context, tokenHash string) (bool, error)
	CleanupExpiredBlacklistEntries(ctx context.Context) error

	// Contact change verification
	CreatePendingContactChange(ctx context.Context, id, userID, changeType, newValue, code string, expiresAt time.Time) error
	GetPendingContactChange(ctx context.Context, userID, changeType, code string) (*PendingContactChange, error)
	DeletePendingContactChange(ctx context.Context, id string) error

	// External OAuth accounts
	CreateExternalAccount(ctx context.Context, account *ExternalAccount) error
	GetExternalAccountByProviderUserID(ctx context.Context, provider, providerUserID string) (*ExternalAccount, error)
	GetExternalAccountsByUserID(ctx context.Context, userID string) ([]*ExternalAccount, error)
	GetExternalAccountByProviderAndEmail(ctx context.Context, provider, email string) (*ExternalAccount, error)
	UpdateExternalAccountTokens(ctx context.Context, provider, providerUserID, accessToken, refreshToken string, expiresAt *time.Time) error
	UpdateExternalAccountProfile(ctx context.Context, provider, providerUserID, name, avatarURL, email string, rawData map[string]interface{}) error
	DeleteExternalAccount(ctx context.Context, id, userID string) error
	DeleteExternalAccountByProvider(ctx context.Context, userID, provider string) error
	CountExternalAccountsByUserID(ctx context.Context, userID string) (int64, error)

	// OAuth state tokens
	CreateOAuthState(ctx context.Context, state *OAuthState) error
	GetOAuthState(ctx context.Context, state string) (*OAuthState, error)
	DeleteOAuthState(ctx context.Context, state string) error
	CleanupExpiredOAuthStates(ctx context.Context) error

	// StoreOutbox stores an event in the outbox table within the current transaction.
	StoreOutbox(ctx context.Context, eventName string, payload any) error

	// Verification
	MarkEmailVerified(ctx context.Context, userID string) error
	MarkPhoneVerified(ctx context.Context, userID string) error
}

// Session represents a user session in the database.
type Session struct {
	ID               string
	UserID           string
	RefreshTokenHash string
	UserAgent        string
	IPAddress        string
	CreatedAt        time.Time
	LastActiveAt     time.Time
	ExpiresAt        time.Time
	RevokedAt        *time.Time
}

// PendingContactChange represents a pending email or phone change.
type PendingContactChange struct {
	ID               string
	UserID           string
	ChangeType       string // "email" or "phone"
	NewValue         string
	VerificationCode string
	CreatedAt        time.Time
	ExpiresAt        time.Time
}

// ExternalAccount represents a linked external OAuth account.
type ExternalAccount struct {
	ID             string
	UserID         string
	Provider       string
	ProviderUserID string
	Email          string
	Name           string
	AvatarURL      string
	//nolint:gosec
	AccessToken string // Encrypted
	//nolint:gosec
	RefreshToken   string // Encrypted
	TokenExpiresAt *time.Time
	RawData        map[string]interface{}
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// OAuthState represents an OAuth state token for CSRF protection.
type OAuthState struct {
	State       string
	Provider    string
	RedirectURL string
	UserID      string // Empty for login, set for linking
	Action      string // "login" or "link"
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

// SQLRepository implements the Repository interface using pgx.
type SQLRepository struct {
	q  *store.Queries
	db *pgxpool.Pool
}

// NewSQLRepository creates a new instance of SQLRepository.
func NewSQLRepository(db *pgxpool.Pool) *SQLRepository {
	return &SQLRepository{
		q:  store.New(db),
		db: db,
	}
}

// WithTx executes the given function within a database transaction.
func (r *SQLRepository) WithTx(ctx context.Context, fn func(Repository) error) error {
	if err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		txRepo := &SQLRepository{
			q:  r.q.WithTx(tx),
			db: r.db,
		}

		return fn(txRepo)
	}); err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

// CreateUser persists a new user record in the database.
func (r *SQLRepository) CreateUser(ctx context.Context, id, email, phone string) error {
	ctx, span := telemetry.RepositorySpan(ctx, "auth", "CreateUser", "user")
	defer span.End()

	if err := r.q.CreateUser(ctx, store.CreateUserParams{
		ID:    id,
		Email: pgtype.Text{String: email, Valid: email != ""},
		Phone: pgtype.Text{String: phone, Valid: phone != ""},
	}); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUserByEmail retrieves a user record by their email address.
func (r *SQLRepository) GetUserByEmail(ctx context.Context, email string) (*store.AuthUser, error) {
	ctx, span := telemetry.RepositorySpan(ctx, "auth", "GetUserByEmail", "user")
	defer span.End()

	u, err := r.q.GetUserByEmail(ctx, pgtype.Text{String: email, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &u, nil
}

// GetUserByPhone retrieves a user record by their phone number.
func (r *SQLRepository) GetUserByPhone(ctx context.Context, phone string) (*store.AuthUser, error) {
	ctx, span := telemetry.RepositorySpan(ctx, "auth", "GetUserByPhone", "user")
	defer span.End()

	u, err := r.q.GetUserByPhone(ctx, pgtype.Text{String: phone, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get user by phone: %w", err)
	}

	return &u, nil
}

// CreateMagicCode persists a new magic code for a user.
func (r *SQLRepository) CreateMagicCode(ctx context.Context, code, email, phone string, expiresAt time.Time) error {
	ctx, span := telemetry.RepositorySpan(ctx, "auth", "CreateMagicCode", "magic_code")
	defer span.End()

	if err := r.q.CreateMagicCode(ctx, store.CreateMagicCodeParams{
		Code:      code,
		UserEmail: pgtype.Text{String: email, Valid: email != ""},
		UserPhone: pgtype.Text{String: phone, Valid: phone != ""},
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	}); err != nil {
		return fmt.Errorf("failed to create magic code: %w", err)
	}

	return nil
}

// GetValidMagicCodeByEmail retrieves a valid magic code by user email and code value.
func (r *SQLRepository) GetValidMagicCodeByEmail(ctx context.Context, email, code string) (*store.AuthMagicCode, error) {
	ctx, span := telemetry.RepositorySpan(ctx, "auth", "GetValidMagicCodeByEmail", "magic_code")
	defer span.End()

	mc, err := r.q.GetValidMagicCodeByEmail(ctx, store.GetValidMagicCodeByEmailParams{
		UserEmail: pgtype.Text{String: email, Valid: true},
		Code:      code,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get valid magic code by email: %w", err)
	}

	return &mc, nil
}

// GetValidMagicCodeByPhone retrieves a valid magic code by user phone and code value.
func (r *SQLRepository) GetValidMagicCodeByPhone(ctx context.Context, phone, code string) (*store.AuthMagicCode, error) {
	ctx, span := telemetry.RepositorySpan(ctx, "auth", "GetValidMagicCodeByPhone", "magic_code")
	defer span.End()

	mc, err := r.q.GetValidMagicCodeByPhone(ctx, store.GetValidMagicCodeByPhoneParams{
		UserPhone: pgtype.Text{String: phone, Valid: true},
		Code:      code,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get valid magic code by phone: %w", err)
	}

	return &mc, nil
}

// InvalidateMagicCodes deletes all magic codes associated with a user.
func (r *SQLRepository) InvalidateMagicCodes(ctx context.Context, email, phone string) error {
	if email != "" {
		if err := r.q.DeleteMagicCodesByEmail(ctx, pgtype.Text{String: email, Valid: true}); err != nil {
			return fmt.Errorf("failed to delete magic codes by email: %w", err)
		}

		return nil
	}

	if phone != "" {
		if err := r.q.DeleteMagicCodesByPhone(ctx, pgtype.Text{String: phone, Valid: true}); err != nil {
			return fmt.Errorf("failed to delete magic codes by phone: %w", err)
		}

		return nil
	}

	return nil
}

// GetUserByID retrieves a user record by their ID.
func (r *SQLRepository) GetUserByID(ctx context.Context, id string) (*store.AuthUser, error) {
	u, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return &u, nil
}

// GetUserRole retrieves the role name for a user.
func (r *SQLRepository) GetUserRole(ctx context.Context, userID string) (string, error) {
	role, err := r.q.GetUserRole(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "user", nil
		}

		return "", fmt.Errorf("failed to get user role: %w", err)
	}

	return role, nil
}

// AssignRole assigns a role to a user.
func (r *SQLRepository) AssignRole(ctx context.Context, userID, roleName string) error {
	if err := r.q.AssignUserRole(ctx, store.AssignUserRoleParams{
		UserID:   userID,
		RoleName: roleName,
	}); err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	return nil
}

// RemoveUserRoles removes all roles from a user.
func (r *SQLRepository) RemoveUserRoles(ctx context.Context, userID string) error {
	if err := r.q.RemoveUserRoles(ctx, userID); err != nil {
		return fmt.Errorf("failed to remove user roles: %w", err)
	}

	return nil
}

// UpdateUserProfile updates a user's display name, avatar URL, and timezone.
func (r *SQLRepository) UpdateUserProfile(ctx context.Context, id, displayName, avatarURL, timezone string) error {
	if err := r.q.UpdateUserProfile(ctx, store.UpdateUserProfileParams{
		ID:          id,
		DisplayName: pgtype.Text{String: displayName, Valid: displayName != ""},
		AvatarUrl:   pgtype.Text{String: avatarURL, Valid: avatarURL != ""},
		Timezone:    pgtype.Text{String: timezone, Valid: timezone != ""},
	}); err != nil {
		return fmt.Errorf("failed to update user profile: %w", err)
	}

	return nil
}

// CreateSession creates a new user session.
func (r *SQLRepository) CreateSession(ctx context.Context, session *Session) error {
	ctx, span := telemetry.RepositorySpan(ctx, "auth", "CreateSession", "session")
	defer span.End()

	if err := r.q.CreateSession(ctx, store.CreateSessionParams{
		ID:               session.ID,
		UserID:           session.UserID,
		RefreshTokenHash: session.RefreshTokenHash,
		UserAgent:        pgtype.Text{String: session.UserAgent, Valid: session.UserAgent != ""},
		IpAddress:        pgtype.Text{String: session.IPAddress, Valid: session.IPAddress != ""},
		ExpiresAt:        pgtype.Timestamptz{Time: session.ExpiresAt, Valid: true},
	}); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetSessionByID retrieves a session by its ID.
func (r *SQLRepository) GetSessionByID(ctx context.Context, id string) (*Session, error) {
	s, err := r.q.GetSessionByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get session by id: %w", err)
	}

	return storeSessionToModel(&s), nil
}

// GetSessionByRefreshTokenHash retrieves a session by refresh token hash.
func (r *SQLRepository) GetSessionByRefreshTokenHash(ctx context.Context, hash string) (*Session, error) {
	s, err := r.q.GetSessionByRefreshTokenHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get session by refresh token hash: %w", err)
	}

	return storeSessionToModel(&s), nil
}

// GetSessionsByUserID retrieves all active sessions for a user.
func (r *SQLRepository) GetSessionsByUserID(ctx context.Context, userID string) ([]*Session, error) {
	sessions, err := r.q.GetSessionsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions by user id: %w", err)
	}

	result := make([]*Session, len(sessions))
	for i, s := range sessions {
		result[i] = storeSessionToModel(&s)
	}

	return result, nil
}

// UpdateSessionActivity updates the last active timestamp of a session.
func (r *SQLRepository) UpdateSessionActivity(ctx context.Context, id string) error {
	if err := r.q.UpdateSessionActivity(ctx, id); err != nil {
		return fmt.Errorf("failed to update session activity: %w", err)
	}

	return nil
}

// RevokeSession marks a session as revoked.
func (r *SQLRepository) RevokeSession(ctx context.Context, id string) error {
	if err := r.q.RevokeSession(ctx, id); err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	return nil
}

// RevokeAllUserSessions revokes all sessions for a user, optionally excluding one.
func (r *SQLRepository) RevokeAllUserSessions(ctx context.Context, userID string, exceptSessionID string) (int, error) {
	count, err := r.q.RevokeAllUserSessions(ctx, store.RevokeAllUserSessionsParams{
		UserID:  userID,
		Column2: exceptSessionID,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to revoke all user sessions: %w", err)
	}

	return int(count), nil
}

// BlacklistToken adds a token to the blacklist.
func (r *SQLRepository) BlacklistToken(ctx context.Context, tokenHash, userID, reason string, expiresAt time.Time) error {
	if err := r.q.BlacklistToken(ctx, store.BlacklistTokenParams{
		TokenHash: tokenHash,
		UserID:    userID,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
		Reason:    pgtype.Text{String: reason, Valid: reason != ""},
	}); err != nil {
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	return nil
}

// IsTokenBlacklisted checks if a token is in the blacklist.
func (r *SQLRepository) IsTokenBlacklisted(ctx context.Context, tokenHash string) (bool, error) {
	blacklisted, err := r.q.IsTokenBlacklisted(ctx, tokenHash)
	if err != nil {
		return false, fmt.Errorf("failed to check token blacklist: %w", err)
	}

	return blacklisted, nil
}

// CleanupExpiredBlacklistEntries removes expired entries from the token blacklist.
func (r *SQLRepository) CleanupExpiredBlacklistEntries(ctx context.Context) error {
	if err := r.q.CleanupExpiredBlacklistEntries(ctx); err != nil {
		return fmt.Errorf("failed to cleanup expired blacklist entries: %w", err)
	}

	return nil
}

// CreatePendingContactChange creates a pending email or phone change request.
func (r *SQLRepository) CreatePendingContactChange(ctx context.Context, id, userID, changeType, newValue, code string, expiresAt time.Time) error {
	if err := r.q.CreatePendingContactChange(ctx, store.CreatePendingContactChangeParams{
		ID:               id,
		UserID:           userID,
		ChangeType:       changeType,
		NewValue:         newValue,
		VerificationCode: code,
		ExpiresAt:        pgtype.Timestamptz{Time: expiresAt, Valid: true},
	}); err != nil {
		return fmt.Errorf("failed to create pending contact change: %w", err)
	}

	return nil
}

// GetPendingContactChange retrieves a pending contact change by user, type, and code.
func (r *SQLRepository) GetPendingContactChange(ctx context.Context, userID, changeType, code string) (*PendingContactChange, error) {
	pcc, err := r.q.GetPendingContactChange(ctx, store.GetPendingContactChangeParams{
		UserID:           userID,
		ChangeType:       changeType,
		VerificationCode: code,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get pending contact change: %w", err)
	}

	return &PendingContactChange{
		ID:               pcc.ID,
		UserID:           pcc.UserID,
		ChangeType:       pcc.ChangeType,
		NewValue:         pcc.NewValue,
		VerificationCode: pcc.VerificationCode,
		CreatedAt:        pcc.CreatedAt.Time,
		ExpiresAt:        pcc.ExpiresAt.Time,
	}, nil
}

// DeletePendingContactChange removes a pending contact change.
func (r *SQLRepository) DeletePendingContactChange(ctx context.Context, id string) error {
	if err := r.q.DeletePendingContactChange(ctx, id); err != nil {
		return fmt.Errorf("failed to delete pending contact change: %w", err)
	}

	return nil
}

// storeSessionToModel converts a store.AuthSession to repository.Session.
func storeSessionToModel(s *store.AuthSession) *Session {
	session := &Session{
		ID:               s.ID,
		UserID:           s.UserID,
		RefreshTokenHash: s.RefreshTokenHash,
		CreatedAt:        s.CreatedAt.Time,
		LastActiveAt:     s.LastActiveAt.Time,
		ExpiresAt:        s.ExpiresAt.Time,
	}

	if s.UserAgent.Valid {
		session.UserAgent = s.UserAgent.String
	}

	if s.IpAddress.Valid {
		session.IPAddress = s.IpAddress.String
	}

	if s.RevokedAt.Valid {
		session.RevokedAt = &s.RevokedAt.Time
	}

	return session
}

// =====================
// External OAuth Accounts
// =====================

// CreateExternalAccount creates a new external OAuth account link.
func (r *SQLRepository) CreateExternalAccount(ctx context.Context, account *ExternalAccount) error {
	var rawData []byte

	if account.RawData != nil {
		data, err := json.Marshal(account.RawData)
		if err != nil {
			return fmt.Errorf("failed to marshal raw data: %w", err)
		}

		rawData = data
	}

	var tokenExpiresAt pgtype.Timestamptz
	if account.TokenExpiresAt != nil {
		tokenExpiresAt = pgtype.Timestamptz{Time: *account.TokenExpiresAt, Valid: true}
	}

	if err := r.q.CreateExternalAccount(ctx, store.CreateExternalAccountParams{
		ID:             account.ID,
		UserID:         account.UserID,
		Provider:       account.Provider,
		ProviderUserID: account.ProviderUserID,
		Email:          pgtype.Text{String: account.Email, Valid: account.Email != ""},
		Name:           pgtype.Text{String: account.Name, Valid: account.Name != ""},
		AvatarUrl:      pgtype.Text{String: account.AvatarURL, Valid: account.AvatarURL != ""},
		AccessToken:    pgtype.Text{String: account.AccessToken, Valid: account.AccessToken != ""},
		RefreshToken:   pgtype.Text{String: account.RefreshToken, Valid: account.RefreshToken != ""},
		TokenExpiresAt: tokenExpiresAt,
		RawData:        rawData,
	}); err != nil {
		return fmt.Errorf("failed to create external account: %w", err)
	}

	return nil
}

// GetExternalAccountByProviderUserID retrieves an external account by provider and provider user ID.
func (r *SQLRepository) GetExternalAccountByProviderUserID(ctx context.Context, provider, providerUserID string) (*ExternalAccount, error) {
	ea, err := r.q.GetExternalAccountByProviderAndUserID(ctx, store.GetExternalAccountByProviderAndUserIDParams{
		Provider:       provider,
		ProviderUserID: providerUserID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get external account: %w", err)
	}

	return storeExternalAccountToModel(&ea)
}

// GetExternalAccountsByUserID retrieves all external accounts for a user.
func (r *SQLRepository) GetExternalAccountsByUserID(ctx context.Context, userID string) ([]*ExternalAccount, error) {
	accounts, err := r.q.GetExternalAccountsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get external accounts: %w", err)
	}

	result := make([]*ExternalAccount, len(accounts))

	for i, ea := range accounts {
		acc, err := storeExternalAccountToModel(&ea)
		if err != nil {
			return nil, err
		}

		result[i] = acc
	}

	return result, nil
}

// GetExternalAccountByProviderAndEmail retrieves an external account by provider and email.
func (r *SQLRepository) GetExternalAccountByProviderAndEmail(ctx context.Context, provider, email string) (*ExternalAccount, error) {
	ea, err := r.q.GetExternalAccountByProviderAndEmail(ctx, store.GetExternalAccountByProviderAndEmailParams{
		Provider: provider,
		Email:    pgtype.Text{String: email, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get external account by email: %w", err)
	}

	return storeExternalAccountToModel(&ea)
}

// UpdateExternalAccountTokens updates the OAuth tokens for an external account.
func (r *SQLRepository) UpdateExternalAccountTokens(ctx context.Context, provider, providerUserID, accessToken, refreshToken string, expiresAt *time.Time) error {
	var tokenExpiresAt pgtype.Timestamptz
	if expiresAt != nil {
		tokenExpiresAt = pgtype.Timestamptz{Time: *expiresAt, Valid: true}
	}

	if err := r.q.UpdateExternalAccountTokens(ctx, store.UpdateExternalAccountTokensParams{
		Provider:       provider,
		ProviderUserID: providerUserID,
		AccessToken:    pgtype.Text{String: accessToken, Valid: accessToken != ""},
		RefreshToken:   pgtype.Text{String: refreshToken, Valid: refreshToken != ""},
		TokenExpiresAt: tokenExpiresAt,
	}); err != nil {
		return fmt.Errorf("failed to update external account tokens: %w", err)
	}

	return nil
}

// UpdateExternalAccountProfile updates the profile info for an external account.
func (r *SQLRepository) UpdateExternalAccountProfile(ctx context.Context, provider, providerUserID, name, avatarURL, email string, rawData map[string]interface{}) error {
	var rawDataNullable []byte

	if rawData != nil {
		data, err := json.Marshal(rawData)
		if err != nil {
			return fmt.Errorf("failed to marshal raw data: %w", err)
		}

		rawDataNullable = data
	}

	if err := r.q.UpdateExternalAccountProfile(ctx, store.UpdateExternalAccountProfileParams{
		Provider:       provider,
		ProviderUserID: providerUserID,
		Name:           pgtype.Text{String: name, Valid: name != ""},
		AvatarUrl:      pgtype.Text{String: avatarURL, Valid: avatarURL != ""},
		Email:          pgtype.Text{String: email, Valid: email != ""},
		RawData:        rawDataNullable,
	}); err != nil {
		return fmt.Errorf("failed to update external account profile: %w", err)
	}

	return nil
}

// DeleteExternalAccount deletes an external account by ID.
func (r *SQLRepository) DeleteExternalAccount(ctx context.Context, id, userID string) error {
	if err := r.q.DeleteExternalAccount(ctx, store.DeleteExternalAccountParams{
		ID:     id,
		UserID: userID,
	}); err != nil {
		return fmt.Errorf("failed to delete external account: %w", err)
	}

	return nil
}

// DeleteExternalAccountByProvider deletes an external account by user ID and provider.
func (r *SQLRepository) DeleteExternalAccountByProvider(ctx context.Context, userID, provider string) error {
	if err := r.q.DeleteExternalAccountByProvider(ctx, store.DeleteExternalAccountByProviderParams{
		UserID:   userID,
		Provider: provider,
	}); err != nil {
		return fmt.Errorf("failed to delete external account by provider: %w", err)
	}

	return nil
}

// CountExternalAccountsByUserID counts the number of external accounts for a user.
func (r *SQLRepository) CountExternalAccountsByUserID(ctx context.Context, userID string) (int64, error) {
	count, err := r.q.CountExternalAccountsByUserID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to count external accounts: %w", err)
	}

	return count, nil
}

// storeExternalAccountToModel converts a store.AuthUserExternalAccount to repository.ExternalAccount.
func storeExternalAccountToModel(ea *store.AuthUserExternalAccount) (*ExternalAccount, error) {
	account := &ExternalAccount{
		ID:             ea.ID,
		UserID:         ea.UserID,
		Provider:       ea.Provider,
		ProviderUserID: ea.ProviderUserID,
		CreatedAt:      ea.CreatedAt.Time,
		UpdatedAt:      ea.UpdatedAt.Time,
	}

	if ea.Email.Valid {
		account.Email = ea.Email.String
	}

	if ea.Name.Valid {
		account.Name = ea.Name.String
	}

	if ea.AvatarUrl.Valid {
		account.AvatarURL = ea.AvatarUrl.String
	}

	if ea.AccessToken.Valid {
		account.AccessToken = ea.AccessToken.String
	}

	if ea.RefreshToken.Valid {
		account.RefreshToken = ea.RefreshToken.String
	}

	if ea.TokenExpiresAt.Valid {
		account.TokenExpiresAt = &ea.TokenExpiresAt.Time
	}

	if len(ea.RawData) > 0 {
		if err := json.Unmarshal(ea.RawData, &account.RawData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal raw data: %w", err)
		}
	}

	return account, nil
}

// =====================
// OAuth State Tokens
// =====================

// CreateOAuthState creates a new OAuth state authtoken.
func (r *SQLRepository) CreateOAuthState(ctx context.Context, state *OAuthState) error {
	if err := r.q.CreateOAuthState(ctx, store.CreateOAuthStateParams{
		State:       state.State,
		Provider:    state.Provider,
		RedirectUrl: pgtype.Text{String: state.RedirectURL, Valid: state.RedirectURL != ""},
		UserID:      pgtype.Text{String: state.UserID, Valid: state.UserID != ""},
		Action:      state.Action,
		ExpiresAt:   pgtype.Timestamptz{Time: state.ExpiresAt, Valid: true},
	}); err != nil {
		return fmt.Errorf("failed to create oauth state: %w", err)
	}

	return nil
}

// GetOAuthState retrieves an OAuth state authtoken.
func (r *SQLRepository) GetOAuthState(ctx context.Context, state string) (*OAuthState, error) {
	s, err := r.q.GetOAuthState(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("failed to get oauth state: %w", err)
	}

	result := &OAuthState{
		State:     s.State,
		Provider:  s.Provider,
		Action:    s.Action,
		CreatedAt: s.CreatedAt.Time,
		ExpiresAt: s.ExpiresAt.Time,
	}

	if s.RedirectUrl.Valid {
		result.RedirectURL = s.RedirectUrl.String
	}

	if s.UserID.Valid {
		result.UserID = s.UserID.String
	}

	return result, nil
}

// DeleteOAuthState deletes an OAuth state authtoken.
func (r *SQLRepository) DeleteOAuthState(ctx context.Context, state string) error {
	if err := r.q.DeleteOAuthState(ctx, state); err != nil {
		return fmt.Errorf("failed to delete oauth state: %w", err)
	}

	return nil
}

// CleanupExpiredOAuthStates removes expired OAuth state tokens.
func (r *SQLRepository) CleanupExpiredOAuthStates(ctx context.Context) error {
	if err := r.q.CleanupExpiredOAuthStates(ctx); err != nil {
		return fmt.Errorf("failed to cleanup expired oauth states: %w", err)
	}

	return nil
}

// MarkEmailVerified marks a user's email as verified.
func (r *SQLRepository) MarkEmailVerified(ctx context.Context, userID string) error {
	if err := r.q.MarkEmailVerified(ctx, userID); err != nil {
		return fmt.Errorf("failed to mark email verified: %w", err)
	}

	return nil
}

// MarkPhoneVerified marks a user's phone as verified.
func (r *SQLRepository) MarkPhoneVerified(ctx context.Context, userID string) error {
	if err := r.q.MarkPhoneVerified(ctx, userID); err != nil {
		return fmt.Errorf("failed to mark phone verified: %w", err)
	}

	return nil
}

// CleanupExpiredSessions removes expired sessions (older than 7 days past expiration).
// Returns the number of sessions deleted.
func (r *SQLRepository) CleanupExpiredSessions(ctx context.Context) (int, error) {
	tag, err := r.db.Exec(ctx, "DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP - INTERVAL '7 days'")
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	return int(tag.RowsAffected()), nil
}

// CleanupExpiredMagicCodes removes expired magic codes.
// Returns the number of magic codes deleted.
func (r *SQLRepository) CleanupExpiredMagicCodes(ctx context.Context) (int, error) {
	tag, err := r.db.Exec(ctx, "DELETE FROM magic_codes WHERE expires_at < CURRENT_TIMESTAMP")
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired magic codes: %w", err)
	}

	return int(tag.RowsAffected()), nil
}

// StoreOutbox stores an event in the outbox table.
func (r *SQLRepository) StoreOutbox(ctx context.Context, eventName string, payload any) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	id := fmt.Sprintf("outbox_%d", time.Now().UnixNano())

	if err := r.q.StoreOutbox(ctx, store.StoreOutboxParams{
		ID:        id,
		EventName: eventName,
		Payload:   payloadBytes,
	}); err != nil {
		return fmt.Errorf("failed to store outbox entry: %w", err)
	}

	return nil
}

// GetUnpublished retrieves unpublished events from the outbox.
func (r *SQLRepository) GetUnpublished(ctx context.Context, limit int) ([]outbox.Entry, error) {
	rows, err := r.q.GetUnpublishedOutbox(ctx, int32(limit)) // #nosec G115
	if err != nil {
		return nil, fmt.Errorf("failed to get unpublished outbox entries: %w", err)
	}

	entries := make([]outbox.Entry, len(rows))
	for i, row := range rows {
		entries[i] = outbox.Entry{
			ID:        row.ID,
			EventName: row.EventName,
			Payload:   row.Payload,
			CreatedAt: row.CreatedAt.Time,
		}
		if row.PublishedAt.Valid {
			entries[i].PublishedAt = &row.PublishedAt.Time
		}
	}

	return entries, nil
}

// MarkPublished marks events as published in the outbox.
func (r *SQLRepository) MarkPublished(ctx context.Context, ids []string) error {
	if err := r.q.MarkOutboxAsPublished(ctx, ids); err != nil {
		return fmt.Errorf("failed to mark outbox entries as published: %w", err)
	}

	return nil
}

// Ensure SQLRepository implements outbox.PublisherRepository
var _ outbox.PublisherRepository = (*SQLRepository)(nil)
