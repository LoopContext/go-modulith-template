// Package config provides configuration management for the application.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

const (
	sourceYAML    = "yaml"
	sourceDotenv  = ".env"
	sourceSystem  = "system"
	sourceDefault = "default"
	envTrue       = "true"
	envProd       = "prod"
)

// AppConfig is the root configuration for the entire modulith
type AppConfig struct {
	AppName  string `yaml:"app_name" env:"APP_NAME"`
	Env      string `yaml:"env" env:"ENV"`
	LogLevel string `yaml:"log_level" env:"LOG_LEVEL"`
	HTTPPort string `yaml:"http_port" env:"HTTP_PORT"`
	GRPCPort string `yaml:"grpc_port" env:"GRPC_PORT"`
	DBDSN    string `yaml:"db_dsn" env:"DB_DSN"`

	// Database connection pool settings
	DBMaxOpenConns    int    `yaml:"db_max_open_conns" env:"DB_MAX_OPEN_CONNS"`
	DBMaxIdleConns    int    `yaml:"db_max_idle_conns" env:"DB_MAX_IDLE_CONNS"`
	DBConnMaxLifetime string `yaml:"db_conn_max_lifetime" env:"DB_CONN_MAX_LIFETIME"`
	DBConnectTimeout  string `yaml:"db_connect_timeout" env:"DB_CONNECT_TIMEOUT"` // Initial connection timeout (e.g., "10s")

	// Observability
	OTLPEndpoint string `yaml:"otlp_endpoint" env:"OTLP_ENDPOINT"`
	ServiceName  string `yaml:"service_name" env:"SERVICE_NAME"` // Service name for OpenTelemetry resource

	// CORS configuration
	CORSAllowedOrigins []string `yaml:"cors_allowed_origins" env:"CORS_ALLOWED_ORIGINS"`

	// Rate limiting
	RateLimitEnabled bool `yaml:"rate_limit_enabled" env:"RATE_LIMIT_ENABLED"`
	RateLimitRPS     int  `yaml:"rate_limit_rps" env:"RATE_LIMIT_RPS"`
	RateLimitBurst   int  `yaml:"rate_limit_burst" env:"RATE_LIMIT_BURST"`

	// Redis/Valkey
	ValkeyAddr         string `yaml:"valkey_addr" env:"VALKEY_ADDR"`
	ValkeyPassword     string `yaml:"valkey_password" env:"VALKEY_PASSWORD"`
	ValkeyDB           int    `yaml:"valkey_db" env:"VALKEY_DB"`
	ValkeyPoolSize     int    `yaml:"valkey_pool_size" env:"VALKEY_POOL_SIZE"`
	ValkeyMinIdleConns int    `yaml:"valkey_min_idle_conns" env:"VALKEY_MIN_IDLE_CONNS"`

	// Timeouts
	ReadTimeout     string `yaml:"read_timeout" env:"READ_TIMEOUT"`         // HTTP server read timeout (e.g., "5s")
	WriteTimeout    string `yaml:"write_timeout" env:"WRITE_TIMEOUT"`       // HTTP server write timeout (e.g., "10s")
	RequestTimeout  string `yaml:"request_timeout" env:"REQUEST_TIMEOUT"`   // Request handler timeout (e.g., "30s")
	ShutdownTimeout string `yaml:"shutdown_timeout" env:"SHUTDOWN_TIMEOUT"` // Graceful shutdown timeout (e.g., "30s")

	// Internationalization
	DefaultLocale string `yaml:"default_locale" env:"DEFAULT_LOCALE"` // Default locale for i18n (e.g., "en", "es")

	// Swagger/OpenAPI documentation
	SwaggerAPITitle string `yaml:"swagger_api_title" env:"SWAGGER_API_TITLE"` // API title shown in Swagger UI

	// Outbox configuration
	OutboxPollInterval string `yaml:"outbox_poll_interval" env:"OUTBOX_POLL_INTERVAL"` // e.g. "5s", "100ms"

	// Module specific configs
	Auth  AuthConfig  `yaml:"auth"`
	KYC   KycConfig   `yaml:"kyc"`
	Feeds FeedsConfig `yaml:"feeds"`
	Seeds SeedConfig  `yaml:"seeds"`
	E2E   E2EConfig   `yaml:"e2e"`
}

