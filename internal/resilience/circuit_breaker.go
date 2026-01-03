// Package resilience provides resilience patterns for external service calls.
// It includes circuit breaker, retry, and timeout patterns.
package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Common errors returned by circuit breaker.
var (
	// ErrCircuitOpen is returned when the circuit breaker is open.
	ErrCircuitOpen = errors.New("circuit breaker is open")
	// ErrTooManyRequests is returned when the circuit breaker is half-open
	// and the maximum number of test requests has been reached.
	ErrTooManyRequests = errors.New("too many requests in half-open state")
)

// State represents the circuit breaker state.
type State int

const (
	// StateClosed means the circuit is closed and requests flow normally.
	StateClosed State = iota
	// StateOpen means the circuit is open and requests are rejected.
	StateOpen
	// StateHalfOpen means the circuit is testing if the service has recovered.
	StateHalfOpen
)

// String returns the string representation of the state.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig configures the circuit breaker behavior.
type CircuitBreakerConfig struct {
	// MaxFailures is the number of failures before the circuit opens.
	MaxFailures int
	// Timeout is how long the circuit stays open before testing.
	Timeout time.Duration
	// MaxHalfOpenRequests is the maximum number of requests allowed in half-open state.
	MaxHalfOpenRequests int
	// SuccessThreshold is the number of successes needed to close the circuit from half-open.
	SuccessThreshold int
	// OnStateChange is called when the circuit state changes.
	OnStateChange func(name string, from, to State)
}

// DefaultCircuitBreakerConfig returns sensible defaults.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxFailures:         5,
		Timeout:             30 * time.Second,
		MaxHalfOpenRequests: 3,
		SuccessThreshold:    2,
		OnStateChange:       nil,
	}
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	name   string
	config CircuitBreakerConfig

	mu                 sync.RWMutex
	state              State
	failures           int
	halfOpenRequests   int
	lastStateChange    time.Time
	consecutiveSuccess int
}

// NewCircuitBreaker creates a new circuit breaker with the given name and config.
func NewCircuitBreaker(name string, config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		name:            name,
		config:          config,
		state:           StateClosed,
		lastStateChange: time.Now(),
	}
}

// Execute runs the given function with circuit breaker protection.
// Returns ErrCircuitOpen if the circuit is open.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	err := fn(ctx)

	cb.afterRequest(err)

	return err
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return cb.state
}

// Failures returns the current failure count.
func (cb *CircuitBreaker) Failures() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return cb.failures
}

// beforeRequest checks if the request should be allowed.
func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return nil

	case StateOpen:
		// Check if timeout has passed
		if now.After(cb.lastStateChange.Add(cb.config.Timeout)) {
			cb.setState(StateHalfOpen)
			cb.halfOpenRequests = 1
			cb.consecutiveSuccess = 0

			return nil
		}

		return ErrCircuitOpen

	case StateHalfOpen:
		// Limit requests in half-open state
		if cb.halfOpenRequests >= cb.config.MaxHalfOpenRequests {
			return ErrTooManyRequests
		}

		cb.halfOpenRequests++

		return nil
	}

	return nil
}

// afterRequest records the result of the request.
func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.onFailure()
	} else {
		cb.onSuccess()
	}
}

// onSuccess handles a successful request.
func (cb *CircuitBreaker) onSuccess() {
	switch cb.state {
	case StateClosed:
		cb.failures = 0

	case StateHalfOpen:
		cb.consecutiveSuccess++
		if cb.consecutiveSuccess >= cb.config.SuccessThreshold {
			cb.setState(StateClosed)
			cb.failures = 0
		}
	}
}

// onFailure handles a failed request.
func (cb *CircuitBreaker) onFailure() {
	cb.failures++

	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.config.MaxFailures {
			cb.setState(StateOpen)
		}

	case StateHalfOpen:
		// Any failure in half-open state opens the circuit
		cb.setState(StateOpen)
	}
}

// setState changes the circuit state and calls the callback if configured.
func (cb *CircuitBreaker) setState(state State) {
	if cb.state == state {
		return
	}

	from := cb.state
	cb.state = state
	cb.lastStateChange = time.Now()

	if cb.config.OnStateChange != nil {
		// Call in goroutine to avoid blocking
		go cb.config.OnStateChange(cb.name, from, state)
	}
}

// Reset manually resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.setState(StateClosed)
	cb.failures = 0
	cb.halfOpenRequests = 0
	cb.consecutiveSuccess = 0
}
