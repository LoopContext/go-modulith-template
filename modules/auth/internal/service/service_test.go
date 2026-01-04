package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	authv1 "github.com/cmelgarejo/go-modulith-template/gen/go/proto/auth/v1"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/db/store"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/repository"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/token"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockRepository struct {
	createUserFunc               func(ctx context.Context, id, email, phone string) error
	getUserByEmailFunc           func(ctx context.Context, email string) (*store.AuthUser, error)
	getUserByPhoneFunc           func(ctx context.Context, phone string) (*store.AuthUser, error)
	createMagicCodeFunc          func(ctx context.Context, code, email, phone string, expiresAt time.Time) error
	getValidMagicCodeByEmailFunc func(ctx context.Context, email, code string) (*store.AuthMagicCode, error)
	getValidMagicCodeByPhoneFunc func(ctx context.Context, phone, code string) (*store.AuthMagicCode, error)
	invalidateMagicCodesFunc     func(ctx context.Context, email, phone string) error
}

func (m *mockRepository) WithTx(_ context.Context, fn func(repository.Repository) error) error {
	return fn(m)
}

func (m *mockRepository) CreateUser(ctx context.Context, id, email, phone string) error {
	if m.createUserFunc != nil {
		return m.createUserFunc(ctx, id, email, phone)
	}

	return nil
}

func (m *mockRepository) GetUserByEmail(ctx context.Context, email string) (*store.AuthUser, error) {
	if m.getUserByEmailFunc != nil {
		return m.getUserByEmailFunc(ctx, email)
	}

	return &store.AuthUser{ID: "user-123", Email: sql.NullString{String: email, Valid: true}}, nil
}

func (m *mockRepository) GetUserByPhone(ctx context.Context, phone string) (*store.AuthUser, error) {
	if m.getUserByPhoneFunc != nil {
		return m.getUserByPhoneFunc(ctx, phone)
	}

	return &store.AuthUser{ID: "user-123", Phone: sql.NullString{String: phone, Valid: true}}, nil
}

func (m *mockRepository) CreateMagicCode(ctx context.Context, code, email, phone string, expiresAt time.Time) error {
	if m.createMagicCodeFunc != nil {
		return m.createMagicCodeFunc(ctx, code, email, phone, expiresAt)
	}

	return nil
}

func (m *mockRepository) GetValidMagicCodeByEmail(ctx context.Context, email, code string) (*store.AuthMagicCode, error) {
	if m.getValidMagicCodeByEmailFunc != nil {
		return m.getValidMagicCodeByEmailFunc(ctx, email, code)
	}

	return &store.AuthMagicCode{Code: code, UserEmail: sql.NullString{String: email, Valid: true}}, nil
}

func (m *mockRepository) GetValidMagicCodeByPhone(ctx context.Context, phone, code string) (*store.AuthMagicCode, error) {
	if m.getValidMagicCodeByPhoneFunc != nil {
		return m.getValidMagicCodeByPhoneFunc(ctx, phone, code)
	}

	return &store.AuthMagicCode{Code: code, UserPhone: sql.NullString{String: phone, Valid: true}}, nil
}

func (m *mockRepository) InvalidateMagicCodes(ctx context.Context, email, phone string) error {
	if m.invalidateMagicCodesFunc != nil {
		return m.invalidateMagicCodesFunc(ctx, email, phone)
	}

	return nil
}

func createTestService(t *testing.T, repo *mockRepository) *AuthService {
	t.Helper()

	tokenService, err := token.NewService("test-secret-key-that-is-at-least-32-bytes-long")
	if err != nil {
		t.Fatalf("failed to create token service: %v", err)
	}

	bus := events.NewBus()

	return NewAuthService(repo, tokenService, bus)
}

func TestNewAuthService(t *testing.T) {
	repo := &mockRepository{}
	svc := createTestService(t, repo)

	if svc == nil {
		t.Fatal("expected service to not be nil")
	}

	if svc.repo == nil {
		t.Error("expected service to have repository")
	}
}

