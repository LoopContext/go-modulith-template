package service

//nolint:goconst
import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	authv1 "github.com/cmelgarejo/go-modulith-template/gen/go/proto/auth/v1"
	"github.com/cmelgarejo/go-modulith-template/internal/audit"
	"github.com/cmelgarejo/go-modulith-template/internal/authn"
	"github.com/cmelgarejo/go-modulith-template/internal/authtoken"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/feature"
	"github.com/cmelgarejo/go-modulith-template/internal/testutil"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/db/store"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/repository/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/metadata"

	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/repository"
)

const testUserID = "user_1"

//nolint:funlen
func TestAuthService_RequestLogin(t *testing.T) {
	tests := []struct {
		name          string
		req           *authv1.RequestLoginRequest
		setup         func(*mocks.MockRepository)
		expectedError string
		checkResponse func(*testing.T, *authv1.RequestLoginResponse)
	}{
		{
			name: "Success Email",
			req: &authv1.RequestLoginRequest{
				ContactInfo: &authv1.RequestLoginRequest_Email{Email: "user@example.com"},
			},
			setup: func(mRepo *mocks.MockRepository) {
				mRepo.EXPECT().GetUserByEmail(gomock.Any(), "user@example.com").Return(&store.AuthUser{ID: testUserID}, nil)
				mRepo.EXPECT().CreateMagicCode(gomock.Any(), gomock.Any(), "user@example.com", "", gomock.Any()).Return(nil)
			},
			checkResponse: func(t *testing.T, resp *authv1.RequestLoginResponse) {
				assert.True(t, resp.Success)
				assert.Equal(t, "If an account exists with this email, you will receive a verification code", resp.Message)
			},
		},
		{
			name: "Success Phone",
			req: &authv1.RequestLoginRequest{
				ContactInfo: &authv1.RequestLoginRequest_Phone{Phone: "+1234567890"},
			},
			setup: func(mRepo *mocks.MockRepository) {
				mRepo.EXPECT().GetUserByPhone(gomock.Any(), "+1234567890").Return(&store.AuthUser{ID: testUserID}, nil)
				mRepo.EXPECT().CreateMagicCode(gomock.Any(), gomock.Any(), "", "+1234567890", gomock.Any()).Return(nil)
			},
			checkResponse: func(t *testing.T, resp *authv1.RequestLoginResponse) {
				assert.True(t, resp.Success)
			},
		},
		{
			name: "User Not Found (Silent Success)",
			req: &authv1.RequestLoginRequest{
				ContactInfo: &authv1.RequestLoginRequest_Email{Email: "missing@example.com"},
			},
			setup: func(mRepo *mocks.MockRepository) {
				mRepo.EXPECT().GetUserByEmail(gomock.Any(), "missing@example.com").Return(nil, pgx.ErrNoRows)
				// CreateMagicCode should NOT be called
			},
			checkResponse: func(t *testing.T, resp *authv1.RequestLoginResponse) {
				assert.True(t, resp.Success)
			},
		},
		{
			name: "DB Error on Create Magic Code",
			req: &authv1.RequestLoginRequest{
				ContactInfo: &authv1.RequestLoginRequest_Email{Email: "user@example.com"},
			},
			setup: func(mRepo *mocks.MockRepository) {
				mRepo.EXPECT().GetUserByEmail(gomock.Any(), "user@example.com").Return(&store.AuthUser{ID: testUserID}, nil)
				mRepo.EXPECT().CreateMagicCode(gomock.Any(), gomock.Any(), "user@example.com", "", gomock.Any()).Return(errors.New("db error"))
			},
			expectedError: "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mRepo := mocks.NewMockRepository(ctrl)
			mRepo.EXPECT().WithTx(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, fn func(repository.Repository) error) error {
				return fn(mRepo)
			}).AnyTimes()
			mRepo.EXPECT().StoreOutbox(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			tokenSvc, _ := authtoken.NewService(testutil.TestJWTPrivateKeyPEM)
			bus := events.NewBus()
			auditLog := &audit.NoopLogger{}
			flags := feature.NewInMemoryManager()

			if tt.setup != nil {
				tt.setup(mRepo)
			}

			svc := NewAuthService(mRepo, tokenSvc, bus, auditLog, flags, "dev")

			resp, err := svc.RequestLogin(context.Background(), tt.req)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)

				if tt.checkResponse != nil {
					tt.checkResponse(t, resp)
				}
			}
		})
	}
}