// FeedsConfig contains configuration for the data feeds module.
type FeedsConfig struct {
	TheOddsAPIKey  string `yaml:"the_odds_api_key" env:"THE_ODDS_API_KEY"`
	APIFootballKey string `yaml:"api_football_key" env:"API_FOOTBALL_KEY"`
}

// SeedConfig contains configuration for seeding data.
type SeedConfig struct {
	Users []SeedUser `yaml:"users"`
}

// SeedUser represents a user to be seeded.
type SeedUser struct {
	Name           string  `yaml:"name"`
	Email          string  `yaml:"email"`
	Role           string  `yaml:"role"`
	Phone          string  `yaml:"phone"`
	InitialBalance float64 `yaml:"initial_balance"`
	Currency       string  `yaml:"currency"`
}

// PlatformEmail returns the email of the platform-role user from the seed config.
// Falls back to "system@opos.dev" if no platform user is configured.
func (s *SeedConfig) PlatformEmail() string {
	for _, u := range s.Users {
		if u.Role == "platform" {
			return u.Email
		}
	}

	return "system@opos.dev"
}

// E2EConfig contains configuration for E2E tests.
type E2EConfig struct {
	GRPCAddr     string `yaml:"grpc_addr" env:"E2E_GRPC_ADDR"`
	CreatorEmail string `yaml:"creator_email" env:"E2E_CREATOR_EMAIL"`
	AdminEmail   string `yaml:"admin_email" env:"E2E_ADMIN_EMAIL"`
}

