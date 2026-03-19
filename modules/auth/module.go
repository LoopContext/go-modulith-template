// Package auth implements the authentication module.
package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	authv1 "github.com/cmelgarejo/go-modulith-template/gen/go/proto/auth/v1"
	"github.com/cmelgarejo/go-modulith-template/internal/audit"
	"github.com/cmelgarejo/go-modulith-template/internal/authtoken"
	"github.com/cmelgarejo/go-modulith-template/internal/config"
	internalEvents "github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/feature"
	"github.com/cmelgarejo/go-modulith-template/internal/outbox"
	"github.com/cmelgarejo/go-modulith-template/internal/registry"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/repository"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/service"
	authSeed "github.com/cmelgarejo/go-modulith-template/modules/auth/resources/db/seed"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config is an alias for backwards compatibility.
//
// Deprecated: Use config.AuthConfig instead.
type Config = config.AuthConfig

// Module implements the registry.Module interface for auth.
type Module struct {
	svc    *service.AuthService
	outbox *outbox.Publisher
}

// NewModule creates a new auth module instance.
func NewModule() *Module {
	return &Module{}
}

// Service returns the auth service server interface for cross-module communication.
func (m *Module) Service() authv1.AuthServiceServer {
	return m.svc
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

	if cfg.Auth.JWTPrivateKeyPEM == "" {
		return fmt.Errorf("JWT private key (JWT_PRIVATE_KEY) is required to initialize auth module (RS256)")
	}

	tokenService, err := authtoken.NewService(cfg.Auth.JWTPrivateKeyPEM)
	if err != nil {
		return fmt.Errorf("failed to init token service: %w", err)
	}

	repo := repository.NewSQLRepository(r.DB())
	m.svc = service.NewAuthService(repo, tokenService, r.EventBus(), r.AuditLogger(), r.FlagManager(), cfg.Env)

	m.outbox = outbox.NewPublisher(repo, func(ctx context.Context, name string, payload interface{}) {
		r.EventBus().Publish(ctx, internalEvents.Event{Name: name, Payload: payload})
	})

	if cfg.OutboxPollInterval != "" {
		if d, err := time.ParseDuration(cfg.OutboxPollInterval); err == nil {
			m.outbox.SetPollInterval(d)
		}
	}

	// Handle dummy email verification
	r.EventBus().Subscribe(internalEvents.EventAuthEmailVerificationRequested, m.handleEmailVerificationRequested)

	return nil
}

func (m *Module) handleEmailVerificationRequested(ctx context.Context, event internalEvents.Event) error {
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		return nil
	}

	userID, _ := payload["user_id"].(string)
	if userID == "" {
		return nil
	}

	if err := m.svc.HandleEmailVerificationRequested(ctx, userID); err != nil {
		return fmt.Errorf("failed to handle email verification request: %w", err)
	}

	return nil
}

// OnStart starts the outbox publisher.
func (m *Module) OnStart(ctx context.Context) error {
	if m.outbox != nil {
		go m.outbox.Start(ctx)
	}

	return nil
}

// OnStop stops the outbox publisher.
func (m *Module) OnStop(_ context.Context) error {
	if m.outbox != nil {
		m.outbox.Stop()
	}

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

// Seed runs programmatic seed data for the auth module.
func (m *Module) Seed(ctx context.Context, r interface{}) error {
	reg, ok := r.(*registry.Registry)
	if !ok {
		return fmt.Errorf("registry is not *registry.Registry")
	}

	cfg, ok := reg.Config().(*config.AppConfig)
	if !ok {
		return fmt.Errorf("config is not *config.AppConfig")
	}

	if err := authSeed.Seed(ctx, reg.DB(), cfg, reg.AuditLogger()); err != nil {
		return fmt.Errorf("failed to seed auth module: %w", err)
	}

	return nil
}

// PublicEndpoints returns the list of public endpoints that don't require authentication.
func (m *Module) PublicEndpoints() []string {
	return []string{
		"/auth.v1.AuthService/RequestLogin",
		"/auth.v1.AuthService/CompleteLogin",
		"/auth.v1.AuthService/Register",
		"/auth.v1.AuthService/RefreshSession",
		"/auth.v1.AuthService/GetOAuthProviders",
		"/auth.v1.AuthService/InitiateOAuth",
		"/auth.v1.AuthService/CompleteOAuth",
		"/auth.v1.AuthService/GetSystemConfig",
	}
}

// --- Legacy functions for backwards compatibility ---

// Initialize registers the Auth module with the gRPC server (legacy).
//
// Deprecated: Use Module.Initialize with Registry instead.
func Initialize(db *pgxpool.Pool, grpcServer *grpc.Server, bus *internalEvents.Bus, cfg Config, auditLog audit.Logger, flagManager feature.Manager) error {
	if cfg.JWTPrivateKeyPEM == "" {
		return fmt.Errorf("JWT private key (JWT_PRIVATE_KEY) is required to initialize auth module (RS256)")
	}

	tokenService, err := authtoken.NewService(cfg.JWTPrivateKeyPEM)
	if err != nil {
		return fmt.Errorf("failed to init token service: %w", err)
	}

	repo := repository.NewSQLRepository(db)
	svc := service.NewAuthService(repo, tokenService, bus, auditLog, flagManager, "legacy")

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
