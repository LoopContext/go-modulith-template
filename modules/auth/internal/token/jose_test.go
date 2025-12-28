package token

import (
	"testing"
	"time"
)

func TestTokenService(t *testing.T) {
	secret := "test-secret-key-must-be-long-enough"
	ts, err := NewTokenService(secret)
	if err != nil {
		t.Fatalf("Failed to create token service: %v", err)
	}

	t.Run("CreateAndVerifyToken", func(t *testing.T) {
		userID := "user-123"
		role := "admin"
		duration := time.Minute

		token, err := ts.CreateToken(userID, role, duration)
		if err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}

		claims, err := ts.VerifyToken(token)
		if err != nil {
			t.Fatalf("Failed to verify token: %v", err)
		}

		if claims.UserID != userID {
			t.Errorf("Expected userID %s, got %s", userID, claims.UserID)
		}
		if claims.Role != role {
			t.Errorf("Expected role %s, got %s", role, claims.Role)
		}
	})

	t.Run("ExpiredToken", func(t *testing.T) {
		userID := "user-expired"
		role := "user"
		// Negative duration to ensure expiry
		duration := -time.Minute

		token, err := ts.CreateToken(userID, role, duration)
		if err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}

		_, err = ts.VerifyToken(token)
		if err == nil {
			t.Error("Expected error for expired token, got nil")
		}
	})

	t.Run("InvalidSignature", func(t *testing.T) {
		ts2, _ := NewTokenService("different-secret-key-that-is-long")

		token, err := ts.CreateToken("user", "role", time.Minute)
		if err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}

		_, err = ts2.VerifyToken(token)
		if err == nil {
			t.Error("Expected error for invalid signature, got nil")
		}
	})
}
