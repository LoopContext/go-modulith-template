// Package service implements the business logic for the authentication module.
//
//nolint:wrapcheck // gRPC status.Error should not be wrapped
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	authv1 "github.com/cmelgarejo/go-modulith-template/gen/go/proto/auth/v1"
	"github.com/cmelgarejo/go-modulith-template/internal/audit"
	"github.com/cmelgarejo/go-modulith-template/internal/authn"
	"github.com/cmelgarejo/go-modulith-template/internal/authtoken"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/feature"
	"github.com/cmelgarejo/go-modulith-template/internal/i18n"
	"github.com/cmelgarejo/go-modulith-template/internal/notifier"
	"github.com/cmelgarejo/go-modulith-template/internal/telemetry"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/db/store"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/repository"
	"go.jetify.com/typeid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// ModuleName is the name of the module.
const ModuleName = "auth"

// RoleAdmin is the admin role.
const RoleAdmin = "admin"

// RoleUser is the user role.
const RoleUser = "user"

// AuthService implements the authv1.AuthServiceServer interface
type AuthService struct {
	authv1.UnimplementedAuthServiceServer
	repo         repository.Repository
	tokenService *authtoken.Service
	bus          *events.Bus
	audit        audit.Logger
	feature      feature.Manager
	env          string // "dev", "staging", "prod", etc.
}

// NewAuthService creates a new instance of the AuthService
func NewAuthService(repo repository.Repository, svc *authtoken.Service, bus *events.Bus, audit audit.Logger, feature feature.Manager, env string) *AuthService {
	return &AuthService{
		repo:         repo,
		tokenService: svc,
		bus:          bus,
		audit:        audit,
		feature:      feature,
		env:          env,
	}
}

