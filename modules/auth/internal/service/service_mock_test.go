package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	authv1 "github.com/cmelgarejo/go-modulith-template/gen/go/proto/auth/v1"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/db/store"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/repository/mocks"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/service"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/token"
	"go.uber.org/mock/gomock"
)

// TestRequestLogin_WithMock demonstrates using gomock to test the RequestLogin method
func TestRequestLogin_WithMock(t *testing.T) {
	t.Parallel()

	tests := getRequestLoginTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockRepository(ctrl)
			tt.setupMock(mockRepo)

			// Create service with mock repository
			tokenSvc, _ := token.NewService("test-secret-key-with-at-least-32-bytes!")
			bus := events.NewBus()
			authSvc := service.NewAuthService(mockRepo, tokenSvc, bus)

			// Execute
			ctx := context.Background()
			resp, err := authSvc.RequestLogin(ctx, tt.req)

			// Assert
			assertRequestLoginResult(t, resp, err, tt.wantErr, tt.errContains)
		})
	}
}

// TestCompleteLogin_WithMock demonstrates testing CompleteLogin with mocks
func TestCompleteLogin_WithMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)

	// Setup expectations
	email := "test@example.com"
	code := "123456"
	userID := "user_abc123"

	mockRepo.EXPECT().
		GetValidMagicCodeByEmail(gomock.Any(), email, code).
		Return(&store.MagicCode{
			Code:      code,
			UserEmail: sql.NullString{String: email, Valid: true},
			ExpiresAt: time.Now().Add(5 * time.Minute),
		}, nil).
		Times(1)

	mockRepo.EXPECT().
		GetUserByEmail(gomock.Any(), email).
		Return(&store.User{
			ID:    userID,
			Email: sql.NullString{String: email, Valid: true},
		}, nil).
		Times(1)

	mockRepo.EXPECT().
		InvalidateMagicCodes(gomock.Any(), email, "").
		Return(nil).
		Times(1)

	// Create service
	tokenSvc, _ := token.NewService("test-secret-key-with-at-least-32-bytes!")
	bus := events.NewBus()
	authSvc := service.NewAuthService(mockRepo, tokenSvc, bus)

	// Execute
	ctx := context.Background()
	req := &authv1.CompleteLoginRequest{
		ContactInfo: &authv1.CompleteLoginRequest_Email{Email: email},
		Code:        code,
	}

	resp, err := authSvc.CompleteLogin(ctx, req)
	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response but got nil")
	}

	if resp.AccessToken == "" {
		t.Error("expected access_token to be set")
	}

	if resp.ExpiresIn == 0 {
		t.Error("expected expires_in to be set")
	}
}

// getRequestLoginTestCases returns test cases for RequestLogin
func getRequestLoginTestCases() []struct {
	name        string
	req         *authv1.RequestLoginRequest
	setupMock   func(*mocks.MockRepository)
	wantErr     bool
	errContains string
} {
	return []struct {
		name        string
		req         *authv1.RequestLoginRequest
		setupMock   func(*mocks.MockRepository)
		wantErr     bool
		errContains string
	}{
		{
			name: "successful login request",
			req:  &authv1.RequestLoginRequest{ContactInfo: &authv1.RequestLoginRequest_Email{Email: "test@example.com"}},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().GetUserByEmail(gomock.Any(), "test@example.com").Return(&store.User{ID: "user-123"}, nil).Times(1)
				m.EXPECT().CreateMagicCode(gomock.Any(), gomock.Any(), "test@example.com", "", gomock.Any()).Return(nil).Times(1)
			},
			wantErr: false,
		},
		{
			name:        "user not found",
			req:         &authv1.RequestLoginRequest{ContactInfo: &authv1.RequestLoginRequest_Phone{Phone: "+1234567890"}},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().GetUserByPhone(gomock.Any(), "+1234567890").Return(nil, sql.ErrNoRows).Times(1)
			},
			wantErr:     true,
			errContains: "user not found",
		},
		{
			name: "repository error on user lookup",
			req:  &authv1.RequestLoginRequest{ContactInfo: &authv1.RequestLoginRequest_Email{Email: "test@example.com"}},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().GetUserByEmail(gomock.Any(), "test@example.com").Return(nil, errors.New("database error")).Times(1)
			},
			wantErr:     true,
			errContains: "internal server error",
		},
		{
			name: "repository error on create magic code",
			req:  &authv1.RequestLoginRequest{ContactInfo: &authv1.RequestLoginRequest_Email{Email: "test@example.com"}},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().GetUserByEmail(gomock.Any(), "test@example.com").Return(&store.User{ID: "user-123"}, nil).Times(1)
				m.EXPECT().CreateMagicCode(gomock.Any(), gomock.Any(), "test@example.com", "", gomock.Any()).
					Return(errors.New("database error")).Times(1)
			},
			wantErr:     true,
			errContains: "internal server error",
		},
	}
}

// assertRequestLoginResult validates the result of a RequestLogin call
func assertRequestLoginResult(t *testing.T, resp *authv1.RequestLoginResponse, err error, wantErr bool, errContains string) {
	t.Helper()

	if wantErr {
		assertError(t, err, errContains)
		return
	}

	assertSuccess(t, resp, err)
}

// assertError checks that an error occurred and optionally contains a specific message
func assertError(t *testing.T, err error, errContains string) {
	t.Helper()

	if err == nil {
		t.Errorf("expected error but got none")
		return
	}

	if errContains != "" && !contains(err.Error(), errContains) {
		t.Errorf("expected error to contain %q, got %q", errContains, err.Error())
	}
}

// assertSuccess checks that the response is successful
func assertSuccess(t *testing.T, resp *authv1.RequestLoginResponse, err error) {
	t.Helper()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Errorf("expected response but got nil")
		return
	}

	if !resp.Success {
		t.Errorf("expected success=true, got success=false")
	}
}

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
