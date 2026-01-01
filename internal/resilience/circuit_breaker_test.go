package resilience_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/cmelgarejo/go-modulith-template/internal/resilience"
)

var errService = errors.New("service error")

func TestCircuitBreaker_ClosedState(t *testing.T) {
	cb := resilience.NewCircuitBreaker("test", resilience.DefaultCircuitBreakerConfig())

	// Should start closed
	if cb.State() != resilience.StateClosed {
		t.Errorf("expected StateClosed, got %v", cb.State())
	}

	// Successful requests should work
	err := cb.Execute(context.Background(), func(_ context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCircuitBreaker_OpensAfterFailures(t *testing.T) {
	config := resilience.CircuitBreakerConfig{
		MaxFailures:         3,
		Timeout:             1 * time.Second,
		MaxHalfOpenRequests: 1,
		SuccessThreshold:    1,
	}
	cb := resilience.NewCircuitBreaker("test", config)

	// Generate failures
	for i := 0; i < 3; i++ {
		_ = cb.Execute(context.Background(), func(_ context.Context) error {
			return errService
		})
	}

	// Circuit should be open now
	if cb.State() != resilience.StateOpen {
		t.Errorf("expected StateOpen, got %v", cb.State())
	}

	// New requests should be rejected
	err := cb.Execute(context.Background(), func(_ context.Context) error {
		return nil
	})

	if !errors.Is(err, resilience.ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	config := resilience.CircuitBreakerConfig{
		MaxFailures:         2,
		Timeout:             50 * time.Millisecond,
		MaxHalfOpenRequests: 2,
		SuccessThreshold:    1,
	}
	cb := resilience.NewCircuitBreaker("test", config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(_ context.Context) error {
			return errService
		})
	}

	if cb.State() != resilience.StateOpen {
		t.Fatalf("expected StateOpen, got %v", cb.State())
	}

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Next request should transition to half-open
	_ = cb.Execute(context.Background(), func(_ context.Context) error {
		return nil
	})

	// Should be closed after success in half-open
	if cb.State() != resilience.StateClosed {
		t.Errorf("expected StateClosed after success in half-open, got %v", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	config := resilience.CircuitBreakerConfig{
		MaxFailures:         2,
		Timeout:             50 * time.Millisecond,
		MaxHalfOpenRequests: 2,
		SuccessThreshold:    2,
	}
	cb := resilience.NewCircuitBreaker("test", config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(_ context.Context) error {
			return errService
		})
	}

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Fail in half-open state
	_ = cb.Execute(context.Background(), func(_ context.Context) error {
		return errService
	})

	// Should be open again
	if cb.State() != resilience.StateOpen {
		t.Errorf("expected StateOpen after failure in half-open, got %v", cb.State())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := resilience.CircuitBreakerConfig{
		MaxFailures:         2,
		Timeout:             1 * time.Minute,
		MaxHalfOpenRequests: 1,
		SuccessThreshold:    1,
	}
	cb := resilience.NewCircuitBreaker("test", config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(_ context.Context) error {
			return errService
		})
	}

	if cb.State() != resilience.StateOpen {
		t.Fatalf("expected StateOpen, got %v", cb.State())
	}

	// Reset
	cb.Reset()

	if cb.State() != resilience.StateClosed {
		t.Errorf("expected StateClosed after reset, got %v", cb.State())
	}

	// Should work again
	err := cb.Execute(context.Background(), func(_ context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("unexpected error after reset: %v", err)
	}
}

func TestCircuitBreaker_StateCallback(t *testing.T) {
	var (
		mu                 sync.Mutex
		called             bool
		fromState, toState resilience.State
	)

	config := resilience.CircuitBreakerConfig{
		MaxFailures:         1,
		Timeout:             1 * time.Second,
		MaxHalfOpenRequests: 1,
		SuccessThreshold:    1,
		OnStateChange: func(_ string, from, to resilience.State) {
			mu.Lock()
			defer mu.Unlock()
			called = true
			fromState = from
			toState = to
		},
	}
	cb := resilience.NewCircuitBreaker("test", config)

	// Trigger state change
	_ = cb.Execute(context.Background(), func(_ context.Context) error {
		return errService
	})

	// Wait for async callback with retries
	for i := 0; i < 100; i++ {
		mu.Lock()
		wasCalled := called
		mu.Unlock()
		if wasCalled {
			break
		}
		time.Sleep(1 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()

	if !called {
		t.Error("state change callback was not called")
	}

	if fromState != resilience.StateClosed {
		t.Errorf("expected fromState to be StateClosed, got %v", fromState)
	}

	if toState != resilience.StateOpen {
		t.Errorf("expected toState to be StateOpen, got %v", toState)
	}
}
