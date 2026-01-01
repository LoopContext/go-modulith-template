package token

import (
	"testing"
	"time"
)

const testSecret = "test-secret-key-must-be-long-enough"

func TestTokenService(t *testing.T) {
	ts, err := NewService(testSecret)
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

		if claims.Subject != userID {
			t.Errorf("Expected userID %s, got %s", userID, claims.Subject)
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
		ts2, _ := NewService("different-secret-key-that-is-long")

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

func TestNewService_EmptySecret(t *testing.T) {
	_, err := NewService("")
	if err == nil {
		t.Error("Expected error when secret is empty")
	}
}

func TestNewService_ShortSecret(t *testing.T) {
	_, err := NewService("short")
	if err == nil {
		t.Error("Expected error when secret is too short")
	}
}

func TestNewService_ExactlyMinLength(t *testing.T) {
	secret := "12345678901234567890123456789012" // exactly 32 bytes

	ts, err := NewService(secret)
	if err != nil {
		t.Errorf("Expected no error for 32-byte secret, got: %v", err)
	}

	if ts == nil {
		t.Error("Expected token service to not be nil")
	}
}

func TestCreateToken_EmptyUserID(t *testing.T) {
	ts, err := NewService(testSecret)
	if err != nil {
		t.Fatalf("Failed to create token service: %v", err)
	}



	token, err := ts.CreateToken("", "user", time.Minute)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	claims, err := ts.VerifyToken(token)
	if err != nil {
		t.Fatalf("Failed to verify token: %v", err)
	}

	if claims.Subject != "" {
		t.Errorf("Expected empty userID, got %s", claims.Subject)
	}
}

func TestCreateToken_EmptyRole(t *testing.T) {
	ts, err := NewService(testSecret)
	if err != nil {
		t.Fatalf("Failed to create token service: %v", err)
	}



	token, err := ts.CreateToken("user-123", "", time.Minute)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	claims, err := ts.VerifyToken(token)
	if err != nil {
		t.Fatalf("Failed to verify token: %v", err)
	}

	if claims.Role != "" {
		t.Errorf("Expected empty role, got %s", claims.Role)
	}
}

func TestVerifyToken_MalformedToken(t *testing.T) {
	ts, err := NewService(testSecret)
	if err != nil {
		t.Fatalf("Failed to create token service: %v", err)
	}



	_, err = ts.VerifyToken("not.a.valid.token")
	if err == nil {
		t.Error("Expected error for malformed token")
	}
}

func TestVerifyToken_EmptyToken(t *testing.T) {
	ts, err := NewService(testSecret)
	if err != nil {
		t.Fatalf("Failed to create token service: %v", err)
	}



	_, err = ts.VerifyToken("")
	if err == nil {
		t.Error("Expected error for empty token")
	}
}
