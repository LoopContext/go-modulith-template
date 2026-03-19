package authn

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// Verifier validates an auth token and returns claims.
type Verifier interface {
	VerifyToken(ctx context.Context, tokenString string) (*Claims, error)
}

// JWTVerifier verifies RS256 JWT tokens that include a "sub" and a "role" claim.
type JWTVerifier struct {
	publicKey *rsa.PublicKey
}

// NewJWTVerifier creates a verifier for RS256 tokens using the given PEM-encoded RSA public key.
func NewJWTVerifier(publicKeyPEM string) (*JWTVerifier, error) {
	if publicKeyPEM == "" {
		return nil, fmt.Errorf("jwt public key PEM is empty")
	}

	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode JWT public key PEM")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		// Try PKCS1
		pub, err = x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JWT public key: %w", err)
		}
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("JWT public key must be RSA, got %T", pub)
	}

	return &JWTVerifier{publicKey: rsaPub}, nil
}

// VerifyToken parses and validates the given token string (RS256), returning claims if valid.
func (v *JWTVerifier) VerifyToken(_ context.Context, tokenString string) (*Claims, error) {
	tok, err := jwt.ParseSigned(tokenString, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	stdClaims := jwt.Claims{}
	privateClaims := make(map[string]interface{})

	if err := tok.Claims(v.publicKey, &stdClaims, &privateClaims); err != nil {
		return nil, fmt.Errorf("failed to deserialize claims: %w", err)
	}

	if err := stdClaims.Validate(jwt.Expected{
		Time:   time.Now(),
		Issuer: "opos-auth-service",
	}); err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	if err := validateAudience(stdClaims.Audience); err != nil {
		return nil, err
	}

	if stdClaims.ID == "" {
		return nil, fmt.Errorf("missing token ID (jti)")
	}

	role, _ := privateClaims["role"].(string)

	if stdClaims.Subject == "" {
		return nil, fmt.Errorf("missing subject claim")
	}

	return &Claims{
		UserID: stdClaims.Subject,
		Role:   role,
	}, nil
}

func validateAudience(audiences []string) error {
	if len(audiences) == 0 {
		return nil
	}

	for _, aud := range audiences {
		if aud == "opos-microservices" || aud == "opos-frontend" {
			return nil
		}
	}

	return fmt.Errorf("invalid token audience")
}
