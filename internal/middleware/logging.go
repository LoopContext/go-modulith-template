// Package middleware provides HTTP middleware components.
package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture status code and bytes written.
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // default status
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

//nolint:wrapcheck // Wrapper passes through ResponseWriter errors unchanged
func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n

	return n, err
}

// LoggingConfig configures the logging middleware behavior.
type LoggingConfig struct {
	// SkipPaths are paths that should not be logged (e.g., health checks).
	SkipPaths []string
	// LogRequestBody enables logging of request body (use with caution for PII).
	LogRequestBody bool
	// LogResponseBody enables logging of response body (use with caution).
	LogResponseBody bool
	// SlowRequestThreshold marks requests slower than this as "slow".
	SlowRequestThreshold time.Duration
}

// DefaultLoggingConfig returns sensible defaults for request logging.
func DefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		SkipPaths: []string{
			"/healthz",
			"/readyz",
			"/metrics",
		},
		LogRequestBody:       false,
		LogResponseBody:      false,
		SlowRequestThreshold: 500 * time.Millisecond,
	}
}

// Logging returns a middleware that logs HTTP requests.
// It captures method, path, status code, duration, and integrates with request ID.
func Logging(config LoggingConfig) func(http.Handler) http.Handler {
	skipSet := make(map[string]struct{}, len(config.SkipPaths))
	for _, path := range config.SkipPaths {
		skipSet[path] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip logging for certain paths
			if _, skip := skipSet[r.URL.Path]; skip {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			wrapped := newResponseWriter(w)

			// Process request
			next.ServeHTTP(wrapped, r)

			// Calculate duration
			duration := time.Since(start)

			// Build log attributes
			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", wrapped.statusCode),
				slog.Duration("duration", duration),
				slog.Int("bytes", wrapped.bytesWritten),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
			}

			// Add request ID if present
			if reqID := GetRequestID(r.Context()); reqID != "" {
				attrs = append(attrs, slog.String("request_id", reqID))
			}

			// Add query params if present (be careful with PII)
			if r.URL.RawQuery != "" {
				attrs = append(attrs, slog.String("query", r.URL.RawQuery))
			}

			// Determine log level based on status and duration
			ctx := r.Context()
			msg := "HTTP request"

			switch {
			case wrapped.statusCode >= 500:
				slog.LogAttrs(ctx, slog.LevelError, msg, attrs...)
			case wrapped.statusCode >= 400:
				slog.LogAttrs(ctx, slog.LevelWarn, msg, attrs...)
			case duration > config.SlowRequestThreshold:
				attrs = append(attrs, slog.Bool("slow", true))
				slog.LogAttrs(ctx, slog.LevelWarn, msg, attrs...)
			default:
				slog.LogAttrs(ctx, slog.LevelInfo, msg, attrs...)
			}
		})
	}
}

// LoggingWithDefaults returns the logging middleware with default configuration.
func LoggingWithDefaults() func(http.Handler) http.Handler {
	return Logging(DefaultLoggingConfig())
}

