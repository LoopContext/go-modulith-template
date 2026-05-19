package queue

// Task represents a background task.
type Task struct {
	typename string
	payload  []byte
}

// NewTask creates a new Task with a type and payload.
func NewTask(typename string, payload []byte) *Task {
	return &Task{
		typename: typename,
		payload:  payload,
	}
}

// Type returns the task type/name.
func (t *Task) Type() string {
	return t.typename
}

// Payload returns the raw task payload.
func (t *Task) Payload() []byte {
	return t.payload
}
