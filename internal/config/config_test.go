package config

import (
	"os"
	"path/filepath"
	"testing"
)

const envDev = "dev"

func TestLoad_Defaults(t *testing.T) {
	// Clear environment
	clearTestEnv(t)

	cfg, err := Load("", map[string]string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Env != envDev {
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

	if cfg.Auth.JWTPublicKeyPEM != "test-public-key-pem-that-is-at-least-100-characters-long-to-pass-validation-logic-for-pem-format-checking-in-config-validation" {
		t.Errorf("expected JWTPublicKeyPEM to be set")
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
  jwt_public_key: "yaml-public-key-pem-that-is-at-least-100-characters-long-to-pass-validation-logic-for-pem-format-checking-in-config-validation"
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

	if cfg.Auth.JWTPublicKeyPEM != "yaml-public-key-pem-that-is-at-least-100-characters-long-to-pass-validation-logic-for-pem-format-checking-in-config-validation" {
		t.Errorf("expected JWTPublicKeyPEM to be set from YAML")
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
			Auth: AuthConfig{
				JWTPublicKeyPEM: "valid-public-key-pem-that-is-at-least-100-characters-long-to-pass-validation-logic-for-pem-format-checking-in-config-validation",
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
			Auth: AuthConfig{
				JWTPublicKeyPEM: "",
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
			Auth: AuthConfig{
				JWTPublicKeyPEM: "valid-public-key-pem-that-is-at-least-100-characters-long-to-pass-validation-logic-for-pem-format-checking-in-config-validation",
			},
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestValidate_JWTPEMLength(t *testing.T) {
	t.Run("JWT public key too short", func(t *testing.T) {
		cfg := &AppConfig{
			Env:      envDev,
			HTTPPort: "8080",
			GRPCPort: "9050",
			Auth: AuthConfig{
				JWTPublicKeyPEM: "short-pem",
			},
		}

		err := cfg.Validate()
		if err == nil {
			t.Fatal("expected error for short JWT public key")
		}
	})


	t.Run("JWT public key long enough", func(t *testing.T) {
		cfg := &AppConfig{
			Env:      envDev,
			HTTPPort: "8080",
			GRPCPort: "9050",
			Auth: AuthConfig{
				JWTPublicKeyPEM: "valid-public-key-pem-that-is-at-least-100-characters-long-to-pass-validation-logic-for-pem-format-checking-in-config-validation",
			},
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("expected no error for long enough PEM, got %v", err)
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
		"JWT_PUBLIC_KEY": "test-public-key-pem-that-is-at-least-100-characters-long-to-pass-validation-logic-for-pem-format-checking-in-config-validation",
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
	_ = os.Unsetenv("JWT_PUBLIC_KEY")
	_ = os.Unsetenv("JWT_PRIVATE_KEY")

	// OAuth vars
	_ = os.Unsetenv("OAUTH_ENABLED")
	_ = os.Unsetenv("OAUTH_AUTO_LINK_BY_EMAIL")
	_ = os.Unsetenv("OAUTH_BASE_URL")
	_ = os.Unsetenv("OAUTH_TOKEN_ENCRYPTION_KEY")
	_ = os.Unsetenv("GOOGLE_CLIENT_ID")
	_ = os.Unsetenv("GOOGLE_CLIENT_SECRET")
	_ = os.Unsetenv("FACEBOOK_CLIENT_ID")
	_ = os.Unsetenv("FACEBOOK_CLIENT_SECRET")
	_ = os.Unsetenv("GITHUB_CLIENT_ID")
	_ = os.Unsetenv("GITHUB_CLIENT_SECRET")
}

func TestLoad_OAuthConfiguration(t *testing.T) {
	clearTestEnv(t)
	setupOAuthEnv(t)

	defer clearTestEnv(t)

	cfg, err := Load("", map[string]string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	t.Run("OAuth settings loaded", func(t *testing.T) {
		if !cfg.Auth.OAuth.Enabled {
			t.Error("expected OAuth to be enabled")
		}

		if !cfg.Auth.OAuth.AutoLinkByEmail {
			t.Error("expected OAuth auto-link to be enabled")
		}

		if cfg.Auth.OAuth.BaseURL != "http://localhost:8080" {
			t.Errorf("expected OAuth base URL 'http://localhost:8080', got %s", cfg.Auth.OAuth.BaseURL)
		}

		if cfg.Auth.OAuth.TokenEncryptionKey != "12345678901234567890123456789012" {
			t.Errorf("expected OAuth token encryption key, got %s", cfg.Auth.OAuth.TokenEncryptionKey)
		}
	})

	t.Run("OAuth providers loaded", func(t *testing.T) {
		if !cfg.Auth.OAuth.Providers.Google.Enabled {
			t.Error("expected Google provider to be enabled")
		}

		if cfg.Auth.OAuth.Providers.Google.ClientID != "google-client-id" {
			t.Errorf("expected Google client ID, got %s", cfg.Auth.OAuth.Providers.Google.ClientID)
		}

		if !cfg.Auth.OAuth.Providers.Facebook.Enabled {
			t.Error("expected Facebook provider to be enabled")
		}

		if !cfg.Auth.OAuth.Providers.GitHub.Enabled {
			t.Error("expected GitHub provider to be enabled")
		}
	})
}

func setupOAuthEnv(t *testing.T) {
	t.Helper()

	_ = os.Setenv("OAUTH_ENABLED", "true")
	_ = os.Setenv("OAUTH_AUTO_LINK_BY_EMAIL", "true")
	_ = os.Setenv("OAUTH_BASE_URL", "http://localhost:8080")
	_ = os.Setenv("OAUTH_TOKEN_ENCRYPTION_KEY", "12345678901234567890123456789012")
	_ = os.Setenv("GOOGLE_CLIENT_ID", "google-client-id")
	_ = os.Setenv("GOOGLE_CLIENT_SECRET", "google-client-secret")
	_ = os.Setenv("FACEBOOK_CLIENT_ID", "facebook-client-id")
	_ = os.Setenv("FACEBOOK_CLIENT_SECRET", "facebook-client-secret")
	_ = os.Setenv("GITHUB_CLIENT_ID", "github-client-id")
	_ = os.Setenv("GITHUB_CLIENT_SECRET", "github-client-secret")
	_ = os.Setenv("JWT_PUBLIC_KEY", "test-public-key-pem-that-is-at-least-100-characters-long-to-pass-validation-logic-for-pem-format-checking-in-config-validation")
}

func TestLoad_OAuthYAMLConfiguration(t *testing.T) {
	clearTestEnv(t)

	yamlPath := createOAuthYAMLConfig(t)

	cfg, err := Load(yamlPath, map[string]string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	t.Run("OAuth settings from YAML", func(t *testing.T) {
		if !cfg.Auth.OAuth.Enabled {
			t.Error("expected OAuth to be enabled from YAML")
		}

		if cfg.Auth.OAuth.BaseURL != "http://yaml:8080" {
			t.Errorf("expected OAuth base URL from YAML, got %s", cfg.Auth.OAuth.BaseURL)
		}
	})

	t.Run("OAuth providers from YAML", func(t *testing.T) {
		providers := []struct {
			name    string
			enabled bool
		}{
			{"Google", cfg.Auth.OAuth.Providers.Google.Enabled},
			{"Facebook", cfg.Auth.OAuth.Providers.Facebook.Enabled},
			{"GitHub", cfg.Auth.OAuth.Providers.GitHub.Enabled},
			{"Apple", cfg.Auth.OAuth.Providers.Apple.Enabled},
			{"Microsoft", cfg.Auth.OAuth.Providers.Microsoft.Enabled},
			{"Twitter", cfg.Auth.OAuth.Providers.Twitter.Enabled},
		}

		for _, p := range providers {
			if !p.enabled {
				t.Errorf("expected %s provider to be enabled from YAML", p.name)
			}
		}
	})
}

func createOAuthYAMLConfig(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "test-oauth-config.yaml")

	yamlContent := `env: dev
http_port: "8080"
grpc_port: "9050"
auth:
  jwt_public_key: "test-public-key-pem-that-is-at-least-100-characters-long-to-pass-validation-logic-for-pem-format-checking-in-config-validation"
  oauth:
    enabled: true
    auto_link_by_email: true
    base_url: "http://yaml:8080"
    token_encryption_key: "12345678901234567890123456789012"
    providers:
      google:
        enabled: true
        client_id: "yaml-google-id"
        client_secret: "yaml-google-secret"
      facebook:
        enabled: true
        client_id: "yaml-facebook-id"
        client_secret: "yaml-facebook-secret"
      github:
        enabled: true
        client_id: "yaml-github-id"
        client_secret: "yaml-github-secret"
      apple:
        enabled: true
        client_id: "yaml-apple-id"
        team_id: "apple-team-id"
        key_id: "apple-key-id"
        private_key_path: "/path/to/key.p8"
      microsoft:
        enabled: true
        client_id: "yaml-microsoft-id"
        client_secret: "yaml-microsoft-secret"
      twitter:
        enabled: true
        client_id: "yaml-twitter-id"
        client_secret: "yaml-twitter-secret"
`

	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	return yamlPath
}

func TestValidate_OAuthConfig(t *testing.T) {
	t.Run("OAuth enabled with valid encryption key", func(t *testing.T) {
		cfg := validOAuthConfig()

		err := cfg.Validate()
		if err != nil {
			t.Errorf("expected no error for valid OAuth config, got %v", err)
		}
	})

	t.Run("OAuth enabled with missing base URL", func(t *testing.T) {
		cfg := validOAuthConfig()
		cfg.Auth.OAuth.BaseURL = ""

		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for OAuth enabled without base URL")
		}
	})

	t.Run("OAuth provider enabled with missing credentials", func(t *testing.T) {
		cfg := validOAuthConfig()
		cfg.Auth.OAuth.Providers.Google.ClientID = ""
		cfg.Auth.OAuth.Providers.Google.ClientSecret = ""

		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for OAuth provider enabled without credentials")
		}
	})

	t.Run("OAuth disabled, no validation required", func(t *testing.T) {
		cfg := &AppConfig{
			Env:      envDev,
			HTTPPort: "8080",
			GRPCPort: "9050",
			Auth: AuthConfig{
				JWTPublicKeyPEM: "valid-public-key-pem-that-is-at-least-100-characters-long-to-pass-validation-logic-for-pem-format-checking-in-config-validation",
				OAuth: OAuthConfig{
					Enabled: false,
				},
			},
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("expected no error for disabled OAuth, got %v", err)
		}
	})
}

func validOAuthConfig() *AppConfig {
	return &AppConfig{
		Env:      envDev,
		HTTPPort: "8080",
		GRPCPort: "9050",
		Auth: AuthConfig{
			JWTPublicKeyPEM: "valid-public-key-pem-that-is-at-least-100-characters-long-to-pass-validation-logic-for-pem-format-checking-in-config-validation",
			OAuth: OAuthConfig{
				Enabled:            true,
				BaseURL:            "http://localhost:8080",
				TokenEncryptionKey: "12345678901234567890123456789012", // Exactly 32 bytes
				Providers: OAuthProviders{
					Google: OAuthProviderConfig{
						Enabled:      true,
						ClientID:     "google-id",
						ClientSecret: "google-secret",
					},
				},
			},
		},
	}
}
