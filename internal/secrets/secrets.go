// Package secrets provides an abstraction for secret management.
// This allows the application to use different secret providers (environment variables,
// HashiCorp Vault, AWS Secrets Manager, etc.) without changing business logic.
package secrets

import (
	"context"
	"fmt"
)

// Provider defines the interface for secret management providers.
type Provider interface {
	// GetSecret retrieves a secret value by key.
	// Returns an error if the secret is not found or cannot be retrieved.
	GetSecret(ctx context.Context, key string) (string, error)

	// GetSecretJSON retrieves a secret value and unmarshals it as JSON.
	// This is useful for structured secrets like database connection strings with multiple fields.
	GetSecretJSON(ctx context.Context, key string, v interface{}) error
}

// ErrSecretNotFound is returned when a secret is not found.
var ErrSecretNotFound = fmt.Errorf("secret not found")

// GetSecretOrDefault retrieves a secret or returns a default value if not found.
func GetSecretOrDefault(ctx context.Context, provider Provider, key, defaultValue string) (string, error) {
	value, err := provider.GetSecret(ctx, key)
	if err != nil {
		if err == ErrSecretNotFound {
			return defaultValue, nil
		}

		return "", fmt.Errorf("failed to get secret: %w", err)
	}

	return value, nil
}

