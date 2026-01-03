package saga

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaga_Execute_Success(t *testing.T) {
	ctx := context.Background()

	saga := New()

	executed := make([]string, 0)

	saga.AddStep("step1", func(_ context.Context) error {
		executed = append(executed, "step1")
		return nil
	}, nil)

	saga.AddStep("step2", func(_ context.Context) error {
		executed = append(executed, "step2")
		return nil
	}, nil)

	err := saga.Execute(ctx)
	require.NoError(t, err)

	assert.Equal(t, []string{"step1", "step2"}, executed)
}

func TestSaga_Execute_WithCompensation(t *testing.T) {
	ctx := context.Background()

	saga := New()

	executed := make([]string, 0)
	compensated := make([]string, 0)

	saga.AddStep("step1", func(_ context.Context) error {
		executed = append(executed, "step1")
		return nil
	}, func(_ context.Context) error {
		compensated = append(compensated, "step1")
		return nil
	})

	saga.AddStep("step2", func(_ context.Context) error {
		executed = append(executed, "step2")
		return errors.New("step2 failed")
	}, func(_ context.Context) error {
		compensated = append(compensated, "step2")
		return nil
	})

	err := saga.Execute(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "step2 failed")

	// Step1 should have executed and been compensated
	assert.Equal(t, []string{"step1", "step2"}, executed)
	assert.Equal(t, []string{"step1"}, compensated) // Only step1 was compensated (step2 failed before completion)
}

func TestSaga_Execute_CompensationInReverseOrder(t *testing.T) {
	ctx := context.Background()

	saga := New()

	compensated := make([]string, 0)

	saga.AddStep("step1", func(_ context.Context) error {
		return nil
	}, func(_ context.Context) error {
		compensated = append(compensated, "step1")
		return nil
	})

	saga.AddStep("step2", func(_ context.Context) error {
		return nil
	}, func(_ context.Context) error {
		compensated = append(compensated, "step2")
		return nil
	})

	saga.AddStep("step3", func(_ context.Context) error {
		return errors.New("step3 failed")
	}, nil)

	err := saga.Execute(ctx)
	require.Error(t, err)

	// Compensation should be in reverse order: step2, then step1
	assert.Equal(t, []string{"step2", "step1"}, compensated)
}

func TestSaga_Execute_CompensationFailure(t *testing.T) {
	ctx := context.Background()

	saga := New()

	saga.AddStep("step1", func(_ context.Context) error {
		return nil
	}, func(_ context.Context) error {
		return errors.New("compensation failed")
	})

	saga.AddStep("step2", func(_ context.Context) error {
		return errors.New("step2 failed")
	}, nil)

	err := saga.Execute(ctx)
	require.Error(t, err)

	// Error should mention both the step failure and compensation failure
	assert.Contains(t, err.Error(), "step2 failed")
	assert.Contains(t, err.Error(), "compensation also failed")
}

func TestSaga_Execute_StepWithoutCompensation(t *testing.T) {
	ctx := context.Background()

	saga := New()

	saga.AddStep("step1", func(_ context.Context) error {
		return nil
	}, nil) // No compensation

	saga.AddStep("step2", func(_ context.Context) error {
		return errors.New("step2 failed")
	}, nil)

	err := saga.Execute(ctx)
	require.Error(t, err)

	// Should still fail gracefully even if step1 has no compensation
	assert.Contains(t, err.Error(), "step2 failed")
}

func TestSaga_Steps(t *testing.T) {
	saga := New()

	saga.AddStep("step1", func(_ context.Context) error {
		return nil
	}, nil)

	saga.AddStep("step2", func(_ context.Context) error {
		return nil
	}, nil)

	steps := saga.Steps()
	assert.Len(t, steps, 2)
	assert.Equal(t, "step1", steps[0].Name)
	assert.Equal(t, "step2", steps[1].Name)
}
