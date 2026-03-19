// Package audit provides the audit logging interface and event bus implementation.
package audit

import (
	"context"

	"github.com/cmelgarejo/go-modulith-template/internal/events"
)

// LogParams defines the data for an audit log entry.
type LogParams struct {
	UserID     string         `json:"user_id"`
	ActorID    string         `json:"actor_id"`
	Action     string         `json:"action"`
	Resource   string         `json:"resource"`
	ResourceID string         `json:"resource_id"`
	OldValue   any            `json:"old_value"`
	NewValue   any            `json:"new_value"`
	Metadata   map[string]any `json:"metadata"`
	IPAddress  string         `json:"ip_address"`
	UserAgent  string         `json:"user_agent"`
	Success    bool           `json:"success"`
	ErrorMsg   string         `json:"error_msg"`
}

// Logger is the interface for recording audit events.
type Logger interface {
	Log(ctx context.Context, params LogParams)
}

type eventBusLogger struct {
	bus *events.Bus
}

// NewEventBusLogger creates a new Logger that publishes audit events to the event bus.
func NewEventBusLogger(bus *events.Bus) Logger {
	return &eventBusLogger{bus: bus}
}

// Log publishes an audit event to the event bus.
func (l *eventBusLogger) Log(ctx context.Context, params LogParams) {
	l.bus.Publish(ctx, events.Event{
		Name:    events.EventAuditLogCreated,
		Payload: params,
	})
}

// NoopLogger is a logger that does nothing (e.g., for testing).
type NoopLogger struct{}

// Log does nothing.
func (n *NoopLogger) Log(_ context.Context, _ LogParams) {}
