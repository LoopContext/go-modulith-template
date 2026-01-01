// Package admin provides infrastructure for running administrative tasks and one-off processes.
package admin

import (
	"context"
	"fmt"
	"log/slog"
)

// Task represents an administrative task that can be executed.
type Task interface {
	// Name returns the unique identifier for this task.
	Name() string

	// Description returns a human-readable description of what this task does.
	Description() string

	// Execute runs the task with the provided context.
	Execute(ctx context.Context) error
}

// Runner manages and executes administrative tasks.
type Runner struct {
	tasks map[string]Task
}

// NewRunner creates a new admin task runner.
func NewRunner() *Runner {
	return &Runner{
		tasks: make(map[string]Task),
	}
}

// Register adds a task to the runner.
func (r *Runner) Register(task Task) {
	r.tasks[task.Name()] = task
	slog.Debug("Registered admin task", "task", task.Name())
}

// Run executes a task by name.
func (r *Runner) Run(ctx context.Context, name string) error {
	task, exists := r.tasks[name]
	if !exists {
		return fmt.Errorf("unknown task: %s", name)
	}

	slog.Info("Running admin task", "task", name, "description", task.Description())

	if err := task.Execute(ctx); err != nil {
		return fmt.Errorf("task %s failed: %w", name, err)
	}

	slog.Info("Admin task completed successfully", "task", name)

	return nil
}

// List returns all registered tasks.
func (r *Runner) List() []Task {
	tasks := make([]Task, 0, len(r.tasks))
	for _, task := range r.tasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// Has checks if a task is registered.
func (r *Runner) Has(name string) bool {
	_, exists := r.tasks[name]
	return exists
}

