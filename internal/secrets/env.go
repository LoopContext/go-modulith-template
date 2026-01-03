// Package secrets provides environment variable-based secret provider.
package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// EnvProvider implements the Provider interface using environment variables.
// This is the simplest implementation and is suitable for development and
// containerized deployments where secrets are injected via environment variables.
type EnvProvider struct{}

// NewEnvProvider creates a new environment variable-based secret provider.
func NewEnvProvider() *EnvProvider {
	return &EnvProvider{}
}

// GetSecret retrieves a secret value from environment variables.
func (e *EnvProvider) GetSecret(_ context.Context, key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("%w: %s", ErrSecretNotFound, key)
	}

	return value, nil
}

// GetSecretJSON retrieves a secret value from environment variables and unmarshals it as JSON.
func (e *EnvProvider) GetSecretJSON(ctx context.Context, key string, v interface{}) error {
	value, err := e.GetSecret(ctx, key)
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(value), v); err != nil {
		return fmt.Errorf("failed to unmarshal secret %s as JSON: %w", key, err)
	}

	return nil
}
