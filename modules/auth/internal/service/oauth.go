// Package service implements the business logic for the authentication module.
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.jetify.com/typeid"

	"github.com/LoopContext/go-modulith-template/internal/authtoken"
	"github.com/LoopContext/go-modulith-template/internal/oauth"
	"github.com/LoopContext/go-modulith-template/modules/auth/internal/repository"
)

// OAuthService handles OAuth-related business logic.
type OAuthService struct {
	repo           repository.Repository
	tokenService   *authtoken.Service
	oauthRegistry  *oauth.Registry
	tokenEncryptor TokenEncryptor
	autoLinkEmail  bool
}

// TokenEncryptor defines the interface for encrypting/decrypting OAuth tokens.
type TokenEncryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

// NewOAuthService creates a new OAuthService.
func NewOAuthService(
	repo repository.Repository,
	tokenService *authtoken.Service,
	oauthRegistry *oauth.Registry,
	tokenEncryptor TokenEncryptor,
	autoLinkEmail bool,
) *OAuthService {
	return &OAuthService{
		repo:           repo,
		tokenService:   tokenService,
		oauthRegistry:  oauthRegistry,
		tokenEncryptor: tokenEncryptor,
		autoLinkEmail:  autoLinkEmail,
	}
}

// HandleOAuthCallback processes the OAuth callback and returns JWT tokens.
// This is called by the OAuth HTTP handler after successful OAuth flow.
func (s *OAuthService) HandleOAuthCallback(ctx context.Context, userInfo oauth.UserInfo, stateData *oauth.StateData) (*oauth.Result, error) {
	// Handle based on action type
	if stateData.Action == oauth.ActionLink {
		return s.handleAccountLinking(ctx, userInfo, stateData.UserID)
	}

	return s.handleLogin(ctx, userInfo)
}

// handleLogin handles OAuth login flow.
func (s *OAuthService) handleLogin(ctx context.Context, userInfo oauth.UserInfo) (*oauth.Result, error) {
	// Check if external account already exists
	existingAccount, err := s.repo.GetExternalAccountByProviderUserID(ctx, userInfo.Provider, userInfo.ProviderUserID)
	if err == nil {
		// Account exists, update tokens and profile, then login
		if err := s.updateExternalAccount(ctx, userInfo); err != nil {
			slog.WarnContext(ctx, "Failed to update external account", "error", err)
		}

		return s.generateTokensForUser(existingAccount.UserID, false)
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to check external account: %w", err)
	}

	// Account doesn't exist, check for auto-link by email
	if s.autoLinkEmail && userInfo.Email != "" {
		existingUser, err := s.repo.GetUserByEmail(ctx, userInfo.Email)
		if err == nil {
			// User with this email exists, link the external account
			if err := s.createExternalAccount(ctx, existingUser.ID, userInfo); err != nil {
				return nil, fmt.Errorf("failed to link external account: %w", err)
			}

			return s.generateTokensForUser(existingUser.ID, false)
		}

		if !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to check user by email: %w", err)
		}
	}

	// No existing account or user, blocked (security requirement)
	slog.WarnContext(ctx, "OAuth login attempt for non-existent user blocked",
		"provider", userInfo.Provider,
		"email", userInfo.Email,
	)

	return nil, fmt.Errorf("account not found: %s", userInfo.Email)
}

// handleAccountLinking links an external account to an existing user.
func (s *OAuthService) handleAccountLinking(ctx context.Context, userInfo oauth.UserInfo, userID string) (*oauth.Result, error) {
	// Check if this external account is already linked to another user
	existingAccount, err := s.repo.GetExternalAccountByProviderUserID(ctx, userInfo.Provider, userInfo.ProviderUserID)
	if err == nil && existingAccount.UserID != userID {
		return nil, fmt.Errorf("this %s account is already linked to another user", userInfo.Provider)
	}

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to check external account: %w", err)
	}

	// Create or update the external account
	if existingAccount != nil {
		if err := s.updateExternalAccount(ctx, userInfo); err != nil {
			return nil, fmt.Errorf("failed to update external account: %w", err)
		}
	} else {
		if err := s.createExternalAccount(ctx, userID, userInfo); err != nil {
			return nil, fmt.Errorf("failed to create external account: %w", err)
		}
	}

	return s.generateTokensForUser(userID, false)
}