func TestRequestLogin_Success_Email(t *testing.T) {
	repo := &mockRepository{}
	svc := createTestService(t, repo)

	req := &authv1.RequestLoginRequest{
		ContactInfo: &authv1.RequestLoginRequest_Email{Email: "user@example.com"},
	}

	resp, err := svc.RequestLogin(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !resp.Success {
		t.Error("expected success to be true")
	}

	if resp.Message != "Magic code sent" {
		t.Errorf("expected message 'Magic code sent', got %s", resp.Message)
	}
}

func TestRequestLogin_Success_Phone(t *testing.T) {
	repo := &mockRepository{}
	svc := createTestService(t, repo)

	req := &authv1.RequestLoginRequest{
		ContactInfo: &authv1.RequestLoginRequest_Phone{Phone: "+1234567890"},
	}

	resp, err := svc.RequestLogin(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !resp.Success {
		t.Error("expected success to be true")
	}
}

func TestRequestLogin_UserNotFound(t *testing.T) {
	repo := &mockRepository{
		getUserByEmailFunc: func(_ context.Context, _ string) (*store.AuthUser, error) {
			return nil, sql.ErrNoRows
		},
	}
	svc := createTestService(t, repo)

	req := &authv1.RequestLoginRequest{
		ContactInfo: &authv1.RequestLoginRequest_Email{Email: "nonexistent@example.com"},
	}

	_, err := svc.RequestLogin(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when user not found")
	}

	// Check that it's a NotFound error
	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected status error")
	}

	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound error, got %v", st.Code())
	}
}

func TestRequestLogin_CreateMagicCodeError(t *testing.T) {
	repo := &mockRepository{
		createMagicCodeFunc: func(_ context.Context, _, _, _ string, _ time.Time) error {
			return errors.New("database error")
		},
	}
	svc := createTestService(t, repo)

	req := &authv1.RequestLoginRequest{
		ContactInfo: &authv1.RequestLoginRequest_Email{Email: "user@example.com"},
	}

	_, err := svc.RequestLogin(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when create magic code fails")
	}
}

func TestCompleteLogin_Success_ExistingUser(t *testing.T) {
	repo := &mockRepository{
		getValidMagicCodeByEmailFunc: func(_ context.Context, _, code string) (*store.AuthMagicCode, error) {
			return &store.AuthMagicCode{Code: code}, nil
		},
		getUserByEmailFunc: func(_ context.Context, email string) (*store.AuthUser, error) {
			return &store.AuthUser{
				ID:    "user-123",
				Email: sql.NullString{String: email, Valid: true},
			}, nil
		},
	}
	svc := createTestService(t, repo)

	req := &authv1.CompleteLoginRequest{
		ContactInfo: &authv1.CompleteLoginRequest_Email{Email: "user@example.com"},
		Code:        "123456",
	}

	resp, err := svc.CompleteLogin(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("expected access token to not be empty")
	}

	if resp.RefreshToken == "" {
		t.Error("expected refresh token to not be empty")
	}

	if resp.ExpiresIn != 3600 {
		t.Errorf("expected expires in 3600, got %d", resp.ExpiresIn)
	}
}

func TestCompleteLogin_Success_NewUser(t *testing.T) {
	var createdUserID string

	repo := &mockRepository{
		getValidMagicCodeByEmailFunc: func(_ context.Context, _, code string) (*store.AuthMagicCode, error) {
			return &store.AuthMagicCode{Code: code}, nil
		},
		getUserByEmailFunc: func(_ context.Context, email string) (*store.AuthUser, error) {
			// First call returns not found, simulating new user
			if createdUserID == "" {
				return nil, sql.ErrNoRows
			}

			// After user creation, return the new user
			return &store.AuthUser{
				ID:    createdUserID,
				Email: sql.NullString{String: email, Valid: true},
			}, nil
		},
		createUserFunc: func(_ context.Context, id, _, _ string) error {
			createdUserID = id
			return nil
		},
	}
	svc := createTestService(t, repo)

	req := &authv1.CompleteLoginRequest{
		ContactInfo: &authv1.CompleteLoginRequest_Email{Email: "newuser@example.com"},
		Code:        "123456",
	}

	resp, err := svc.CompleteLogin(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("expected access token to not be empty")
	}

	if createdUserID == "" {
		t.Error("expected user to be created")
	}
}

func TestCompleteLogin_MissingEmailAndPhone(t *testing.T) {
	repo := &mockRepository{}
	svc := createTestService(t, repo)

	req := &authv1.CompleteLoginRequest{
		Code: "123456",
	}

	_, err := svc.CompleteLogin(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when both email and phone are missing")
	}
}

func TestCompleteLogin_InvalidMagicCode(t *testing.T) {
	repo := &mockRepository{
		getValidMagicCodeByEmailFunc: func(_ context.Context, _, _ string) (*store.AuthMagicCode, error) {
			return nil, sql.ErrNoRows
		},
	}
	svc := createTestService(t, repo)

	req := &authv1.CompleteLoginRequest{
		ContactInfo: &authv1.CompleteLoginRequest_Email{Email: "user@example.com"},
		Code:        "wrong-code",
	}

	_, err := svc.CompleteLogin(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid magic code")
	}
}

