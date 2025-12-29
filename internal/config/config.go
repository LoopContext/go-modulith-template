// Package config provides configuration management for the application.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/cmelgarejo/go-modulith-template/modules/auth"
	"gopkg.in/yaml.v3"
)

const (
	sourceYAML    = "yaml"
	sourceDotenv  = ".env"
	sourceSystem  = "system"
	sourceDefault = "default"
)

// AppConfig is the root configuration for the entire modulith
type AppConfig struct {
	Env      string `yaml:"env" env:"ENV"`
	HTTPPort string `yaml:"http_port" env:"HTTP_PORT"`
	GRPCPort string `yaml:"grpc_port" env:"GRPC_PORT"`
	DBDSN    string `yaml:"db_dsn" env:"DB_DSN"`

	// Observability
	OTLPEndpoint string `yaml:"otlp_endpoint" env:"OTLP_ENDPOINT"`

	// Module specific configs
	Auth auth.Config `yaml:"auth"`
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
		Env:      "dev",
		HTTPPort: "8080",
		GRPCPort: "9050",
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

	if endpoint := os.Getenv("OTLP_ENDPOINT"); endpoint != "" {
		c.OTLPEndpoint = endpoint
		sources["OTLP_ENDPOINT"] = sourceName
	}

	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		c.Auth.JWTSecret = secret
		sources["JWT_SECRET"] = sourceName
	}
}

// OverrideWithEnvFromDotenv applies .env file values, but only marks as ".env" if:
// 1. The variable was not in system ENV vars (new variable from .env), OR
// 2. The value changed from system ENV vars (overridden by .env)
func (c *AppConfig) OverrideWithEnvFromDotenv(sources, systemEnvVars map[string]string, sourceName string) {
	c.overrideEnvVar("ENV", func(val string) { c.Env = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("HTTP_PORT", func(val string) { c.HTTPPort = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("GRPC_PORT", func(val string) { c.GRPCPort = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("DB_DSN", func(val string) { c.DBDSN = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("OTLP_ENDPOINT", func(val string) { c.OTLPEndpoint = val }, sources, systemEnvVars, sourceName)
	c.overrideEnvVar("JWT_SECRET", func(val string) { c.Auth.JWTSecret = val }, sources, systemEnvVars, sourceName)
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

	if yamlOnly.OTLPEndpoint != "" {
		cfg.OTLPEndpoint = yamlOnly.OTLPEndpoint
		sources["OTLP_ENDPOINT"] = sourceYAML
	}

	if yamlOnly.Auth.JWTSecret != "" {
		cfg.Auth.JWTSecret = yamlOnly.Auth.JWTSecret
		sources["JWT_SECRET"] = sourceYAML
	}
}

// getSource returns the source for a config key, or default if not found
func getSource(sources map[string]string, key string) string {
	if src, ok := sources[key]; ok {
		return src
	}

	return sourceDefault
}
