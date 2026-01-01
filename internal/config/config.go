// Package config provides configuration management for the application.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

const (
	sourceYAML    = "yaml"
	sourceDotenv  = ".env"
	sourceSystem  = "system"
	sourceDefault = "default"
	envTrue       = "true"
)

// AppConfig is the root configuration for the entire modulith
type AppConfig struct {
	Env      string `yaml:"env" env:"ENV"`
	LogLevel string `yaml:"log_level" env:"LOG_LEVEL"`
	HTTPPort string `yaml:"http_port" env:"HTTP_PORT"`
	GRPCPort string `yaml:"grpc_port" env:"GRPC_PORT"`
	DBDSN    string `yaml:"db_dsn" env:"DB_DSN"`

	// Database connection pool settings
	DBMaxOpenConns    int    `yaml:"db_max_open_conns" env:"DB_MAX_OPEN_CONNS"`
	DBMaxIdleConns    int    `yaml:"db_max_idle_conns" env:"DB_MAX_IDLE_CONNS"`
	DBConnMaxLifetime string `yaml:"db_conn_max_lifetime" env:"DB_CONN_MAX_LIFETIME"`

	// Observability
	OTLPEndpoint string `yaml:"otlp_endpoint" env:"OTLP_ENDPOINT"`

	// CORS configuration
	CORSAllowedOrigins []string `yaml:"cors_allowed_origins" env:"CORS_ALLOWED_ORIGINS"`

	// Rate limiting
	RateLimitEnabled bool `yaml:"rate_limit_enabled" env:"RATE_LIMIT_ENABLED"`
	RateLimitRPS     int  `yaml:"rate_limit_rps" env:"RATE_LIMIT_RPS"`
	RateLimitBurst   int  `yaml:"rate_limit_burst" env:"RATE_LIMIT_BURST"`

	// Module specific configs
	Auth AuthConfig `yaml:"auth"`
}

// Load loads the configuration following this priority order (from lowest to highest):
// 1. System environment variables (already in os.Getenv) - base/default values
// 2. .env file (if exists, loaded by caller via godotenv.Load()) - overrides system ENV vars
// 3. YAML config file (if path is provided and file exists) - highest priority, overrides everything
//
// Priority: YAML > .env > system ENV vars
// systemEnvVars should be captured BEFORE godotenv.Load() is called in main()
func Load(yamlPath string, systemEnvVars map[string]string) (*AppConfig, error) {
	cfg := &AppConfig{
		Env:               "dev",
		LogLevel:          "debug",
		HTTPPort:          "8080",
		GRPCPort:          "9050",
		DBMaxOpenConns:    25,
		DBMaxIdleConns:    25,
		DBConnMaxLifetime: "5m",
		RateLimitEnabled:  false,
		RateLimitRPS:      100,
		RateLimitBurst:    50,
	}

	// Track sources for each config value
	sources := make(map[string]string)

	// Step 1: Start with system environment variables (lowest priority - base values)
	cfg.OverrideWithEnv(sources, sourceSystem)

	// Step 2: Apply .env file values (loaded by caller via godotenv.Load(), now in os.Getenv)
	// Only mark as ".env" if the value changed from system or is new (not in systemEnvVars)
	cfg.OverrideWithEnvFromDotenv(sources, systemEnvVars, sourceDotenv)

	// Step 3: Load YAML config file if exists (highest priority - overrides everything)
	if err := loadYAMLConfig(yamlPath, cfg, sources); err != nil {
		return nil, err
	}

	// Log configuration sources in a readable format
	slog.Info("Configuration sources",
		"ENV", fmt.Sprintf("%s = %s", cfg.Env, getSource(sources, "ENV")),
		"LOG_LEVEL", fmt.Sprintf("%s = %s", cfg.LogLevel, getSource(sources, "LOG_LEVEL")),
		"HTTP_PORT", fmt.Sprintf("%s = %s", cfg.HTTPPort, getSource(sources, "HTTP_PORT")),
		"GRPC_PORT", fmt.Sprintf("%s = %s", cfg.GRPCPort, getSource(sources, "GRPC_PORT")),
		"DB_DSN", fmt.Sprintf("%s = %s", cfg.DBDSN, getSource(sources, "DB_DSN")),
		"OTLP_ENDPOINT", fmt.Sprintf("%s = %s", cfg.OTLPEndpoint, getSource(sources, "OTLP_ENDPOINT")),
		"JWT_SECRET", fmt.Sprintf("[%d bytes] = %s", len(cfg.Auth.JWTSecret), getSource(sources, "JWT_SECRET")),
	)

	// Validation
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// OverrideWithEnv manually maps environment variables to the config struct.
// Only overrides if the environment variable is set and non-empty.
// Updates the sources map to track where each value came from.
// In a production app, consider using a library like cleanenv or envconfig.
func (c *AppConfig) OverrideWithEnv(sources map[string]string, sourceName string) {
	if env := os.Getenv("ENV"); env != "" {
		c.Env = env
		sources["ENV"] = sourceName
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		c.LogLevel = logLevel
		sources["LOG_LEVEL"] = sourceName
	}

	if port := os.Getenv("HTTP_PORT"); port != "" {
		c.HTTPPort = port
		sources["HTTP_PORT"] = sourceName
	}

	if port := os.Getenv("GRPC_PORT"); port != "" {
		c.GRPCPort = port
		sources["GRPC_PORT"] = sourceName
	}

	if dsn := os.Getenv("DB_DSN"); dsn != "" {
		c.DBDSN = dsn
		sources["DB_DSN"] = sourceName
	}

	if maxOpen := os.Getenv("DB_MAX_OPEN_CONNS"); maxOpen != "" {
		if val, err := strconv.Atoi(maxOpen); err == nil {
			c.DBMaxOpenConns = val
			sources["DB_MAX_OPEN_CONNS"] = sourceName
		}
	}

	if maxIdle := os.Getenv("DB_MAX_IDLE_CONNS"); maxIdle != "" {
		if val, err := strconv.Atoi(maxIdle); err == nil {
			c.DBMaxIdleConns = val
			sources["DB_MAX_IDLE_CONNS"] = sourceName
		}
	}

	if maxLifetime := os.Getenv("DB_CONN_MAX_LIFETIME"); maxLifetime != "" {
		c.DBConnMaxLifetime = maxLifetime
		sources["DB_CONN_MAX_LIFETIME"] = sourceName
	}

	if endpoint := os.Getenv("OTLP_ENDPOINT"); endpoint != "" {
		c.OTLPEndpoint = endpoint
		sources["OTLP_ENDPOINT"] = sourceName
	}

	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		c.Auth.JWTSecret = secret
		sources["JWT_SECRET"] = sourceName
	}

	// OAuth configuration
	c.overrideOAuthEnv(sources, sourceName)
}

