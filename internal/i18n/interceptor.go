package i18n

import (
	"context"

	"google.golang.org/grpc"
)

// UnaryServerInterceptor is a gRPC interceptor that detects locale from Accept-Language header
// and injects it into the context.
func UnaryServerInterceptor(defaultLocale string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Detect locale from context (may extract from Accept-Language header)
		locale := DetectLocale(ctx, defaultLocale)

		// Inject locale into context for downstream use
		ctx = ContextWithLocale(ctx, locale)

		return handler(ctx, req)
	}
}
