// Package testutil provides utilities for cross-module testing.
package testutil

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	authv1 "github.com/cmelgarejo/go-modulith-template/gen/go/proto/auth/v1"
	"github.com/cmelgarejo/go-modulith-template/internal/authtoken"
	"github.com/cmelgarejo/go-modulith-template/internal/config"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/migration"
	"github.com/cmelgarejo/go-modulith-template/internal/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// CrossModuleTestSetup provides the canonical setup for testing module interactions.
type CrossModuleTestSetup struct {
	Config         *config.AppConfig
	Postgres       *PostgresContainer
	Pool           *pgxpool.Pool
	Registry       *registry.Registry
	GRPCTestServer *GRPCTestServer
	EventBus       *events.Bus
	EventCollector *EventCollector
	TokenService   *authtoken.Service
	Cleanup        func()
}

// SetupCrossModuleTest creates a reusable integration harness for cross-module gRPC tests.
func SetupCrossModuleTest(
	ctx context.Context,
	t *testing.T,
	modules ...registry.Module,
) (*CrossModuleTestSetup, error) {
	t.Helper()

	container, pool, cfg, eventBus, eventCollector, reg, err := newCrossModuleComponents(ctx, t, modules...)
	if err != nil {
		return nil, err
	}

	grpcTestServer, err := NewGRPCTestServer(cfg, reg)
	if err != nil {
		cleanupCrossModuleResources(ctx, container, pool, eventBus, nil)

		return nil, fmt.Errorf("create gRPC test server: %w", err)
	}

	if err := grpcTestServer.Start(); err != nil {
		cleanupCrossModuleResources(ctx, container, pool, eventBus, grpcTestServer)

		return nil, fmt.Errorf("start gRPC test server: %w", err)
	}

	if err := reg.OnStartAll(ctx); err != nil {
		cleanupCrossModuleResources(ctx, container, pool, eventBus, grpcTestServer)

		return nil, fmt.Errorf("start modules: %w", err)
	}

	tokenService, _ := authtoken.NewService(cfg.Auth.JWTPrivateKeyPEM)

	var cleanupOnce sync.Once

	cleanup := func() {
		cleanupOnce.Do(func() {
			_ = reg.OnStopAll(ctx)
			cleanupCrossModuleResources(ctx, container, pool, eventBus, grpcTestServer)
		})
	}

	t.Cleanup(cleanup)

	return &CrossModuleTestSetup{
		Config:         cfg,
		Postgres:       container,
		Pool:           pool,
		Registry:       reg,
		GRPCTestServer: grpcTestServer,
		EventBus:       eventBus,
		EventCollector: eventCollector,
		TokenService:   tokenService,
		Cleanup:        cleanup,
	}, nil
}

func newCrossModuleComponents(
	ctx context.Context,
	t *testing.T,
	modules ...registry.Module,
) (*PostgresContainer, *pgxpool.Pool, *config.AppConfig, *events.Bus, *EventCollector, *registry.Registry, error) {
	container, err := NewPostgresContainer(ctx, t)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("create postgres container: %w", err)
	}

	pool, err := container.Pool(ctx)
	if err != nil {
		_ = container.Close(ctx)

		return nil, nil, nil, nil, nil, nil, fmt.Errorf("connect postgres pool: %w", err)
	}

	cfg := TestConfig()
	cfg.DBDSN = container.DSN
	cfg.Env = "dev" // Enable magic code bypass for testing

	eventBus := events.NewBus()
	eventCollector := NewEventCollector()
	reg := NewTestRegistryBuilder().
		WithDatabase(pool).
		WithConfig(cfg).
		WithEventBus(eventBus).
		WithModules(modules...).
		Build()

	if err := reg.InitializeAll(); err != nil {
		cleanupCrossModuleResources(ctx, container, pool, eventBus, nil)

		return nil, nil, nil, nil, nil, nil, fmt.Errorf("initialize modules: %w", err)
	}

	if err := RunMigrationsForTest(ctx, container.DSN, reg); err != nil {
		cleanupCrossModuleResources(ctx, container, pool, eventBus, nil)

		return nil, nil, nil, nil, nil, nil, fmt.Errorf("run migrations: %w", err)
	}

	seeder, err := migration.NewSeeder(container.DSN, reg)
	if err != nil {
		cleanupCrossModuleResources(ctx, container, pool, eventBus, nil)

		return nil, nil, nil, nil, nil, nil, fmt.Errorf("create seeder: %w", err)
	}

	defer func() {
		_ = seeder.Close()
	}()

	if err := seeder.SeedAll(ctx); err != nil {
		cleanupCrossModuleResources(ctx, container, pool, eventBus, nil)

		return nil, nil, nil, nil, nil, nil, fmt.Errorf("seed modules: %w", err)
	}

	return container, pool, cfg, eventBus, eventCollector, reg, nil
}

