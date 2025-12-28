package config

import (
	"fmt"
	"os"

	"github.com/cmelgarejo/go-modulith-template/modules/auth"
	"gopkg.in/yaml.v3"
)

// AppConfig is the root configuration for the entire modulith
type AppConfig struct {
	Env      string `yaml:"env" env:"ENV"`
	HTTPPort string `yaml:"http_port" env:"HTTP_PORT"`
	GRPCPort string `yaml:"grpc_port" env:"GRPC_PORT"`
	DBDSN    string `yaml:"db_dsn" env:"DB_DSN"`

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
	if path != "" {
		f, err := os.Open(path)
		if err == nil {
			defer f.Close()
			if err := yaml.NewDecoder(f).Decode(cfg); err != nil {
				return nil, fmt.Errorf("failed to decode yaml config: %w", err)
			}
		}
	}

	// 2. Override with Env Vars (Manual mapping for simplicity in this template)
	if env := os.Getenv("ENV"); env != "" {
		cfg.Env = env
	}
	if port := os.Getenv("HTTP_PORT"); port != "" {
		cfg.HTTPPort = port
	}
	if port := os.Getenv("GRPC_PORT"); port != "" {
		cfg.GRPCPort = port
	}
	if dsn := os.Getenv("DB_DSN"); dsn != "" {
		cfg.DBDSN = dsn
	}
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		cfg.Auth.JWTSecret = secret
	}

	return cfg, nil
}
