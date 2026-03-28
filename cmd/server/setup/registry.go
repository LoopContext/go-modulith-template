// Package setup provides server setup and configuration utilities.
package setup

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/LoopContext/go-modulith-template/internal/audit"
	"github.com/LoopContext/go-modulith-template/internal/cache"
	"github.com/LoopContext/go-modulith-template/internal/config"
	"github.com/LoopContext/go-modulith-template/internal/events"
	"github.com/LoopContext/go-modulith-template/internal/feature"
	"github.com/LoopContext/go-modulith-template/internal/notifier"
	"github.com/LoopContext/go-modulith-template/internal/registry"
	"github.com/LoopContext/go-modulith-template/internal/websocket"
	"github.com/LoopContext/go-modulith-template/modules/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateRegistry creates a new registry with all dependencies.
func CreateRegistry(cfg *config.AppConfig, db *pgxpool.Pool) *registry.Registry {
	// Create shared services
	ebus := events.NewBus()
	wsHub := websocket.NewHub(context.Background())
	ntf := notifier.NewLogNotifier()

	// Initialize WebSocket subscriber
	wsSubscriber := websocket.NewSubscriber(wsHub, ebus)
	wsSubscriber.Subscribe()

	// Initialize notification subscriber with default locale
	ns := notifier.NewSubscriber(ntf, cfg.DefaultLocale)
	ns.SubscribeToEvents(ebus)

	// Start WebSocket hub in background
	go wsHub.Run()

	slog.Info("WebSocket hub initialized")

	// Initialize Audit Logger
	auditLogger := audit.NewEventBusLogger(ebus)

	// Initialize Cache
	var cacheImpl cache.Cache

	valkeyCfg := cache.ValkeyConfig{
		Addr:         cfg.ValkeyAddr,
		Password:     cfg.ValkeyPassword,
		DB:           cfg.ValkeyDB,
		PoolSize:     cfg.ValkeyPoolSize,
		MinIdleConns: cfg.ValkeyMinIdleConns,
	}

	valkeyCache, err := cache.NewValkeyCache(valkeyCfg)
	if err != nil {
		slog.Error("Failed to initialize Valkey cache", "error", err)
		// We could decide to fail here or continue without cache
		// Given it's critical for rate limiting, let's fail in prod
		if cfg.Env == "prod" {
			panic(fmt.Errorf("failed to initialize Valkey cache: %w", err))
		}
	} else {
		slog.Info("Valkey cache initialized", "addr", cfg.ValkeyAddr)

		cacheImpl = valkeyCache
	}

	// Initialize Feature Flag Manager
	featureMgr := feature.NewSQLManager(db)

	// Create registry with all dependencies
	return registry.New(
		registry.WithConfig(cfg),
		registry.WithDatabase(db),
		registry.WithEventBus(ebus),
		registry.WithNotifier(ntf),
		registry.WithWebSocketHub(wsHub),
		registry.WithAuditLogger(auditLogger),
		registry.WithFeature(featureMgr),
		registry.WithCache(cacheImpl),
	)
}

// RegisterModules registers all modules with the registry.
func RegisterModules(reg *registry.Registry) {
	// Register all modules here
	// Order matters: modules that are dependencies must be registered first
	reg.Register(auth.NewModule())
	// Add more modules as needed:
	// reg.Register(wallet.NewModule())
}
