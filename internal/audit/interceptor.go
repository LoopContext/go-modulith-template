package audit

import (
	"context"
	"strings"
	"time"

	"github.com/LoopContext/go-modulith-template/internal/authn"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor returns a new unary server interceptor that audits requests.
func UnaryServerInterceptor(logger Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// filter read-only methods
		if isReadOnly(info.FullMethod) {
			return handler(ctx, req)
		}

		startTime := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(startTime)

		// Extract metadata
		userID, _ := authn.UserIDFromContext(ctx)
		ip := getClientIP(ctx)
		userAgent := getUserAgent(ctx)

		// Determine status
		success := true
		errorMsg := ""

		if err != nil {
			success = false

			st, _ := status.FromError(err)
			if st.Code() != codes.OK {
				errorMsg = st.Message()
			}
		}

		// Parse Method for Resource/Action
		// Format: /package.Service/Method
		parts := strings.Split(info.FullMethod, "/")
		resource := "unknown"
		action := "unknown"

		if len(parts) >= 3 {
			resource = parts[1] // package.Service
			action = parts[2]   // Method
		}

		// Log asynchronously to avoid blocking response
		// In a production system with high volume, this might go to a queue
		// For now, the event bus in the logger handles it non-blocking enough
		go func() {
			// Detach cancellation but preserve request-scoped values for audit logging.
			logCtx := context.WithoutCancel(ctx)

			logger.Log(logCtx, LogParams{
				UserID:    userID,
				ActorID:   userID, // Primary actor is the authenticated user
				Action:    action,
				Resource:  resource,
				IPAddress: ip,
				UserAgent: userAgent,
				Success:   success,
				ErrorMsg:  errorMsg,
				Metadata: map[string]any{
					"duration_ms": duration.Milliseconds(),
					"method":      info.FullMethod,
				},
				// Note: Request/Response bodies (OldValue/NewValue) are hard to capture generically
				// without reflection or specific proto handling.
				// For detailed field auditing, manual calls inside services are still better.
				// This interceptor provides a "Who doing What" baseline.
			})
		}()

		return resp, err
	}
}

// MetadataFromContext extracts metadata from the context.
func MetadataFromContext(ctx context.Context) map[string]any {
	userID, _ := authn.UserIDFromContext(ctx)
	ip := getClientIP(ctx)
	userAgent := getUserAgent(ctx)

	return map[string]any{
		"user_id":    userID,
		"ip_address": ip,
		"user_agent": userAgent,
	}
}

func isReadOnly(method string) bool {
	// Simple heuristic: if method starts with specific verbs
	parts := strings.Split(method, "/")
	if len(parts) < 3 {
		return false
	}

	methodName := parts[2]

	prefixes := []string{"Get", "List", "View", "Check", "Watch", "Stream"}
	for _, p := range prefixes {
		if strings.HasPrefix(methodName, p) {
			return true
		}
	}

	return false
}

func getClientIP(ctx context.Context) string {
	if p, ok := peer.FromContext(ctx); ok {
		return p.Addr.String()
	}

	return ""
}

func getUserAgent(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if userAgents := md.Get("user-agent"); len(userAgents) > 0 {
			return userAgents[0]
		}

		if userAgents := md.Get("grpcgateway-user-agent"); len(userAgents) > 0 {
			return userAgents[0]
		}
	}

	return ""
}
