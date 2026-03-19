// Package testutil provides testing utilities including testcontainers setup.
package testutil

import (
	"github.com/cmelgarejo/go-modulith-template/internal/config"
)

// TestConfig returns a minimal valid config for testing (RS256 JWT keys).
//
//nolint:gosec // G101: test DSN only, not production credentials
func TestConfig() *config.AppConfig {
	return &config.AppConfig{
		Env:      "test",
		LogLevel: "debug",
		HTTPPort: "8080",
		GRPCPort: "9090",
		DBDSN:    "postgres://test:test@localhost:5432/test?sslmode=disable",
		Auth: config.AuthConfig{
			JWTPrivateKeyPEM: TestJWTPrivateKeyPEM,
			JWTPublicKeyPEM:  TestJWTPublicKeyPEM,
		},
		DBMaxOpenConns:     10,
		DBMaxIdleConns:     5,
		DBConnMaxLifetime:  "5m",
		DBConnectTimeout:   "10s",
		DefaultLocale:      "en",
		OutboxPollInterval: "100ms",
		RequestTimeout:     "30s",
		ReadTimeout:        "5s",
		WriteTimeout:       "10s",
		ShutdownTimeout:    "30s",
	}
}
