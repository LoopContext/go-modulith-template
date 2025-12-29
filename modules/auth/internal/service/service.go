// Package service implements the business logic for the authentication module.
//
//nolint:wrapcheck // gRPC status.Error should not be wrapped
package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	authv1 "github.com/cmelgarejo/go-modulith-template/gen/go/proto/auth/v1"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/notifier"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/db/store"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/repository"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/token"
	"go.jetify.com/typeid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

// RequestLogin generates a magic code and emits an event to send it to the user
func (s *AuthService) RequestLogin(ctx context.Context, req *authv1.RequestLoginRequest) (*authv1.RequestLoginResponse, error) {
	if req.Email == "" && req.Phone == "" {
		return nil, status.Error(codes.InvalidArgument, "email or phone must be provided")
	}

	// Generate 6 digit code
	code, err := generateRandomCode(6)
	if err != nil {
		slog.ErrorContext(ctx, "failed to generate random code", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	expiresAt := time.Now().Add(15 * time.Minute)

	err = s.repo.CreateMagicCode(ctx, code, req.Email, req.Phone, expiresAt)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create magic code", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// Emit event for notification (decoupled/async)
	s.bus.Publish(ctx, events.Event{
		Name: notifier.EventMagicCodeRequested,
		Payload: map[string]string{
			"email": req.Email,
			"phone": req.Phone,
			"code":  code,
		},
	})

	slog.InfoContext(ctx, "magic code event published")

	return &authv1.RequestLoginResponse{
		Success: true,
		Message: "Magic code sent",
	}, nil
}

// CompleteLogin verifies the magic code and generates tokens for the user
func (s *AuthService) CompleteLogin(ctx context.Context, req *authv1.CompleteLoginRequest) (*authv1.CompleteLoginResponse, error) {
	if err := s.verifyLoginRequest(ctx, req); err != nil {
		return nil, err
	}

	user, err := s.getOrCreateUser(ctx, req.Email, req.Phone)
	if err != nil {
		return nil, err
	}

	// Clean up codes
	if err := s.repo.InvalidateMagicCodes(ctx, req.Email, req.Phone); err != nil {
		slog.ErrorContext(ctx, "failed to invalidate magic codes", "error", err)
	}

	return s.generateLoginResponse(user)
}

func (s *AuthService) verifyLoginRequest(ctx context.Context, req *authv1.CompleteLoginRequest) error {
	if req.Email == "" && req.Phone == "" {
		return status.Error(codes.InvalidArgument, "email or phone required")
	}

	if req.Email != "" {
		return s.verifyMagicCodeByEmail(ctx, req.Email, req.Code)
	}

	return s.verifyMagicCodeByPhone(ctx, req.Phone, req.Code)
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

func (s *AuthService) getOrCreateUser(ctx context.Context, email, phone string) (*store.User, error) {
	var user *store.User

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

func (s *AuthService) handleSignup(ctx context.Context, email, phone string) (*store.User, error) {
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

func (s *AuthService) generateLoginResponse(user *store.User) (*authv1.CompleteLoginResponse, error) {
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
