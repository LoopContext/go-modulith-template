package migration

import (
	"context"
	"testing"

	"github.com/cmelgarejo/go-modulith-template/internal/registry"
)

type mockSeederRegistry struct {
	modules []registry.Module
}

func (m *mockSeederRegistry) Modules() []registry.Module {
	return m.modules
}

func TestSeeder_SeedAll_NoModules(t *testing.T) {
	reg := &mockSeederRegistry{modules: []registry.Module{}}

	// Create seeder (skip DB connection for this test)
	seeder := &Seeder{
		registry: reg,
		provider: &registryAdapter{registry: reg},
	}

	// Should not error with no modules
	if err := seeder.SeedAll(context.TODO()); err != nil {
		t.Errorf("SeedAll() with no modules should not error, got: %v", err)
	}
}

func TestSeeder_SeedAll_ModuleWithoutSeeder(t *testing.T) {
	// mockModule implements registry.Module
	type mockModule struct {
		registry.Module
		name string
	}

	mod := &mockModule{name: "test"}
	reg := &mockSeederRegistry{modules: []registry.Module{mod}}

	seeder := &Seeder{
		registry: reg,
		provider: &registryAdapter{registry: reg},
	}

	// Should not error when module doesn't implement ModuleSeeder
	if err := seeder.SeedAll(context.TODO()); err != nil {
		t.Errorf("SeedAll() with non-seeder module should not error, got: %v", err)
	}
}
