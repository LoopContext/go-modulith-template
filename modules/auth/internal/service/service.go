// Package service implements the business logic for the authentication module.
//
//nolint:wrapcheck // gRPC status.Error should not be wrapped
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	authv1 "github.com/cmelgarejo/go-modulith-template/gen/go/proto/auth/v1"
	"github.com/cmelgarejo/go-modulith-template/internal/authn"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/i18n"
	"github.com/cmelgarejo/go-modulith-template/internal/notifier"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/db/store"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/repository"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/token"
	"go.jetify.com/typeid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AuthService implements the authv1.AuthServiceServer interface
type AuthService struct {
	authv1.UnimplementedAuthServiceServer
	repo         repository.Repository
	tokenService *token.Service
	bus          *events.Bus
}

// NewAuthService creates a new instance of the AuthService
func NewAuthService(repo repository.Repository, svc *token.Service, bus *events.Bus) *AuthService {
	return &AuthService{
		repo:         repo,
		tokenService: svc,
		bus:          bus,
	}
}

// RequestLogin generates a magic code and emits an event to send it to the user.
// Note: Field format validation (email format, phone pattern) and oneof requirement
// are handled by the validation interceptor. This method handles business logic only.
// Only sends codes to existing users (returns NotFound if user doesn't exist).
func (s *AuthService) RequestLogin(ctx context.Context, req *authv1.RequestLoginRequest) (*authv1.RequestLoginResponse, error) {
	// With oneof, exactly one field will be set (validated by protovalidate)
	email := req.GetEmail()
	phone := req.GetPhone()

	// Check if user exists before sending code
	if err := s.verifyUserExists(ctx, email, phone); err != nil {
		return nil, err
	}

	// Generate 6 digit code
	code, err := generateRandomCode(6)
	if err != nil {
		slog.ErrorContext(ctx, "failed to generate random code", "error", err)

		return nil, status.Error(codes.Internal, "internal server error")
	}

	expiresAt := time.Now().Add(15 * time.Minute)

	err = s.repo.CreateMagicCode(ctx, code, email, phone, expiresAt)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create magic code", "error", err)

		return nil, status.Error(codes.Internal, "internal server error")
	}

	// Emit event for notification (decoupled/async)
	s.publishMagicCodeEvent(ctx, email, phone, code)

	return &authv1.RequestLoginResponse{
		Success: true,
		Message: "Magic code sent",
	}, nil
}

// verifyUserExists checks if a user exists by email or phone, returning NotFound if not found.
func (s *AuthService) verifyUserExists(ctx context.Context, email, phone string) error {
	var err error

	if email != "" {
		_, err = s.repo.GetUserByEmail(ctx, email)
	} else {
		_, err = s.repo.GetUserByPhone(ctx, phone)
	}

	if err != nil {
		// Check for sql.ErrNoRows (works with wrapped errors via %w)
		if errors.Is(err, sql.ErrNoRows) {
			slog.InfoContext(ctx, "user not found for login request",
				"email", email,
				"phone", phone,
			)

			return status.Error(codes.NotFound, "user not found")
		}

		slog.ErrorContext(ctx, "failed to lookup user", "error", err)

		return status.Error(codes.Internal, "internal server error")
	}

	return nil
}

// publishMagicCodeEvent publishes an event for magic code notification.
func (s *AuthService) publishMagicCodeEvent(ctx context.Context, email, phone, code string) {
	locale := i18n.LocaleFromContext(ctx)
	if locale == "" {
		// Detect locale if not already in context
		locale = i18n.DetectLocale(ctx, "en")
	}

	s.bus.Publish(ctx, events.Event{
		Name: notifier.EventMagicCodeRequested,
		Payload: map[string]interface{}{
			"email":  email,
			"phone":  phone,
			"code":   code,
			"locale": locale,
		},
	})

	slog.InfoContext(ctx, "magic code event published")
}

// CompleteLogin verifies the magic code and generates tokens for the user
func (s *AuthService) CompleteLogin(ctx context.Context, req *authv1.CompleteLoginRequest) (*authv1.CompleteLoginResponse, error) {
	if err := s.verifyLoginRequest(ctx, req); err != nil {
		return nil, err
	}

	// With oneof, exactly one field will be set (validated by protovalidate)
	email := req.GetEmail()
	phone := req.GetPhone()

	user, err := s.getOrCreateUser(ctx, email, phone)
	if err != nil {
		return nil, err
	}

	// Clean up codes
	if err := s.repo.InvalidateMagicCodes(ctx, email, phone); err != nil {
		slog.ErrorContext(ctx, "failed to invalidate magic codes", "error", err)
	}

	return s.generateLoginResponse(user)
}