func cleanupCrossModuleResources(
	ctx context.Context,
	container *PostgresContainer,
	pool *pgxpool.Pool,
	eventBus *events.Bus,
	grpcTestServer *GRPCTestServer,
) {
	if grpcTestServer != nil {
		_ = grpcTestServer.Stop()
	}

	if pool != nil {
		pool.Close()
	}

	if eventBus != nil {
		_ = eventBus.Close()
	}

	if container != nil {
		_ = container.Close(ctx)
	}
}

// Client returns the active client connection for generated gRPC clients.
func (s *CrossModuleTestSetup) Client() *grpc.ClientConn {
	if s == nil || s.GRPCTestServer == nil {
		return nil
	}

	return s.GRPCTestServer.Client()
}

// NewServiceClient creates a typed generated gRPC client from the shared harness.
func NewServiceClient[T any](setup *CrossModuleTestSetup, factory func(grpc.ClientConnInterface) T) T {
	return factory(setup.Client())
}

// AuthenticatedContext returns a new context with a valid JWT token for the given user.
func (s *CrossModuleTestSetup) AuthenticatedContext(ctx context.Context, userID, role string) (context.Context, string, error) {
	if s.TokenService == nil {
		return nil, "", fmt.Errorf("token service not initialized")
	}

	token, _, err := s.TokenService.CreateToken(userID, role, time.Hour)
	if err != nil {
		return nil, "", fmt.Errorf("create token: %w", err)
	}

	md := metadata.Pairs("authorization", "Bearer "+token)

	return metadata.NewOutgoingContext(ctx, md), token, nil
}

// RegisterUser creates a new user via the Auth module if available.
func (s *CrossModuleTestSetup) RegisterUser(ctx context.Context, email, displayName string) (*authv1.User, error) {
	authClient := NewServiceClient(s, authv1.NewAuthServiceClient)

	resp, err := authClient.Register(ctx, &authv1.RegisterRequest{
		ContactInfo: &authv1.RegisterRequest_Email{
			Email: email,
		},
		DisplayName:    displayName,
		Nationality:    "US",
		DocumentType:   "PASSPORT",
		DocumentNumber: "123456789",
	})
	if err != nil {
		return nil, fmt.Errorf("register user: %w", err)
	}

	return resp.User, nil
}

// SubscribeEvents wires the event collector to the provided event names.
func (s *CrossModuleTestSetup) SubscribeEvents(eventNames ...string) {
	for _, eventName := range eventNames {
		s.EventCollector.Subscribe(s.EventBus, eventName)
	}
}

// WaitForEvent waits until the named event is observed or the timeout expires.
func (s *CrossModuleTestSetup) WaitForEvent(eventName string, timeout time.Duration) (events.Event, error) {
	return s.EventCollector.WaitForEventByName(eventName, timeout)
}

// AssertEventPublished checks if an event with the given name was published.
func AssertEventPublished(
	collector *EventCollector,
	eventName string,
) error {
	if collector.HasEvent(eventName) {
		return nil
	}

	return fmt.Errorf("event %s was not published", eventName)
}