// Load loads the configuration following this priority order (from lowest to highest):
// 1. System environment variables (already in os.Getenv) - base/default values
// 2. .env file (if exists, loaded by caller via godotenv.Load()) - overrides system ENV vars
// 3. YAML config file (if path is provided and file exists) - highest priority, overrides everything
//
// Priority: YAML > .env > system ENV vars
// systemEnvVars should be captured BEFORE godotenv.Load() is called in main()
//
//nolint:funlen // Configuration loading requires many sequential steps
func Load(yamlPath string, systemEnvVars map[string]string) (*AppConfig, error) {
	cfg := &AppConfig{
		AppName:            "Modulith Project",
		Env:                "dev",
		LogLevel:           "debug",
		HTTPPort:           "8080",
		GRPCPort:           "9050",
		ServiceName:        "modulith-server",
		DBMaxOpenConns:     25,
		DBMaxIdleConns:     25,
		DBConnMaxLifetime:  "5m",
		DBConnectTimeout:   "10s",
		RateLimitEnabled:   false,
		RateLimitRPS:       100,
		RateLimitBurst:     50,
		ReadTimeout:        "5s",
		WriteTimeout:       "10s",
		RequestTimeout:     "30s",
		ShutdownTimeout:    "30s",
		DefaultLocale:      "en",
		SwaggerAPITitle:    "Modulith API",
		ValkeyAddr:         "localhost:6379",
		ValkeyPassword:     "",
		ValkeyDB:           0,
		ValkeyPoolSize:     10,
		ValkeyMinIdleConns: 2,
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

	// Step 3.5: Load Seeds YAML config file based on env
	seedPath := fmt.Sprintf("configs/seeds/%s.yaml", cfg.Env)
	if _, err := os.Stat(seedPath); err == nil {
		if err := loadYAMLConfig(seedPath, cfg, sources); err != nil {
			return nil, fmt.Errorf("failed to load seed config: %w", err)
		}
	} else if cfg.Env == "dev" {
		// Fallback to development.yaml if env is dev and file missing?
		// Or just ignore if missing.
		slog.Info("Seed config not found, skipping", "path", seedPath)
	}

	// Step 4: Apply standard PORT env var (12-factor app compliance)
	// PORT takes precedence over HTTP_PORT for compatibility with Heroku, Cloud Run, Railway, etc.
	// Priority: PORT > HTTP_PORT > default
	if port := os.Getenv("PORT"); port != "" {
		cfg.HTTPPort = port
		sources["HTTP_PORT"] = "PORT" // Track that it came from PORT
	}

	// Log configuration sources in a readable format
	slog.Info("Configuration sources",
		"APP_NAME", formatConfigSourceValue(cfg.AppName, getSource(sources, "APP_NAME")),
		"ENV", formatConfigSourceValue(cfg.Env, getSource(sources, "ENV")),
		"LOG_LEVEL", formatConfigSourceValue(cfg.LogLevel, getSource(sources, "LOG_LEVEL")),
		"HTTP_PORT", formatConfigSourceValue(cfg.HTTPPort, getSource(sources, "HTTP_PORT")),
		"GRPC_PORT", formatConfigSourceValue(cfg.GRPCPort, getSource(sources, "GRPC_PORT")),
		"DB_DSN", formatConfigSecretValue(cfg.DBDSN, getSource(sources, "DB_DSN"), "chars"),
		"OTLP_ENDPOINT", formatConfigSourceValue(cfg.OTLPEndpoint, getSource(sources, "OTLP_ENDPOINT")),
		"SERVICE_NAME", formatConfigSourceValue(cfg.ServiceName, getSource(sources, "SERVICE_NAME")),
		"JWT_PRIVATE_KEY", formatConfigSecretValue(cfg.Auth.JWTPrivateKeyPEM, getSource(sources, "JWT_PRIVATE_KEY"), "bytes"),
		"JWT_PUBLIC_KEY", formatConfigSecretValue(cfg.Auth.JWTPublicKeyPEM, getSource(sources, "JWT_PUBLIC_KEY"), "bytes"),
		"KYC_ENFORCEMENT_ENABLED", formatConfigBoolValue(cfg.KYC.EnforcementEnabled, getSource(sources, "KYC_ENFORCEMENT_ENABLED")),
		"RATE_LIMIT_ENABLED", formatConfigBoolValue(cfg.RateLimitEnabled, getSource(sources, "RATE_LIMIT_ENABLED")),
		"CORS_ALLOWED_ORIGINS", formatConfigSourceValue(strings.Join(cfg.CORSAllowedOrigins, ","), getSource(sources, "CORS_ALLOWED_ORIGINS")),
		"SWAGGER_API_TITLE", formatConfigSourceValue(cfg.SwaggerAPITitle, getSource(sources, "SWAGGER_API_TITLE")),
		"THE_ODDS_API_KEY", formatConfigSecretValue(cfg.Feeds.TheOddsAPIKey, getSource(sources, "THE_ODDS_API_KEY"), "chars"),
		"API_FOOTBALL_KEY", formatConfigSecretValue(cfg.Feeds.APIFootballKey, getSource(sources, "API_FOOTBALL_KEY"), "chars"),
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
//
//nolint:cyclop,funlen,gocognit,gocyclo // Configuration parsing requires many sequential environment variable checks
//nolint:gocyclo // Complexity is needed for config overrides
func (c *AppConfig) OverrideWithEnv(sources map[string]string, sourceName string) {
	if appName := os.Getenv("APP_NAME"); appName != "" {
		c.AppName = appName
		sources["APP_NAME"] = sourceName
	}

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

	if connectTimeout := os.Getenv("DB_CONNECT_TIMEOUT"); connectTimeout != "" {
		c.DBConnectTimeout = connectTimeout
		sources["DB_CONNECT_TIMEOUT"] = sourceName
	}

	if endpoint := os.Getenv("OTLP_ENDPOINT"); endpoint != "" {
		c.OTLPEndpoint = endpoint
		sources["OTLP_ENDPOINT"] = sourceName
	}

	if serviceName := os.Getenv("SERVICE_NAME"); serviceName != "" {
		c.ServiceName = serviceName
		sources["SERVICE_NAME"] = sourceName
	}

	if key := os.Getenv("JWT_PRIVATE_KEY"); key != "" {
		c.Auth.JWTPrivateKeyPEM = key
		sources["JWT_PRIVATE_KEY"] = sourceName
	}

	if key := os.Getenv("JWT_PUBLIC_KEY"); key != "" {
		c.Auth.JWTPublicKeyPEM = key
		sources["JWT_PUBLIC_KEY"] = sourceName
	}

	if timeout := os.Getenv("READ_TIMEOUT"); timeout != "" {
		c.ReadTimeout = timeout
		sources["READ_TIMEOUT"] = sourceName
	}

	if timeout := os.Getenv("WRITE_TIMEOUT"); timeout != "" {
		c.WriteTimeout = timeout
		sources["WRITE_TIMEOUT"] = sourceName
	}

	if timeout := os.Getenv("REQUEST_TIMEOUT"); timeout != "" {
		c.RequestTimeout = timeout
		sources["REQUEST_TIMEOUT"] = sourceName
	}

	if timeout := os.Getenv("SHUTDOWN_TIMEOUT"); timeout != "" {
		c.ShutdownTimeout = timeout
		sources["SHUTDOWN_TIMEOUT"] = sourceName
	}

	// CORS configuration
	if corsOrigins := os.Getenv("CORS_ALLOWED_ORIGINS"); corsOrigins != "" {
		// Parse comma-separated list
		origins := parseCommaSeparated(corsOrigins)
		if len(origins) > 0 {
			c.CORSAllowedOrigins = origins
			sources["CORS_ALLOWED_ORIGINS"] = sourceName
		}
	}

	// Rate limiting
	if enabled := os.Getenv("RATE_LIMIT_ENABLED"); enabled == envTrue {
		c.RateLimitEnabled = true
		sources["RATE_LIMIT_ENABLED"] = sourceName
	}

	if rps := os.Getenv("RATE_LIMIT_RPS"); rps != "" {
		if val, err := strconv.Atoi(rps); err == nil {
			c.RateLimitRPS = val
			sources["RATE_LIMIT_RPS"] = sourceName
		}
	}

	if burst := os.Getenv("RATE_LIMIT_BURST"); burst != "" {
		if val, err := strconv.Atoi(burst); err == nil {
			c.RateLimitBurst = val
			sources["RATE_LIMIT_BURST"] = sourceName
		}
	}

	// Internationalization
	if locale := os.Getenv("DEFAULT_LOCALE"); locale != "" {
		c.DefaultLocale = locale
		sources["DEFAULT_LOCALE"] = sourceName
	}

	// Swagger/OpenAPI
	if title := os.Getenv("SWAGGER_API_TITLE"); title != "" {
		c.SwaggerAPITitle = title
		sources["SWAGGER_API_TITLE"] = sourceName
	}

	// OAuth configuration
	c.overrideOAuthEnv(sources, sourceName)

	// KYC configuration
	if enabled := os.Getenv("KYC_ENFORCEMENT_ENABLED"); enabled == envTrue {
		c.KYC.EnforcementEnabled = true
		sources["KYC_ENFORCEMENT_ENABLED"] = sourceName
	}

	if apiKey := os.Getenv("THE_ODDS_API_KEY"); apiKey != "" {
		c.Feeds.TheOddsAPIKey = apiKey
		sources["THE_ODDS_API_KEY"] = sourceName
	}

	if apiKey := os.Getenv("API_FOOTBALL_KEY"); apiKey != "" {
		c.Feeds.APIFootballKey = apiKey
		sources["API_FOOTBALL_KEY"] = sourceName
	}
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
	c.overrideEnvVar("APP_NAME", func(val string) { c.AppName = val }, sources, systemEnvVars, sourceName)
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
	c.overrideEnvVar("DB_CONNECT_TIMEOUT", func(val string) { c.DBConnectTimeout = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("OTLP_ENDPOINT", func(val string) { c.OTLPEndpoint = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("SERVICE_NAME", func(val string) { c.ServiceName = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("JWT_PRIVATE_KEY", func(val string) { c.Auth.JWTPrivateKeyPEM = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("JWT_PUBLIC_KEY", func(val string) { c.Auth.JWTPublicKeyPEM = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("READ_TIMEOUT", func(val string) { c.ReadTimeout = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("WRITE_TIMEOUT", func(val string) { c.WriteTimeout = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("REQUEST_TIMEOUT", func(val string) { c.RequestTimeout = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("SHUTDOWN_TIMEOUT", func(val string) { c.ShutdownTimeout = val }, sources, systemEnvVars, sourceName)

	// CORS configuration from .env
	c.overrideEnvVar("CORS_ALLOWED_ORIGINS", func(val string) {
		origins := parseCommaSeparated(val)
		if len(origins) > 0 {
			c.CORSAllowedOrigins = origins
		}
	}, sources, systemEnvVars, sourceName)

	// Rate limiting from .env
	c.overrideEnvVar("RATE_LIMIT_ENABLED", func(val string) { c.RateLimitEnabled = val == envTrue }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("RATE_LIMIT_RPS", func(val string) {
		if v, err := strconv.Atoi(val); err == nil {
			c.RateLimitRPS = v
		}
	}, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("RATE_LIMIT_BURST", func(val string) {
		if v, err := strconv.Atoi(val); err == nil {
			c.RateLimitBurst = v
		}
	}, sources, systemEnvVars, sourceName)

	// Internationalization from .env
	c.overrideEnvVar("DEFAULT_LOCALE", func(val string) { c.DefaultLocale = val }, sources, systemEnvVars, sourceName)

	// Swagger/OpenAPI from .env
	c.overrideEnvVar("SWAGGER_API_TITLE", func(val string) { c.SwaggerAPITitle = val }, sources, systemEnvVars, sourceName)

	// KYC from .env
	c.overrideEnvVar("KYC_ENFORCEMENT_ENABLED", func(val string) { c.KYC.EnforcementEnabled = val == envTrue }, sources, systemEnvVars, sourceName)

	// Feeds from .env
	c.overrideEnvVar("THE_ODDS_API_KEY", func(val string) { c.Feeds.TheOddsAPIKey = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("API_FOOTBALL_KEY", func(val string) { c.Feeds.APIFootballKey = val }, sources, systemEnvVars, sourceName)

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
	c.overrideEnvVar("GOOGLE_CLIENT_ID", func(val string) {
		c.Auth.OAuth.Providers.Google.ClientID = val
		c.Auth.OAuth.Providers.Google.Enabled = val != ""
	}, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("GOOGLE_CLIENT_SECRET", func(val string) { c.Auth.OAuth.Providers.Google.ClientSecret = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("FACEBOOK_CLIENT_ID", func(val string) {
		c.Auth.OAuth.Providers.Facebook.ClientID = val
		c.Auth.OAuth.Providers.Facebook.Enabled = val != ""
	}, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("FACEBOOK_CLIENT_SECRET", func(val string) { c.Auth.OAuth.Providers.Facebook.ClientSecret = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("GITHUB_CLIENT_ID", func(val string) {
		c.Auth.OAuth.Providers.GitHub.ClientID = val
		c.Auth.OAuth.Providers.GitHub.Enabled = val != ""
	}, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("GITHUB_CLIENT_SECRET", func(val string) { c.Auth.OAuth.Providers.GitHub.ClientSecret = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("MICROSOFT_CLIENT_ID", func(val string) {
		c.Auth.OAuth.Providers.Microsoft.ClientID = val
		c.Auth.OAuth.Providers.Microsoft.Enabled = val != ""
	}, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("MICROSOFT_CLIENT_SECRET", func(val string) { c.Auth.OAuth.Providers.Microsoft.ClientSecret = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("TWITTER_CLIENT_ID", func(val string) {
		c.Auth.OAuth.Providers.Twitter.ClientID = val
		c.Auth.OAuth.Providers.Twitter.Enabled = val != ""
	}, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("TWITTER_CLIENT_SECRET", func(val string) { c.Auth.OAuth.Providers.Twitter.ClientSecret = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("APPLE_CLIENT_ID", func(val string) {
		c.Auth.OAuth.Providers.Apple.ClientID = val
		c.Auth.OAuth.Providers.Apple.Enabled = val != ""
	}, sources, systemEnvVars, sourceName)
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
	if c.Env == envProd {
		if c.DBDSN == "" {
			return fmt.Errorf("DB_DSN is required in production")
		}

		if c.Auth.JWTPublicKeyPEM == "" {
			return fmt.Errorf("JWT_PUBLIC_KEY is required in production (RS256 verification)")
		}
	}

	if err := c.validateJWT(); err != nil {
		return err
	}

	if err := c.validateCORS(); err != nil {
		return err
	}

	return c.validateOAuthConfig()
}

func (c *AppConfig) validateJWT() error {
	// RS256: public key required for verification; private key required only for auth/admin token issuance
	if c.Auth.JWTPublicKeyPEM != "" {
		// Basic PEM structure check
		if len(c.Auth.JWTPublicKeyPEM) < 100 {
			return fmt.Errorf("JWT_PUBLIC_KEY appears invalid (PEM too short)")
		}
	}

	if c.Auth.JWTPrivateKeyPEM != "" && len(c.Auth.JWTPrivateKeyPEM) < 100 {
		return fmt.Errorf("JWT_PRIVATE_KEY appears invalid (PEM too short)")
	}

	return nil
}

func (c *AppConfig) validateCORS() error {
	// Enforce strict CORS in production
	if c.Env == envProd {
		for _, origin := range c.CORSAllowedOrigins {
			if origin == "*" {
				return fmt.Errorf("wildcard CORS origin '*' is not allowed in production")
			}
		}
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
//
//nolint:cyclop,funlen,gocyclo,gocognit // Configuration parsing requires many sequential YAML value checks
func applyYAMLConfig(cfg, yamlOnly *AppConfig, sources map[string]string) {
	if len(yamlOnly.Seeds.Users) > 0 {
		cfg.Seeds = yamlOnly.Seeds
		sources["SEEDS"] = sourceYAML
	}

	if yamlOnly.E2E.GRPCAddr != "" {
		cfg.E2E = yamlOnly.E2E
		sources["E2E"] = sourceYAML
	}

	if yamlOnly.AppName != "" {
		cfg.AppName = yamlOnly.AppName
		sources["APP_NAME"] = sourceYAML
	}

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

	if yamlOnly.ValkeyAddr != "" {
		cfg.ValkeyAddr = yamlOnly.ValkeyAddr
		sources["VALKEY_ADDR"] = sourceYAML
	}

	if yamlOnly.ValkeyPassword != "" {
		cfg.ValkeyPassword = yamlOnly.ValkeyPassword
		sources["VALKEY_PASSWORD"] = sourceYAML
	}

	if yamlOnly.ValkeyDB >= 0 {
		cfg.ValkeyDB = yamlOnly.ValkeyDB
		sources["VALKEY_DB"] = sourceYAML
	}

	if yamlOnly.ValkeyPoolSize > 0 {
		cfg.ValkeyPoolSize = yamlOnly.ValkeyPoolSize
		sources["VALKEY_POOL_SIZE"] = sourceYAML
	}

	if yamlOnly.ValkeyMinIdleConns > 0 {
		cfg.ValkeyMinIdleConns = yamlOnly.ValkeyMinIdleConns
		sources["VALKEY_MIN_IDLE_CONNS"] = sourceYAML
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

	if yamlOnly.DBConnectTimeout != "" {
		cfg.DBConnectTimeout = yamlOnly.DBConnectTimeout
		sources["DB_CONNECT_TIMEOUT"] = sourceYAML
	}

	if yamlOnly.OTLPEndpoint != "" {
		cfg.OTLPEndpoint = yamlOnly.OTLPEndpoint
		sources["OTLP_ENDPOINT"] = sourceYAML
	}

	if yamlOnly.ServiceName != "" {
		cfg.ServiceName = yamlOnly.ServiceName
		sources["SERVICE_NAME"] = sourceYAML
	}

	if yamlOnly.Auth.JWTPrivateKeyPEM != "" {
		cfg.Auth.JWTPrivateKeyPEM = yamlOnly.Auth.JWTPrivateKeyPEM
		sources["JWT_PRIVATE_KEY"] = sourceYAML
	}

	if yamlOnly.Auth.JWTPublicKeyPEM != "" {
		cfg.Auth.JWTPublicKeyPEM = yamlOnly.Auth.JWTPublicKeyPEM
		sources["JWT_PUBLIC_KEY"] = sourceYAML
	}

	if yamlOnly.ReadTimeout != "" {
		cfg.ReadTimeout = yamlOnly.ReadTimeout
		sources["READ_TIMEOUT"] = sourceYAML
	}

	if yamlOnly.WriteTimeout != "" {
		cfg.WriteTimeout = yamlOnly.WriteTimeout
		sources["WRITE_TIMEOUT"] = sourceYAML
	}

	if yamlOnly.RequestTimeout != "" {
		cfg.RequestTimeout = yamlOnly.RequestTimeout
		sources["REQUEST_TIMEOUT"] = sourceYAML
	}

	if yamlOnly.ShutdownTimeout != "" {
		cfg.ShutdownTimeout = yamlOnly.ShutdownTimeout
		sources["SHUTDOWN_TIMEOUT"] = sourceYAML
	}

	// CORS configuration from YAML
	if len(yamlOnly.CORSAllowedOrigins) > 0 {
		cfg.CORSAllowedOrigins = yamlOnly.CORSAllowedOrigins
		sources["CORS_ALLOWED_ORIGINS"] = sourceYAML
	}

	// Rate limiting from YAML
	if yamlOnly.RateLimitEnabled {
		cfg.RateLimitEnabled = true
		sources["RATE_LIMIT_ENABLED"] = sourceYAML
	}

	if yamlOnly.RateLimitRPS > 0 {
		cfg.RateLimitRPS = yamlOnly.RateLimitRPS
		sources["RATE_LIMIT_RPS"] = sourceYAML
	}

	if yamlOnly.RateLimitBurst > 0 {
		cfg.RateLimitBurst = yamlOnly.RateLimitBurst
		sources["RATE_LIMIT_BURST"] = sourceYAML
	}

	if yamlOnly.DefaultLocale != "" {
		cfg.DefaultLocale = yamlOnly.DefaultLocale
		sources["DEFAULT_LOCALE"] = sourceYAML
	}

	if yamlOnly.SwaggerAPITitle != "" {
		cfg.SwaggerAPITitle = yamlOnly.SwaggerAPITitle
		sources["SWAGGER_API_TITLE"] = sourceYAML
	}

	// Apply OAuth configuration from YAML
	applyYAMLOAuthConfig(cfg, yamlOnly, sources)

	if yamlOnly.Feeds.TheOddsAPIKey != "" {
		cfg.Feeds.TheOddsAPIKey = yamlOnly.Feeds.TheOddsAPIKey
		sources["THE_ODDS_API_KEY"] = sourceYAML
	}

	if yamlOnly.Feeds.APIFootballKey != "" {
		cfg.Feeds.APIFootballKey = yamlOnly.Feeds.APIFootballKey
		sources["API_FOOTBALL_KEY"] = sourceYAML
	}
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

	// KYC
	if yamlOnly.KYC.EnforcementEnabled {
		cfg.KYC.EnforcementEnabled = true
		sources["KYC_ENFORCEMENT_ENABLED"] = sourceYAML
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

func formatConfigSourceValue(value, source string) string {
	return fmt.Sprintf("%s = %s", sanitizeConfigLogValue(value), sanitizeConfigLogValue(source))
}

func formatConfigBoolValue(value bool, source string) string {
	return fmt.Sprintf("%t = %s", value, sanitizeConfigLogValue(source))
}

func formatConfigSecretValue(value, source, unit string) string {
	return fmt.Sprintf("[%d %s] = %s", len(value), unit, sanitizeConfigLogValue(source))
}

func sanitizeConfigLogValue(value string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			return ' '
		case unicode.IsPrint(r):
			return r
		default:
			return -1
		}
	}, value)
}

// parseCommaSeparated parses a comma-separated string into a slice of strings.
// Trims whitespace from each element.
func parseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