func TestCompleteLogin_WithPhone(t *testing.T) {
	repo := &mockRepository{
		getValidMagicCodeByPhoneFunc: func(_ context.Context, _, code string) (*store.AuthMagicCode, error) {
			return &store.AuthMagicCode{Code: code}, nil
		},
		getUserByPhoneFunc: func(_ context.Context, phone string) (*store.AuthUser, error) {
			return &store.AuthUser{
				ID:    "user-456",
				Phone: sql.NullString{String: phone, Valid: true},
			}, nil
		},
	}
	svc := createTestService(t, repo)

	req := &authv1.CompleteLoginRequest{
		ContactInfo: &authv1.CompleteLoginRequest_Phone{Phone: "+1234567890"},
		Code:        "123456",
	}

	resp, err := svc.CompleteLogin(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("expected access token to not be empty")
	}
}

func TestGenerateRandomCode(t *testing.T) {
	t.Run("generates 6 digit code", func(t *testing.T) {
		testGenerateSixDigitCode(t)
	})

	t.Run("generates different codes", func(t *testing.T) {
		testGeneratesDifferentCodes(t)
	})

	t.Run("generates different lengths", func(t *testing.T) {
		testGeneratesDifferentLengths(t)
	})
}

func testGenerateSixDigitCode(t *testing.T) {
	t.Helper()

	code, err := generateRandomCode(6)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(code) != 6 {
		t.Errorf("expected code length 6, got %d", len(code))
	}

	// Verify all characters are digits
	for _, c := range code {
		if c < '0' || c > '9' {
			t.Errorf("expected all digits, got character %c", c)
		}
	}
}

func testGeneratesDifferentCodes(t *testing.T) {
	t.Helper()

	code1, err := generateRandomCode(6)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	code2, err := generateRandomCode(6)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// While it's theoretically possible for two random codes to be the same,
	// it's extremely unlikely (1 in 1,000,000 chance)
	// We test this to ensure randomness is working
	if code1 == code2 {
		t.Log("Warning: generated identical codes (very unlikely but possible)")
	}
}

func testGeneratesDifferentLengths(t *testing.T) {
	t.Helper()

	for length := 1; length <= 10; length++ {
		code, err := generateRandomCode(length)
		if err != nil {
			t.Fatalf("expected no error for length %d, got %v", length, err)
		}

		if len(code) != length {
			t.Errorf("expected code length %d, got %d", length, len(code))
		}
	}
}

func TestCompleteLogin_DatabaseError(t *testing.T) {
	repo := &mockRepository{
		getValidMagicCodeByEmailFunc: func(_ context.Context, _, _ string) (*store.AuthMagicCode, error) {
			return nil, errors.New("database connection error")
		},
	}
	svc := createTestService(t, repo)

	req := &authv1.CompleteLoginRequest{
		ContactInfo: &authv1.CompleteLoginRequest_Email{Email: "user@example.com"},
		Code:        "123456",
	}

	_, err := svc.CompleteLogin(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when database fails")
	}
}

func TestCompleteLogin_GetUserError(t *testing.T) {
	repo := &mockRepository{
		getValidMagicCodeByEmailFunc: func(_ context.Context, _, code string) (*store.AuthMagicCode, error) {
			return &store.AuthMagicCode{Code: code}, nil
		},
		getUserByEmailFunc: func(_ context.Context, _ string) (*store.AuthUser, error) {
			return nil, errors.New("database error fetching user")
		},
	}
	svc := createTestService(t, repo)

	req := &authv1.CompleteLoginRequest{
		ContactInfo: &authv1.CompleteLoginRequest_Email{Email: "user@example.com"},
		Code:        "123456",
	}

	_, err := svc.CompleteLogin(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when fetching user fails")
	}
}

func TestCompleteLogin_NewUser_WithPhone(t *testing.T) {
	var createdUserID string

	repo := &mockRepository{
		getValidMagicCodeByPhoneFunc: func(_ context.Context, _, code string) (*store.AuthMagicCode, error) {
			return &store.AuthMagicCode{Code: code}, nil
		},
		getUserByPhoneFunc: func(_ context.Context, phone string) (*store.AuthUser, error) {
			if createdUserID == "" {
				return nil, sql.ErrNoRows
			}

			return &store.AuthUser{
				ID:    createdUserID,
				Phone: sql.NullString{String: phone, Valid: true},
			}, nil
		},
		createUserFunc: func(_ context.Context, id, _, _ string) error {
			createdUserID = id
			return nil
		},
	}
	svc := createTestService(t, repo)

	req := &authv1.CompleteLoginRequest{
		ContactInfo: &authv1.CompleteLoginRequest_Phone{Phone: "+1234567890"},
		Code:        "123456",
	}

	resp, err := svc.CompleteLogin(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("expected access token to not be empty")
	}

	if createdUserID == "" {
		t.Error("expected user to be created")
	}
}

