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

	vals := md.Get("authorization")
	if len(vals) == 0 {
		return "", fmt.Errorf("authorization header not found")
	}

	// Take the first value.
	v := strings.TrimSpace(vals[0])
	if v == "" {
		return "", fmt.Errorf("authorization header empty")
	}

	const prefix = "bearer "
	if len(v) < len(prefix) || strings.ToLower(v[:len(prefix)]) != prefix {
		return "", fmt.Errorf("authorization header is not bearer token")
	}

	token := strings.TrimSpace(v[len(prefix):])
	if token == "" {
		return "", fmt.Errorf("bearer token empty")
	}

	return token, nil
}