//nolint:funlen
func TestAuthService_CompleteLogin(t *testing.T) {
	tests := []struct {
		name          string
		req           *authv1.CompleteLoginRequest
		setup         func(*mocks.MockRepository)
		expectedError string
	}{
		{
			name: "Success Existing User",
			req: &authv1.CompleteLoginRequest{
				ContactInfo: &authv1.CompleteLoginRequest_Email{Email: "user@example.com"},
				Code:        "123456",
			},
			setup: func(mRepo *mocks.MockRepository) {
				mRepo.EXPECT().GetValidMagicCodeByEmail(gomock.Any(), "user@example.com", "123456").
					Return(&store.AuthMagicCode{Code: "123456"}, nil)

				mRepo.EXPECT().GetUserByEmail(gomock.Any(), "user@example.com").
					Return(&store.AuthUser{
						ID:    testUserID,
						Email: pgtype.Text{String: "user@example.com", Valid: true},
					}, nil)

				mRepo.EXPECT().GetUserRole(gomock.Any(), testUserID).Return("user", nil)

				mRepo.EXPECT().InvalidateMagicCodes(gomock.Any(), "user@example.com", "").Return(nil)

				mRepo.EXPECT().CreateSession(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name: "New User (Blocked)",
			req: &authv1.CompleteLoginRequest{
				ContactInfo: &authv1.CompleteLoginRequest_Email{Email: "new@example.com"},
				Code:        "123456",
			},
			setup: func(mRepo *mocks.MockRepository) {
				mRepo.EXPECT().GetValidMagicCodeByEmail(gomock.Any(), "new@example.com", "123456").
					Return(&store.AuthMagicCode{Code: "123456"}, nil)

				// Check returns not found
				mRepo.EXPECT().GetUserByEmail(gomock.Any(), "new@example.com").
					Return(nil, pgx.ErrNoRows)
			},
			expectedError: "user not found",
		},
		{
			name: "Invalid Code",
			req: &authv1.CompleteLoginRequest{
				ContactInfo: &authv1.CompleteLoginRequest_Email{Email: "user@example.com"},
				Code:        "wrong",
			},
			setup: func(mRepo *mocks.MockRepository) {
				mRepo.EXPECT().GetValidMagicCodeByEmail(gomock.Any(), "user@example.com", "wrong").
					Return(nil, pgx.ErrNoRows)
			},
			expectedError: "invalid or expired code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mRepo := mocks.NewMockRepository(ctrl)
			mRepo.EXPECT().WithTx(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, fn func(repository.Repository) error) error {
				return fn(mRepo)
			}).AnyTimes()
			mRepo.EXPECT().StoreOutbox(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			tokenSvc, _ := authtoken.NewService(testutil.TestJWTPrivateKeyPEM)
			bus := events.NewBus()
			auditLog := &audit.NoopLogger{}
			flags := feature.NewInMemoryManager()

			if tt.setup != nil {
				tt.setup(mRepo)
			}

			svc := NewAuthService(mRepo, tokenSvc, bus, auditLog, flags, "dev")

			resp, err := svc.CompleteLogin(context.Background(), tt.req)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, resp.AccessToken)
			}
		})
	}
}

