package admin

import (
	"context"
	"errors"
	"testing"
)

type mockTask struct {
	name        string
	description string
	executeFunc func(context.Context) error
}

func (m *mockTask) Name() string        { return m.name }
func (m *mockTask) Description() string { return m.description }
func (m *mockTask) Execute(ctx context.Context) error {
	if m.executeFunc != nil {
		return m.executeFunc(ctx)
	}

	return nil
}

func TestRunner_Register(t *testing.T) {
	runner := NewRunner()
	task := &mockTask{name: "test", description: "test task"}

	runner.Register(task)

	if !runner.Has("test") {
		t.Error("Expected task to be registered")
	}
}

func TestRunner_Run(t *testing.T) {
	tests := []struct {
		name      string
		taskName  string
		task      *mockTask
		wantError bool
	}{
		{
			name:     "successful execution",
			taskName: "success",
			task: &mockTask{
				name:        "success",
				description: "successful task",
			},
			wantError: false,
		},
		{
			name:     "failed execution",
			taskName: "fail",
			task: &mockTask{
				name:        "fail",
				description: "failing task",
				executeFunc: func(context.Context) error {
					return errors.New("task error")
				},
			},
			wantError: true,
		},
		{
			name:      "unknown task",
			taskName:  "unknown",
			task:      nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewRunner()
			if tt.task != nil {
				runner.Register(tt.task)
			}

			err := runner.Run(context.Background(), tt.taskName)
			if (err != nil) != tt.wantError {
				t.Errorf("Run() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestRunner_List(t *testing.T) {
	runner := NewRunner()
	task1 := &mockTask{name: "task1", description: "first task"}
	task2 := &mockTask{name: "task2", description: "second task"}

	runner.Register(task1)
	runner.Register(task2)

	tasks := runner.List()
	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}
}

