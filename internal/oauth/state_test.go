package oauth

import (
	"testing"
	"time"
)

func TestStateManager_GenerateState(t *testing.T) {
	key := []byte("test-secret-key-for-hmac-1234567")
	sm := NewStateManager(key)

	state1, err := sm.GenerateState()
	if err != nil {
		t.Fatalf("GenerateState() error = %v", err)
	}

	if state1 == "" {
		t.Error("GenerateState() returned empty string")
	}

	state2, err := sm.GenerateState()
	if err != nil {
		t.Fatalf("GenerateState() error = %v", err)
	}

	// States should be unique
	if state1 == state2 {
		t.Error("GenerateState() returned same state twice")
	}
}

func TestStateManager_ValidateState(t *testing.T) {
	key := []byte("test-secret-key-for-hmac-1234567")
	sm := NewStateManager(key)

	state, err := sm.GenerateState()
	if err != nil {
		t.Fatalf("GenerateState() error = %v", err)
	}

	if !sm.ValidateState(state) {
		t.Error("ValidateState() returned false for valid state")
	}
}

func TestStateManager_ValidateState_Invalid(t *testing.T) {
	key := []byte("test-secret-key-for-hmac-1234567")
	sm := NewStateManager(key)

	tests := []struct {
		name  string
		state string
	}{
		{name: "empty", state: ""},
		{name: "invalid base64", state: "not-valid-base64!!!"},
		{name: "too short", state: "dG9vLXNob3J0"}, // "too-short" base64 encoded
		{name: "random garbage", state: "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if sm.ValidateState(tt.state) {
				t.Error("ValidateState() should return false for invalid state")
			}
		})
	}
}

func TestStateManager_ValidateState_WrongKey(t *testing.T) {
	key1 := []byte("test-secret-key-for-hmac-1234567")
	key2 := []byte("different-secret-key-12345678901")

	sm1 := NewStateManager(key1)
	sm2 := NewStateManager(key2)

	state, err := sm1.GenerateState()
	if err != nil {
		t.Fatalf("GenerateState() error = %v", err)
	}

	// Should validate with same key
	if !sm1.ValidateState(state) {
		t.Error("ValidateState() with same key should return true")
	}

	// Should NOT validate with different key
	if sm2.ValidateState(state) {
		t.Error("ValidateState() with different key should return false")
	}
}

func TestStateManager_CreateStateData(t *testing.T) {
	key := []byte("test-secret-key-for-hmac-1234567")
	sm := NewStateManager(key)

	data, err := sm.CreateStateData(testProviderGoogle, "https://example.com/callback", "", ActionLogin)
	if err != nil {
		t.Fatalf("CreateStateData() error = %v", err)
	}

	if data.Provider != testProviderGoogle {
		t.Errorf("Provider = %s, want google", data.Provider)
	}

	if data.RedirectURL != "https://example.com/callback" {
		t.Errorf("RedirectURL = %s, want https://example.com/callback", data.RedirectURL)
	}

	if data.Action != ActionLogin {
		t.Errorf("Action = %s, want login", data.Action)
	}

	if data.UserID != "" {
		t.Errorf("UserID = %s, want empty", data.UserID)
	}

	// Verify state is valid
	if !sm.ValidateState(data.State) {
		t.Error("Generated state should be valid")
	}

	// Verify expiration is in the future
	if data.ExpiresAt.Before(time.Now()) {
		t.Error("ExpiresAt should be in the future")
	}
}

func TestStateManager_CreateStateData_WithUserID(t *testing.T) {
	key := []byte("test-secret-key-for-hmac-1234567")
	sm := NewStateManager(key)

	data, err := sm.CreateStateData("github", "", "user_12345", ActionLink)
	if err != nil {
		t.Fatalf("CreateStateData() error = %v", err)
	}

	if data.Provider != "github" {
		t.Errorf("Provider = %s, want github", data.Provider)
	}

	if data.UserID != "user_12345" {
		t.Errorf("UserID = %s, want user_12345", data.UserID)
	}

	if data.Action != ActionLink {
		t.Errorf("Action = %s, want link", data.Action)
	}
}

func TestStateManager_CreateStateData_AllProviders(t *testing.T) {
	key := []byte("test-secret-key-for-hmac-1234567")
	sm := NewStateManager(key)

	providers := []string{testProviderGoogle, "facebook", "github", "apple", "microsoft", "twitter"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			data, err := sm.CreateStateData(provider, "", "", ActionLogin)
			if err != nil {
				t.Fatalf("CreateStateData() error = %v", err)
			}

			if data.Provider != provider {
				t.Errorf("Provider = %s, want %s", data.Provider, provider)
			}

			if !sm.ValidateState(data.State) {
				t.Error("Generated state should be valid")
			}
		})
	}
}

func TestStateData_ExpiresAfter(t *testing.T) {
	key := []byte("test-secret-key-for-hmac-1234567")
	sm := NewStateManager(key)

	data, err := sm.CreateStateData(testProviderGoogle, "", "", ActionLogin)
	if err != nil {
		t.Fatalf("CreateStateData() error = %v", err)
	}

	// Should expire after default expiration time
	expectedExpiry := time.Now().Add(DefaultStateExpiration)
	if data.ExpiresAt.Before(expectedExpiry.Add(-time.Second)) || data.ExpiresAt.After(expectedExpiry.Add(time.Second)) {
		t.Errorf("ExpiresAt = %v, expected ~%v", data.ExpiresAt, expectedExpiry)
	}
}

func TestNewStateManager(t *testing.T) {
	key := []byte("any-key-will-work")
	sm := NewStateManager(key)

	if sm == nil {
		t.Error("NewStateManager() returned nil")
	}

	// Should be able to generate and validate states
	state, err := sm.GenerateState()
	if err != nil {
		t.Fatalf("GenerateState() error = %v", err)
	}

	if !sm.ValidateState(state) {
		t.Error("ValidateState() returned false for valid state")
	}
}
