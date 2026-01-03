// Package oauth provides OAuth 2.0 integration with external providers.
package oauth

import (
	"time"

	"github.com/markbates/goth"
)

// ProviderInfo contains information about an OAuth provider.
type ProviderInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Enabled     bool   `json:"enabled"`
}

// UserInfo contains user information from an OAuth provider.
type UserInfo struct {
	Provider       string    `json:"provider"`
	ProviderUserID string    `json:"provider_user_id"`
	Email          string    `json:"email"`
	Name           string    `json:"name"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	AvatarURL      string    `json:"avatar_url"`
	AccessToken    string    `json:"-"`
	RefreshToken   string    `json:"-"`
	ExpiresAt      time.Time `json:"expires_at"`
	RawData        map[string]interface{}
}

// FromGothUser converts a goth.User to our UserInfo struct.
func FromGothUser(user goth.User) UserInfo {
	return UserInfo{
		Provider:       user.Provider,
		ProviderUserID: user.UserID,
		Email:          user.Email,
		Name:           user.Name,
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		AvatarURL:      user.AvatarURL,
		AccessToken:    user.AccessToken,
		RefreshToken:   user.RefreshToken,
		ExpiresAt:      user.ExpiresAt,
		RawData:        user.RawData,
	}
}

// StateAction defines what action to perform after OAuth callback.
type StateAction string

const (
	// ActionLogin indicates OAuth is being used for login.
	ActionLogin StateAction = "login"
	// ActionLink indicates OAuth is being used to link an account.
	ActionLink StateAction = "link"
)

// StateData holds information about an OAuth state token.
type StateData struct {
	State       string
	Provider    string
	RedirectURL string
	UserID      string // Non-empty when linking an account
	Action      StateAction
	ExpiresAt   time.Time
}
