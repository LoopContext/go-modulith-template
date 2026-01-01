package oauth

import (
	"testing"

	"github.com/cmelgarejo/go-modulith-template/internal/config"
)

func TestNewRegistry_Disabled(t *testing.T) {
	cfg := config.OAuthConfig{
		Enabled: false,
	}

	r, err := NewRegistry(cfg)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}

	if r.IsEnabled() {
		t.Error("IsEnabled() = true, want false")
	}

	providers := r.GetEnabledProviders()
	if len(providers) != 0 {
		t.Errorf("GetEnabledProviders() = %d providers, want 0", len(providers))
	}
}

func TestNewRegistry_WithProviders(t *testing.T) {
	cfg := config.OAuthConfig{
		Enabled: true,
		BaseURL: "http://localhost:8080",
		Providers: config.OAuthProviders{
			Google: config.OAuthProviderConfig{
				Enabled:      true,
				ClientID:     "test-google-client-id",
				ClientSecret: "test-google-secret",
			},
			GitHub: config.OAuthProviderConfig{
				Enabled:      true,
				ClientID:     "test-github-client-id",
				ClientSecret: "test-github-secret",
			},
		},
	}

	r, err := NewRegistry(cfg)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	if !r.IsEnabled() {
		t.Error("IsEnabled() = false, want true")
	}

	providers := r.GetEnabledProviders()
	if len(providers) != 2 {
		t.Errorf("GetEnabledProviders() = %d providers, want 2", len(providers))
	}

	if !r.IsProviderEnabled("google") {
		t.Error("IsProviderEnabled(google) = false, want true")
	}

	if !r.IsProviderEnabled("github") {
		t.Error("IsProviderEnabled(github) = false, want true")
	}

	if r.IsProviderEnabled("facebook") {
		t.Error("IsProviderEnabled(facebook) = true, want false")
	}
}

func TestRegistry_GetProvider(t *testing.T) {
	cfg := config.OAuthConfig{
		Enabled: true,
		BaseURL: "http://localhost:8080",
		Providers: config.OAuthProviders{
			Google: config.OAuthProviderConfig{
				Enabled:      true,
				ClientID:     "test-google-client-id",
				ClientSecret: "test-google-secret",
			},
		},
	}

	r, err := NewRegistry(cfg)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	provider, err := r.GetProvider("google")
	if err != nil {
		t.Errorf("GetProvider(google) error = %v", err)
	}

	if provider == nil {
		t.Error("GetProvider(google) returned nil provider")
	}

	_, err = r.GetProvider("nonexistent")
	if err == nil {
		t.Error("GetProvider(nonexistent) should return error")
	}
}

func TestRegistry_GetDisplayName(t *testing.T) {
	cfg := config.OAuthConfig{
		Enabled: false,
	}

	r, err := NewRegistry(cfg)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	tests := []struct {
		provider    string
		displayName string
	}{
		{"google", "Google"},
		{"facebook", "Facebook"},
		{"github", "GitHub"},
		{"apple", "Apple"},
		{"microsoftonline", "Microsoft"},
		{"twitter", "X (Twitter)"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			got := r.GetDisplayName(tt.provider)
			if got != tt.displayName {
				t.Errorf("GetDisplayName(%s) = %s, want %s", tt.provider, got, tt.displayName)
			}
		})
	}
}

func TestRegistry_AutoLinkByEmail(t *testing.T) {
	t.Run("enabled", func(t *testing.T) {
		cfg := config.OAuthConfig{
			Enabled:         false,
			AutoLinkByEmail: true,
		}

		r, err := NewRegistry(cfg)
		if err != nil {
			t.Fatalf("NewRegistry() error = %v", err)
		}

		if !r.AutoLinkByEmail() {
			t.Error("AutoLinkByEmail() = false, want true")
		}
	})

	t.Run("disabled", func(t *testing.T) {
		cfg := config.OAuthConfig{
			Enabled:         false,
			AutoLinkByEmail: false,
		}

		r, err := NewRegistry(cfg)
		if err != nil {
			t.Fatalf("NewRegistry() error = %v", err)
		}

		if r.AutoLinkByEmail() {
			t.Error("AutoLinkByEmail() = true, want false")
		}
	})
}

func TestRegistry_AllProviders(t *testing.T) {
	cfg := config.OAuthConfig{
		Enabled: true,
		BaseURL: "http://localhost:8080",
		Providers: config.OAuthProviders{
			Google: config.OAuthProviderConfig{
				Enabled:      true,
				ClientID:     "google-id",
				ClientSecret: "google-secret",
			},
			Facebook: config.OAuthProviderConfig{
				Enabled:      true,
				ClientID:     "facebook-id",
				ClientSecret: "facebook-secret",
			},
			GitHub: config.OAuthProviderConfig{
				Enabled:      true,
				ClientID:     "github-id",
				ClientSecret: "github-secret",
			},
			Microsoft: config.OAuthProviderConfig{
				Enabled:      true,
				ClientID:     "microsoft-id",
				ClientSecret: "microsoft-secret",
			},
			Twitter: config.OAuthProviderConfig{
				Enabled:      true,
				ClientID:     "twitter-id",
				ClientSecret: "twitter-secret",
			},
			Apple: config.AppleOAuthConfig{
				Enabled:        true,
				ClientID:       "apple-services-id",
				PrivateKeyPath: "/path/to/key.p8",
			},
		},
	}

	r, err := NewRegistry(cfg)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	providers := r.GetEnabledProviders()
	if len(providers) != 6 {
		t.Errorf("GetEnabledProviders() = %d providers, want 6", len(providers))
	}

	expectedProviders := []string{"google", "facebook", "github", "microsoftonline", "twitter", "apple"}
	for _, name := range expectedProviders {
		if !r.IsProviderEnabled(name) {
			t.Errorf("IsProviderEnabled(%s) = false, want true", name)
		}
	}
}
