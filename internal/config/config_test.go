package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear environment
	clearTestEnv(t)

	cfg, err := Load("", map[string]string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Env != "dev" {
		t.Errorf("expected default env 'dev', got %s", cfg.Env)
	}

	if cfg.HTTPPort != "8080" {
		t.Errorf("expected default HTTP port '8080', got %s", cfg.HTTPPort)
	}

	if cfg.GRPCPort != "9050" {
		t.Errorf("expected default gRPC port '9050', got %s", cfg.GRPCPort)
	}
}

func TestLoad_EnvironmentVariables(t *testing.T) {
	clearTestEnv(t)
	setupTestEnv(t)

	defer clearTestEnv(t)

	cfg, err := Load("", map[string]string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Env != "test" {
		t.Errorf("expected env 'test', got %s", cfg.Env)
	}

	if cfg.HTTPPort != "3000" {
		t.Errorf("expected HTTP port '3000', got %s", cfg.HTTPPort)
	}

	if cfg.GRPCPort != "4000" {
		t.Errorf("expected gRPC port '4000', got %s", cfg.GRPCPort)
	}

	if cfg.DBDSN != "postgres://localhost/test" {
		t.Errorf("expected DB DSN 'postgres://localhost/test', got %s", cfg.DBDSN)
	}

	if cfg.OTLPEndpoint != "http://localhost:4317" {
		t.Errorf("expected OTLP endpoint 'http://localhost:4317', got %s", cfg.OTLPEndpoint)
	}

	if cfg.Auth.JWTSecret != "test-secret-key-that-is-at-least-32-bytes-long" {
		t.Errorf("expected JWT secret to be set")
	}
}

func TestLoad_YAMLConfigFile(t *testing.T) {
	clearTestEnv(t)

	// Create temporary YAML config
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "test-config.yaml")

	yamlContent := `env: staging
http_port: "8888"
grpc_port: "9999"
db_dsn: "postgres://localhost/staging"
otlp_endpoint: "http://otlp:4317"
auth:
  jwt_secret: "yaml-secret-key-that-is-at-least-32-bytes-long"
`

	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(yamlPath, map[string]string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Env != "staging" {
		t.Errorf("expected env 'staging', got %s", cfg.Env)
	}

	if cfg.HTTPPort != "8888" {
		t.Errorf("expected HTTP port '8888', got %s", cfg.HTTPPort)
	}

	if cfg.GRPCPort != "9999" {
		t.Errorf("expected gRPC port '9999', got %s", cfg.GRPCPort)
	}

	if cfg.DBDSN != "postgres://localhost/staging" {
		t.Errorf("expected DB DSN 'postgres://localhost/staging', got %s", cfg.DBDSN)
	}

	if cfg.OTLPEndpoint != "http://otlp:4317" {
		t.Errorf("expected OTLP endpoint 'http://otlp:4317', got %s", cfg.OTLPEndpoint)
	}

	if cfg.Auth.JWTSecret != "yaml-secret-key-that-is-at-least-32-bytes-long" {
		t.Errorf("expected JWT secret to be set from YAML")
	}
}

func TestLoad_PriorityOrder(t *testing.T) {
	clearTestEnv(t)

	// System env vars (captured before .env)
	systemEnvVars := map[string]string{
		"ENV": "system",
	}

	// Simulated .env file (now in os.Getenv)
	setupPriorityTestEnv(t)

	defer clearTestEnv(t)

	// Create YAML config
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "test-config.yaml")

	yamlContent := `env: yaml
http_port: "6666"
`

	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(yamlPath, systemEnvVars)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// YAML should have highest priority
	if cfg.Env != "yaml" {
		t.Errorf("expected env 'yaml' (from YAML), got %s", cfg.Env)
	}

	if cfg.HTTPPort != "6666" {
		t.Errorf("expected HTTP port '6666' (from YAML), got %s", cfg.HTTPPort)
	}

	// GRPCPort not in YAML, should use default
	if cfg.GRPCPort != "9050" {
		t.Errorf("expected gRPC port '9050' (default), got %s", cfg.GRPCPort)
	}
}

func TestLoad_NonExistentYAML(t *testing.T) {
	clearTestEnv(t)

	cfg, err := Load("/path/to/nonexistent/config.yaml", map[string]string{})
	if err != nil {
		t.Fatalf("expected no error for nonexistent YAML, got %v", err)
	}

	// Should use defaults
	if cfg.Env != "dev" {
		t.Errorf("expected default env 'dev', got %s", cfg.Env)
	}
}

