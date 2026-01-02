// Package testutil provides testing utilities including testcontainers setup.
package testutil

import (
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/cmelgarejo/go-modulith-template/internal/authn"
	"github.com/cmelgarejo/go-modulith-template/internal/config"
	"github.com/cmelgarejo/go-modulith-template/internal/i18n"
	"github.com/cmelgarejo/go-modulith-template/internal/registry"
	"github.com/cmelgarejo/go-modulith-template/internal/validation"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

// GRPCTestServer wraps a gRPC server for testing.
type GRPCTestServer struct {
	Server   *grpc.Server
	Conn     *grpc.ClientConn
	listener net.Listener
	cfg      *config.AppConfig
	reg      *registry.Registry
}

// NewGRPCTestServer creates a new gRPC test server with the given modules.
func NewGRPCTestServer(cfg *config.AppConfig, reg *registry.Registry) (*GRPCTestServer, error) {
	if cfg == nil {
		cfg = TestConfig()
	}

	lis, err := net.Listen("tcp", ":0") // Use random port
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	verifier, err := authn.NewJWTVerifier(cfg.Auth.JWTSecret)
	if err != nil {
		_ = lis.Close()
		return nil, fmt.Errorf("failed to init jwt verifier: %w", err)
	}

	public := reg.GetPublicEndpoints()

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			i18n.UnaryServerInterceptor(cfg.DefaultLocale),
			validation.UnaryServerInterceptor(),
			authn.UnaryServerInterceptor(authn.InterceptorConfig{
				Verifier:      verifier,
				PublicMethods: public,
			}),
		),
	)

	reg.RegisterGRPCAll(grpcServer)
	reflection.Register(grpcServer)

	return &GRPCTestServer{
		Server:   grpcServer,
		listener: lis,
		cfg:      cfg,
		reg:      reg,
	}, nil
}

// Start starts the gRPC server in a goroutine.
func (s *GRPCTestServer) Start() error {
	go func() {
		if err := s.Server.Serve(s.listener); err != nil {
			slog.Error("gRPC test server error", "error", err)
		}
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	// Create client connection
	addr := s.listener.Addr().String()
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to dial gRPC server: %w", err)
	}

	s.Conn = conn

	return nil
}

// Stop stops the gRPC server and closes the connection.
func (s *GRPCTestServer) Stop() error {
	if s.Conn != nil {
		if err := s.Conn.Close(); err != nil {
			slog.Error("Failed to close gRPC client connection", "error", err)
		}
	}

	if s.Server != nil {
		s.Server.GracefulStop()
	}

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			return fmt.Errorf("failed to close listener: %w", err)
		}
	}

	return nil
}

// Client returns the gRPC client connection.
func (s *GRPCTestServer) Client() *grpc.ClientConn {
	return s.Conn
}

// Address returns the address the server is listening on.
func (s *GRPCTestServer) Address() string {
	if s.listener == nil {
		return ""
	}

	return s.listener.Addr().String()
}