// verifyLoginRequest validates the login request.
// Note: Field format validation is handled by the validation interceptor.
// This method handles business logic validation (magic code verification).
// Note: The oneof requirement (email or phone) is handled by the validation interceptor.
func (s *AuthService) verifyLoginRequest(ctx context.Context, req *authv1.CompleteLoginRequest) error {
	email := req.GetEmail()
	phone := req.GetPhone()

	// Validate that at least one of email or phone is provided
	if email == "" && phone == "" {
		return status.Error(codes.InvalidArgument, "either email or phone must be provided")
	}

	// With oneof, exactly one field will be set (validated by protovalidate)
	if email != "" {
		return s.verifyMagicCodeByEmail(ctx, email, req.Code)
	}

	return s.verifyMagicCodeByPhone(ctx, phone, req.Code)
}

func (s *AuthService) verifyMagicCodeByEmail(ctx context.Context, email, code string) error {
	_, err := s.repo.GetValidMagicCodeByEmail(ctx, email, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.DebugContext(ctx, "magic code not found or expired",
				"email", email,
				"code", code,
			)

			return status.Error(codes.Unauthenticated, "invalid or expired code")
		}

		slog.ErrorContext(ctx, "failed to verify magic code", "error", err, "email", email)

		return status.Error(codes.Internal, "internal server error")
	}

	return nil
}

func (s *AuthService) verifyMagicCodeByPhone(ctx context.Context, phone, code string) error {
	_, err := s.repo.GetValidMagicCodeByPhone(ctx, phone, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.DebugContext(ctx, "magic code not found or expired",
				"phone", phone,
				"code", code,
			)

			return status.Error(codes.Unauthenticated, "invalid or expired code")
		}

		slog.ErrorContext(ctx, "failed to verify magic code", "error", err, "phone", phone)

		return status.Error(codes.Internal, "internal server error")
	}

	return nil
}

