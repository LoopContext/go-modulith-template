package telemetry

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	// gRPC metrics
	grpcRequestsTotal  *Counter
	grpcErrorsTotal    *Counter
	grpcLatencySeconds *Histogram
)

// InitGRPCMetrics initializes the standardized gRPC metrics.
func InitGRPCMetrics() error {
	var err error

	grpcRequestsTotal, err = NewCounter("grpc_requests_total", "Total number of gRPC requests")
	if err != nil {
		return fmt.Errorf("failed to create grpc_requests_total metric: %w", err)
	}

	grpcErrorsTotal, err = NewCounter("grpc_errors_total", "Total number of gRPC errors")
	if err != nil {
		return fmt.Errorf("failed to create grpc_errors_total metric: %w", err)
	}

	grpcLatencySeconds, err = NewHistogram("grpc_request_duration_seconds", "gRPC request latency in seconds", "s")
	if err != nil {
		return fmt.Errorf("failed to create grpc_request_duration_seconds metric: %w", err)
	}

	return nil
}

// UnaryServerInterceptor returns a new unary server interceptor that records metrics.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Execute handler
		resp, err := handler(ctx, req)

		duration := time.Since(start)
		method := info.FullMethod
		code := status.Code(err).String()

		// Record metrics
		if grpcRequestsTotal != nil {
			grpcRequestsTotal.WithAttributes(
				Attr("method", method),
				Attr("code", code),
			).Inc(ctx)
		}

		if err != nil && grpcErrorsTotal != nil {
			grpcErrorsTotal.WithAttributes(
				Attr("method", method),
				Attr("code", code),
			).Inc(ctx)
		}

		if grpcLatencySeconds != nil {
			grpcLatencySeconds.WithAttributes(
				RecordAttr("method", method),
				RecordAttr("code", code),
			).RecordDuration(ctx, duration)
		}

		return resp, err
	}
}