func TestValidate_ProductionRequirements(t *testing.T) {
	t.Run("prod missing DB_DSN", func(t *testing.T) {
		cfg := &AppConfig{
			Env:      "prod",
			HTTPPort: "8080",
			GRPCPort: "9050",
			DBDSN:    "",
			Auth: struct {
				JWTSecret string `yaml:"jwt_secret"`
			}{
				JWTSecret: "valid-secret-key-that-is-at-least-32-bytes-long",
			},
		}

		err := cfg.Validate()
		if err == nil {
			t.Fatal("expected error for missing DB_DSN in production")
		}
	})

	t.Run("prod missing JWT_SECRET", func(t *testing.T) {
		cfg := &AppConfig{
			Env:      "prod",
			HTTPPort: "8080",
			GRPCPort: "9050",
			DBDSN:    "postgres://localhost/db",
			Auth: struct {
				JWTSecret string `yaml:"jwt_secret"`
			}{
				JWTSecret: "",
			},
		}

		err := cfg.Validate()
		if err == nil {
			t.Fatal("expected error for missing JWT_SECRET in production")
		}
	})

	t.Run("prod with valid config", func(t *testing.T) {
		cfg := &AppConfig{
			Env:      "prod",
			HTTPPort: "8080",
			GRPCPort: "9050",
			DBDSN:    "postgres://localhost/db",
			Auth: struct {
				JWTSecret string `yaml:"jwt_secret"`
			}{
				JWTSecret: "valid-secret-key-that-is-at-least-32-bytes-long",
			},
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestValidate_JWTSecretLength(t *testing.T) {
	t.Run("JWT secret too short", func(t *testing.T) {
		cfg := &AppConfig{
			Env:      "dev",
			HTTPPort: "8080",
			GRPCPort: "9050",
			Auth: struct {
				JWTSecret string `yaml:"jwt_secret"`
			}{
				JWTSecret: "short",
			},
		}

		err := cfg.Validate()
		if err == nil {
			t.Fatal("expected error for short JWT secret")
		}
	})

	t.Run("JWT secret exactly 32 bytes", func(t *testing.T) {
		cfg := &AppConfig{
			Env:      "dev",
			HTTPPort: "8080",
			GRPCPort: "9050",
			Auth: struct {
				JWTSecret string `yaml:"jwt_secret"`
			}{
				JWTSecret: "12345678901234567890123456789012",
			},
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("expected no error for 32-byte secret, got %v", err)
		}
	})
}

func TestLoad_InvalidYAMLFile(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "invalid.yaml")

	invalidYAML := `invalid: yaml: content: [[[`
	if err := os.WriteFile(yamlPath, []byte(invalidYAML), 0o600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := Load(yamlPath, map[string]string{})
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

// Helper functions
func setupTestEnv(t *testing.T) {
	t.Helper()

	envVars := map[string]string{
		"ENV":           "test",
		"HTTP_PORT":     "3000",
		"GRPC_PORT":     "4000",
		"DB_DSN":        "postgres://localhost/test",
		"OTLP_ENDPOINT": "http://localhost:4317",
		"JWT_SECRET":    "test-secret-key-that-is-at-least-32-bytes-long",
	}

	for key, value := range envVars {
		if err := os.Setenv(key, value); err != nil {
			t.Fatalf("failed to set %s: %v", key, err)
		}
	}
}

func setupPriorityTestEnv(t *testing.T) {
	t.Helper()

	envVars := map[string]string{
		"ENV":       "dotenv",
		"HTTP_PORT": "7777",
	}

	for key, value := range envVars {
		if err := os.Setenv(key, value); err != nil {
			t.Fatalf("failed to set %s: %v", key, err)
		}
	}
}

func clearTestEnv(t *testing.T) {
	t.Helper()

	_ = os.Unsetenv("ENV")
	_ = os.Unsetenv("HTTP_PORT")
	_ = os.Unsetenv("GRPC_PORT")
	_ = os.Unsetenv("DB_DSN")
	_ = os.Unsetenv("OTLP_ENDPOINT")
	_ = os.Unsetenv("JWT_SECRET")
}

