// Package registry provides a simple dependency injection container
// using the Registry Pattern for managing application services and modules.
package registry

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

// Module defines the interface that all application modules must implement.
// This allows for consistent initialization and registration across modules.
type Module interface {
	// Name returns the unique identifier for this module.
	Name() string

	// Initialize sets up the module with its dependencies from the registry.
	// This is called once during application startup.
	Initialize(r *Registry) error

	// RegisterGRPC registers the module's gRPC services with the server.
	RegisterGRPC(server *grpc.Server)

	// RegisterGateway registers the module's HTTP gateway handlers.
	RegisterGateway(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error
}

// ModuleHealth provides optional health check capabilities for modules.
type ModuleHealth interface {
	// HealthCheck returns an error if the module is unhealthy.
	HealthCheck(ctx context.Context) error
}

// ModuleLifecycle provides optional lifecycle hooks for modules.
type ModuleLifecycle interface {
	// OnStart is called after all modules are initialized, before serving.
	OnStart(ctx context.Context) error

	// OnStop is called during graceful shutdown.
	OnStop(ctx context.Context) error
}

// ModuleMigrations provides optional database migration support for modules.
type ModuleMigrations interface {
	// MigrationPath returns the path to the module's migration directory.
	// Return empty string if the module has no migrations.
	MigrationPath() string
}

// ModuleAuth provides optional authentication configuration for modules.
type ModuleAuth interface {
	// PublicEndpoints returns a list of gRPC method paths that don't require authentication.
	// Format: "/package.service/Method"
	PublicEndpoints() []string
}

// ModuleSeeder defines the interface for modules that provide seed data via SQL files.
type ModuleSeeder interface {
	// SeedPath returns the path to the module's seed directory.
	// Return empty string if the module has no seed data.
	SeedPath() string
}

// ModuleProgrammaticSeeder defines the interface for modules that provide seed data programmatically via Go.
type ModuleProgrammaticSeeder interface {
	// Seed runs programmatic seed data using the application's registry/dependencies.
	Seed(ctx context.Context, r interface{}) error
}

