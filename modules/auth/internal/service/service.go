package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"log/slog"
	"math/big"
	"time"

	authv1 "github.com/cmelgarejo/go-modulith-template/gen/go/proto/auth/v1"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/db/store"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/repository"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/token"
	"go.jetify.com/typeid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthService struct {
	authv1.UnimplementedAuthServiceServer
	repo         repository.Repository
	tokenService *token.TokenService
}

func NewAuthService(repo repository.Repository, ts *token.TokenService) *AuthService {
	return &AuthService{
		repo:         repo,
		tokenService: ts,
	}
}

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

	// TODO: Send code via Email/SMS provider.
	// For now, we log it (INSECURE for production, but okay for dev as per docs note).
	slog.InfoContext(ctx, "magic code generated", "code", code)

	return &authv1.RequestLoginResponse{
		Success: true,
		Message: "Magic code sent",
	}, nil
}

func (s *AuthService) CompleteLogin(ctx context.Context, req *authv1.CompleteLoginRequest) (*authv1.CompleteLoginResponse, error) {
	// Validate code
	if req.Email == "" && req.Phone == "" {
		return nil, status.Error(codes.InvalidArgument, "email or phone required")
	}

	var err error
	if req.Email != "" {
		_, err = s.repo.GetValidMagicCodeByEmail(ctx, req.Email, req.Code)
	} else {
		_, err = s.repo.GetValidMagicCodeByPhone(ctx, req.Phone, req.Code)
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired code")
		}
		slog.ErrorContext(ctx, "failed to verify magic code", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// Verify User exists or Create (Implicit Signup)
	var user *store.User
	if req.Email != "" {
		user, err = s.repo.GetUserByEmail(ctx, req.Email)
	} else {
		user, err = s.repo.GetUserByPhone(ctx, req.Phone)
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Create User (Implicit Signup)
			tid, err := typeid.WithPrefix("user")
			if err != nil {
				slog.ErrorContext(ctx, "failed to generate user typeid", "error", err)
				return nil, status.Error(codes.Internal, "internal server error")
			}
			userID := tid.String()
			err = s.repo.CreateUser(ctx, userID, req.Email, req.Phone)
			if err != nil {
				slog.ErrorContext(ctx, "failed to create user", "error", err)
				return nil, status.Error(codes.Internal, "internal server error")
			}

			// Refetch user to get full struct (or just use local values)
			if req.Email != "" {
				user, _ = s.repo.GetUserByEmail(ctx, req.Email)
			} else {
				user, _ = s.repo.GetUserByPhone(ctx, req.Phone)
			}
		} else {
			slog.ErrorContext(ctx, "failed to lookup user", "error", err)
			return nil, status.Error(codes.Internal, "internal server error")
		}
	}

	if user == nil {
		return nil, status.Error(codes.Internal, "failed to retrieve or create user")
	}

	// Clean up codes
	_ = s.repo.InvalidateMagicCodes(ctx, req.Email, req.Phone)

	// Generate Tokens
	accessToken, err := s.tokenService.CreateToken(user.ID, "user", 1*time.Hour)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create access token", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}
	refreshToken, err := s.tokenService.CreateToken(user.ID, "user", 24*time.Hour)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create refresh token", "error", err)
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
			return "", err
		}
		ret[i] = digits[num.Int64()]
	}
	return string(ret), nil
}
