package middleware

import (
	"net/http"
)

// SecurityHeaders returns a middleware that adds standard security headers to the response.
func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// HSTS: Strict-Transport-Security (1 year)
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

			// X-Frame-Options: Prevent clickjacking
			w.Header().Set("X-Frame-Options", "DENY")

			// X-Content-Type-Options: Prevent MIME sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// Referrer-Policy: Control referrer information
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Content-Security-Policy: Modern security policy
			// Note: This is a restrictive default, might need adjustments based on UI needs.
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; object-src 'none';")

			// X-XSS-Protection: Legacy but still useful for some browsers
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			next.ServeHTTP(w, r)
		})
	}
}
