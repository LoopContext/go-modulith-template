package registry

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/notifier"
	"github.com/cmelgarejo/go-modulith-template/internal/websocket"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

// Registry is the central dependency container for the application.
// It holds all shared services and manages module lifecycle.
type Registry struct {
	// Core dependencies - config is any to avoid import cycles
	config   any
	db       *sql.DB
	bus      *events.Bus
	notifier notifier.Notifier
	wsHub    *websocket.Hub

	// Infrastructure
	metricsHandler http.Handler

	// Registered modules
	modules []Module
}

// New creates a new Registry with the provided options.
func New(opts ...Option) *Registry {
	r := &Registry{
		modules: make([]Module, 0),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Register adds a module to the registry.
// Modules are initialized in the order they are registered.
func (r *Registry) Register(m Module) {
	r.modules = append(r.modules, m)
}

// InitializeAll initializes all registered modules.
// Returns an error if any module fails to initialize.
func (r *Registry) InitializeAll() error {
	for _, m := range r.modules {
		slog.Info("Initializing module", "module", m.Name())

		if err := m.Initialize(r); err != nil {
			return fmt.Errorf("failed to initialize module %s: %w", m.Name(), err)
		}
	}

	return nil
}

// RegisterGRPCAll registers all modules with the gRPC server.
func (r *Registry) RegisterGRPCAll(server *grpc.Server) {
	for _, m := range r.modules {
		slog.Debug("Registering gRPC services", "module", m.Name())
		m.RegisterGRPC(server)
	}
}

// RegisterGatewayAll registers all modules with the HTTP gateway.
func (r *Registry) RegisterGatewayAll(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	for _, m := range r.modules {
		slog.Debug("Registering gateway handlers", "module", m.Name())

		if err := m.RegisterGateway(ctx, mux, conn); err != nil {
			return fmt.Errorf("failed to register gateway for module %s: %w", m.Name(), err)
		}
	}

	return nil
}

// OnStartAll calls OnStart on all modules that implement ModuleLifecycle.
func (r *Registry) OnStartAll(ctx context.Context) error {
	for _, m := range r.modules {
		if lc, ok := m.(ModuleLifecycle); ok {
			slog.Debug("Calling OnStart", "module", m.Name())

			if err := lc.OnStart(ctx); err != nil {
				return fmt.Errorf("module %s OnStart failed: %w", m.Name(), err)
			}
		}
	}

	return nil
}

// OnStopAll calls OnStop on all modules that implement ModuleLifecycle.
// Modules are stopped in reverse order of registration.
func (r *Registry) OnStopAll(ctx context.Context) error {
	var errs []error

	for i := len(r.modules) - 1; i >= 0; i-- {
		m := r.modules[i]
		if lc, ok := m.(ModuleLifecycle); ok {
			slog.Debug("Calling OnStop", "module", m.Name())

			if err := lc.OnStop(ctx); err != nil {
				errs = append(errs, fmt.Errorf("module %s OnStop failed: %w", m.Name(), err))
			}
		}
	}

	if len(errs) > 0 {
		return errs[0] // Return first error
	}

	return nil
}

// HealthCheckAll checks health of all modules that implement ModuleHealth.
func (r *Registry) HealthCheckAll(ctx context.Context) error {
	for _, m := range r.modules {
		if hc, ok := m.(ModuleHealth); ok {
			if err := hc.HealthCheck(ctx); err != nil {
				return fmt.Errorf("module %s health check failed: %w", m.Name(), err)
			}
		}
	}

	return nil
}

// Getters for dependencies - modules use these to access shared services

// Config returns the application configuration as an interface.
// Modules should type-assert to *config.AppConfig.
func (r *Registry) Config() any {
	return r.config
}

// DB returns the database connection.
func (r *Registry) DB() *sql.DB {
	return r.db
}

// EventBus returns the event bus.
func (r *Registry) EventBus() *events.Bus {
	return r.bus
}

// Notifier returns the notification service.
func (r *Registry) Notifier() notifier.Notifier {
	return r.notifier
}

// WebSocketHub returns the WebSocket hub.
func (r *Registry) WebSocketHub() *websocket.Hub {
	return r.wsHub
}

// MetricsHandler returns the metrics HTTP handler.
func (r *Registry) MetricsHandler() http.Handler {
	return r.metricsHandler
}

// Modules returns all registered modules.
func (r *Registry) Modules() []Module {
	return r.modules
}

// GetModule returns a module by name, or nil if not found.
func (r *Registry) GetModule(name string) Module {
	for _, m := range r.modules {
		if m.Name() == name {
			return m
		}
	}

	return nil
}

// GetPublicEndpoints returns a map of all public endpoints from registered modules.
func (r *Registry) GetPublicEndpoints() map[string]struct{} {
	endpoints := make(map[string]struct{})

	for _, m := range r.modules {
		if authMod, ok := m.(ModuleAuth); ok {
			for _, endpoint := range authMod.PublicEndpoints() {
				endpoints[endpoint] = struct{}{}
			}
		}
	}

	return endpoints
}
