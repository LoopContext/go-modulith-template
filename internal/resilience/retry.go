// Package resilience provides resilience patterns for external service calls.
package resilience

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"
)

// RetryConfig configures the retry behavior.
type RetryConfig struct {
	// MaxAttempts is the maximum number of attempts (including the first one).
	MaxAttempts int
	// InitialDelay is the delay before the first retry.
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries.
	MaxDelay time.Duration
	// Multiplier is the factor by which the delay increases.
	Multiplier float64
	// Jitter adds randomness to the delay (0.0 to 1.0).
	Jitter float64
	// RetryIf determines if an error should be retried.
	RetryIf func(error) bool
}

// DefaultRetryConfig returns sensible defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
		RetryIf:      nil, // Retry all errors
	}
}

// Retry executes the given function with retry logic.
// It uses exponential backoff with jitter.
//
//nolint:wrapcheck // Context errors are passed through unchanged
func Retry(ctx context.Context, config RetryConfig, fn func(context.Context) error) error {
	var (
		lastErr error
		delay   = config.InitialDelay
	)

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Check context before attempting
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry this error
		if config.RetryIf != nil && !config.RetryIf(err) {
			return err
		}

		// Don't sleep after the last attempt
		if attempt == config.MaxAttempts-1 {
			break
		}

		// Calculate delay with jitter
		jitteredDelay := addJitter(delay, config.Jitter)

		// Wait before next attempt
		select {
		case <-time.After(jitteredDelay):
		case <-ctx.Done():
			return ctx.Err()
		}

		// Increase delay for next attempt
		delay = time.Duration(float64(delay) * config.Multiplier)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
	}

	return lastErr
}

// addJitter adds randomness to the delay.
func addJitter(delay time.Duration, jitter float64) time.Duration {
	if jitter <= 0 {
		return delay
	}

	// jitter is a percentage, so we calculate a random factor between (1-jitter) and (1+jitter)
	jitterFactor := 1 + (rand.Float64()*2-1)*jitter //nolint:gosec // Random jitter doesn't need crypto/rand

	return time.Duration(float64(delay) * jitterFactor)
}

// RetryWithBackoff is a convenience function that uses default retry config
// with exponential backoff.
func RetryWithBackoff(ctx context.Context, maxAttempts int, fn func(context.Context) error) error {
	config := DefaultRetryConfig()
	config.MaxAttempts = maxAttempts

	return Retry(ctx, config, fn)
}

// IsRetryable checks if an error is a common retryable error.
// This can be used as the RetryIf function.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Context errors are not retryable
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Circuit breaker errors are not retryable
	if errors.Is(err, ErrCircuitOpen) || errors.Is(err, ErrTooManyRequests) {
		return false
	}

	// By default, retry all other errors
	return true
}

// ExponentialBackoff calculates the delay for a given attempt using exponential backoff.
func ExponentialBackoff(attempt int, initialDelay, maxDelay time.Duration, multiplier float64) time.Duration {
	delay := float64(initialDelay) * math.Pow(multiplier, float64(attempt))

	if delay > float64(maxDelay) {
		return maxDelay
	}

	return time.Duration(delay)
}