// RequestLogin generates a magic code and emits an event to send it to the user.
// Note: This endpoint always returns success to prevent email enumeration attacks.
// If the user doesn't exist, no code is sent but the response looks identical.
func (s *AuthService) RequestLogin(ctx context.Context, req *authv1.RequestLoginRequest) (*authv1.RequestLoginResponse, error) {
	ctx, span := telemetry.ServiceSpan(ctx, ModuleName, "RequestLogin")
	defer span.End()

	// With oneof, exactly one field will be set (validated by protovalidate)
	email := req.GetEmail()
	phone := req.GetPhone()

	// Security: Always return success to prevent email enumeration.
	// If user doesn't exist, we simply don't send a code but return the same response.
	userExists := s.checkUserExists(ctx, email, phone)
	if !userExists {
		slog.InfoContext(ctx, "login request for non-existent user (silent fail)",
			"email_hash", hashContactInfo(email),
			"phone_hash", hashContactInfo(phone),
		)
		// Return success without sending a code
		return &authv1.RequestLoginResponse{
			Success: true,
			Message: "If an account exists with this email, you will receive a verification code",
		}, nil
	}

	// Generate 6 digit code
	code, err := generateRandomCode(6)
	if err != nil {
		slog.ErrorContext(ctx, "failed to generate random code", "error", err)

		return nil, status.Error(codes.Internal, "internal server error")
	}

	expiresAt := time.Now().Add(15 * time.Minute)

	err = s.repo.WithTx(ctx, func(txRepo repository.Repository) error {
		if txErr := txRepo.CreateMagicCode(ctx, code, email, phone, expiresAt); txErr != nil {
			return txErr
		}

		locale := i18n.LocaleFromContext(ctx)
		if locale == "" {
			locale = i18n.DetectLocale(ctx, "en")
		}

		if txErr := txRepo.StoreOutbox(ctx, notifier.EventMagicCodeRequested, map[string]interface{}{
			"email":  email,
			"phone":  phone,
			"code":   code,
			"locale": locale,
		}); txErr != nil {
			return txErr
		}

		return nil
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to create magic code or outbox event", "error", err)

		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &authv1.RequestLoginResponse{
		Success: true,
		Message: "If an account exists with this email, you will receive a verification code",
	}, nil
}

// checkUserExists checks if a user exists without returning an error.
// This is used to silently check for user existence without revealing it to clients.
func (s *AuthService) checkUserExists(ctx context.Context, email, phone string) bool {
	var err error

	if email != "" {
		_, err = s.repo.GetUserByEmail(ctx, email)
	} else {
		_, err = s.repo.GetUserByPhone(ctx, phone)
	}

	return err == nil
}

// hashContactInfo creates a simple hash of contact info for secure logging.
// This prevents PII from being logged while still allowing log correlation.
func hashContactInfo(info string) string {
	if info == "" {
		return ""
	}

	hash := sha256.Sum256([]byte(info))

	return hex.EncodeToString(hash[:8]) // Only first 8 bytes for brevity
}

// CompleteLogin verifies the magic code and generates tokens for the user
func (s *AuthService) CompleteLogin(ctx context.Context, req *authv1.CompleteLoginRequest) (*authv1.CompleteLoginResponse, error) {
	ctx, span := telemetry.ServiceSpan(ctx, ModuleName, "CompleteLogin")
	defer span.End()

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

	// Clean up codes and publish login event within transaction
	err = s.repo.WithTx(ctx, func(txRepo repository.Repository) error {
		if txErr := txRepo.InvalidateMagicCodes(ctx, email, phone); txErr != nil {
			slog.ErrorContext(ctx, "failed to invalidate magic codes", "error", txErr)
		}

		return txRepo.StoreOutbox(ctx, events.EventAuthUserLoggedIn, map[string]interface{}{
			"user_id":      user.ID,
			"email":        email,
			"phone":        phone,
			"login_method": "magic_code",
			"timestamp":    time.Now().UTC(),
		})
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to create outbox event for login", "error", err)
	}

	// Audit Log
	s.audit.Log(ctx, audit.LogParams{
		UserID:   user.ID,
		Action:   "LOGIN",
		Resource: ModuleName,
		Metadata: map[string]any{
			"method": "magic_code",
			"email":  email,
			"phone":  phone,
		},
		Success: true,
	})

	// Record business metrics
	if telemetry.AuthLoginTotal != nil {
		telemetry.AuthLoginTotal.Inc(ctx)
	}

	resp, err := s.generateLoginResponse(ctx, user)
	if err != nil {
		return nil, err
	}
	// Set HttpOnly cookies for access and refresh tokens (web clients)
	s.setAuthCookies(ctx, resp.AccessToken, resp.RefreshToken)

	return resp, nil
}

// Register creates a new user account.
//
//nolint:funlen // Register logic is naturally sequential: check exists, create user/profile/role/event in tx, fetch, audit, return
func (s *AuthService) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	ctx, span := telemetry.ServiceSpan(ctx, ModuleName, "Register")
	defer span.End()

	email := req.GetEmail()
	phone := req.GetPhone()
	displayName := req.GetDisplayName()
	nationality := req.GetNationality()
	docType := req.GetDocumentType()
	docNumber := req.GetDocumentNumber()

	// Check if user already exists
	if s.checkUserExists(ctx, email, phone) {
		return nil, status.Error(codes.AlreadyExists, "user already exists")
	}

	// Generate new user ID
	uid, err := typeid.WithPrefix("user")
	if err != nil {
		slog.ErrorContext(ctx, "failed to generate user id", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	userID := uid.String()

	// Use transaction to ensure user and role are created together
	if err := s.repo.WithTx(ctx, func(tx repository.Repository) error {
		if err := tx.CreateUser(ctx, userID, email, phone); err != nil {
			return err
		}

		if err := tx.UpdateUserProfile(ctx, userID, displayName, "", ""); err != nil {
			return err
		}

		if err := tx.AssignRole(ctx, userID, RoleUser); err != nil {
			return err
		}

		// Publish registration event via outbox
		return tx.StoreOutbox(ctx, events.EventAuthUserRegistered, events.NewUserRegisteredPayload(
			userID, email, phone, displayName, nationality, docType, docNumber,
		))
	}); err != nil {
		slog.ErrorContext(ctx, "failed to register user", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// Fetch created user
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch created user", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// Audit Log
	s.audit.Log(ctx, audit.LogParams{
		UserID:   userID,
		Action:   "REGISTER",
		Resource: ModuleName,
		Metadata: map[string]any{
			"email":        email,
			"phone":        phone,
			"display_name": displayName,
		},
		Success: true,
	})

	slog.InfoContext(ctx, "Dummy verification email sent", "email", email)

	return &authv1.RegisterResponse{
		Success: true,
		Message: "Account created successfully. Please log in to continue.",
		User:    userToProto(user),
	}, nil
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

	// Bypass magic code in development for automated testing with AI
	// SECURITY: Only allowed in dev environment
	if req.Code == "000000" && s.env == "dev" {
		slog.WarnContext(ctx, "DEV ONLY: bypassing magic code check", "email", email, "phone", phone)
		return nil
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
		if errors.Is(err, pgx.ErrNoRows) {
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
		if errors.Is(err, pgx.ErrNoRows) {
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
	var (
		user *store.AuthUser
		err  error
	)

	if email != "" {
		user, err = s.repo.GetUserByEmail(ctx, email)
	} else {
		user, err = s.repo.GetUserByPhone(ctx, phone)
	}

	if err == nil {
		return user, nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		slog.WarnContext(ctx, "login attempt for non-existent user blocked",
			"email_hash", hashContactInfo(email),
			"phone_hash", hashContactInfo(phone),
		)

		return nil, status.Error(codes.NotFound, "user not found")
	}

	slog.ErrorContext(ctx, "failed to lookup user", "error", err)

	return nil, status.Error(codes.Internal, "internal server error")
}

func (s *AuthService) generateLoginResponse(ctx context.Context, user *store.AuthUser) (*authv1.CompleteLoginResponse, error) {
	role, err := s.repo.GetUserRole(ctx, user.ID)
	if err != nil {
		slog.WarnContext(ctx, "failed to get user role, defaulting to user", "user_id", user.ID, "error", err)

		role = RoleUser
	}

	accessToken, _, err := s.tokenService.CreateToken(user.ID, role, 1*time.Hour)
	if err != nil {
		slog.Error("failed to create access token", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	refreshToken, jti, err := s.tokenService.CreateToken(user.ID, role, 24*time.Hour)
	if err != nil {
		slog.Error("failed to create refresh token", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// Create session in DB
	err = s.repo.CreateSession(ctx, &repository.Session{
		ID:               jti,
		UserID:           user.ID,
		RefreshTokenHash: hashToken(refreshToken),
		UserAgent:        "", // Optional: Could extract from context if available
		IPAddress:        "", // Optional: Could extract from context if available
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
		ExpiresAt:        time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		slog.Error("failed to create session", "error", err)
		// Non-blocking for now? Or return error?
		// Usually we want session creation to be mandatory for tracking.
	}

	// Set HttpOnly cookies in gRPC metadata (will be forwarded by gateway)
	s.setAuthCookies(ctx, accessToken, refreshToken)

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

// getJTIFromContext extracts the JTI from the token in the context.
func (s *AuthService) getJTIFromContext(ctx context.Context) string {
	rawToken := getTokenFromContext(ctx)
	if rawToken == "" {
		return ""
	}

	claims, err := s.tokenService.VerifyToken(rawToken)
	if err != nil {
		return ""
	}

	return claims.ID
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

// setAuthCookies sends Set-Cookie via gRPC response metadata (HttpOnly, Secure, SameSite=Lax).
func (s *AuthService) setAuthCookies(ctx context.Context, accessToken, refreshToken string) {
	isProd := s.env == "prod"

	accessCookie := &http.Cookie{ //nolint:gosec // G601: attributes are dynamically set
		Name:     authn.AccessTokenCookieName,
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   isProd,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600, // 1 hour
	}

	refreshCookie := &http.Cookie{ //nolint:gosec // G601: attributes are dynamically set
		Name:     authn.RefreshTokenCookieName,
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   isProd,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   24 * 3600, // 24 hours
	}

	// Add both cookies to metadata
	_ = grpc.SetHeader(ctx, metadata.Pairs(
		"set-cookie", accessCookie.String(),
		"set-cookie", refreshCookie.String(),
	))
}

// clearAuthCookies sends Set-Cookie to clear auth cookies.
func (s *AuthService) clearAuthCookies(ctx context.Context) {
	isProd := s.env == "prod"

	accessCookie := &http.Cookie{ //nolint:gosec // G601: attributes are dynamically set
		Name:     authn.AccessTokenCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   isProd,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}

	refreshCookie := &http.Cookie{ //nolint:gosec // G601: attributes are dynamically set
		Name:     authn.RefreshTokenCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   isProd,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}

	_ = grpc.SetHeader(ctx, metadata.Pairs(
		"set-cookie", accessCookie.String(),
		"set-cookie", refreshCookie.String(),
	))
}

// getRefreshTokenFromCookie extracts refresh_token from the Cookie header in gRPC metadata.
func getRefreshTokenFromCookie(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	// Gateway passes Cookie header; gRPC normalizes keys to lowercase
	cookieHeaders := md.Get("cookie")
	if len(cookieHeaders) == 0 {
		return ""
	}

	prefix := authn.RefreshTokenCookieName + "="

	for _, line := range cookieHeaders {
		for _, part := range strings.Split(line, ";") {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, prefix) {
				val := strings.TrimPrefix(part, prefix)

				return strings.TrimSpace(val)
			}
		}
	}

	return ""
}

// RefreshToken refreshes an expired session if the refresh token is valid.
//
//nolint:funlen // Security verifications inherently run long
func (s *AuthService) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	ctx, span := telemetry.ServiceSpan(ctx, ModuleName, "RefreshToken")
	defer span.End()

	refreshToken := getRefreshTokenFromCookie(ctx)
	if refreshToken == "" {
		refreshToken = req.RefreshToken
	}

	if refreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required (body or cookie)")
	}

	// Verify the refresh token
	claims, err := s.tokenService.VerifyToken(refreshToken)
	if err != nil {
		slog.DebugContext(ctx, "invalid refresh token", "error", err)
		return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
	}

	// Check if token is blacklisted
	tokenHash := hashToken(refreshToken)

	blacklisted, err := s.repo.IsTokenBlacklisted(ctx, tokenHash)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check token blacklist", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	if blacklisted {
		return nil, status.Error(codes.Unauthenticated, "token has been revoked")
	}

	// Generate new tokens
	accessToken, _, err := s.tokenService.CreateToken(claims.Subject, claims.Role, 1*time.Hour) //nolint:mnd
	if err != nil {
		slog.ErrorContext(ctx, "failed to create access token", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	refreshToken, jti, err := s.tokenService.CreateToken(claims.Subject, claims.Role, 24*time.Hour) //nolint:mnd
	if err != nil {
		slog.ErrorContext(ctx, "failed to create refresh token", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// Create new session in DB
	err = s.repo.CreateSession(ctx, &repository.Session{
		ID:               jti,
		UserID:           claims.Subject,
		RefreshTokenHash: hashToken(refreshToken),
		ExpiresAt:        time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to create session on refresh", "error", err)
	}

	// Blacklist the old refresh token
	expiresAt := time.Unix(claims.ExpiresAt, 0)
	if err := s.repo.BlacklistToken(ctx, tokenHash, claims.Subject, "refresh", expiresAt); err != nil {
		slog.WarnContext(ctx, "failed to blacklist old refresh token", "error", err)
	}

	// Set new auth cookies for web clients
	s.setAuthCookies(ctx, accessToken, refreshToken)

	return &authv1.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
	}, nil
}

// Logout invalidates the current session and blacklists the authtoken.
func (s *AuthService) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	ctx, span := telemetry.ServiceSpan(ctx, ModuleName, "Logout")
	defer span.End()

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

	// Audit Log
	s.audit.Log(ctx, audit.LogParams{
		UserID:   userID,
		Action:   "LOGOUT",
		Resource: ModuleName,
		Metadata: map[string]any{
			"revoke_all": req.RevokeAll,
		},
		Success: true,
	})

	// Clear auth cookies
	s.clearAuthCookies(ctx)

	return &authv1.LogoutResponse{
		Success: true,
		Message: "Successfully logged out",
	}, nil
}

// GetProfile returns the current user's profile.
func (s *AuthService) GetProfile(ctx context.Context, _ *authv1.GetProfileRequest) (*authv1.GetProfileResponse, error) {
	ctx, span := telemetry.ServiceSpan(ctx, ModuleName, "GetProfile")
	defer span.End()

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "user not found")
		}

		slog.ErrorContext(ctx, "failed to get user", "error", err, "userID", userID)

		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &authv1.GetProfileResponse{
		User: userToProto(user),
	}, nil
}

// RequestEmailVerification initiates the email verification process.
func (s *AuthService) RequestEmailVerification(ctx context.Context, _ *authv1.RequestEmailVerificationRequest) (*authv1.RequestEmailVerificationResponse, error) {
	ctx, span := telemetry.ServiceSpan(ctx, ModuleName, "RequestEmailVerification")
	defer span.End()

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "internal server error")
	}

	if user.EmailVerified {
		return &authv1.RequestEmailVerificationResponse{
			Success: true,
			Message: "Email already verified",
		}, nil
	}

	// Publish event via outbox
	if err := s.repo.StoreOutbox(ctx, events.EventAuthEmailVerificationRequested, map[string]any{
		"user_id": userID,
		"email":   user.Email.String,
	}); err != nil {
		slog.ErrorContext(ctx, "failed to store outbox event", "error", err)
	}

	return &authv1.RequestEmailVerificationResponse{
		Success: true,
		Message: "Verification email sent",
	}, nil
}

// HandleEmailVerificationRequested handles the dummy verification process.
func (s *AuthService) HandleEmailVerificationRequested(ctx context.Context, userID string) error {
	slog.InfoContext(ctx, "Dummy verification: marking email as verified", "user_id", userID)

	// Simulating async verification
	time.Sleep(1 * time.Second)

	if err := s.repo.MarkEmailVerified(ctx, userID); err != nil {
		slog.ErrorContext(ctx, "failed to mark email as verified", "error", err, "user_id", userID)
		return err
	}

	// Audit Log
	s.audit.Log(ctx, audit.LogParams{
		UserID:   userID,
		Action:   "EMAIL_VERIFIED",
		Resource: ModuleName,
		Metadata: map[string]any{
			"method": "dummy",
		},
		Success: true,
	})

	return nil
}

// UpdateProfile updates the current user's profile.
func (s *AuthService) UpdateProfile(ctx context.Context, req *authv1.UpdateProfileRequest) (*authv1.UpdateProfileResponse, error) {
	ctx, span := telemetry.ServiceSpan(ctx, ModuleName, "UpdateProfile")
	defer span.End()

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpdateUserProfile(ctx, userID, req.DisplayName, req.AvatarUrl, req.Timezone); err != nil {
		slog.ErrorContext(ctx, "failed to update profile", "error", err, "userID", userID)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// Fetch updated user
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get updated user", "error", err, "userID", userID)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// Audit Log
	s.audit.Log(ctx, audit.LogParams{
		UserID:     userID,
		Action:     events.EventAuthProfileUpdated,
		Resource:   ModuleName,
		ResourceID: userID,
		Metadata: map[string]any{
			"display_name": req.DisplayName,
			"avatar_url":   req.AvatarUrl,
		},
		Success: true,
	})

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

	err = s.repo.WithTx(ctx, func(txRepo repository.Repository) error {
		if txErr := txRepo.CreatePendingContactChange(ctx, changeID.String(), userID, "email", req.NewEmail, code, expiresAt); txErr != nil {
			return txErr
		}

		return txRepo.StoreOutbox(ctx, notifier.EventMagicCodeRequested, map[string]string{
			"email": req.NewEmail,
			"code":  code,
		})
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to create pending email change or outbox event", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// Audit Log
	s.audit.Log(ctx, audit.LogParams{
		UserID:   userID,
		Action:   "auth.contact_change_requested",
		Resource: ModuleName,
		Metadata: map[string]any{
			"type":      "email",
			"new_email": req.NewEmail,
		},
		Success: true,
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

	err = s.repo.WithTx(ctx, func(txRepo repository.Repository) error {
		if txErr := txRepo.CreatePendingContactChange(ctx, changeID.String(), userID, "phone", req.NewPhone, code, expiresAt); txErr != nil {
			return txErr
		}

		return txRepo.StoreOutbox(ctx, notifier.EventMagicCodeRequested, map[string]string{
			"phone": req.NewPhone,
			"code":  code,
		})
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to create pending phone change or outbox event", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// Audit Log
	s.audit.Log(ctx, audit.LogParams{
		UserID:   userID,
		Action:   "auth.contact_change_requested",
		Resource: ModuleName,
		Metadata: map[string]any{
			"type":      "phone",
			"new_phone": req.NewPhone,
		},
		Success: true,
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

	currentJTI := s.getJTIFromContext(ctx)

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
			IsCurrent:    sess.ID == currentJTI,
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
		if errors.Is(err, pgx.ErrNoRows) {
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

	// Audit Log
	s.audit.Log(ctx, audit.LogParams{
		UserID:     userID,
		Action:     "REVOKE_SESSION",
		Resource:   ModuleName,
		ResourceID: req.SessionId,
		Success:    true,
	})

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
		exceptSessionID = s.getJTIFromContext(ctx)
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
		CreatedAt: timestamppb.New(user.CreatedAt.Time),
		UpdatedAt: timestamppb.New(user.UpdatedAt.Time),
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

	if user.Timezone.Valid {
		u.Timezone = user.Timezone.String
	}

	u.EmailVerified = user.EmailVerified
	u.PhoneVerified = user.PhoneVerified

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
		if errors.Is(err, pgx.ErrNoRows) {
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

// GetSystemConfig returns public system configurations and feature flags.
func (s *AuthService) GetSystemConfig(ctx context.Context, _ *authv1.GetSystemConfigRequest) (*authv1.GetSystemConfigResponse, error) {
	config := make(map[string]string)

	// kyc_enabled is a public feature flag
	kycEnabled := s.feature.IsEnabled(ctx, "kyc_enabled")
	if kycEnabled {
		config["kyc_enabled"] = "true"
	} else {
		config["kyc_enabled"] = "false"
	}

	return &authv1.GetSystemConfigResponse{
		Configs: config,
	}, nil
}
