// Package testutil provides testing utilities including testcontainers setup.
package testutil

import (
	"database/sql"

	"github.com/cmelgarejo/go-modulith-template/internal/config"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/registry"
)

// TestRegistryBuilder helps build test registries with common configurations.
type TestRegistryBuilder struct {
	db       *sql.DB
	bus      *events.Bus
	cfg      *config.AppConfig
	modules  []registry.Module
}

// NewTestRegistryBuilder creates a new test registry builder.
func NewTestRegistryBuilder() *TestRegistryBuilder {
	return &TestRegistryBuilder{
		modules: make([]registry.Module, 0),
	}
}

// WithDatabase sets the database for the registry.
func (b *TestRegistryBuilder) WithDatabase(db *sql.DB) *TestRegistryBuilder {
	b.db = db
	return b
}

// WithEventBus sets the event bus for the registry.
func (b *TestRegistryBuilder) WithEventBus(bus *events.Bus) *TestRegistryBuilder {
	b.bus = bus
	return b
}

// WithConfig sets the configuration for the registry.
func (b *TestRegistryBuilder) WithConfig(cfg *config.AppConfig) *TestRegistryBuilder {
	b.cfg = cfg
	return b
}

// WithModules adds modules to the registry.
func (b *TestRegistryBuilder) WithModules(modules ...registry.Module) *TestRegistryBuilder {
	b.modules = append(b.modules, modules...)
	return b
}

// Build creates and returns the registry with all configured components.
func (b *TestRegistryBuilder) Build() *registry.Registry {
	// Use defaults if not provided
	if b.cfg == nil {
		b.cfg = TestConfig()
	}

	if b.bus == nil {
		b.bus = events.NewBus()
	}

	opts := []registry.Option{
		registry.WithConfig(b.cfg),
		registry.WithEventBus(b.bus),
	}

	if b.db != nil {
		opts = append(opts, registry.WithDatabase(b.db))
	}

	reg := registry.New(opts...)

	// Register all modules
	for _, m := range b.modules {
		reg.Register(m)
	}

	return reg
}

