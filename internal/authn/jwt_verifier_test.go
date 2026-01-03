package authn

import (
	"context"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

func TestNewJWTVerifier(t *testing.T) {
	t.Run("valid secret", func(t *testing.T) {
		verifier, err := NewJWTVerifier("my-secret-key")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if verifier == nil {
			t.Fatal("expected verifier to not be nil")
		}
	})

	t.Run("empty secret", func(t *testing.T) {
		verifier, err := NewJWTVerifier("")
		if err == nil {
			t.Fatal("expected error for empty secret")
		}

		if verifier != nil {
			t.Error("expected verifier to be nil")
		}
	})
}

func TestJWTVerifier_VerifyToken(t *testing.T) {
	secret := "test-secret-key-that-is-at-least-32-bytes-long"

	verifier, err := NewJWTVerifier(secret)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	t.Run("valid token", func(t *testing.T) {
		testValidToken(t, verifier, secret)
	})

	t.Run("expired token", func(t *testing.T) {
		testExpiredToken(t, verifier, secret)
	})

	t.Run("missing subject", func(t *testing.T) {
		testMissingSubject(t, verifier, secret)
	})

	t.Run("invalid signature", func(t *testing.T) {
		testInvalidSignature(t, verifier)
	})

	t.Run("malformed token", func(t *testing.T) {
		testMalformedToken(t, verifier)
	})

	t.Run("token without role claim", func(t *testing.T) {
		testTokenWithoutRole(t, verifier, secret)
	})

	t.Run("empty token string", func(t *testing.T) {
		testEmptyTokenString(t, verifier)
	})
}

func testValidToken(t *testing.T, verifier *JWTVerifier, secret string) {
	t.Helper()

	token := createTestToken(t, secret, "user-123", "admin", time.Now().Add(time.Hour))

	claims, err := verifier.VerifyToken(context.Background(), token)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("expected user ID 'user-123', got %s", claims.UserID)
	}

	if claims.Role != "admin" {
		t.Errorf("expected role 'admin', got %s", claims.Role)
	}
}

func testExpiredToken(t *testing.T, verifier *JWTVerifier, secret string) {
	t.Helper()

	token := createTestToken(t, secret, "user-123", "admin", time.Now().Add(-time.Hour))

	_, err := verifier.VerifyToken(context.Background(), token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func testMissingSubject(t *testing.T, verifier *JWTVerifier, secret string) {
	t.Helper()

	token := createTestToken(t, secret, "", "admin", time.Now().Add(time.Hour))

	_, err := verifier.VerifyToken(context.Background(), token)
	if err == nil {
		t.Fatal("expected error for missing subject")
	}
}

func testInvalidSignature(t *testing.T, verifier *JWTVerifier) {
	t.Helper()

	wrongSecret := "wrong-secret-key-that-is-at-least-32-bytes-long-x"
	token := createTestToken(t, wrongSecret, "user-123", "admin", time.Now().Add(time.Hour))

	_, err := verifier.VerifyToken(context.Background(), token)
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

func testMalformedToken(t *testing.T, verifier *JWTVerifier) {
	t.Helper()

	_, err := verifier.VerifyToken(context.Background(), "not-a-valid-jwt")
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}

func testTokenWithoutRole(t *testing.T, verifier *JWTVerifier, secret string) {
	t.Helper()

	token := createTestToken(t, secret, "user-456", "", time.Now().Add(time.Hour))

	claims, err := verifier.VerifyToken(context.Background(), token)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if claims.UserID != "user-456" {
		t.Errorf("expected user ID 'user-456', got %s", claims.UserID)
	}

	if claims.Role != "" {
		t.Errorf("expected empty role, got %s", claims.Role)
	}
}

func testEmptyTokenString(t *testing.T, verifier *JWTVerifier) {
	t.Helper()

	_, err := verifier.VerifyToken(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty token string")
	}
}

func TestJWTVerifier_DifferentSecrets(t *testing.T) {
	secret1 := "first-secret-key-that-is-at-least-32-bytes-long"
	secret2 := "second-secret-key-that-is-at-least-32-bytes-long"

	verifier1, err := NewJWTVerifier(secret1)
	if err != nil {
		t.Fatalf("failed to create verifier1: %v", err)
	}

	verifier2, err := NewJWTVerifier(secret2)
	if err != nil {
		t.Fatalf("failed to create verifier2: %v", err)
	}

	token := createTestToken(t, secret1, "user-123", "admin", time.Now().Add(time.Hour))

	// verifier1 should succeed
	_, err = verifier1.VerifyToken(context.Background(), token)
	if err != nil {
		t.Errorf("verifier1 should verify token signed with secret1, got error: %v", err)
	}

	// verifier2 should fail
	_, err = verifier2.VerifyToken(context.Background(), token)
	if err == nil {
		t.Error("verifier2 should not verify token signed with secret1")
	}
}

// Helper function to create a test JWT token
func createTestToken(t *testing.T, secret, subject, role string, expiresAt time.Time) string {
	t.Helper()

	key := []byte(secret)

	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.HS256, Key: key},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	claims := jwt.Claims{
		Subject:  subject,
		Expiry:   jwt.NewNumericDate(expiresAt),
		IssuedAt: jwt.NewNumericDate(time.Now()),
	}

	privateClaims := make(map[string]interface{})
	if role != "" {
		privateClaims["role"] = role
	}

	token, err := jwt.Signed(signer).Claims(claims).Claims(privateClaims).Serialize()
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	return token
}
