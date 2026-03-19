package authn

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Cookie names used for auth tokens (HttpOnly cookies).
const (
	AccessTokenCookieName  = "access_token"
	RefreshTokenCookieName = "refresh_token"
)

// InterceptorConfig configures the gRPC auth interceptor.
type InterceptorConfig struct {
	Verifier      Verifier
	PublicMethods map[string]struct{}
}

// UnaryServerInterceptor validates Bearer tokens and injects claims into context.
//
// If a method is in PublicMethods, it will bypass authentication.
func UnaryServerInterceptor(cfg InterceptorConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if _, ok := cfg.PublicMethods[info.FullMethod]; ok {
			return handler(ctx, req)
		}

		if cfg.Verifier == nil {
			return nil, status.Error(codes.Internal, "auth verifier not configured")
		}

		tokenString, err := bearerTokenFromMetadata(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "missing or invalid authorization token")
		}

		claims, err := cfg.Verifier.VerifyToken(ctx, tokenString)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
		}

		ctx = ContextWithClaims(ctx, *claims)

		return handler(ctx, req)
	}
}

func bearerTokenFromMetadata(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", fmt.Errorf("missing metadata")
	}

	// 1. Try Authorization header (standard Bearer token)
	vals := md.Get("authorization")
	if len(vals) > 0 {
		v := strings.TrimSpace(vals[0])

		const prefix = "bearer "
		if len(v) >= len(prefix) && strings.ToLower(v[:len(prefix)]) == prefix {
			token := strings.TrimSpace(v[len(prefix):])
			if token != "" {
				return token, nil
			}
		}
	}

	// 2. Try Cookie header (HttpOnly cookies)
	cookies := md.Get("cookie")
	if len(cookies) > 0 {
		// Cookie header can contain multiple cookies: "name1=val1; name2=val2"
		for _, cookieStr := range cookies {
			parts := strings.Split(cookieStr, ";")

			for _, part := range parts {
				part = strings.TrimSpace(part)

				prefix := AccessTokenCookieName + "="
				if strings.HasPrefix(part, prefix) {
					return strings.TrimPrefix(part, prefix), nil
				}
			}
		}
	}

	return "", fmt.Errorf("authorization token not found in header or cookie")
}
