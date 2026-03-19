package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	authv1 "github.com/cmelgarejo/go-modulith-template/gen/go/proto/auth/v1"
	"github.com/cmelgarejo/go-modulith-template/internal/audit"
	"github.com/cmelgarejo/go-modulith-template/internal/authtoken"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/feature"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/db/store"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/repository"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/repository/mocks"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/service"
	"go.uber.org/mock/gomock"
)

// Real-looking dummy RSA private key for tests
const testRSAPrivateKey = `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQCgKaue+zKl57/5
QuzzKZIm0nQe5Jopmd10ie/fB8k3nAReUwQ0aiaVws9FmeT1fylKzuLrEN4Xh0wy
ZYrEwV0xTaxBOu708yZikVCMz1bF16mhoODBrm2+cNE0bfpxzwoFt/zyP6AigxWJ
5XHJzJHFoaDw3334oLvaG1lkcDjfFUEKMbIk+CN2hXCbI6BSJCo989y4RPoFkZBH
eNgKiRiHZm5ypsNEdvjItlRGM7hwtAH81v+OtdlTeWp+mlz3SCUyCagEP1Gs3L0Y
aeoYOEA1ylpmapaDhKnobk4oFb9ujF60CGkLt/eOjt63AvQQtmAKJLK4Y0EgS7hi
3kh9ZxDJAgMBAAECggEACx+px3jR2Ggp0wspzGxynD1zCpXWlGIXDLOFB4JebTBp
6A7JYXlbGBaq8T4ST7yjs3B+arrfefBDKgYXmh+5GoxqLmOd86d5kh0rZFEE/IIR
SkqTbmnWpGq1SCtrpzQTRNKcMxgyxbYN+Zq7PIh3oJH5TN49o2/ibCNvId5epmh5
Qyvy2FhYZGhtxg3K+WApQxfeTOq/o+BbNdSUrcQaLDeKe3PS3KFykCj+dno3EiFn
dyEwLQcP63dSoUqW6ObR634DSIRR0CNWqRyeWD0SxRbjNV9bIk/bOjJ4FrPjEuRB
gT/LhMsD1fthTMyAyNpryxDknc2mYCrHd/ix5nEakwKBgQDMdfe0CpSbfNqix9T6
TAasGZaXVSBJ3n4GCwFOn5KaJfPhAB64n9x82YvliWOyl5u16SgxnBj3vKGLskCP
DXSLvQWBheZBFoPxEKsGXp2ddEFXf7zVcjG4nYz8Z0Kn5JGImhjwajcQKIVJBCvR
vTwCWl3/9spKARs0Zue6hBd0owKBgQDIiRzDJlonRL6TCS8bJT/LBdWGIn1Syz/A
zbssfD9Qh89TL5i7zfPcGm4Yzk+Z4zbh1/67D33GvMPr1aKnzcbR4+4+xZiVaZjl
m0tDONGFxrZAyvbdHLJiXZBujoRO96bGsjZtyEZ+hG+MV0s+FCX7fkFWJa1+vpyv
aAkZcrjPowKBgQCK76bRC1eMiT0w3EYXh84I6KJyV4BHcg+FH7lVqg2+/gdJYAGA
R/FWTaZI5iF/XJKM/NE5VO+KeP31pb1E+Em4I0w4hbq/hANIrqDpBSZptnQodz7k
dGLhJv6FDc43tJRIlR5ZUHP2YPKheVolfkfm+W1i4Fr6CuJnq33QOq6NrQKBgFml
Oa9fiLO/PnZah61Z5H+stvxElMObSn+1OHQ1gtRMMflc8Kkb82S0h/0c1WbUtOcW
+K/EyBQ8tFTL5u+exL91Zj63dHNuhkQ2PNnrH3bvEvA6C0tjFbd1XiieGzV17h8q
8bv36NOL/pW9PEyfEy+vDCQnqbxcF40uM8slhsqDAoGBAMGCthWkf2eG0Y4Scksf
r/gNlU+15OnndSq0UQt2xjiy+0XQ5CVHaIyyaLiFiYjsLYdaxfOckMMrvP3RqObE
8b9897yqs3ENFV+lJA7z/gZntQFLmlfzQadbGRuVeZfh+u7NqM4j73SRNMubEBEd
7mlsQJQ+USaHSReSju9xmzH8
-----END PRIVATE KEY-----`

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

			// Helper to handle WithTx
			mockRepo.EXPECT().WithTx(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, f func(repository.Repository) error) error {
				return f(mockRepo)
			}).AnyTimes()

			// Create service with mock repository
			tokenSvc, _ := authtoken.NewService(testRSAPrivateKey)
			bus := events.NewBus()
			auditLog := &audit.NoopLogger{}
			featureMgr := feature.NewInMemoryManager()
			authSvc := service.NewAuthService(mockRepo, tokenSvc, bus, auditLog, featureMgr, "dev")

			// Execute
			ctx := context.Background()
			resp, err := authSvc.RequestLogin(ctx, tt.req)

			// Assert
			assertRequestLoginResult(t, resp, err, tt.wantErr, tt.errContains)
		})
	}
}

