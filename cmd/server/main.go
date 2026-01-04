// Package main is the entry point for the server application.
package main

import (
	"context"
	"database/sql"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cmelgarejo/go-modulith-template/cmd/server/commands"
	"github.com/cmelgarejo/go-modulith-template/cmd/server/observability"
	"github.com/cmelgarejo/go-modulith-template/cmd/server/setup"
	"github.com/cmelgarejo/go-modulith-template/internal/config"
	"github.com/cmelgarejo/go-modulith-template/internal/migration"
	"github.com/cmelgarejo/go-modulith-template/internal/registry"
	"google.golang.org/grpc"
)

var (
	migrateOnly = flag.Bool("migrate", false, "Run migrations only and exit")
	seedOnly    = flag.Bool("seed", false, "Run seed data only and exit")
)

func main() {
	flag.Parse()

	// Check for subcommands (non-flag arguments)
	args := flag.Args()
	if len(args) > 0 {
		handleSubcommand(args)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := setup.LoadConfig()
	if cfg == nil {
		return
	}

	shutdownObs, db := initializeServices(ctx, cfg)
	if db == nil {
		return
	}

	defer shutdownObs()
	defer setup.CloseDB(db)

	// Create registry with all dependencies
	reg := setup.CreateRegistry(cfg, db)

	// Register modules
	setup.RegisterModules(reg)

	// Initialize all modules
	if err := reg.InitializeAll(); err != nil {
		slog.Error("Failed to initialize modules", "error", err)
		return
	}

	// Run migrations for all modules
	if err := migration.NewRunner(cfg.DBDSN, reg).RunAll(); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		return
	}

	// Handle special flags (migrate-only, seed-only)
	if handleSpecialFlags(cfg.DBDSN, reg) {
		return
	}

	// Start and run the server
	runServer(ctx, cfg, reg, stop)
}

func handleSubcommand(args []string) {
	command := args[0]

	switch command {
	case "migrate":
		commands.RunMigrateCommand()
	case "migrate-down":
		commands.RunMigrateDownCommand()
	case "seed":
		commands.RunSeedCommand()
	case "admin":
		if len(args) < 2 {
			slog.Error("Usage: admin <task_name>")
			os.Exit(1)
		}

		commands.RunAdminCommand(args[1])
	default:
		slog.Error("Unknown command", "command", command)
		slog.Info("Available commands: migrate, migrate-down, seed, admin")
		os.Exit(1)
	}
}

func initializeServices(ctx context.Context, cfg *config.AppConfig) (func(), *sql.DB) {
	shutdownObs, err := observability.InitObservability(ctx, cfg)
	if err != nil {
		slog.Error("Failed to initialize observability", "error", err)
		return func() {}, nil
	}

	db, err := setup.InitDB(cfg)
	if err != nil {
		return shutdownObs, nil
	}

	return shutdownObs, db
}

func handleSpecialFlags(dbDSN string, reg *registry.Registry) bool {
	// If migrate-only flag is set, exit after migrations
	if *migrateOnly {
		slog.Info("✅ Migrations completed successfully")
		return true
	}

	// If seed-only flag is set, run seed data and exit
	if *seedOnly {
		if err := commands.RunSeedData(dbDSN, reg); err != nil {
			slog.Error("Failed to run seed data", "error", err)
			return true
		}

		slog.Info("✅ Seed data completed successfully")

		return true
	}

	return false
}

func runServer(ctx context.Context, cfg *config.AppConfig, reg *registry.Registry, stop context.CancelFunc) {
	// Call module lifecycle OnStart hooks
	if err := reg.OnStartAll(ctx); err != nil {
		slog.Error("Failed to start modules", "error", err)
		return
	}

	grpcServer, httpServer, gatewayConn := setup.AndStartServers(ctx, cfg, reg, stop)
	if grpcServer == nil {
		return
	}

	defer closeGatewayConn(gatewayConn)

	// Ensure OnStopAll is called during shutdown
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := reg.OnStopAll(shutdownCtx); err != nil {
			slog.Error("Failed to stop modules gracefully", "error", err)
		}
	}()

	<-ctx.Done()
	setup.ShutdownServers(cfg, httpServer, grpcServer, reg.WebSocketHub())
	// runServer returns after graceful shutdown, main() will exit with code 0
}

func closeGatewayConn(conn *grpc.ClientConn) {
	if conn != nil {
		if err := conn.Close(); err != nil {
			slog.Error("Failed to close gateway gRPC connection", "error", err)
		}
	}
}
