package middleware

import (
	"fmt"
	"net/http"
	"strings"
)

// CORSConfig holds the configuration for CORS middleware.
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns a CORS configuration with sensible defaults.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
		},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-Request-ID",
		},
		ExposedHeaders: []string{
			"X-Request-ID",
		},
		AllowCredentials: true,
		MaxAge:           3600,
	}
}

// CORS returns a middleware that handles Cross-Origin Resource Sharing (CORS).
func CORS(config CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			setCORSHeaders(w, r, config)

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// setCORSHeaders sets the appropriate CORS headers on the response.
func setCORSHeaders(w http.ResponseWriter, r *http.Request, config CORSConfig) {
	setOriginHeaders(w, r, config)

	if config.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	if len(config.AllowedMethods) > 0 {
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
	}

	if len(config.AllowedHeaders) > 0 {
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
	}

	if len(config.ExposedHeaders) > 0 {
		w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
	}

	if config.MaxAge > 0 {
		w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
	}
}

func setOriginHeaders(w http.ResponseWriter, r *http.Request, config CORSConfig) {
	origin := r.Header.Get("Origin")
	hasGlobalWildcard := false

	for _, allowed := range config.AllowedOrigins {
		if allowed == "*" {
			hasGlobalWildcard = true
			break
		}
	}

	if origin != "" && isOriginAllowed(origin, config.AllowedOrigins) {
		setAllowedOriginHeader(w, origin, hasGlobalWildcard, config.AllowCredentials)
		return
	}

	if hasGlobalWildcard {
		if config.AllowCredentials {
			w.Header().Set("Access-Control-Allow-Origin", "null")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
	}
}

func setAllowedOriginHeader(w http.ResponseWriter, origin string, hasGlobalWildcard, allowCredentials bool) {
	switch {
	case hasGlobalWildcard && allowCredentials:
		w.Header().Set("Access-Control-Allow-Origin", "null")
	case hasGlobalWildcard:
		w.Header().Set("Access-Control-Allow-Origin", "*")
	default:
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}
}

// isOriginAllowed checks if the origin is in the allowed origins list.
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
		// Support wildcard subdomains like *.example.com
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*")
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}

	return false
}
