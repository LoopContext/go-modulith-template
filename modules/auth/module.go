// Package auth implements the authentication module.
package auth

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	authv1 "github.com/cmelgarejo/go-modulith-template/gen/go/proto/auth/v1"
	"github.com/cmelgarejo/go-modulith-template/internal/config"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/registry"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/repository"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/service"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/token"
)

// Config is an alias for backwards compatibility.
//
// Deprecated: Use config.AuthConfig instead.
type Config = config.AuthConfig

// Module implements the registry.Module interface for auth.
type Module struct {
	svc *service.AuthService
}

// NewModule creates a new auth module instance.
func NewModule() *Module {
	return &Module{}
}

// Name returns the module identifier.
func (m *Module) Name() string {
	return "auth"
}

// Initialize sets up the auth module with dependencies from the registry.
func (m *Module) Initialize(r *registry.Registry) error {
	cfg, ok := r.Config().(*config.AppConfig)
	if !ok {
		return fmt.Errorf("invalid config type, expected *config.AppConfig")
	}

	if cfg.Auth.JWTSecret == "" {
		return fmt.Errorf("JWT secret is empty, cannot initialize auth module")
	}

	tokenService, err := token.NewService(cfg.Auth.JWTSecret)
	if err != nil {
		return fmt.Errorf("failed to init token service: %w", err)
	}

	repo := repository.NewSQLRepository(r.DB())
	m.svc = service.NewAuthService(repo, tokenService, r.EventBus())

	return nil
}

// RegisterGRPC registers the auth gRPC services.
func (m *Module) RegisterGRPC(server *grpc.Server) {
	authv1.RegisterAuthServiceServer(server, m.svc)
}

// RegisterGateway registers the auth HTTP gateway handlers.
func (m *Module) RegisterGateway(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	if err := authv1.RegisterAuthServiceHandler(ctx, mux, conn); err != nil {
		return fmt.Errorf("failed to register auth gateway: %w", err)
	}

	return nil
}

// MigrationPath returns the path to the auth module's migration directory.
func (m *Module) MigrationPath() string {
	return "modules/auth/resources/db/migration"
}

// SeedPath returns the path to the auth module's seed data directory.
func (m *Module) SeedPath() string {
	return "modules/auth/resources/db/seed"
}

// PublicEndpoints returns the list of public endpoints that don't require authentication.
func (m *Module) PublicEndpoints() []string {
	return []string{
		"/auth.v1.AuthService/RequestLogin",
		"/auth.v1.AuthService/CompleteLogin",
	}
}

// --- Legacy functions for backwards compatibility ---

// Initialize registers the Auth module with the gRPC server (legacy).
//
// Deprecated: Use Module.Initialize with Registry instead.
func Initialize(db *sql.DB, grpcServer *grpc.Server, bus *events.Bus, cfg Config) error {
	if cfg.JWTSecret == "" {
		return fmt.Errorf("JWT secret is empty, cannot initialize auth module")
	}

	tokenService, err := token.NewService(cfg.JWTSecret)
	if err != nil {
		return fmt.Errorf("failed to init token service: %w", err)
	}

	repo := repository.NewSQLRepository(db)
	svc := service.NewAuthService(repo, tokenService, bus)

	authv1.RegisterAuthServiceServer(grpcServer, svc)

	return nil
}

// RegisterGateway registers the gRPC-Gateway handler (legacy).
//
// Deprecated: Use Module.RegisterGateway instead.
func RegisterGateway(ctx context.Context, mux *runtime.ServeMux, grpcEndpoint string, opts []grpc.DialOption) error {
	if err := authv1.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, grpcEndpoint, opts); err != nil {
		return fmt.Errorf("failed to register auth gateway: %w", err)
	}

	return nil
}

// RegisterGatewayWithConn registers the gRPC-Gateway handler using an explicit connection (legacy).
//
// Deprecated: Use Module.RegisterGateway instead.
func RegisterGatewayWithConn(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	if err := authv1.RegisterAuthServiceHandler(ctx, mux, conn); err != nil {
		return fmt.Errorf("failed to register auth gateway: %w", err)
	}

	return nil
}