// TestCompleteLogin_WithMock demonstrates testing CompleteLogin with mocks
//nolint:funlen // Complex login flow requires comprehensive test cases
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
		Return(&store.AuthMagicCode{
			Code:      code,
			UserEmail: pgtype.Text{String: email, Valid: true},
			ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(5 * time.Minute), Valid: true},
		}, nil).
		Times(1)

	mockRepo.EXPECT().
		GetUserByEmail(gomock.Any(), email).
		Return(&store.AuthUser{
			ID:    userID,
			Email: pgtype.Text{String: email, Valid: true},
		}, nil).
		Times(1)

	// Handle WithTx for CompleteLogin
	mockRepo.EXPECT().WithTx(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, f func(repository.Repository) error) error {
		return f(mockRepo)
	}).Times(1)

	mockRepo.EXPECT().
		InvalidateMagicCodes(gomock.Any(), email, "").
		Return(nil).
		Times(1)

	mockRepo.EXPECT().
		StoreOutbox(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	mockRepo.EXPECT().
		GetUserRole(gomock.Any(), userID).
		Return("user", nil).
		Times(1)

	mockRepo.EXPECT().
		CreateSession(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	// Create service
	tokenSvc, _ := authtoken.NewService(testRSAPrivateKey)
	bus := events.NewBus()
	auditLog := &audit.NoopLogger{}
	featureMgr := feature.NewInMemoryManager()
	authSvc := service.NewAuthService(mockRepo, tokenSvc, bus, auditLog, featureMgr, "dev")

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
				m.EXPECT().GetUserByEmail(gomock.Any(), "test@example.com").Return(&store.AuthUser{ID: "user-123"}, nil).Times(1)
				m.EXPECT().CreateMagicCode(gomock.Any(), gomock.Any(), "test@example.com", "", gomock.Any()).Return(nil).AnyTimes()
				m.EXPECT().StoreOutbox(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "user not found (silent success)",
			req:  &authv1.RequestLoginRequest{ContactInfo: &authv1.RequestLoginRequest_Phone{Phone: "+1234567890"}},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().GetUserByPhone(gomock.Any(), "+1234567890").Return(nil, sql.ErrNoRows).Times(1)
			},
			wantErr: false,
		},
		{
			name: "repository error on user lookup (silent success)",
			req:  &authv1.RequestLoginRequest{ContactInfo: &authv1.RequestLoginRequest_Email{Email: "test@example.com"}},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().GetUserByEmail(gomock.Any(), "test@example.com").Return(nil, errors.New("database error")).Times(1)
			},
			wantErr: false,
		},
		{
			name: "repository error on create magic code",
			req:  &authv1.RequestLoginRequest{ContactInfo: &authv1.RequestLoginRequest_Email{Email: "test@example.com"}},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().GetUserByEmail(gomock.Any(), "test@example.com").Return(&store.AuthUser{ID: "user-123"}, nil).Times(1)
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
