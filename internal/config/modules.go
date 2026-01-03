package config

// AuthConfig holds the Auth module settings.
// This is defined here to avoid import cycles between config and modules.
type AuthConfig struct {
	JWTSecret string      `yaml:"jwt_secret"`
	OAuth     OAuthConfig `yaml:"oauth"`
}

// OAuthConfig holds OAuth provider settings.
type OAuthConfig struct {
	Enabled         bool   `yaml:"enabled"`
	AutoLinkByEmail bool   `yaml:"auto_link_by_email"`
	BaseURL         string `yaml:"base_url"` // e.g., "https://api.example.com"
	// Encryption key for storing OAuth tokens (AES-256, must be 32 bytes)
	TokenEncryptionKey string `yaml:"token_encryption_key"`

	Providers OAuthProviders `yaml:"providers"`
}

// OAuthProviders holds configuration for all supported OAuth providers.
type OAuthProviders struct {
	Google    OAuthProviderConfig `yaml:"google"`
	Facebook  OAuthProviderConfig `yaml:"facebook"`
	GitHub    OAuthProviderConfig `yaml:"github"`
	Apple     AppleOAuthConfig    `yaml:"apple"`
	Microsoft OAuthProviderConfig `yaml:"microsoft"`
	Twitter   OAuthProviderConfig `yaml:"twitter"`
}

// OAuthProviderConfig holds common OAuth provider settings.
type OAuthProviderConfig struct {
	Enabled      bool     `yaml:"enabled"`
	ClientID     string   `yaml:"client_id"`
	ClientSecret string   `yaml:"client_secret"`
	Scopes       []string `yaml:"scopes"`
}

// AppleOAuthConfig holds Apple-specific OAuth settings.
type AppleOAuthConfig struct {
	Enabled        bool     `yaml:"enabled"`
	ClientID       string   `yaml:"client_id"` // Services ID
	TeamID         string   `yaml:"team_id"`
	KeyID          string   `yaml:"key_id"`
	PrivateKeyPath string   `yaml:"private_key_path"` // Path to .p8 file
	Scopes         []string `yaml:"scopes"`
}
