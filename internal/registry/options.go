package registry

import (
	"database/sql"
	"net/http"

	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/notifier"
	"github.com/cmelgarejo/go-modulith-template/internal/websocket"
)

// Option is a functional option for configuring the Registry.
type Option func(*Registry)

// WithConfig sets the application configuration.
// The config parameter is an interface{} to avoid import cycles.
// Modules should type-assert to *config.AppConfig.
func WithConfig(cfg any) Option {
	return func(r *Registry) {
		r.config = cfg
	}
}

// WithDatabase sets the database connection.
func WithDatabase(db *sql.DB) Option {
	return func(r *Registry) {
		r.db = db
	}
}

// WithEventBus sets the event bus for pub/sub.
func WithEventBus(bus *events.Bus) Option {
	return func(r *Registry) {
		r.bus = bus
	}
}

// WithOutbox enables the outbox pattern for reliable event publishing.
// This is a placeholder for future outbox integration.
// Outbox pattern implementation is in internal/outbox package.
// Usage is explicit in modules when outbox is needed.
func WithOutbox(outboxRepo interface{}) Option {
	return func(_ *Registry) {
		// Outbox integration is optional and explicit in modules
		_ = outboxRepo
	}
}

// WithNotifier sets the notification service.
func WithNotifier(n notifier.Notifier) Option {
	return func(r *Registry) {
		r.notifier = n
	}
}

// WithWebSocketHub sets the WebSocket hub.
func WithWebSocketHub(hub *websocket.Hub) Option {
	return func(r *Registry) {
		r.wsHub = hub
	}
}

// WithMetricsHandler sets the metrics HTTP handler.
func WithMetricsHandler(h http.Handler) Option {
	return func(r *Registry) {
		r.metricsHandler = h
	}
}