// createExternalAccount creates a new external account link.
func (s *OAuthService) createExternalAccount(ctx context.Context, userID string, userInfo oauth.UserInfo) error {
	tid, err := typeid.WithPrefix("extacc")
	if err != nil {
		return fmt.Errorf("failed to generate external account typeid: %w", err)
	}

	// Encrypt tokens
	encryptedAccessToken, err := s.tokenEncryptor.Encrypt(userInfo.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt access token: %w", err)
	}

	encryptedRefreshToken, err := s.tokenEncryptor.Encrypt(userInfo.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt refresh token: %w", err)
	}

	var tokenExpiresAt *time.Time
	if !userInfo.ExpiresAt.IsZero() {
		tokenExpiresAt = &userInfo.ExpiresAt
	}

	account := &repository.ExternalAccount{
		ID:             tid.String(),
		UserID:         userID,
		Provider:       userInfo.Provider,
		ProviderUserID: userInfo.ProviderUserID,
		Email:          userInfo.Email,
		Name:           userInfo.Name,
		AvatarURL:      userInfo.AvatarURL,
		AccessToken:    encryptedAccessToken,
		RefreshToken:   encryptedRefreshToken,
		TokenExpiresAt: tokenExpiresAt,
		RawData:        userInfo.RawData,
	}

	if err := s.repo.CreateExternalAccount(ctx, account); err != nil {
		return fmt.Errorf("failed to create external account: %w", err)
	}

	return nil
}

// updateExternalAccount updates an existing external account.
func (s *OAuthService) updateExternalAccount(ctx context.Context, userInfo oauth.UserInfo) error {
	// Encrypt tokens
	encryptedAccessToken, err := s.tokenEncryptor.Encrypt(userInfo.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt access token: %w", err)
	}

	encryptedRefreshToken, err := s.tokenEncryptor.Encrypt(userInfo.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt refresh token: %w", err)
	}

	var tokenExpiresAt *time.Time
	if !userInfo.ExpiresAt.IsZero() {
		tokenExpiresAt = &userInfo.ExpiresAt
	}

	// Update tokens
	if err := s.repo.UpdateExternalAccountTokens(ctx, userInfo.Provider, userInfo.ProviderUserID,
		encryptedAccessToken, encryptedRefreshToken, tokenExpiresAt); err != nil {
		return fmt.Errorf("failed to update tokens: %w", err)
	}

	// Update profile
	if err := s.repo.UpdateExternalAccountProfile(ctx, userInfo.Provider, userInfo.ProviderUserID,
		userInfo.Name, userInfo.AvatarURL, userInfo.Email, userInfo.RawData); err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	return nil
}

// generateTokensForUser generates JWT tokens for a user.
func (s *OAuthService) generateTokensForUser(userID string, isNewUser bool) (*oauth.Result, error) {
	accessToken, _, err := s.tokenService.CreateToken(userID, "user", 1*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to create access token: %w", err)
	}

	refreshToken, _, err := s.tokenService.CreateToken(userID, "user", 24*time.Hour) //nolint:mnd
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	return &oauth.Result{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
		IsNewUser:    isNewUser,
		UserID:       userID,
	}, nil
}

// RepositoryStateStore implements oauth.StateStore using the repository.
type RepositoryStateStore struct {
	repo repository.Repository
}

// NewRepositoryStateStore creates a new RepositoryStateStore.
func NewRepositoryStateStore(repo repository.Repository) *RepositoryStateStore {
	return &RepositoryStateStore{repo: repo}
}

// SaveState saves an OAuth state authtoken.
func (s *RepositoryStateStore) SaveState(ctx context.Context, data *oauth.StateData) error {
	state := &repository.OAuthState{
		State:       data.State,
		Provider:    data.Provider,
		RedirectURL: data.RedirectURL,
		UserID:      data.UserID,
		Action:      string(data.Action),
		ExpiresAt:   data.ExpiresAt,
	}

	if err := s.repo.CreateOAuthState(ctx, state); err != nil {
		return fmt.Errorf("failed to save oauth state: %w", err)
	}

	return nil
}

// GetState retrieves an OAuth state authtoken.
func (s *RepositoryStateStore) GetState(ctx context.Context, state string) (*oauth.StateData, error) {
	repoState, err := s.repo.GetOAuthState(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("failed to get oauth state: %w", err)
	}

	return &oauth.StateData{
		State:       repoState.State,
		Provider:    repoState.Provider,
		RedirectURL: repoState.RedirectURL,
		UserID:      repoState.UserID,
		Action:      oauth.StateAction(repoState.Action),
		ExpiresAt:   repoState.ExpiresAt,
	}, nil
}

// DeleteState deletes an OAuth state authtoken.
func (s *RepositoryStateStore) DeleteState(ctx context.Context, state string) error {
	if err := s.repo.DeleteOAuthState(ctx, state); err != nil {
		return fmt.Errorf("failed to delete oauth state: %w", err)
	}

	return nil
}
