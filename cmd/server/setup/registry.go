// Package setup provides server setup and configuration utilities.
package setup

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/cmelgarejo/go-modulith-template/internal/config"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/notifier"
	"github.com/cmelgarejo/go-modulith-template/internal/registry"
	"github.com/cmelgarejo/go-modulith-template/internal/websocket"
	"github.com/cmelgarejo/go-modulith-template/modules/auth"
)

// CreateRegistry creates a new registry with all dependencies.
func CreateRegistry(cfg *config.AppConfig, db *sql.DB) *registry.Registry {
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

	// Create registry with all dependencies
	return registry.New(
		registry.WithConfig(cfg),
		registry.WithDatabase(db),
		registry.WithEventBus(ebus),
		registry.WithNotifier(ntf),
		registry.WithWebSocketHub(wsHub),
	)
}

// RegisterModules registers all modules with the registry.
func RegisterModules(reg *registry.Registry) {
	// Register all modules here
	reg.Register(auth.NewModule())
	// Add more modules as needed:
	// reg.Register(order.NewModule())
	// reg.Register(payment.NewModule())
}
