package authn

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// Test RSA key pair for RS256 (testing only). Must match internal/testutil/jwt_keys.go.
const testJWTPublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAzdZn/gHvspc7YC+BUnRN
3k0AB/zcrjrMzi16wyEAIeGaaXvtudPWqWzDwcK3VhUrgxTFRKRmiiXDVdhcDkJu
0b5OAiKSbxoKwY+OQOZhjOn+jGK5TtD15uPtjxU7WLnegI2z6m+OnBbxUxL/zBh9
y5V1eI0RBJm0EbPKy2QSKZIvvjPv1i74X6vphQWV2+OAHQLEed++wFQ6FfcNqTSK
C6QEKSrx/hbSwb6OIPV7H/35mLCubb/rFiwz+NGUeahlvu0kpMRRLwJA3pJjr/Im
aw+CW96WpImu79LdcYTUZ3k/N1CCnba8KjMsvHQVxWGHjUegqA5dq/eng9SPPMuj
fQIDAQAB
-----END PUBLIC KEY-----`

const testJWTPrivateKeyPEM = `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDN1mf+Ae+ylztg
L4FSdE3eTQAH/NyuOszOLXrDIQAh4Zppe+2509apbMPBwrdWFSuDFMVEpGaKJcNV
2FwOQm7Rvk4CIpJvGgrBj45A5mGM6f6MYrlO0PXm4+2PFTtYud6AjbPqb46cFvFT
Ev/MGH3LlXV4jREEmbQRs8rLZBIpki++M+/WLvhfq+mFBZXb44AdAsR5377AVDoV
9w2pNIoLpAQpKvH+FtLBvo4g9Xsf/fmYsK5tv+sWLDP40ZR5qGW+7SSkxFEvAkDe
kmOv8iZrD4Jb3pakia7v0t1xhNRneT83UIKdtrwqMyy8dBXFYYeNR6CoDl2r96eD
1I88y6N9AgMBAAECggEACCYVTtp/xUe0a43l5kBBduwAdNB/Ygxk4EKvqfrr+Oto
BAYKdsFarbFnHIwbWvaSluljF+EUSCLPlV3v4wahQX9xsibxOiHDTD9lJ8+XDA+V
arRb1rFyErZySKhUBaKyGs/BUCYjdK1510qYwtkzXbRohqG7Cz4UgWDnRd8L0wZq
21at4l+bDWKxa8vCIZAzvI3XMvWCs+wfvU416XYEC8kBNjEYOqESHwZw6NFA+iOv
haciwkpWAVG1jWMG4jPPLzXEtz/BLjXDHp62gYtZ89dxdKzl2NcD/JFVulI3idTf
GeWbc1lj8pgPmHomt/QJTEbFItY/GWM4fS8Pj51VoQKBgQDssb3AM4OAGhQzUwbG
iFEJRKfa41NQoNguKSfqEoHP+7W+9qK6wy1FEr9MyKr1GVaAhI+Oa9640HVJz/cN
EjdcZ1+dwswxqACpQCoikfIKjA7TVGBAQSYgw02n+VyvpnGs32CdNkq5zPX+uaYz
TKyT/GoX8mhhq3pOaS4u07gYMQKBgQDeoF1l2XtsunF+/YjOs+QI81V16r4xfymX
c6WgcF8zpMUTZhs+BCuKBfgCasaShLB8QIPztjCyCPY5boHMNDZwAao/S7KEqvoo
0t30n1JmcmCDNT3arn2SdTVnoLc6tBA2QfZfwmldfNypWFO2KIMZIN4vGz2yO4/D
In1Bm6e5DQKBgA36GPhmkldYMuUs+/NxTUe81CSq09qpBNsE9yRtX1kGxh62tblN
mTjA+KbyGpZKnr8MFOYWHJrRRHvNWgtdjgNY316TiDdOcmuMLHDKKX7R8nYsP1rL
/hJlNgq7QOvmakQJFM1zzUnXfpdCIzxYRMCgYSt01xEdbSWANIfzXKWhAoGBAMQT
Z89BegRsPXQEZw7uv4PmlTly06qSfhZHI/QnpKG+mFiakJnRYGuDEElIs7XuKeZ1
iAIJT+AuJna0zpsEzYFe5gwzZnqUgBmehyBhhlh2mmxVYzIMhsqMcsnfciHA35p6
BD2Y4+YUB+EayzfffH+QREAm9PLapKbP5JP5PQKtAoGBAMjaHnQYPicw0lWjO6r0
8xwDcbyZIEKAZKGNp1kiETwSjfILJUjuzrWsIBPiKRrIuvFmjjs8R5sL2bdzTb+Q
xz3LbR46bN3fWjDqdmhlUplbqMdw/r69Nx0PiDKblwZQFuvSE8+hb+FJGSz3WirL
BstKaGEUzkQp5SFKsepviqfS
-----END PRIVATE KEY-----`

func TestNewJWTVerifier(t *testing.T) {
	t.Run("valid public key", func(t *testing.T) {
		verifier, err := NewJWTVerifier(testJWTPublicKeyPEM)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if verifier == nil {
			t.Fatal("expected verifier to not be nil")
		}
	})

	t.Run("empty public key", func(t *testing.T) {
		verifier, err := NewJWTVerifier("")
		if err == nil {
			t.Fatal("expected error for empty public key")
		}

		if verifier != nil {
			t.Error("expected verifier to be nil")
		}
	})
}

func TestJWTVerifier_VerifyToken(t *testing.T) {
	verifier, err := NewJWTVerifier(testJWTPublicKeyPEM)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	t.Run("valid token", func(t *testing.T) {
		testValidToken(t, verifier)
	})

	t.Run("expired token", func(t *testing.T) {
		testExpiredToken(t, verifier)
	})

	t.Run("missing subject", func(t *testing.T) {
		testMissingSubject(t, verifier)
	})

	t.Run("invalid signature", func(t *testing.T) {
		testInvalidSignature(t, verifier)
	})

	t.Run("malformed token", func(t *testing.T) {
		testMalformedToken(t, verifier)
	})

	t.Run("token without role claim", func(t *testing.T) {
		testTokenWithoutRole(t, verifier)
	})

	t.Run("empty token string", func(t *testing.T) {
		testEmptyTokenString(t, verifier)
	})
}

func testValidToken(t *testing.T, verifier *JWTVerifier) {
	t.Helper()

	token := createTestTokenRS256(t, "user-123", "admin", time.Now().Add(time.Hour))

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

func testExpiredToken(t *testing.T, verifier *JWTVerifier) {
	t.Helper()

	token := createTestTokenRS256(t, "user-123", "admin", time.Now().Add(-time.Hour))

	_, err := verifier.VerifyToken(context.Background(), token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func testMissingSubject(t *testing.T, verifier *JWTVerifier) {
	t.Helper()

	token := createTestTokenRS256WithSubject(t, "", "admin", time.Now().Add(time.Hour))

	_, err := verifier.VerifyToken(context.Background(), token)
	if err == nil {
		t.Fatal("expected error for missing subject")
	}
}

func testInvalidSignature(t *testing.T, verifier *JWTVerifier) {
	t.Helper()

	// Token signed with different key (we use another key or tampered payload)
	token := createTestTokenRS256(t, "user-123", "admin", time.Now().Add(time.Hour))
	// Tamper with token (change one character in signature part)
	if len(token) > 10 {
		token = token[:len(token)-2] + "xx"
	}

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

func testTokenWithoutRole(t *testing.T, verifier *JWTVerifier) {
	t.Helper()

	token := createTestTokenRS256WithRole(t, "user-123", "", time.Now().Add(time.Hour))

	claims, err := verifier.VerifyToken(context.Background(), token)
	if err != nil {
		t.Fatalf("expected no error for token without role, got %v", err)
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

func TestJWTVerifier_DifferentKeys(t *testing.T) {
	verifier1, err := NewJWTVerifier(testJWTPublicKeyPEM)
	if err != nil {
		t.Fatalf("failed to create verifier1: %v", err)
	}

	// Verifier1 uses test key; token is signed with test private key so it should verify
	token := createTestTokenRS256(t, "user-123", "admin", time.Now().Add(time.Hour))

	_, err = verifier1.VerifyToken(context.Background(), token)
	if err != nil {
		t.Errorf("verifier1 should verify token signed with test key, got error: %v", err)
	}
}

// createTestTokenRS256 creates a JWT signed with the test private key (RS256).
//
//nolint:unparam // subject/role vary by test
func createTestTokenRS256(t *testing.T, subject, role string, expiresAt time.Time) string {
	return createTestTokenRS256WithSubject(t, subject, role, expiresAt)
}

func createTestTokenRS256WithSubject(t *testing.T, subject, role string, expiresAt time.Time) string {
	t.Helper()
	return createTestTokenRS256WithRole(t, subject, role, expiresAt)
}

func createTestTokenRS256WithRole(t *testing.T, subject, role string, expiresAt time.Time) string {
	t.Helper()

	block, _ := pem.Decode([]byte(testJWTPrivateKeyPEM))
	if block == nil {
		t.Fatal("failed to decode test private key PEM")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse private key: %v", err)
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		t.Fatal("private key is not RSA")
	}

	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: rsaKey},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	claims := jwt.Claims{
		Subject:  subject,
		Expiry:   jwt.NewNumericDate(expiresAt),
		IssuedAt: jwt.NewNumericDate(time.Now()),
		Issuer:   "opos-auth-service",
		Audience: []string{"opos-microservices"},
		ID:       "test-jti",
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
