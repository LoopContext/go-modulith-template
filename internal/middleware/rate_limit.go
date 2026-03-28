package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/LoopContext/go-modulith-template/internal/cache"
	"golang.org/x/time/rate"
)

// RateLimiter holds the rate limiter configuration and state.
type RateLimiter struct {
	limiters map[string]*limiterWithUsage
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	cache    cache.Cache // Optional: for centralized rate limiting
	maxAge   time.Duration
}

type limiterWithUsage struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// NewRateLimiter creates a new rate limiter middleware.
// rate is requests per second, burst is the maximum burst size.
func NewRateLimiter(rps int, burst int, c cache.Cache) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*limiterWithUsage),
		rate:     rate.Limit(rps),
		burst:    burst,
		cache:    c,
		maxAge:   10 * time.Minute,
	}

	// Start cleanup goroutine
	go rl.CleanupExpired(5 * time.Minute)

	return rl
}

// getLimiter returns a rate limiter for the given key (typically IP address).
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	l, exists := rl.limiters[key]
	if exists {
		l.lastAccess = time.Now()
		return l.limiter
	}

	l = &limiterWithUsage{
		limiter:    rate.NewLimiter(rl.rate, rl.burst),
		lastAccess: time.Now(),
	}
	rl.limiters[key] = l

	return l.limiter
}

// Middleware returns an HTTP middleware that rate limits requests.
func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use IP address as the key for rate limiting
			ip := getIPAddress(r)

			// Centralized limiting check if cache is available
			if rl.cache != nil {
				if blocked := rl.checkCentralizedLimit(r.Context(), ip); blocked {
					http.Error(w, "Too Many Requests (Centralized)", http.StatusTooManyRequests)
					return
				}
			}

			limiter := rl.getLimiter(ip)

			if !limiter.Allow() {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (rl *RateLimiter) checkCentralizedLimit(ctx context.Context, ip string) bool {
	key := "ratelimit:" + ip

	count, err := rl.cache.Increment(ctx, key)
	if err != nil {
		return false
	}

	if count == 1 {
		_ = rl.cache.Expire(ctx, key, 1*time.Second)
	}

	// If centralized count exceeds (rps * 2) or some threshold, block.
	return count > int64(rl.burst*2)
}

// CleanupExpired removes limiters that haven't been used recently to prevent memory leaks.
func (rl *RateLimiter) CleanupExpired(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()

		now := time.Now()
		for ip, l := range rl.limiters {
			if now.Sub(l.lastAccess) > rl.maxAge {
				delete(rl.limiters, ip)
			}
		}

		rl.mu.Unlock()
	}
}

// getIPAddress extracts the client IP address from the request.
// It prioritizes X-Forwarded-For but takes only the first IP to mitigate spoofing.
func getIPAddress(r *http.Request) string {
	// Check X-Forwarded-For header (set by proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take only the first IP in the list to prevent spoofing of internal IPs
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
