// Package setup provides server setup and configuration utilities.
package setup

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/cmelgarejo/go-modulith-template/cmd/server/observability"
	"github.com/cmelgarejo/go-modulith-template/internal/appversion"
	"github.com/cmelgarejo/go-modulith-template/internal/config"
	"github.com/cmelgarejo/go-modulith-template/internal/i18n"
)

// LoadDotenv loads environment variables from .env file.
func LoadDotenv() error {
	if err := godotenv.Load(); err != nil {
		return fmt.Errorf("failed to load .env file: %w", err)
	}

	return nil
}

// LoadConfig loads configuration from YAML, .env, and environment variables.
func LoadConfig() *config.AppConfig {
	observability.InitLoggerEarly()

	systemEnvVars := CaptureSystemEnvVars()
	_ = LoadDotenv()

	cfg, err := config.Load("configs/server.yaml", systemEnvVars)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		return nil
	}

	observability.InitLogger(cfg.Env, cfg.LogLevel)

	// Initialize i18n
	if err := i18n.Init(cfg.DefaultLocale); err != nil {
		slog.Error("Failed to initialize i18n", "error", err)
		return nil
	}

	slog.Info("Starting application", "version", appversion.Info())

	return cfg
}

// CaptureSystemEnvVars captures system environment variables before .env is loaded.
func CaptureSystemEnvVars() map[string]string {
	systemEnvVars := make(map[string]string)
	if env := os.Getenv("ENV"); env != "" {
		systemEnvVars["ENV"] = env
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		systemEnvVars["LOG_LEVEL"] = logLevel
	}

	if port := os.Getenv("HTTP_PORT"); port != "" {
		systemEnvVars["HTTP_PORT"] = port
	}

	if port := os.Getenv("GRPC_PORT"); port != "" {
		systemEnvVars["GRPC_PORT"] = port
	}

	if dsn := os.Getenv("DB_DSN"); dsn != "" {
		systemEnvVars["DB_DSN"] = dsn
	}

	if endpoint := os.Getenv("OTLP_ENDPOINT"); endpoint != "" {
		systemEnvVars["OTLP_ENDPOINT"] = endpoint
	}

	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		systemEnvVars["JWT_SECRET"] = secret
	}

	if key := os.Getenv("JWT_PRIVATE_KEY"); key != "" {
		systemEnvVars["JWT_PRIVATE_KEY"] = key
	}

	if key := os.Getenv("JWT_PUBLIC_KEY"); key != "" {
		systemEnvVars["JWT_PUBLIC_KEY"] = key
	}

	return systemEnvVars
}
