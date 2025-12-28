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

// Load loads the configuration from a YAML file and overrides with environment variables
func Load(path string) (*AppConfig, error) {
	cfg := &AppConfig{
		Env:      "dev",
		HTTPPort: "8080",
		GRPCPort: "9050",
	}

	// 1. Load from YAML if exists
	if path == "" {
		return cfg, nil
	}

	cleanPath := filepath.Clean(path)
	if _, err := os.Stat(cleanPath); err != nil {
		return cfg, nil
	}

	f, err := os.Open(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}

	defer func() {
		if err := f.Close(); err != nil {
			slog.Error("failed to close config file", "error", err)
		}
	}()

	if err := yaml.NewDecoder(f).Decode(cfg); err != nil {
		return nil, fmt.Errorf("failed to decode yaml config: %w", err)
	}

	// 2. Override with Env Vars
	cfg.overrideWithEnv()

	// 3. Validation
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// overrideWithEnv manually maps environment variables to the config struct.
// In a production app, consider using a library like cleanenv or envconfig.
func (c *AppConfig) overrideWithEnv() {
	if env := os.Getenv("ENV"); env != "" {
		c.Env = env
	}

	if port := os.Getenv("HTTP_PORT"); port != "" {
		c.HTTPPort = port
	}

	if port := os.Getenv("GRPC_PORT"); port != "" {
		c.GRPCPort = port
	}

	if dsn := os.Getenv("DB_DSN"); dsn != "" {
		c.DBDSN = dsn
	}

	if endpoint := os.Getenv("OTLP_ENDPOINT"); endpoint != "" {
		c.OTLPEndpoint = endpoint
	}

	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		c.Auth.JWTSecret = secret
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

	return nil
}
