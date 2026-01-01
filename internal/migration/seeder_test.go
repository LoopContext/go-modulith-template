package migration

import (
	"context"
	"testing"
)

type mockSeederRegistry struct {
	modules []interface{}
}

func (m *mockSeederRegistry) Modules() []interface{} {
	return m.modules
}

func TestSeeder_SeedAll_NoModules(t *testing.T) {
	registry := &mockSeederRegistry{modules: []interface{}{}}
	adapter := &registryAdapter{registry: registry}

	// Create seeder (skip DB connection for this test)
	seeder := &Seeder{
		provider: adapter,
	}

	// Should not error with no modules
	if err := seeder.SeedAll(context.TODO()); err != nil {
		t.Errorf("SeedAll() with no modules should not error, got: %v", err)
	}
}

func TestSeeder_SeedAll_ModuleWithoutSeeder(t *testing.T) {
	type moduleWithoutSeeder struct {
		name string
	}

	mod := &moduleWithoutSeeder{name: "test"}
	registry := &mockSeederRegistry{modules: []interface{}{mod}}
	adapter := &registryAdapter{registry: registry}

	seeder := &Seeder{
		provider: adapter,
	}

	// Should not error when module doesn't implement ModuleSeeder
	if err := seeder.SeedAll(context.TODO()); err != nil {
		t.Errorf("SeedAll() with non-seeder module should not error, got: %v", err)
	}
}

