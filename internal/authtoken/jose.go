// Package authtoken provides token generation and validation services using RS256.
package authtoken

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// Service handles JOSE token operations with RS256 (asymmetric) signing.
type Service struct {
	signer    jose.Signer
	publicKey *rsa.PublicKey
}

// NewService creates a new Service using the given PEM-encoded RSA private key (RS256).
// The public key for verification is derived from the private key.
func NewService(privateKeyPEM string) (*Service, error) {
	if privateKeyPEM == "" {
		return nil, fmt.Errorf("JWT private key PEM cannot be empty")
	}

	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode JWT private key PEM")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8
		key, parseErr := x509.ParsePKCS8PrivateKey(block.Bytes)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse JWT private key: %w", err)
		}

		var ok bool

		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("JWT private key must be RSA, got %T", key)
		}
	}

	sig, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: privateKey},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create RS256 signer: %w", err)
	}

	return &Service{
		signer:    sig,
		publicKey: &privateKey.PublicKey,
	}, nil
}

// Claims represents the custom claims in the JWT
type Claims struct {
	Subject   string   `json:"sub"`
	Role      string   `json:"role"`
	Scope     []string `json:"scope,omitempty"`
	ExpiresAt int64    `json:"exp"`
	Issuer    string   `json:"iss,omitempty"`
	Audience  []string `json:"aud,omitempty"`
	ID        string   `json:"jti,omitempty"`
}

// CreateToken generates a new signed JWT for the given user and role (RS256).
func (s *Service) CreateToken(userID, role string, duration time.Duration) (string, string, error) {
	now := time.Now()
	jti := fmt.Sprintf("%d-%s", now.UnixNano(), userID)

	claims := jwt.Claims{
		Subject:   userID,
		Expiry:    jwt.NewNumericDate(now.Add(duration)),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		Issuer:    "loopcontext-auth-service",
		Audience:  []string{"loopcontext-services", "loopcontext-frontend"},
		ID:        jti,
	}
	privateClaims := map[string]interface{}{
		"role": role,
	}

	raw, err := jwt.Signed(s.signer).Claims(claims).Claims(privateClaims).Serialize()
	if err != nil {
		return "", "", fmt.Errorf("failed to sign token: %w", err)
	}

	return raw, jti, nil
}

// VerifyToken parses and validates the given token string (RS256), returning the claims if valid.
func (s *Service) VerifyToken(tokenString string) (*Claims, error) {
	tok, err := jwt.ParseSigned(tokenString, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims := jwt.Claims{}
	privateClaims := make(map[string]interface{})

	if err := tok.Claims(s.publicKey, &claims, &privateClaims); err != nil {
		return nil, fmt.Errorf("failed to deserialize claims: %w", err)
	}

	if err := claims.Validate(jwt.Expected{Time: time.Now()}); err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	role, _ := privateClaims["role"].(string)

	var expiresAt int64
	if claims.Expiry != nil {
		expiresAt = claims.Expiry.Time().Unix()
	}

	return &Claims{
		Subject:   claims.Subject,
		Role:      role,
		ExpiresAt: expiresAt,
		ID:        claims.ID,
	}, nil
}
