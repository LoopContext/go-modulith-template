package registry_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/registry"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

// mockModule is a test implementation of Module
type mockModule struct {
	name          string
	initErr       error
	registerErr   error
	onStartCalled bool
	onStopCalled  bool
	onStartErr    error
	onStopErr     error
	healthErr     error
}

func (m *mockModule) Name() string {
	return m.name
}

func (m *mockModule) Initialize(_ *registry.Registry) error {
	return m.initErr
}

func (m *mockModule) RegisterGRPC(_ *grpc.Server) {}

func (m *mockModule) RegisterGateway(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	return m.registerErr
}

func (m *mockModule) OnStart(_ context.Context) error {
	m.onStartCalled = true

	return m.onStartErr
}

func (m *mockModule) OnStop(_ context.Context) error {
	m.onStopCalled = true

	return m.onStopErr
}

func (m *mockModule) HealthCheck(_ context.Context) error {
	return m.healthErr
}

func TestNew(t *testing.T) {
	t.Parallel()

	cfg := struct{ Env string }{Env: "test"}
	bus := events.NewBus()

	r := registry.New(
		registry.WithConfig(&cfg),
		registry.WithEventBus(bus),
	)

	if r == nil {
		t.Fatal("expected registry to be created")
	}

	if r.Config() != &cfg {
		t.Error("expected config to be set")
	}

	if r.EventBus() != bus {
		t.Error("expected event bus to be set")
	}
}

func TestRegistry_Register(t *testing.T) {
	t.Parallel()

	r := registry.New()

	m1 := &mockModule{name: "module1"}
	m2 := &mockModule{name: "module2"}

	r.Register(m1)
	r.Register(m2)

	modules := r.Modules()
	if len(modules) != 2 {
		t.Errorf("expected 2 modules, got %d", len(modules))
	}
}

func TestRegistry_InitializeAll(t *testing.T) {
	t.Parallel()

	r := registry.New()

	r.Register(&mockModule{name: "ok1"})
	r.Register(&mockModule{name: "ok2"})

	err := r.InitializeAll()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRegistry_InitializeAll_Error(t *testing.T) {
	t.Parallel()

	r := registry.New()

	r.Register(&mockModule{name: "ok"})
	r.Register(&mockModule{name: "fail", initErr: errors.New("init failed")})
	r.Register(&mockModule{name: "never"})

	err := r.InitializeAll()
	if err == nil {
		t.Error("expected error")
	}

	if err.Error() != "failed to initialize module fail: init failed" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRegistry_GetModule(t *testing.T) {
	t.Parallel()

	r := registry.New()

	m1 := &mockModule{name: "auth"}
	m2 := &mockModule{name: "orders"}

	r.Register(m1)
	r.Register(m2)

	found := r.GetModule("auth")
	if found == nil {
		t.Fatal("expected to find module 'auth'")
	}

	if found.Name() != "auth" {
		t.Errorf("expected 'auth', got '%s'", found.Name())
	}

	notFound := r.GetModule("nonexistent")
	if notFound != nil {
		t.Error("expected nil for nonexistent module")
	}
}

func TestRegistry_OnStartAll(t *testing.T) {
	t.Parallel()

	r := registry.New()

	m1 := &mockModule{name: "m1"}
	m2 := &mockModule{name: "m2"}

	r.Register(m1)
	r.Register(m2)

	err := r.OnStartAll(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !m1.onStartCalled {
		t.Error("expected OnStart to be called on m1")
	}

	if !m2.onStartCalled {
		t.Error("expected OnStart to be called on m2")
	}
}

func TestRegistry_OnStopAll(t *testing.T) {
	t.Parallel()

	r := registry.New()

	m1 := &mockModule{name: "m1"}
	m2 := &mockModule{name: "m2"}

	r.Register(m1)
	r.Register(m2)

	err := r.OnStopAll(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !m1.onStopCalled {
		t.Error("expected OnStop to be called on m1")
	}

	if !m2.onStopCalled {
		t.Error("expected OnStop to be called on m2")
	}
}

func TestRegistry_HealthCheckAll(t *testing.T) {
	t.Parallel()

	r := registry.New()

	r.Register(&mockModule{name: "healthy"})
	r.Register(&mockModule{name: "unhealthy", healthErr: errors.New("unhealthy")})

	err := r.HealthCheckAll(context.Background())
	if err == nil {
		t.Error("expected error from unhealthy module")
	}
}

func TestRegistry_RegisterGatewayAll_Error(t *testing.T) {
	t.Parallel()

	r := registry.New()

	r.Register(&mockModule{name: "fail", registerErr: errors.New("gateway failed")})

	err := r.RegisterGatewayAll(context.Background(), nil, nil)
	if err == nil {
		t.Error("expected error")
	}
}

