package oauth

import (
	"testing"
	"time"

	"github.com/markbates/goth"
)

const testProviderGoogle = "google"

func TestStateAction_Constants(t *testing.T) {
	// Test that action constants are defined correctly
	tests := []struct {
		action   StateAction
		expected string
	}{
		{ActionLogin, "login"},
		{ActionLink, "link"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.action) != tt.expected {
				t.Errorf("Action = %s, want %s", tt.action, tt.expected)
			}
		})
	}
}

//nolint:cyclop // Test function with many assertions
func TestUserInfo_Fields(t *testing.T) {
	info := UserInfo{
		ProviderUserID: "123456789",
		Provider:       testProviderGoogle,
		Email:          "user@example.com",
		Name:           "Test User",
		FirstName:      "Test",
		LastName:       "User",
		AvatarURL:      "https://example.com/avatar.jpg",
		AccessToken:    "access-token-value",
		RefreshToken:   "refresh-token-value",
		ExpiresAt:      time.Now().Add(time.Hour),
		RawData: map[string]interface{}{
			"verified_email": true,
			"locale":         "en",
		},
	}

	if info.ProviderUserID != "123456789" {
		t.Errorf("ProviderUserID = %s, want 123456789", info.ProviderUserID)
	}

	if info.Provider != testProviderGoogle {
		t.Errorf("Provider = %s, want google", info.Provider)
	}

	if info.Email != "user@example.com" {
		t.Errorf("Email = %s, want user@example.com", info.Email)
	}

	if info.Name != "Test User" {
		t.Errorf("Name = %s, want Test User", info.Name)
	}

	if info.FirstName != "Test" {
		t.Errorf("FirstName = %s, want Test", info.FirstName)
	}

	if info.LastName != "User" {
		t.Errorf("LastName = %s, want User", info.LastName)
	}

	if info.AvatarURL != "https://example.com/avatar.jpg" {
		t.Errorf("AvatarURL = %s, want https://example.com/avatar.jpg", info.AvatarURL)
	}

	if info.AccessToken != "access-token-value" {
		t.Errorf("AccessToken = %s, want access-token-value", info.AccessToken)
	}

	if info.RefreshToken != "refresh-token-value" {
		t.Errorf("RefreshToken = %s, want refresh-token-value", info.RefreshToken)
	}

	if len(info.RawData) != 2 {
		t.Errorf("RawData length = %d, want 2", len(info.RawData))
	}
}

func TestProviderInfo_Fields(t *testing.T) {
	info := ProviderInfo{
		Name:        testProviderGoogle,
		DisplayName: "Google",
		Enabled:     true,
	}

	if info.Name != testProviderGoogle {
		t.Errorf("Name = %s, want google", info.Name)
	}

	if info.DisplayName != "Google" {
		t.Errorf("DisplayName = %s, want Google", info.DisplayName)
	}

	if !info.Enabled {
		t.Error("Enabled = false, want true")
	}
}

func TestUserInfo_EmptyFields(t *testing.T) {
	info := UserInfo{}

	if info.ProviderUserID != "" {
		t.Errorf("ProviderUserID should be empty, got %s", info.ProviderUserID)
	}

	if info.Provider != "" {
		t.Errorf("Provider should be empty, got %s", info.Provider)
	}

	if info.Email != "" {
		t.Errorf("Email should be empty, got %s", info.Email)
	}

	if info.RawData != nil {
		t.Error("RawData should be nil")
	}
}

func TestFromGothUser(t *testing.T) {
	gothUser := goth.User{
		Provider:     "github",
		UserID:       "gh_123456",
		Email:        "test@github.com",
		Name:         "GitHub User",
		FirstName:    "GitHub",
		LastName:     "User",
		AvatarURL:    "https://avatars.githubusercontent.com/u/123",
		AccessToken:  "gh_access_token",
		RefreshToken: "gh_refresh_token",
		ExpiresAt:    time.Now().Add(time.Hour),
		RawData: map[string]interface{}{
			"login": "githubuser",
			"bio":   "Developer",
		},
	}

	info := FromGothUser(gothUser)

	if info.Provider != "github" {
		t.Errorf("Provider = %s, want github", info.Provider)
	}

	if info.ProviderUserID != "gh_123456" {
		t.Errorf("ProviderUserID = %s, want gh_123456", info.ProviderUserID)
	}

	if info.Email != "test@github.com" {
		t.Errorf("Email = %s, want test@github.com", info.Email)
	}

	if info.Name != "GitHub User" {
		t.Errorf("Name = %s, want GitHub User", info.Name)
	}

	if info.FirstName != "GitHub" {
		t.Errorf("FirstName = %s, want GitHub", info.FirstName)
	}

	if info.LastName != "User" {
		t.Errorf("LastName = %s, want User", info.LastName)
	}

	if info.AccessToken != "gh_access_token" {
		t.Errorf("AccessToken = %s, want gh_access_token", info.AccessToken)
	}

	if info.RefreshToken != "gh_refresh_token" {
		t.Errorf("RefreshToken = %s, want gh_refresh_token", info.RefreshToken)
	}

	if len(info.RawData) != 2 {
		t.Errorf("RawData length = %d, want 2", len(info.RawData))
	}
}

func TestStateData_Fields(t *testing.T) {
	now := time.Now()

	data := StateData{
		State:       "random-state-token",
		Provider:    testProviderGoogle,
		RedirectURL: "https://example.com/callback",
		UserID:      "user_12345",
		Action:      ActionLink,
		ExpiresAt:   now.Add(15 * time.Minute),
	}

	if data.State != "random-state-token" {
		t.Errorf("State = %s, want random-state-token", data.State)
	}

	if data.Provider != testProviderGoogle {
		t.Errorf("Provider = %s, want google", data.Provider)
	}

	if data.RedirectURL != "https://example.com/callback" {
		t.Errorf("RedirectURL = %s, want https://example.com/callback", data.RedirectURL)
	}

	if data.UserID != "user_12345" {
		t.Errorf("UserID = %s, want user_12345", data.UserID)
	}

	if data.Action != ActionLink {
		t.Errorf("Action = %s, want link", data.Action)
	}
}
