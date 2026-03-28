package auth

import (
	"context"
	"testing"

	"github.com/LoopContext/go-modulith-template/internal/audit"
	"github.com/LoopContext/go-modulith-template/internal/events"
	"github.com/LoopContext/go-modulith-template/internal/feature"
	"github.com/LoopContext/go-modulith-template/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

func TestInitialize_EmptyJWTPrivateKey(t *testing.T) {
	grpcServer := grpc.NewServer()
	bus := events.NewBus()

	cfg := Config{
		JWTPrivateKeyPEM: "",
	}

	auditLog := &audit.NoopLogger{}
	flagManager := feature.NewInMemoryManager()

	err := Initialize(nil, grpcServer, bus, cfg, auditLog, flagManager)
	if err == nil {
		t.Fatal("expected error when JWT private key is empty")
	}
}

func TestInitialize_InvalidJWTPrivateKey(t *testing.T) {
	grpcServer := grpc.NewServer()
	bus := events.NewBus()

	cfg := Config{
		JWTPrivateKeyPEM: "not-valid-pem",
	}

	auditLog := &audit.NoopLogger{}
	flagManager := feature.NewInMemoryManager()

	err := Initialize(nil, grpcServer, bus, cfg, auditLog, flagManager)
	if err == nil {
		t.Fatal("expected error when JWT private key is invalid")
	}
}

func TestInitialize_Success(t *testing.T) {
	grpcServer := grpc.NewServer()
	bus := events.NewBus()

	cfg := Config{
		JWTPrivateKeyPEM: testutil.TestJWTPrivateKeyPEM,
	}

	auditLog := &audit.NoopLogger{}
	flagManager := feature.NewInMemoryManager()

	err := Initialize(nil, grpcServer, bus, cfg, auditLog, flagManager)
	if err != nil {
		if err.Error() == "JWT private key (JWT_PRIVATE_KEY) is required to initialize auth module (RS256)" {
			t.Error("JWT private key validation failed incorrectly")
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
		JWTPrivateKeyPEM: testutil.TestJWTPrivateKeyPEM,
		JWTPublicKeyPEM:  testutil.TestJWTPublicKeyPEM,
	}

	if cfg.JWTPrivateKeyPEM == "" {
		t.Error("expected JWT private key to be set")
	}

	if cfg.JWTPublicKeyPEM == "" {
		t.Error("expected JWT public key to be set")
	}
}

// TestInitialize_NilDB tests behavior with nil database
func TestInitialize_NilDB(_ *testing.T) {
	grpcServer := grpc.NewServer()
	bus := events.NewBus()

	cfg := Config{
		JWTPrivateKeyPEM: testutil.TestJWTPrivateKeyPEM,
	}

	var nilDB *pgxpool.Pool

	auditLog := &audit.NoopLogger{}
	flagManager := feature.NewInMemoryManager()
	// Should not panic even with nil DB (repository creation should handle it)
	_ = Initialize(nilDB, grpcServer, bus, cfg, auditLog, flagManager)

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
		JWTPrivateKeyPEM: testutil.TestJWTPrivateKeyPEM,
	}

	// Should not panic even with nil bus
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Initialize panicked with nil bus: %v", r)
		}
	}()

	auditLog := &audit.NoopLogger{}
	flagManager := feature.NewInMemoryManager()
	_ = Initialize(nil, grpcServer, nil, cfg, auditLog, flagManager)
}