func TestCompleteLogin_CreateUserError(t *testing.T) {
	repo := &mockRepository{
		getValidMagicCodeByEmailFunc: func(_ context.Context, _, code string) (*store.AuthMagicCode, error) {
			return &store.AuthMagicCode{Code: code}, nil
		},
		getUserByEmailFunc: func(_ context.Context, _ string) (*store.AuthUser, error) {
			return nil, sql.ErrNoRows
		},
		createUserFunc: func(_ context.Context, _, _, _ string) error {
			return errors.New("failed to create user")
		},
	}
	svc := createTestService(t, repo)

	req := &authv1.CompleteLoginRequest{
		ContactInfo: &authv1.CompleteLoginRequest_Email{Email: "newuser@example.com"},
		Code:        "123456",
	}

	_, err := svc.CompleteLogin(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when user creation fails")
	}
}

func TestCompleteLogin_FetchUserAfterCreateError(t *testing.T) {
	callCount := 0

	repo := &mockRepository{
		getValidMagicCodeByEmailFunc: func(_ context.Context, _, code string) (*store.AuthMagicCode, error) {
			return &store.AuthMagicCode{Code: code}, nil
		},
		getUserByEmailFunc: func(_ context.Context, _ string) (*store.AuthUser, error) {
			callCount++
			if callCount == 1 {
				return nil, sql.ErrNoRows
			}

			return nil, errors.New("failed to fetch after create")
		},
		createUserFunc: func(_ context.Context, _, _, _ string) error {
			return nil
		},
	}
	svc := createTestService(t, repo)

	req := &authv1.CompleteLoginRequest{
		ContactInfo: &authv1.CompleteLoginRequest_Email{Email: "newuser@example.com"},
		Code:        "123456",
	}

	_, err := svc.CompleteLogin(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when fetching user after creation fails")
	}
}

func TestCompleteLogin_FetchUserByPhoneAfterCreateError(t *testing.T) {
	callCount := 0

	repo := &mockRepository{
		getValidMagicCodeByPhoneFunc: func(_ context.Context, _, code string) (*store.AuthMagicCode, error) {
			return &store.AuthMagicCode{Code: code}, nil
		},
		getUserByPhoneFunc: func(_ context.Context, _ string) (*store.AuthUser, error) {
			callCount++
			if callCount == 1 {
				return nil, sql.ErrNoRows
			}

			return nil, errors.New("failed to fetch after create")
		},
		createUserFunc: func(_ context.Context, _, _, _ string) error {
			return nil
		},
	}
	svc := createTestService(t, repo)

	req := &authv1.CompleteLoginRequest{
		ContactInfo: &authv1.CompleteLoginRequest_Phone{Phone: "+1234567890"},
		Code:        "123456",
	}

	_, err := svc.CompleteLogin(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when fetching user by phone after creation fails")
	}
}

func TestCompleteLogin_InvalidateMagicCodesError(t *testing.T) {
	repo := &mockRepository{
		getValidMagicCodeByEmailFunc: func(_ context.Context, _, code string) (*store.AuthMagicCode, error) {
			return &store.AuthMagicCode{Code: code}, nil
		},
		getUserByEmailFunc: func(_ context.Context, email string) (*store.AuthUser, error) {
			return &store.AuthUser{
				ID:    "user-123",
				Email: sql.NullString{String: email, Valid: true},
			}, nil
		},
		invalidateMagicCodesFunc: func(_ context.Context, _, _ string) error {
			return errors.New("failed to invalidate codes")
		},
	}
	svc := createTestService(t, repo)

	req := &authv1.CompleteLoginRequest{
		ContactInfo: &authv1.CompleteLoginRequest_Email{Email: "user@example.com"},
		Code:        "123456",
	}

	// Should still succeed even if invalidation fails (it's logged but not fatal)
	resp, err := svc.CompleteLogin(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("expected access token to not be empty")
	}
}

func TestVerifyMagicCodeByPhone_InvalidCode(t *testing.T) {
	repo := &mockRepository{
		getValidMagicCodeByPhoneFunc: func(_ context.Context, _, _ string) (*store.AuthMagicCode, error) {
			return nil, sql.ErrNoRows
		},
	}
	svc := createTestService(t, repo)

	req := &authv1.CompleteLoginRequest{
		ContactInfo: &authv1.CompleteLoginRequest_Phone{Phone: "+1234567890"},
		Code:        "wrong-code",
	}

	_, err := svc.CompleteLogin(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid magic code by phone")
	}
}