// overrideOAuthEnv handles OAuth-specific environment variables.
//
//nolint:cyclop,funlen // OAuth configuration has many provider-specific env vars
func (c *AppConfig) overrideOAuthEnv(sources map[string]string, sourceName string) {
	if enabled := os.Getenv("OAUTH_ENABLED"); enabled == envTrue {
		c.Auth.OAuth.Enabled = true
		sources["OAUTH_ENABLED"] = sourceName
	}

	if autoLink := os.Getenv("OAUTH_AUTO_LINK_BY_EMAIL"); autoLink == envTrue {
		c.Auth.OAuth.AutoLinkByEmail = true
		sources["OAUTH_AUTO_LINK_BY_EMAIL"] = sourceName
	}

	if baseURL := os.Getenv("OAUTH_BASE_URL"); baseURL != "" {
		c.Auth.OAuth.BaseURL = baseURL
		sources["OAUTH_BASE_URL"] = sourceName
	}

	if key := os.Getenv("OAUTH_TOKEN_ENCRYPTION_KEY"); key != "" {
		c.Auth.OAuth.TokenEncryptionKey = key
		sources["OAUTH_TOKEN_ENCRYPTION_KEY"] = sourceName
	}

	// Google
	if id := os.Getenv("GOOGLE_CLIENT_ID"); id != "" {
		c.Auth.OAuth.Providers.Google.ClientID = id
		c.Auth.OAuth.Providers.Google.Enabled = true
		sources["GOOGLE_CLIENT_ID"] = sourceName
	}

	if secret := os.Getenv("GOOGLE_CLIENT_SECRET"); secret != "" {
		c.Auth.OAuth.Providers.Google.ClientSecret = secret
		sources["GOOGLE_CLIENT_SECRET"] = sourceName
	}

	// Facebook
	if id := os.Getenv("FACEBOOK_CLIENT_ID"); id != "" {
		c.Auth.OAuth.Providers.Facebook.ClientID = id
		c.Auth.OAuth.Providers.Facebook.Enabled = true
		sources["FACEBOOK_CLIENT_ID"] = sourceName
	}

	if secret := os.Getenv("FACEBOOK_CLIENT_SECRET"); secret != "" {
		c.Auth.OAuth.Providers.Facebook.ClientSecret = secret
		sources["FACEBOOK_CLIENT_SECRET"] = sourceName
	}

	// GitHub
	if id := os.Getenv("GITHUB_CLIENT_ID"); id != "" {
		c.Auth.OAuth.Providers.GitHub.ClientID = id
		c.Auth.OAuth.Providers.GitHub.Enabled = true
		sources["GITHUB_CLIENT_ID"] = sourceName
	}

	if secret := os.Getenv("GITHUB_CLIENT_SECRET"); secret != "" {
		c.Auth.OAuth.Providers.GitHub.ClientSecret = secret
		sources["GITHUB_CLIENT_SECRET"] = sourceName
	}

	// Microsoft
	if id := os.Getenv("MICROSOFT_CLIENT_ID"); id != "" {
		c.Auth.OAuth.Providers.Microsoft.ClientID = id
		c.Auth.OAuth.Providers.Microsoft.Enabled = true
		sources["MICROSOFT_CLIENT_ID"] = sourceName
	}

	if secret := os.Getenv("MICROSOFT_CLIENT_SECRET"); secret != "" {
		c.Auth.OAuth.Providers.Microsoft.ClientSecret = secret
		sources["MICROSOFT_CLIENT_SECRET"] = sourceName
	}

	// Twitter
	if id := os.Getenv("TWITTER_CLIENT_ID"); id != "" {
		c.Auth.OAuth.Providers.Twitter.ClientID = id
		c.Auth.OAuth.Providers.Twitter.Enabled = true
		sources["TWITTER_CLIENT_ID"] = sourceName
	}

	if secret := os.Getenv("TWITTER_CLIENT_SECRET"); secret != "" {
		c.Auth.OAuth.Providers.Twitter.ClientSecret = secret
		sources["TWITTER_CLIENT_SECRET"] = sourceName
	}

	// Apple (uses different env vars)
	if id := os.Getenv("APPLE_CLIENT_ID"); id != "" {
		c.Auth.OAuth.Providers.Apple.ClientID = id
		c.Auth.OAuth.Providers.Apple.Enabled = true
		sources["APPLE_CLIENT_ID"] = sourceName
	}

	if teamID := os.Getenv("APPLE_TEAM_ID"); teamID != "" {
		c.Auth.OAuth.Providers.Apple.TeamID = teamID
		sources["APPLE_TEAM_ID"] = sourceName
	}

	if keyID := os.Getenv("APPLE_KEY_ID"); keyID != "" {
		c.Auth.OAuth.Providers.Apple.KeyID = keyID
		sources["APPLE_KEY_ID"] = sourceName
	}

	if keyPath := os.Getenv("APPLE_PRIVATE_KEY_PATH"); keyPath != "" {
		c.Auth.OAuth.Providers.Apple.PrivateKeyPath = keyPath
		sources["APPLE_PRIVATE_KEY_PATH"] = sourceName
	}
}

