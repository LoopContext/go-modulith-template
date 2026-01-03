// Package testutil provides utilities for cross-module testing.
package testutil

import (
	"context"
	"fmt"

	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/registry"
	"google.golang.org/grpc"
)

// CrossModuleTestSetup provides setup for testing cross-module interactions.
type CrossModuleTestSetup struct {
	Registry       *registry.Registry
	GRPCServer     *grpc.Server
	EventBus       *events.Bus
	EventCollector *EventCollector
	Cleanup        func()
}

// SetupCrossModuleTest creates a test setup with registry and gRPC server for cross-module testing.
// This helper makes it easy to test interactions between modules via gRPC.
func SetupCrossModuleTest(_ context.Context, modules ...registry.Module) (*CrossModuleTestSetup, error) {
	// Create event bus
	eventBus := events.NewBus()
	eventCollector := NewEventCollector()

	// Create registry
	reg := registry.New(
		registry.WithEventBus(eventBus),
	)

	// Register modules
	for _, mod := range modules {
		reg.Register(mod)
	}

	// Initialize modules
	if err := reg.InitializeAll(); err != nil {
		return nil, fmt.Errorf("failed to initialize modules: %w", err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()
	reg.RegisterGRPCAll(grpcServer)

	cleanup := func() {
		grpcServer.Stop()

		_ = eventBus.Close()
	}

	return &CrossModuleTestSetup{
		Registry:       reg,
		GRPCServer:     grpcServer,
		EventBus:       eventBus,
		EventCollector: eventCollector,
		Cleanup:        cleanup,
	}, nil
}

// GetModuleServiceClient returns a gRPC client for a specific module service.
// This helper makes it easy to get a client for testing module-to-module calls.
func GetModuleServiceClient[T any](
	_ context.Context,
	_ *grpc.Server,
	conn *grpc.ClientConn,
	_ func(grpc.ServiceRegistrar, T),
	getClientFunc func(*grpc.ClientConn) interface{},
) (interface{}, error) {
	// For now, this is a placeholder that documents the pattern
	// Actual implementation depends on the specific service type
	// Users should use the generated gRPC client directly
	return getClientFunc(conn), nil
}

// AssertEventPublished checks if an event with the given name was published.
// This is a convenience function that checks if any collected event matches.
func AssertEventPublished(
	collector *EventCollector,
	eventName string,
) error {
	events := collector.AllEvents()
	for _, event := range events {
		if event.Name == eventName {
			return nil
		}
	}

	return fmt.Errorf("event %s was not published", eventName)
}
