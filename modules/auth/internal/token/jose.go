package token

import (
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// TokenService handles JOSE token operations
type TokenService struct {
	signer jose.Signer
	key    []byte
}

func NewTokenService(secretKey string) (*TokenService, error) {
	key := []byte(secretKey)
	sig, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: key}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		return nil, err
	}
	return &TokenService{
		signer: sig,
		key:    key,
	}, nil
}

type Claims struct {
	UserID string   `json:"sub"`
	Role   string   `json:"role"`
	Scope  []string `json:"scope,omitempty"`
}

func (s *TokenService) CreateToken(userID, role string, duration time.Duration) (string, error) {
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
		return "", err
	}
	return raw, nil
}

func (s *TokenService) VerifyToken(tokenString string) (*Claims, error) {
	tok, err := jwt.ParseSigned(tokenString, []jose.SignatureAlgorithm{jose.HS256})
	if err != nil {
		return nil, err
	}

	claims := jwt.Claims{}
	privateClaims := make(map[string]interface{})

	if err := tok.Claims(s.key, &claims, &privateClaims); err != nil {
		return nil, err
	}

	if err := claims.Validate(jwt.Expected{Time: time.Now()}); err != nil {
		return nil, err
	}

	role, _ := privateClaims["role"].(string)

	return &Claims{
		UserID: claims.Subject,
		Role:   role,
	}, nil
}