// OverrideWithEnvFromDotenv applies .env file values, but only marks as ".env" if:
// 1. The variable was not in system ENV vars (new variable from .env), OR
// 2. The value changed from system ENV vars (overridden by .env)
func (c *AppConfig) OverrideWithEnvFromDotenv(sources, systemEnvVars map[string]string, sourceName string) {
	c.overrideEnvVar("ENV", func(val string) { c.Env = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("LOG_LEVEL", func(val string) { c.LogLevel = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("HTTP_PORT", func(val string) { c.HTTPPort = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("GRPC_PORT", func(val string) { c.GRPCPort = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("DB_DSN", func(val string) { c.DBDSN = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("DB_MAX_OPEN_CONNS", func(val string) {
		if v, err := strconv.Atoi(val); err == nil {
			c.DBMaxOpenConns = v
		}
	}, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("DB_MAX_IDLE_CONNS", func(val string) {
		if v, err := strconv.Atoi(val); err == nil {
			c.DBMaxIdleConns = v
		}
	}, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("DB_CONN_MAX_LIFETIME", func(val string) { c.DBConnMaxLifetime = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("OTLP_ENDPOINT", func(val string) { c.OTLPEndpoint = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("JWT_SECRET", func(val string) { c.Auth.JWTSecret = val }, sources, systemEnvVars, sourceName)

	// OAuth configuration from .env
	c.overrideOAuthEnvFromDotenv(sources, systemEnvVars, sourceName)
}

// overrideOAuthEnvFromDotenv handles OAuth-specific environment variables from .env file.
func (c *AppConfig) overrideOAuthEnvFromDotenv(sources, systemEnvVars map[string]string, sourceName string) {
	c.overrideEnvVar("OAUTH_ENABLED", func(val string) { c.Auth.OAuth.Enabled = val == envTrue }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("OAUTH_AUTO_LINK_BY_EMAIL", func(val string) { c.Auth.OAuth.AutoLinkByEmail = val == envTrue }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("OAUTH_BASE_URL", func(val string) { c.Auth.OAuth.BaseURL = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("OAUTH_TOKEN_ENCRYPTION_KEY", func(val string) { c.Auth.OAuth.TokenEncryptionKey = val }, sources, systemEnvVars, sourceName)

	// Provider-specific env vars
	c.overrideEnvVar("GOOGLE_CLIENT_ID", func(val string) { c.Auth.OAuth.Providers.Google.ClientID = val; c.Auth.OAuth.Providers.Google.Enabled = val != "" }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("GOOGLE_CLIENT_SECRET", func(val string) { c.Auth.OAuth.Providers.Google.ClientSecret = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("FACEBOOK_CLIENT_ID", func(val string) { c.Auth.OAuth.Providers.Facebook.ClientID = val; c.Auth.OAuth.Providers.Facebook.Enabled = val != "" }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("FACEBOOK_CLIENT_SECRET", func(val string) { c.Auth.OAuth.Providers.Facebook.ClientSecret = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("GITHUB_CLIENT_ID", func(val string) { c.Auth.OAuth.Providers.GitHub.ClientID = val; c.Auth.OAuth.Providers.GitHub.Enabled = val != "" }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("GITHUB_CLIENT_SECRET", func(val string) { c.Auth.OAuth.Providers.GitHub.ClientSecret = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("MICROSOFT_CLIENT_ID", func(val string) { c.Auth.OAuth.Providers.Microsoft.ClientID = val; c.Auth.OAuth.Providers.Microsoft.Enabled = val != "" }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("MICROSOFT_CLIENT_SECRET", func(val string) { c.Auth.OAuth.Providers.Microsoft.ClientSecret = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("TWITTER_CLIENT_ID", func(val string) { c.Auth.OAuth.Providers.Twitter.ClientID = val; c.Auth.OAuth.Providers.Twitter.Enabled = val != "" }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("TWITTER_CLIENT_SECRET", func(val string) { c.Auth.OAuth.Providers.Twitter.ClientSecret = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("APPLE_CLIENT_ID", func(val string) { c.Auth.OAuth.Providers.Apple.ClientID = val; c.Auth.OAuth.Providers.Apple.Enabled = val != "" }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("APPLE_TEAM_ID", func(val string) { c.Auth.OAuth.Providers.Apple.TeamID = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("APPLE_KEY_ID", func(val string) { c.Auth.OAuth.Providers.Apple.KeyID = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("APPLE_PRIVATE_KEY_PATH", func(val string) { c.Auth.OAuth.Providers.Apple.PrivateKeyPath = val }, sources, systemEnvVars, sourceName)
}

// overrideEnvVar is a helper to override an env var and track its source
func (c *AppConfig) overrideEnvVar(key string, setter func(string), sources, systemEnvVars map[string]string, sourceName string) {
	if val := os.Getenv(key); val != "" {
		setter(val)
		// Only mark as sourceName if it's new or changed from system
		if sysVal, wasInSystem := systemEnvVars[key]; !wasInSystem || sysVal != val {
			sources[key] = sourceName
		}
	}
}

// Validate ensures the configuration is semantically correct.
func (c *AppConfig) Validate() error {
	if c.DBDSN == "" && c.Env == "prod" {
		return fmt.Errorf("DB_DSN is required in production")
	}

	if c.Auth.JWTSecret == "" && c.Env == "prod" {
		return fmt.Errorf("JWT_SECRET is required in production")
	}

	// Validate JWT secret length (HS256 requires at least 32 bytes)
	if c.Auth.JWTSecret != "" && len(c.Auth.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 bytes (256 bits) for HS256 algorithm, got %d bytes", len(c.Auth.JWTSecret))
	}

	// Validate OAuth configuration
	if err := c.validateOAuthConfig(); err != nil {
		return err
	}

	return nil
}

// validateOAuthConfig validates OAuth-specific configuration.
//
//nolint:cyclop // Validation requires checking each provider individually
func (c *AppConfig) validateOAuthConfig() error {
	oauth := c.Auth.OAuth

	if !oauth.Enabled {
		return nil // OAuth is disabled, no validation needed
	}

	// BaseURL is required when OAuth is enabled
	if oauth.BaseURL == "" {
		return fmt.Errorf("OAUTH_BASE_URL is required when OAuth is enabled")
	}

	// Token encryption key must be 32 bytes for AES-256
	if oauth.TokenEncryptionKey != "" && len(oauth.TokenEncryptionKey) != 32 {
		return fmt.Errorf("OAUTH_TOKEN_ENCRYPTION_KEY must be exactly 32 bytes for AES-256, got %d bytes", len(oauth.TokenEncryptionKey))
	}

	// Validate each enabled provider
	if oauth.Providers.Google.Enabled && oauth.Providers.Google.ClientSecret == "" {
		return fmt.Errorf("GOOGLE_CLIENT_SECRET is required when Google OAuth is enabled")
	}

	if oauth.Providers.Facebook.Enabled && oauth.Providers.Facebook.ClientSecret == "" {
		return fmt.Errorf("FACEBOOK_CLIENT_SECRET is required when Facebook OAuth is enabled")
	}

	if oauth.Providers.GitHub.Enabled && oauth.Providers.GitHub.ClientSecret == "" {
		return fmt.Errorf("GITHUB_CLIENT_SECRET is required when GitHub OAuth is enabled")
	}

	if oauth.Providers.Microsoft.Enabled && oauth.Providers.Microsoft.ClientSecret == "" {
		return fmt.Errorf("MICROSOFT_CLIENT_SECRET is required when Microsoft OAuth is enabled")
	}

	if oauth.Providers.Twitter.Enabled && oauth.Providers.Twitter.ClientSecret == "" {
		return fmt.Errorf("TWITTER_CLIENT_SECRET is required when Twitter OAuth is enabled")
	}

	// Apple requires additional configuration
	if oauth.Providers.Apple.Enabled {
		if oauth.Providers.Apple.TeamID == "" {
			return fmt.Errorf("APPLE_TEAM_ID is required when Apple OAuth is enabled")
		}

		if oauth.Providers.Apple.KeyID == "" {
			return fmt.Errorf("APPLE_KEY_ID is required when Apple OAuth is enabled")
		}

		if oauth.Providers.Apple.PrivateKeyPath == "" {
			return fmt.Errorf("APPLE_PRIVATE_KEY_PATH is required when Apple OAuth is enabled")
		}
	}

	return nil
}

// loadYAMLConfig loads YAML configuration from file if it exists
func loadYAMLConfig(yamlPath string, cfg *AppConfig, sources map[string]string) error {
	if yamlPath == "" {
		return nil
	}

	cleanPath := filepath.Clean(yamlPath)
	if _, err := os.Stat(cleanPath); err != nil {
		return nil // File doesn't exist, that's OK
	}

	// YAML file exists, load it into temporary struct first
	f, err := os.Open(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}

	var yamlOnly AppConfig
	if err := yaml.NewDecoder(f).Decode(&yamlOnly); err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to decode yaml config: %w", err)
	}

	if err := f.Close(); err != nil {
		slog.Error("failed to close config file", "error", err)
	}

	// Now apply YAML values to cfg and mark fields that exist in YAML as "yaml"
	// If a field is set in YAML (non-empty), it comes from YAML (highest priority)
	// This overwrites any previous source marking
	applyYAMLConfig(cfg, &yamlOnly, sources)

	return nil
}

// applyYAMLConfig applies YAML configuration values to the config struct
func applyYAMLConfig(cfg, yamlOnly *AppConfig, sources map[string]string) {
	if yamlOnly.Env != "" {
		cfg.Env = yamlOnly.Env
		sources["ENV"] = sourceYAML
	}

	if yamlOnly.LogLevel != "" {
		cfg.LogLevel = yamlOnly.LogLevel
		sources["LOG_LEVEL"] = sourceYAML
	}

	if yamlOnly.HTTPPort != "" {
		cfg.HTTPPort = yamlOnly.HTTPPort
		sources["HTTP_PORT"] = sourceYAML
	}

	if yamlOnly.GRPCPort != "" {
		cfg.GRPCPort = yamlOnly.GRPCPort
		sources["GRPC_PORT"] = sourceYAML
	}

	if yamlOnly.DBDSN != "" {
		cfg.DBDSN = yamlOnly.DBDSN
		sources["DB_DSN"] = sourceYAML
	}

	if yamlOnly.DBMaxOpenConns > 0 {
		cfg.DBMaxOpenConns = yamlOnly.DBMaxOpenConns
		sources["DB_MAX_OPEN_CONNS"] = sourceYAML
	}

	if yamlOnly.DBMaxIdleConns > 0 {
		cfg.DBMaxIdleConns = yamlOnly.DBMaxIdleConns
		sources["DB_MAX_IDLE_CONNS"] = sourceYAML
	}

	if yamlOnly.DBConnMaxLifetime != "" {
		cfg.DBConnMaxLifetime = yamlOnly.DBConnMaxLifetime
		sources["DB_CONN_MAX_LIFETIME"] = sourceYAML
	}

	if yamlOnly.OTLPEndpoint != "" {
		cfg.OTLPEndpoint = yamlOnly.OTLPEndpoint
		sources["OTLP_ENDPOINT"] = sourceYAML
	}

	if yamlOnly.Auth.JWTSecret != "" {
		cfg.Auth.JWTSecret = yamlOnly.Auth.JWTSecret
		sources["JWT_SECRET"] = sourceYAML
	}

	// Apply OAuth configuration from YAML
	applyYAMLOAuthConfig(cfg, yamlOnly, sources)
}

// applyYAMLOAuthConfig applies OAuth configuration from YAML.
func applyYAMLOAuthConfig(cfg, yamlOnly *AppConfig, sources map[string]string) {
	oauth := yamlOnly.Auth.OAuth

	if oauth.Enabled {
		cfg.Auth.OAuth.Enabled = true
		sources["OAUTH_ENABLED"] = sourceYAML
	}

	if oauth.AutoLinkByEmail {
		cfg.Auth.OAuth.AutoLinkByEmail = true
		sources["OAUTH_AUTO_LINK_BY_EMAIL"] = sourceYAML
	}

	if oauth.BaseURL != "" {
		cfg.Auth.OAuth.BaseURL = oauth.BaseURL
		sources["OAUTH_BASE_URL"] = sourceYAML
	}

	if oauth.TokenEncryptionKey != "" {
		cfg.Auth.OAuth.TokenEncryptionKey = oauth.TokenEncryptionKey
		sources["OAUTH_TOKEN_ENCRYPTION_KEY"] = sourceYAML
	}

	// Apply provider configurations
	applyYAMLOAuthProviders(cfg, yamlOnly, sources)
}

// applyYAMLOAuthProviders applies OAuth provider configurations from YAML.
func applyYAMLOAuthProviders(cfg, yamlOnly *AppConfig, sources map[string]string) {
	providers := yamlOnly.Auth.OAuth.Providers

	// Google
	if providers.Google.ClientID != "" {
		cfg.Auth.OAuth.Providers.Google = providers.Google
		sources["GOOGLE_CLIENT_ID"] = sourceYAML
	}

	// Facebook
	if providers.Facebook.ClientID != "" {
		cfg.Auth.OAuth.Providers.Facebook = providers.Facebook
		sources["FACEBOOK_CLIENT_ID"] = sourceYAML
	}

	// GitHub
	if providers.GitHub.ClientID != "" {
		cfg.Auth.OAuth.Providers.GitHub = providers.GitHub
		sources["GITHUB_CLIENT_ID"] = sourceYAML
	}

	// Microsoft
	if providers.Microsoft.ClientID != "" {
		cfg.Auth.OAuth.Providers.Microsoft = providers.Microsoft
		sources["MICROSOFT_CLIENT_ID"] = sourceYAML
	}

	// Twitter
	if providers.Twitter.ClientID != "" {
		cfg.Auth.OAuth.Providers.Twitter = providers.Twitter
		sources["TWITTER_CLIENT_ID"] = sourceYAML
	}

	// Apple
	if providers.Apple.ClientID != "" {
		cfg.Auth.OAuth.Providers.Apple = providers.Apple
		sources["APPLE_CLIENT_ID"] = sourceYAML
	}
}

// getSource returns the source for a config key, or default if not found
func getSource(sources map[string]string, key string) string {
	if src, ok := sources[key]; ok {
		return src
	}

	return sourceDefault
}
