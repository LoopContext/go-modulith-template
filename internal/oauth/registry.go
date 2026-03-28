package oauth

import (
	"fmt"
	"log/slog"

	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/apple"
	"github.com/markbates/goth/providers/facebook"
	"github.com/markbates/goth/providers/github"
	gothgoogle "github.com/markbates/goth/providers/google"
	"github.com/markbates/goth/providers/microsoftonline"
	"github.com/markbates/goth/providers/twitter"

	"github.com/LoopContext/go-modulith-template/internal/config"
)

// providerDisplayNames maps provider names to display names.
var providerDisplayNames = map[string]string{
	"google":          "Google",
	"facebook":        "Facebook",
	"github":          "GitHub",
	"apple":           "Apple",
	"microsoftonline": "Microsoft",
	"twitter":         "X (Twitter)",
}

// Registry manages OAuth providers.
type Registry struct {
	cfg       config.OAuthConfig
	providers map[string]ProviderInfo
}

// NewRegistry creates a new OAuth provider registry.
func NewRegistry(cfg config.OAuthConfig) (*Registry, error) {
	r := &Registry{
		cfg:       cfg,
		providers: make(map[string]ProviderInfo),
	}

	if !cfg.Enabled {
		slog.Info("OAuth is disabled")

		return r, nil
	}

	r.initializeProviders()

	return r, nil
}

// initializeProviders sets up all enabled OAuth providers.
//
//nolint:cyclop,funlen // Provider initialization requires checking each provider
func (r *Registry) initializeProviders() {
	var providers []goth.Provider

	callbackBase := r.cfg.BaseURL + "/v1/auth/oauth/callback"

	// Google
	if r.cfg.Providers.Google.Enabled {
		scopes := r.cfg.Providers.Google.Scopes
		if len(scopes) == 0 {
			scopes = []string{"email", "profile"}
		}

		p := gothgoogle.New(
			r.cfg.Providers.Google.ClientID,
			r.cfg.Providers.Google.ClientSecret,
			callbackBase+"?provider=google",
			scopes...,
		)
		providers = append(providers, p)
		r.providers["google"] = ProviderInfo{Name: "google", DisplayName: "Google", Enabled: true}

		slog.Info("OAuth provider enabled", "provider", "google")
	}

	// Facebook
	if r.cfg.Providers.Facebook.Enabled {
		scopes := r.cfg.Providers.Facebook.Scopes
		if len(scopes) == 0 {
			scopes = []string{"email", "public_profile"}
		}

		p := facebook.New(
			r.cfg.Providers.Facebook.ClientID,
			r.cfg.Providers.Facebook.ClientSecret,
			callbackBase+"?provider=facebook",
			scopes...,
		)
		providers = append(providers, p)
		r.providers["facebook"] = ProviderInfo{Name: "facebook", DisplayName: "Facebook", Enabled: true}

		slog.Info("OAuth provider enabled", "provider", "facebook")
	}

	// GitHub
	if r.cfg.Providers.GitHub.Enabled {
		scopes := r.cfg.Providers.GitHub.Scopes
		if len(scopes) == 0 {
			scopes = []string{"user:email", "read:user"}
		}

		p := github.New(
			r.cfg.Providers.GitHub.ClientID,
			r.cfg.Providers.GitHub.ClientSecret,
			callbackBase+"?provider=github",
			scopes...,
		)
		providers = append(providers, p)
		r.providers["github"] = ProviderInfo{Name: "github", DisplayName: "GitHub", Enabled: true}

		slog.Info("OAuth provider enabled", "provider", "github")
	}

	// Microsoft
	if r.cfg.Providers.Microsoft.Enabled {
		scopes := r.cfg.Providers.Microsoft.Scopes
		if len(scopes) == 0 {
			scopes = []string{"openid", "email", "profile"}
		}

		p := microsoftonline.New(
			r.cfg.Providers.Microsoft.ClientID,
			r.cfg.Providers.Microsoft.ClientSecret,
			callbackBase+"?provider=microsoftonline",
			scopes...,
		)
		providers = append(providers, p)
		r.providers["microsoftonline"] = ProviderInfo{Name: "microsoftonline", DisplayName: "Microsoft", Enabled: true}

		slog.Info("OAuth provider enabled", "provider", "microsoft")
	}

	// Twitter
	if r.cfg.Providers.Twitter.Enabled {
		p := twitter.New(
			r.cfg.Providers.Twitter.ClientID,
			r.cfg.Providers.Twitter.ClientSecret,
			callbackBase+"?provider=twitter",
		)
		providers = append(providers, p)
		r.providers["twitter"] = ProviderInfo{Name: "twitter", DisplayName: "X (Twitter)", Enabled: true}

		slog.Info("OAuth provider enabled", "provider", "twitter")
	}

	// Apple
	// Note: Apple Sign In requires special setup with a Services ID and private key
	// The goth apple provider expects the private key content as the "secret" parameter
	if r.cfg.Providers.Apple.Enabled {
		scopes := r.cfg.Providers.Apple.Scopes
		if len(scopes) == 0 {
			scopes = []string{"email", "name"}
		}

		// Apple provider: clientID (Services ID), secret (private key path), callbackURL, httpClient, scopes
		p := apple.New(
			r.cfg.Providers.Apple.ClientID,
			r.cfg.Providers.Apple.PrivateKeyPath,
			callbackBase+"?provider=apple",
			nil,
			scopes...,
		)
		providers = append(providers, p)
		r.providers["apple"] = ProviderInfo{Name: "apple", DisplayName: "Apple", Enabled: true}

		slog.Info("OAuth provider enabled", "provider", "apple")
	}

	if len(providers) > 0 {
		goth.UseProviders(providers...)
	}
}

// GetEnabledProviders returns a list of enabled OAuth providers.
func (r *Registry) GetEnabledProviders() []ProviderInfo {
	result := make([]ProviderInfo, 0, len(r.providers))

	for _, p := range r.providers {
		result = append(result, p)
	}

	return result
}

// IsProviderEnabled checks if a specific provider is enabled.
func (r *Registry) IsProviderEnabled(provider string) bool {
	_, ok := r.providers[provider]

	return ok
}

// GetProvider returns the goth provider by name.
func (r *Registry) GetProvider(name string) (goth.Provider, error) {
	provider, err := goth.GetProvider(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider %s: %w", name, err)
	}

	return provider, nil
}

// GetDisplayName returns the display name for a provider.
func (r *Registry) GetDisplayName(provider string) string {
	if name, ok := providerDisplayNames[provider]; ok {
		return name
	}

	return provider
}

// IsEnabled returns whether OAuth is enabled.
func (r *Registry) IsEnabled() bool {
	return r.cfg.Enabled
}

// AutoLinkByEmail returns whether auto-linking by email is enabled.
func (r *Registry) AutoLinkByEmail() bool {
	return r.cfg.AutoLinkByEmail
}
