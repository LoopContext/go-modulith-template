// Package middleware provides HTTP middleware for request handling.
package middleware

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// Timeout creates a middleware that enforces a maximum duration for request handling.
// If the timeout is exceeded, the middleware returns a 504 Gateway Timeout response.
//
// The timeout is propagated to the request context, allowing handlers to respect
// the deadline and cancel long-running operations.
func Timeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip timeout for WebSocket connections
			if strings.HasPrefix(r.URL.Path, "/ws") {
				next.ServeHTTP(w, r)
				return
			}

			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Create a response writer that tracks if a response was written
			rw := &timeoutResponseWriter{
				ResponseWriter: w,
				timeout:        timeout,
			}

			// Handle the request with timeout context
			done := make(chan bool, 1)

			go func() {
				next.ServeHTTP(rw, r.WithContext(ctx))

				done <- true
			}()

			select {
			case <-done:
				// Request completed within timeout
			case <-ctx.Done():
				// Timeout exceeded
				if !rw.wroteHeader {
					rw.ResponseWriter.WriteHeader(http.StatusGatewayTimeout)
					_, _ = rw.ResponseWriter.Write([]byte("Gateway Timeout"))
				}
			}
		})
	}
}

// timeoutResponseWriter wraps http.ResponseWriter to track if headers were written.
type timeoutResponseWriter struct {
	http.ResponseWriter
	timeout     time.Duration
	wroteHeader bool
}

func (rw *timeoutResponseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *timeoutResponseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}

	n, err := rw.ResponseWriter.Write(b)
	if err != nil {
		return n, fmt.Errorf("failed to write response: %w", err)
	}

	return n, nil
}

func (rw *timeoutResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := rw.ResponseWriter.(http.Hijacker); ok {
		conn, rw, err := hj.Hijack()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to hijack connection: %w", err)
		}

		return conn, rw, nil
	}

	return nil, nil, fmt.Errorf("http.ResponseWriter does not implement http.Hijacker")
}