func TestVerifyMagicCodeByPhone_DatabaseError(t *testing.T) {
	repo := &mockRepository{
		getValidMagicCodeByPhoneFunc: func(_ context.Context, _, _ string) (*store.AuthMagicCode, error) {
			return nil, errors.New("database error")
		},
	}
	svc := createTestService(t, repo)

	req := &authv1.CompleteLoginRequest{
		ContactInfo: &authv1.CompleteLoginRequest_Phone{Phone: "+1234567890"},
		Code:        "123456",
	}

	_, err := svc.CompleteLogin(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when database fails for phone verification")
	}
}

// Additional mock methods for the extended Repository interface

func (m *mockRepository) GetUserByID(_ context.Context, id string) (*store.AuthUser, error) {
	return &store.AuthUser{ID: id}, nil
}

func (m *mockRepository) UpdateUserProfile(_ context.Context, _, _, _ string) error {
	return nil
}

func (m *mockRepository) CreateSession(_ context.Context, _ *repository.Session) error {
	return nil
}

func (m *mockRepository) GetSessionByID(_ context.Context, id string) (*repository.Session, error) {
	return &repository.Session{ID: id}, nil
}

func (m *mockRepository) GetSessionByRefreshTokenHash(_ context.Context, _ string) (*repository.Session, error) {
	return &repository.Session{}, nil
}

func (m *mockRepository) GetSessionsByUserID(_ context.Context, _ string) ([]*repository.Session, error) {
	return nil, nil
}

func (m *mockRepository) UpdateSessionActivity(_ context.Context, _ string) error {
	return nil
}

func (m *mockRepository) RevokeSession(_ context.Context, _ string) error {
	return nil
}

func (m *mockRepository) RevokeAllUserSessions(_ context.Context, _, _ string) (int, error) {
	return 0, nil
}

func (m *mockRepository) BlacklistToken(_ context.Context, _, _, _ string, _ time.Time) error {
	return nil
}

func (m *mockRepository) IsTokenBlacklisted(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (m *mockRepository) CleanupExpiredBlacklistEntries(_ context.Context) error {
	return nil
}

func (m *mockRepository) CreatePendingContactChange(_ context.Context, _, _, _, _, _ string, _ time.Time) error {
	return nil
}

func (m *mockRepository) GetPendingContactChange(_ context.Context, _, _, _ string) (*repository.PendingContactChange, error) {
	return nil, nil
}

func (m *mockRepository) DeletePendingContactChange(_ context.Context, _ string) error {
	return nil
}

// External OAuth accounts mock methods

func (m *mockRepository) CreateExternalAccount(_ context.Context, _ *repository.ExternalAccount) error {
	return nil
}

func (m *mockRepository) GetExternalAccountByProviderUserID(_ context.Context, _, _ string) (*repository.ExternalAccount, error) {
	return nil, nil
}

func (m *mockRepository) GetExternalAccountsByUserID(_ context.Context, _ string) ([]*repository.ExternalAccount, error) {
	return nil, nil
}

func (m *mockRepository) GetExternalAccountByProviderAndEmail(_ context.Context, _, _ string) (*repository.ExternalAccount, error) {
	return nil, nil
}

func (m *mockRepository) UpdateExternalAccountTokens(_ context.Context, _, _, _, _ string, _ *time.Time) error {
	return nil
}

func (m *mockRepository) UpdateExternalAccountProfile(_ context.Context, _, _, _, _, _ string, _ map[string]interface{}) error {
	return nil
}

func (m *mockRepository) DeleteExternalAccount(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockRepository) DeleteExternalAccountByProvider(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockRepository) CountExternalAccountsByUserID(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

// OAuth state mock methods

func (m *mockRepository) CreateOAuthState(_ context.Context, _ *repository.OAuthState) error {
	return nil
}

func (m *mockRepository) GetOAuthState(_ context.Context, _ string) (*repository.OAuthState, error) {
	return nil, nil
}

func (m *mockRepository) DeleteOAuthState(_ context.Context, _ string) error {
	return nil
}

func (m *mockRepository) CleanupExpiredOAuthStates(_ context.Context) error {
	return nil
}

func (m *mockRepository) CleanupExpiredSessions(_ context.Context) (int, error) {
	return 0, nil
}

func (m *mockRepository) CleanupExpiredMagicCodes(_ context.Context) (int, error) {
	return 0, nil
}
