package authn

import (
	"context"
	"fmt"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// Verifier validates an auth token and returns claims.
type Verifier interface {
	VerifyToken(ctx context.Context, tokenString string) (*Claims, error)
}

// JWTVerifier verifies HS256 JWT tokens that include a "sub" and a "role" claim.
type JWTVerifier struct {
	key []byte
}

// NewJWTVerifier creates a verifier for HS256 tokens.
func NewJWTVerifier(secret string) (*JWTVerifier, error) {
	if secret == "" {
		return nil, fmt.Errorf("jwt secret is empty")
	}

	return &JWTVerifier{key: []byte(secret)}, nil
}

// VerifyToken parses and validates the given token string, returning claims if valid.
func (v *JWTVerifier) VerifyToken(_ context.Context, tokenString string) (*Claims, error) {
	tok, err := jwt.ParseSigned(tokenString, []jose.SignatureAlgorithm{jose.HS256})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	stdClaims := jwt.Claims{}
	privateClaims := make(map[string]interface{})

	if err := tok.Claims(v.key, &stdClaims, &privateClaims); err != nil {
		return nil, fmt.Errorf("failed to deserialize claims: %w", err)
	}

	if err := stdClaims.Validate(jwt.Expected{Time: time.Now()}); err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
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
