// Package saga provides simple saga orchestration helpers for multi-step operations
// with compensation support.
//
// This is a simple, in-memory implementation suitable for development and prototyping.
// For production use, consider using Temporal (https://temporal.io/) which provides
// durable, distributed saga orchestration with retries, timeouts, and state management.
//
// Usage:
//
//	saga := saga.New()
//	saga.AddStep("step1", executeStep1, compensateStep1)
//	saga.AddStep("step2", executeStep2, compensateStep2)
//	if err := saga.Execute(ctx); err != nil {
//	    // Compensation is automatically executed for completed steps
//	}
package saga

import (
	"context"
	"fmt"
	"log/slog"
)

// Step represents a single step in a saga.
type Step struct {
	Name       string
	Execute    func(ctx context.Context) error
	Compensate func(ctx context.Context) error // Optional, can be nil
}

// Saga orchestrates multiple steps with compensation support.
type Saga struct {
	steps []Step
}

// New creates a new saga orchestrator.
func New() *Saga {
	return &Saga{
		steps: make([]Step, 0),
	}
}

// AddStep adds a step to the saga.
// The compensate function is optional - if nil, the step cannot be compensated.
func (s *Saga) AddStep(name string, execute func(ctx context.Context) error, compensate func(ctx context.Context) error) {
	s.steps = append(s.steps, Step{
		Name:       name,
		Execute:    execute,
		Compensate: compensate,
	})
}

// Execute executes all steps in order. If any step fails, compensation is executed
// for all completed steps in reverse order.
func (s *Saga) Execute(ctx context.Context) error {
	completedSteps := make([]Step, 0)

	// Execute steps in order
	for _, step := range s.steps {
		slog.DebugContext(ctx, "Executing saga step", "step", step.Name)

		if err := step.Execute(ctx); err != nil {
			slog.ErrorContext(ctx, "Saga step failed, starting compensation", "step", step.Name, "error", err)

			// Step failed - execute compensation for all completed steps
			compensateErr := s.compensate(ctx, completedSteps)
			if compensateErr != nil {
				return fmt.Errorf("step %s failed: %w, compensation also failed: %w", step.Name, err, compensateErr)
			}

			return fmt.Errorf("step %s failed: %w", step.Name, err)
		}

		completedSteps = append(completedSteps, step)
		slog.DebugContext(ctx, "Saga step completed", "step", step.Name)
	}

	return nil
}

// compensate executes compensation for steps in reverse order.
func (s *Saga) compensate(ctx context.Context, steps []Step) error {
	var lastErr error

	// Execute compensation in reverse order
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]

		if step.Compensate == nil {
			slog.WarnContext(ctx, "Step has no compensation function, skipping", "step", step.Name)
			continue
		}

		slog.DebugContext(ctx, "Compensating saga step", "step", step.Name)

		if err := step.Compensate(ctx); err != nil {
			slog.ErrorContext(ctx, "Compensation failed", "step", step.Name, "error", err)
			lastErr = err
			// Continue compensating other steps even if one fails
		} else {
			slog.DebugContext(ctx, "Step compensation completed", "step", step.Name)
		}
	}

	return lastErr
}

// Steps returns all steps in the saga (for inspection/testing).
func (s *Saga) Steps() []Step {
	return s.steps
}
