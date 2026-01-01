package oauth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

const (
	// DefaultStateExpiration is the default lifetime for OAuth state tokens.
	DefaultStateExpiration = 10 * time.Minute
	// StateTokenBytes is the number of random bytes in a state token.
	StateTokenBytes = 32
)

// StateManager handles creation and validation of OAuth state tokens.
type StateManager struct {
	hmacKey []byte
}

// NewStateManager creates a new StateManager with the given HMAC key.
// The key should be at least 32 bytes for security.
func NewStateManager(key []byte) *StateManager {
	return &StateManager{
		hmacKey: key,
	}
}

// GenerateState creates a new cryptographically secure state token.
func (m *StateManager) GenerateState() (string, error) {
	randomBytes := make([]byte, StateTokenBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Create HMAC of the random bytes for integrity
	mac := hmac.New(sha256.New, m.hmacKey)
	mac.Write(randomBytes)
	signature := mac.Sum(nil)

	// Combine random bytes and signature
	combined := make([]byte, 0, len(randomBytes)+len(signature))
	combined = append(combined, randomBytes...)
	combined = append(combined, signature...)

	return base64.URLEncoding.EncodeToString(combined), nil
}

// ValidateState validates a state token.
// It checks the HMAC signature to ensure the token wasn't tampered with.
func (m *StateManager) ValidateState(state string) bool {
	combined, err := base64.URLEncoding.DecodeString(state)
	if err != nil {
		return false
	}

	expectedLen := StateTokenBytes + sha256.Size
	if len(combined) != expectedLen {
		return false
	}

	randomBytes := combined[:StateTokenBytes]
	providedSignature := combined[StateTokenBytes:]

	// Recalculate HMAC
	mac := hmac.New(sha256.New, m.hmacKey)
	mac.Write(randomBytes)
	expectedSignature := mac.Sum(nil)

	return hmac.Equal(providedSignature, expectedSignature)
}

// CreateStateData creates a new StateData with an expiration time.
func (m *StateManager) CreateStateData(provider, redirectURL, userID string, action StateAction) (*StateData, error) {
	state, err := m.GenerateState()
	if err != nil {
		return nil, err
	}

	return &StateData{
		State:       state,
		Provider:    provider,
		RedirectURL: redirectURL,
		UserID:      userID,
		Action:      action,
		ExpiresAt:   time.Now().Add(DefaultStateExpiration),
	}, nil
}

