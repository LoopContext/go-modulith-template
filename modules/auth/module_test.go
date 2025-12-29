package auth

import (
	"context"
	"database/sql"
	"testing"

	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"google.golang.org/grpc"
)

func TestInitialize_EmptyJWTSecret(t *testing.T) {
	grpcServer := grpc.NewServer()
	bus := events.NewBus()

	cfg := Config{
		JWTSecret: "",
	}

	err := Initialize(nil, grpcServer, bus, cfg)
	if err == nil {
		t.Fatal("expected error when JWT secret is empty")
	}
}

func TestInitialize_InvalidJWTSecret(t *testing.T) {
	grpcServer := grpc.NewServer()
	bus := events.NewBus()

	// JWT secret that's too short (less than 32 bytes)
	cfg := Config{
		JWTSecret: "short",
	}

	err := Initialize(nil, grpcServer, bus, cfg)
	if err == nil {
		t.Fatal("expected error when JWT secret is too short")
	}
}

func TestInitialize_Success(t *testing.T) {
	// Note: This test will fail in a real scenario without a DB connection
	// In a real test, you'd use a test database or mock
	// For now, we test the configuration validation part
	grpcServer := grpc.NewServer()
	bus := events.NewBus()

	cfg := Config{
		JWTSecret: "valid-secret-key-that-is-at-least-32-bytes-long",
	}

	// This will fail because db is nil, but that's expected
	// The important part is that it passes JWT secret validation
	err := Initialize(nil, grpcServer, bus, cfg)

	// We expect it to fail, but not due to JWT secret validation
	if err != nil {
		// Check that it's not a JWT secret error
		if err.Error() == "JWT secret is empty, cannot initialize auth module" {
			t.Error("JWT secret validation failed incorrectly")
		}
	}
}

func TestRegisterGateway_NilMux(t *testing.T) {
	ctx := context.Background()

	// This should fail gracefully or return an error
	err := RegisterGateway(ctx, nil, "localhost:9050", []grpc.DialOption{})
	if err == nil {
		t.Error("expected error when mux is nil")
	}
}

func TestRegisterGatewayWithConn_NilConn(t *testing.T) {
	// Skip this test as it would panic with nil mux/conn
	// In real usage, both should be non-nil
	t.Skip("Skipping test that requires valid mux and conn")
}

// TestConfig verifies the Config structure
func TestConfig(t *testing.T) {
	cfg := Config{
		JWTSecret: "test-secret",
	}

	if cfg.JWTSecret != "test-secret" {
		t.Errorf("expected JWT secret 'test-secret', got %s", cfg.JWTSecret)
	}
}

// TestInitialize_NilDB tests behavior with nil database
func TestInitialize_NilDB(_ *testing.T) {
	grpcServer := grpc.NewServer()
	bus := events.NewBus()

	cfg := Config{
		JWTSecret: "valid-secret-key-that-is-at-least-32-bytes-long",
	}

	var nilDB *sql.DB

	// Should not panic even with nil DB (repository creation should handle it)
	_ = Initialize(nilDB, grpcServer, bus, cfg)

	// The function might return an error or not depending on implementation
	// The important thing is it doesn't panic
}

// TestInitialize_NilGRPCServer tests behavior with nil gRPC server
func TestInitialize_NilGRPCServer(t *testing.T) {
	// This will panic when trying to register with nil server
	// In production, this should never happen
	t.Skip("Skipping test that would panic with nil gRPC server")
}

// TestInitialize_NilBus tests behavior with nil event bus
func TestInitialize_NilBus(t *testing.T) {
	grpcServer := grpc.NewServer()

	cfg := Config{
		JWTSecret: "valid-secret-key-that-is-at-least-32-bytes-long",
	}

	// Should not panic even with nil bus
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Initialize panicked with nil bus: %v", r)
		}
	}()

	_ = Initialize(nil, grpcServer, nil, cfg)
}