func TestAuthService_Logout(t *testing.T) {
	tests := []struct {
		name          string
		token         string
		setup         func(*mocks.MockRepository)
		expectedError string
	}{
		{
			name:  "Success",
			token: "valid-token",
			setup: func(mRepo *mocks.MockRepository) {
				mRepo.EXPECT().BlacklistToken(gomock.Any(), gomock.Any(), gomock.Any(), "logout", gomock.Any()).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mRepo := mocks.NewMockRepository(ctrl)
			mRepo.EXPECT().WithTx(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, fn func(repository.Repository) error) error {
				return fn(mRepo)
			}).AnyTimes()
			mRepo.EXPECT().StoreOutbox(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			tokenSvc, _ := authtoken.NewService(testutil.TestJWTPrivateKeyPEM)
			bus := events.NewBus()
			auditLog := &audit.NoopLogger{}
			flags := feature.NewInMemoryManager()

			if tt.setup != nil {
				tt.setup(mRepo)
			}

			svc := NewAuthService(mRepo, tokenSvc, bus, auditLog, flags, "dev")

			// Mock context with token
			// We inject user claims via middleware usually, but Logout parses the token.
			// Since we use a real token service, we can't easily generate a valid token that passes verification
			// unless we generate one first.
			// However, Logout logic parses token TO blacklist it.
			// If verification fails, it proceeds to blacklist anyway (best effort) in some implementations,
			// or fails.
			// Looking at implementation:
			// claims, err := s.tokenService.VerifyAccessToken(token)
			// if err != nil { return ... }

			// So we need a valid token.
			validToken, _, err := tokenSvc.CreateToken(testUserID, "user", 1*time.Hour)
			if err != nil {
				t.Fatalf("failed to create token: %v", err)
			}

			// Add metadata (token)
			mdCtx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"authorization": "Bearer " + validToken}))
			// Add claims (user_id)
			ctx := authn.ContextWithClaims(mdCtx, authn.Claims{UserID: testUserID, Role: "user"})

			// Ensure token is valid (nbf)
			time.Sleep(1 * time.Second)

			_, err = svc.Logout(ctx, &authv1.LogoutRequest{})

			if tt.expectedError != "" {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthService_GetProfile(t *testing.T) {
	userID := testUserID //nolint:goconst
	tests := []struct {
		name          string
		setup         func(*mocks.MockRepository)
		ctx           context.Context
		expectedError string
	}{
		{
			name: "Success",
			ctx:  authn.ContextWithClaims(context.Background(), authn.Claims{UserID: userID}),
			setup: func(mRepo *mocks.MockRepository) {
				mRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(&store.AuthUser{
					ID:    userID,
					Email: pgtype.Text{String: "test@example.com", Valid: true},
				}, nil)
			},
		},
		{
			name: "User Not Found",
			ctx:  authn.ContextWithClaims(context.Background(), authn.Claims{UserID: userID}),
			setup: func(mRepo *mocks.MockRepository) {
				mRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(nil, pgx.ErrNoRows)
			},
			expectedError: "user not found", // or whatever implicit error map returns, checking status usually
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mRepo := mocks.NewMockRepository(ctrl)
			mRepo.EXPECT().WithTx(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, fn func(repository.Repository) error) error {
				return fn(mRepo)
			}).AnyTimes()
			mRepo.EXPECT().StoreOutbox(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			tokenSvc, _ := authtoken.NewService(testutil.TestJWTPrivateKeyPEM)
			svc := NewAuthService(mRepo, tokenSvc, events.NewBus(), &audit.NoopLogger{}, feature.NewInMemoryManager(), "dev")

			if tt.setup != nil {
				tt.setup(mRepo)
			}

			resp, err := svc.GetProfile(tt.ctx, &authv1.GetProfileRequest{})

			if tt.expectedError != "" {
				assert.Error(t, err)
				// Basic check, might be status error
			} else {
				assert.NoError(t, err)
				assert.Equal(t, userID, resp.User.Id)
			}
		})
	}
}

//nolint:funlen
func TestAuthService_UpdateProfile(t *testing.T) {
	userID := testUserID
	tests := []struct {
		name          string
		req           *authv1.UpdateProfileRequest
		setup         func(*mocks.MockRepository)
		ctx           context.Context
		expectedError string
	}{
		{
			name: "Success",
			req: &authv1.UpdateProfileRequest{
				DisplayName: "New Name",
				AvatarUrl:   "http://avatar.com",
			},
			ctx: authn.ContextWithClaims(context.Background(), authn.Claims{UserID: userID}),
			setup: func(mRepo *mocks.MockRepository) {
				mRepo.EXPECT().UpdateUserProfile(gomock.Any(), userID, "New Name", "http://avatar.com", "").Return(nil)
				mRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(&store.AuthUser{
					ID:          userID,
					DisplayName: pgtype.Text{String: "New Name", Valid: true},
				}, nil)
			},
		},
		{
			name: "WithTimezone",
			req: &authv1.UpdateProfileRequest{
				DisplayName: "New Name",
				Timezone:    "America/New_York",
			},
			ctx: authn.ContextWithClaims(context.Background(), authn.Claims{UserID: userID}),
			setup: func(mRepo *mocks.MockRepository) {
				mRepo.EXPECT().UpdateUserProfile(gomock.Any(), userID, "New Name", "", "America/New_York").Return(nil)
				mRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(&store.AuthUser{
					ID:          userID,
					DisplayName: pgtype.Text{String: "New Name", Valid: true},
					Timezone:    pgtype.Text{String: "America/New_York", Valid: true},
				}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mRepo := mocks.NewMockRepository(ctrl)
			mRepo.EXPECT().WithTx(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, fn func(repository.Repository) error) error {
				return fn(mRepo)
			}).AnyTimes()
			mRepo.EXPECT().StoreOutbox(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			svc := NewAuthService(mRepo, nil, events.NewBus(), &audit.NoopLogger{}, nil, "dev")

			if tt.setup != nil {
				tt.setup(mRepo)
			}

			resp, err := svc.UpdateProfile(tt.ctx, tt.req)

			if tt.expectedError != "" {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "New Name", resp.User.DisplayName)
			}
		})
	}
}

func TestAuthService_ChangeEmail(t *testing.T) {
	userID := testUserID
	tests := []struct {
		name          string
		req           *authv1.ChangeEmailRequest
		setup         func(*mocks.MockRepository)
		ctx           context.Context
		expectedError string
	}{
		{
			name: "Success",
			req:  &authv1.ChangeEmailRequest{NewEmail: "new@example.com"},
			ctx:  authn.ContextWithClaims(context.Background(), authn.Claims{UserID: userID}),
			setup: func(mRepo *mocks.MockRepository) {
				mRepo.EXPECT().CreatePendingContactChange(gomock.Any(), gomock.Any(), userID, "email", "new@example.com", gomock.Any(), gomock.Any()).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mRepo := mocks.NewMockRepository(ctrl)
			mRepo.EXPECT().WithTx(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, fn func(repository.Repository) error) error {
				return fn(mRepo)
			}).AnyTimes()
			mRepo.EXPECT().StoreOutbox(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			svc := NewAuthService(mRepo, nil, events.NewBus(), &audit.NoopLogger{}, nil, "dev")

			if tt.setup != nil {
				tt.setup(mRepo)
			}

			resp, err := svc.ChangeEmail(tt.ctx, tt.req)

			if tt.expectedError != "" {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, resp.Success)
			}
		})
	}
}

