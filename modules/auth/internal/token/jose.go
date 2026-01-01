// Package token provides token generation and validation services.
package token

import (
	"fmt"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// Service handles JOSE token operations
type Service struct {
	signer jose.Signer
	key    []byte
}

// NewService creates a new instance of Service with the given secret key
// The secret key must be at least 32 bytes (256 bits) for HS256 algorithm
func NewService(secretKey string) (*Service, error) {
	if secretKey == "" {
		return nil, fmt.Errorf("JWT secret key cannot be empty")
	}

	key := []byte(secretKey)
	if len(key) < 32 {
		return nil, fmt.Errorf("JWT secret key must be at least 32 bytes (256 bits) for HS256, got %d bytes", len(key))
	}

	sig, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: key}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	return &Service{
		signer: sig,
		key:    key,
	}, nil
}

// Claims represents the custom claims in the JWT
type Claims struct {
	Subject   string   `json:"sub"`
	Role      string   `json:"role"`
	Scope     []string `json:"scope,omitempty"`
	ExpiresAt int64    `json:"exp"`
}

// CreateToken generates a new signed JWT for the given user and role
func (s *Service) CreateToken(userID, role string, duration time.Duration) (string, error) {
	claims := jwt.Claims{
		Subject:   userID,
		Expiry:    jwt.NewNumericDate(time.Now().Add(duration)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
	}
	privateClaims := map[string]interface{}{
		"role": role,
	}

	raw, err := jwt.Signed(s.signer).Claims(claims).Claims(privateClaims).Serialize()
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return raw, nil
}

// VerifyToken parses and validates the given token string, returning the claims if valid
func (s *Service) VerifyToken(tokenString string) (*Claims, error) {
	tok, err := jwt.ParseSigned(tokenString, []jose.SignatureAlgorithm{jose.HS256})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims := jwt.Claims{}
	privateClaims := make(map[string]interface{})

	if err := tok.Claims(s.key, &claims, &privateClaims); err != nil {
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
	}, nil
}