func (s *AuthService) getOrCreateUser(ctx context.Context, email, phone string) (*store.AuthUser, error) {
	var user *store.AuthUser

	var err error

	if email != "" {
		user, err = s.repo.GetUserByEmail(ctx, email)
	} else {
		user, err = s.repo.GetUserByPhone(ctx, phone)
	}

	if err == nil {
		return user, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		slog.ErrorContext(ctx, "failed to lookup user", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return s.handleSignup(ctx, email, phone)
}

func (s *AuthService) handleSignup(ctx context.Context, email, phone string) (*store.AuthUser, error) {
	tid, err := typeid.WithPrefix("user")
	if err != nil {
		slog.ErrorContext(ctx, "failed to generate user typeid", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	userID := tid.String()
	if err := s.repo.CreateUser(ctx, userID, email, phone); err != nil {
		slog.ErrorContext(ctx, "failed to create user", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	if email != "" {
		user, err := s.repo.GetUserByEmail(ctx, email)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user by email: %w", err)
		}

		return user, nil
	}

	user, err := s.repo.GetUserByPhone(ctx, phone)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user by phone: %w", err)
	}

	return user, nil
}

func (s *AuthService) generateLoginResponse(user *store.AuthUser) (*authv1.CompleteLoginResponse, error) {
	accessToken, err := s.tokenService.CreateToken(user.ID, "user", 1*time.Hour)
	if err != nil {
		slog.Error("failed to create access token", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	refreshToken, err := s.tokenService.CreateToken(user.ID, "user", 24*time.Hour)
	if err != nil {
		slog.Error("failed to create refresh token", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &authv1.CompleteLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
	}, nil
}

func generateRandomCode(length int) (string, error) {
	const digits = "0123456789"

	ret := make([]byte, length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", fmt.Errorf("random number generation failed: %w", err)
		}

		ret[i] = digits[num.Int64()]
	}

	return string(ret), nil
}

// hashToken creates a SHA-256 hash of a token for storage.
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// getUserIDFromContext extracts the user ID from the gRPC context.
func getUserIDFromContext(ctx context.Context) (string, error) {
	userID, ok := authn.UserIDFromContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "not authenticated")
	}

	return userID, nil
}

// getTokenFromContext extracts the raw token from the gRPC metadata.
func getTokenFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	authHeader := md.Get("authorization")
	if len(authHeader) == 0 {
		return ""
	}

	// Remove "Bearer " prefix
	if len(authHeader[0]) > 7 {
		return authHeader[0][7:]
	}

	return ""
}

// RefreshToken exchanges a refresh token for a new access token.
func (s *AuthService) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
	}

	// Verify the refresh token
	claims, err := s.tokenService.VerifyToken(req.RefreshToken)
	if err != nil {
		slog.DebugContext(ctx, "invalid refresh token", "error", err)
		return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
	}

	// Check if token is blacklisted
	tokenHash := hashToken(req.RefreshToken)

	blacklisted, err := s.repo.IsTokenBlacklisted(ctx, tokenHash)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check token blacklist", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	if blacklisted {
		return nil, status.Error(codes.Unauthenticated, "token has been revoked")
	}

	// Generate new tokens
	accessToken, err := s.tokenService.CreateToken(claims.Subject, claims.Role, 1*time.Hour) //nolint:mnd
	if err != nil {
		slog.ErrorContext(ctx, "failed to create access token", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	refreshToken, err := s.tokenService.CreateToken(claims.Subject, claims.Role, 24*time.Hour) //nolint:mnd
	if err != nil {
		slog.ErrorContext(ctx, "failed to create refresh token", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// Blacklist the old refresh token
	expiresAt := time.Unix(claims.ExpiresAt, 0)
	if err := s.repo.BlacklistToken(ctx, tokenHash, claims.Subject, "refresh", expiresAt); err != nil {
		slog.WarnContext(ctx, "failed to blacklist old refresh token", "error", err)
	}

	return &authv1.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
	}, nil
}

// Logout invalidates the current session and blacklists the token.
func (s *AuthService) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Get the current token and blacklist it
	rawToken := getTokenFromContext(ctx)
	if rawToken != "" {
		claims, err := s.tokenService.VerifyToken(rawToken)
		if err == nil {
			tokenHash := hashToken(rawToken)
			expiresAt := time.Unix(claims.ExpiresAt, 0)

			if err := s.repo.BlacklistToken(ctx, tokenHash, userID, "logout", expiresAt); err != nil {
				slog.WarnContext(ctx, "failed to blacklist token on logout", "error", err)
			}
		}
	}

	// If revoke_all is set, revoke all sessions
	if req.RevokeAll {
		_, err := s.repo.RevokeAllUserSessions(ctx, userID, "")
		if err != nil {
			slog.ErrorContext(ctx, "failed to revoke all sessions", "error", err)
			return nil, status.Error(codes.Internal, "internal server error")
		}
	}

	return &authv1.LogoutResponse{
		Success: true,
		Message: "Successfully logged out",
	}, nil
}

// GetProfile returns the current user's profile.
func (s *AuthService) GetProfile(ctx context.Context, _ *authv1.GetProfileRequest) (*authv1.GetProfileResponse, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "user not found")
		}

		slog.ErrorContext(ctx, "failed to get user", "error", err, "userID", userID)

		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &authv1.GetProfileResponse{
		User: userToProto(user),
	}, nil
}

// UpdateProfile updates the current user's profile.
func (s *AuthService) UpdateProfile(ctx context.Context, req *authv1.UpdateProfileRequest) (*authv1.UpdateProfileResponse, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpdateUserProfile(ctx, userID, req.DisplayName, req.AvatarUrl); err != nil {
		slog.ErrorContext(ctx, "failed to update profile", "error", err, "userID", userID)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// Fetch updated user
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get updated user", "error", err, "userID", userID)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &authv1.UpdateProfileResponse{
		User: userToProto(user),
	}, nil
}