func TestAuthService_ChangePhone(t *testing.T) {
	userID := testUserID
	tests := []struct {
		name          string
		req           *authv1.ChangePhoneRequest
		setup         func(*mocks.MockRepository)
		ctx           context.Context
		expectedError string
	}{
		{
			name: "Success",
			req:  &authv1.ChangePhoneRequest{NewPhone: "+1234567890"},
			ctx:  authn.ContextWithClaims(context.Background(), authn.Claims{UserID: userID}),
			setup: func(mRepo *mocks.MockRepository) {
				mRepo.EXPECT().CreatePendingContactChange(gomock.Any(), gomock.Any(), userID, "phone", "+1234567890", gomock.Any(), gomock.Any()).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mRepo := mocks.NewMockRepository(ctrl)
			mRepo.EXPECT().WithTx(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, fn func(repository.Repository) error) error {
				return fn(mRepo)
			}).AnyTimes()
			mRepo.EXPECT().StoreOutbox(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			svc := NewAuthService(mRepo, nil, events.NewBus(), &audit.NoopLogger{}, nil, "dev")

			if tt.setup != nil {
				tt.setup(mRepo)
			}

			resp, err := svc.ChangePhone(tt.ctx, tt.req)

			if tt.expectedError != "" {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, resp.Success)
			}
		})
	}
}
