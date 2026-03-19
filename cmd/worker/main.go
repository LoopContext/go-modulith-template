// Package main is the entry point for the worker process.
// Workers handle background tasks, event consumers, and scheduled jobs.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cmelgarejo/go-modulith-template/internal/config"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/migration"
	"github.com/cmelgarejo/go-modulith-template/internal/notifier"
	"github.com/cmelgarejo/go-modulith-template/internal/registry"
	"github.com/cmelgarejo/go-modulith-template/internal/version"
	"github.com/cmelgarejo/go-modulith-template/modules/auth"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := loadConfig()
	if cfg == nil {
		return
	}

	db := initDB(cfg)
	if db == nil {
		return
	}

	defer closeDB(db)

	// Create registry with all dependencies
	reg := createRegistry(cfg, db)

	// Register modules
	registerModules(reg)

	// Initialize all modules
	if err := reg.InitializeAll(); err != nil {
		slog.Error("Failed to initialize modules", "error", err)
		return
	}

	// Run migrations for all modules
	if err := runMigrations(cfg.DBDSN, reg); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		return
	}

	// Start worker
	runWorker(ctx, reg, stop)
}

func loadConfig() *config.AppConfig {
	initLoggerEarly()

	systemEnvVars := captureSystemEnvVars()
	_ = godotenv.Load()

	cfg, err := config.Load("configs/worker.yaml", systemEnvVars)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		return nil
	}

	initLogger(cfg.Env, cfg.LogLevel)

	slog.Info("Starting worker", "version", version.Info())

	return cfg
}

func initDB(cfg *config.AppConfig) *pgxpool.Pool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolCfg, err := pgxpool.ParseConfig(cfg.DBDSN)
	if err != nil {
		slog.Error("Failed to parse DB config", "error", err)
		return nil
	}

	// Configure connection pool
	poolCfg.MaxConns = int32(cfg.DBMaxOpenConns) //nolint:gosec // G115: Configured limits are within safe int32 range
	poolCfg.MinConns = int32(cfg.DBMaxIdleConns) //nolint:gosec // G115: Configured limits are within safe int32 range

	if cfg.DBConnMaxLifetime != "" {
		if lifetime, err := time.ParseDuration(cfg.DBConnMaxLifetime); err == nil {
			poolCfg.MaxConnLifetime = lifetime
		}
	}

	db, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		slog.Error("Failed to create DB pool", "error", err)
		return nil
	}

	if err := db.Ping(ctx); err != nil {
		slog.Error("Failed to ping DB", "error", err)
		db.Close()

		return nil
	}

	slog.Info("Connected to Database")

	return db
}

func closeDB(db *pgxpool.Pool) {
	db.Close()
}

func createRegistry(cfg *config.AppConfig, db *pgxpool.Pool) *registry.Registry {
	// Create shared services
	ebus := events.NewBus()
	ntf := notifier.NewLogNotifier()

	// Initialize notification subscriber with default locale
	ns := notifier.NewSubscriber(ntf, cfg.DefaultLocale)
	ns.SubscribeToEvents(ebus)

	// Create registry with all dependencies
	return registry.New(
		registry.WithConfig(cfg),
		registry.WithDatabase(db),
		registry.WithEventBus(ebus),
		registry.WithNotifier(ntf),
	)
}

func registerModules(reg *registry.Registry) {
	// Register all modules here
	reg.Register(auth.NewModule())
	// Add more modules as needed:
	// reg.Register(order.NewModule())
	// reg.Register(payment.NewModule())
}

func runMigrations(dbDSN string, reg *registry.Registry) error {
	runner := migration.NewRunner(dbDSN, reg)
	if err := runner.RunAll(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func runWorker(ctx context.Context, reg *registry.Registry, _ context.CancelFunc) {
	// Call module lifecycle OnStart hooks
	if err := reg.OnStartAll(ctx); err != nil {
		slog.Error("Failed to start modules", "error", err)
		return
	}

	// Ensure OnStopAll is called during shutdown
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := reg.OnStopAll(shutdownCtx); err != nil {
			slog.Error("Failed to stop modules gracefully", "error", err)
		}
	}()

	slog.Info("Worker started", "event_bus_ready", reg.EventBus() != nil)

	// TODO: Add event consumers here
	// Example:
	// bus := reg.EventBus()
	// bus.Subscribe("user.created", func(ctx context.Context, e events.Event) error {
	//     slog.Info("Processing user.created event", "payload", e.Payload)
	//     return nil
	// })

	// TODO: Add scheduled tasks here
	// Example:
	// ticker := time.NewTicker(1 * time.Hour)
	// defer ticker.Stop()
	// go func() {
	//     for {
	//         select {
	//         case <-ticker.C:
	//             // Run scheduled task
	//         case <-ctx.Done():
	//             return
	//         }
	//     }
	// }()

	// Wait for shutdown signal
	<-ctx.Done()
	slog.Info("Worker shutting down")
}

func initLoggerEarly() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func initLogger(env string, logLevel string) {
	var handler slog.Handler

	var level slog.Level

	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	if env == "prod" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func captureSystemEnvVars() map[string]string {
	systemEnvVars := make(map[string]string)
	if env := os.Getenv("ENV"); env != "" {
		systemEnvVars["ENV"] = env
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		systemEnvVars["LOG_LEVEL"] = logLevel
	}

	if dsn := os.Getenv("DB_DSN"); dsn != "" {
		systemEnvVars["DB_DSN"] = dsn
	}

	return systemEnvVars
}