// ChangeEmail initiates email change by sending verification to new email.
func (s *AuthService) ChangeEmail(ctx context.Context, req *authv1.ChangeEmailRequest) (*authv1.ChangeEmailResponse, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.NewEmail == "" {
		return nil, status.Error(codes.InvalidArgument, "new_email is required")
	}

	// Generate verification code
	code, err := generateRandomCode(6)
	if err != nil {
		slog.ErrorContext(ctx, "failed to generate code", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	changeID, err := typeid.WithPrefix("change")
	if err != nil {
		slog.ErrorContext(ctx, "failed to generate change id", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	expiresAt := time.Now().Add(15 * time.Minute)

	if err := s.repo.CreatePendingContactChange(ctx, changeID.String(), userID, "email", req.NewEmail, code, expiresAt); err != nil {
		slog.ErrorContext(ctx, "failed to create pending email change", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// Emit event for notification
	s.bus.Publish(ctx, events.Event{
		Name: notifier.EventMagicCodeRequested,
		Payload: map[string]string{
			"email": req.NewEmail,
			"code":  code,
		},
	})

	return &authv1.ChangeEmailResponse{
		Success: true,
		Message: "Verification code sent to new email",
	}, nil
}

// ChangePhone initiates phone change by sending verification to new phone.
func (s *AuthService) ChangePhone(ctx context.Context, req *authv1.ChangePhoneRequest) (*authv1.ChangePhoneResponse, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.NewPhone == "" {
		return nil, status.Error(codes.InvalidArgument, "new_phone is required")
	}

	// Generate verification code
	code, err := generateRandomCode(6)
	if err != nil {
		slog.ErrorContext(ctx, "failed to generate code", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	changeID, err := typeid.WithPrefix("change")
	if err != nil {
		slog.ErrorContext(ctx, "failed to generate change id", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	expiresAt := time.Now().Add(15 * time.Minute)

	if err := s.repo.CreatePendingContactChange(ctx, changeID.String(), userID, "phone", req.NewPhone, code, expiresAt); err != nil {
		slog.ErrorContext(ctx, "failed to create pending phone change", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// Emit event for notification
	s.bus.Publish(ctx, events.Event{
		Name: notifier.EventMagicCodeRequested,
		Payload: map[string]string{
			"phone": req.NewPhone,
			"code":  code,
		},
	})

	return &authv1.ChangePhoneResponse{
		Success: true,
		Message: "Verification code sent to new phone",
	}, nil
}

// ListSessions returns all active sessions for the current user.
func (s *AuthService) ListSessions(ctx context.Context, _ *authv1.ListSessionsRequest) (*authv1.ListSessionsResponse, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	sessions, err := s.repo.GetSessionsByUserID(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get sessions", "error", err, "userID", userID)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	protoSessions := make([]*authv1.Session, len(sessions))
	for i, sess := range sessions {
		protoSessions[i] = &authv1.Session{
			Id:           sess.ID,
			UserAgent:    sess.UserAgent,
			IpAddress:    sess.IPAddress,
			CreatedAt:    timestamppb.New(sess.CreatedAt),
			LastActiveAt: timestamppb.New(sess.LastActiveAt),
			IsCurrent:    false, // TODO: compare with current session
		}
	}

	return &authv1.ListSessionsResponse{
		Sessions: protoSessions,
	}, nil
}

// RevokeSession revokes a specific session.
func (s *AuthService) RevokeSession(ctx context.Context, req *authv1.RevokeSessionRequest) (*authv1.RevokeSessionResponse, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}

	// Verify the session belongs to the user
	session, err := s.repo.GetSessionByID(ctx, req.SessionId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "session not found")
		}

		slog.ErrorContext(ctx, "failed to get session", "error", err)

		return nil, status.Error(codes.Internal, "internal server error")
	}

	if session.UserID != userID {
		return nil, status.Error(codes.PermissionDenied, "cannot revoke session of another user")
	}

	if err := s.repo.RevokeSession(ctx, req.SessionId); err != nil {
		slog.ErrorContext(ctx, "failed to revoke session", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &authv1.RevokeSessionResponse{
		Success: true,
	}, nil
}

// RevokeAllSessions revokes all sessions except the current one.
func (s *AuthService) RevokeAllSessions(ctx context.Context, req *authv1.RevokeAllSessionsRequest) (*authv1.RevokeAllSessionsResponse, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	exceptSessionID := ""
	if !req.IncludeCurrent {
		// TODO: get current session ID from context
		exceptSessionID = ""
	}

	count, err := s.repo.RevokeAllUserSessions(ctx, userID, exceptSessionID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to revoke all sessions", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &authv1.RevokeAllSessionsResponse{
		RevokedCount: int32(count), //nolint:gosec // count is always small
	}, nil
}

// userToProto converts a store.AuthUser to a protobuf User message.
func userToProto(user *store.AuthUser) *authv1.User {
	u := &authv1.User{
		Id:        user.ID,
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}

	if user.Email.Valid {
		u.Email = user.Email.String
	}

	if user.Phone.Valid {
		u.Phone = user.Phone.String
	}

	if user.DisplayName.Valid {
		u.DisplayName = user.DisplayName.String
	}

	if user.AvatarUrl.Valid {
		u.AvatarUrl = user.AvatarUrl.String
	}

	return u
}

// =====================
// OAuth Methods
// =====================

// GetOAuthProviders returns the list of enabled OAuth providers.
func (s *AuthService) GetOAuthProviders(_ context.Context, _ *authv1.GetOAuthProvidersRequest) (*authv1.GetOAuthProvidersResponse, error) {
	// This is typically handled by the OAuth registry, but we provide a stub here
	// The actual provider list should come from the OAuth registry in the module
	return &authv1.GetOAuthProvidersResponse{
		Providers: []*authv1.OAuthProvider{},
	}, nil
}

// InitiateOAuth starts the OAuth flow for a provider.
func (s *AuthService) InitiateOAuth(_ context.Context, _ *authv1.InitiateOAuthRequest) (*authv1.InitiateOAuthResponse, error) {
	// OAuth flow is handled via HTTP, not gRPC
	return nil, status.Error(codes.Unimplemented, "use HTTP endpoint /v1/auth/oauth/{provider}/start")
}

// CompleteOAuth handles the OAuth callback and returns tokens.
func (s *AuthService) CompleteOAuth(_ context.Context, _ *authv1.CompleteOAuthRequest) (*authv1.CompleteOAuthResponse, error) {
	// OAuth callback is handled via HTTP, not gRPC
	return nil, status.Error(codes.Unimplemented, "use HTTP endpoint /v1/auth/oauth/callback")
}

// LinkExternalAccount links an external provider to the current user.
func (s *AuthService) LinkExternalAccount(_ context.Context, _ *authv1.LinkExternalAccountRequest) (*authv1.LinkExternalAccountResponse, error) {
	// Linking is handled via HTTP OAuth flow
	return nil, status.Error(codes.Unimplemented, "use HTTP endpoint /v1/auth/oauth/{provider}/link")
}

// UnlinkExternalAccount unlinks an external provider from the current user.
//
//nolint:cyclop // Account unlinking requires multiple validation checks
func (s *AuthService) UnlinkExternalAccount(ctx context.Context, req *authv1.UnlinkExternalAccountRequest) (*authv1.UnlinkExternalAccountResponse, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Provider == "" {
		return nil, status.Error(codes.InvalidArgument, "provider is required")
	}

	// Check if user has other login methods before unlinking
	// (they need at least one way to authenticate)
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get user", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// Count external accounts
	accountCount, err := s.repo.CountExternalAccountsByUserID(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to count external accounts", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// User needs at least email/phone OR another external account
	hasEmail := user.Email.Valid && user.Email.String != ""
	hasPhone := user.Phone.Valid && user.Phone.String != ""

	if !hasEmail && !hasPhone && accountCount <= 1 {
		return nil, status.Error(codes.FailedPrecondition, "cannot unlink last authentication method")
	}

	if err := s.repo.DeleteExternalAccountByProvider(ctx, userID, req.Provider); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "external account not found")
		}

		slog.ErrorContext(ctx, "failed to unlink external account", "error", err)

		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &authv1.UnlinkExternalAccountResponse{
		Success: true,
		Message: "External account unlinked successfully",
	}, nil
}

// ListLinkedAccounts returns the external accounts linked to the current user.
func (s *AuthService) ListLinkedAccounts(ctx context.Context, _ *authv1.ListLinkedAccountsRequest) (*authv1.ListLinkedAccountsResponse, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	accounts, err := s.repo.GetExternalAccountsByUserID(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get linked accounts", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	protoAccounts := make([]*authv1.ExternalAccount, len(accounts))
	for i, acc := range accounts {
		protoAccounts[i] = &authv1.ExternalAccount{
			Id:             acc.ID,
			Provider:       acc.Provider,
			ProviderUserId: acc.ProviderUserID,
			Email:          acc.Email,
			Name:           acc.Name,
			AvatarUrl:      acc.AvatarURL,
			LinkedAt:       timestamppb.New(acc.CreatedAt),
		}
	}

	return &authv1.ListLinkedAccountsResponse{
		Accounts: protoAccounts,
	}, nil
}
