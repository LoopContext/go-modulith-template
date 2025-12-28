// Package main is the entry point for the auth service.
package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/cmelgarejo/go-modulith-template/internal/config"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/notifier"
	"github.com/cmelgarejo/go-modulith-template/modules/auth"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	_ = godotenv.Load()

	// Load Configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := sql.Open("pgx", cfg.DBDSN)
	if err != nil {
		slog.Error("Failed to open DB", "error", err)
		os.Exit(1)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		slog.Error("Failed to listen", "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()

	// Initialize Event Bus
	ebus := events.NewBus()

	// Initialize Notifier & Subscriber (Asynchronous delivery)
	ntf := notifier.NewLogNotifier()
	ns := notifier.NewSubscriber(ntf)
	ns.SubscribeToEvents(ebus)

	// Initialize ONLY the Auth module
	if err := auth.Initialize(db, grpcServer, ebus, cfg.Auth); err != nil {
		slog.Error("Failed to initialize auth module", "error", err)
		os.Exit(1)
	}

	reflection.Register(grpcServer)

	slog.Info("Auth Microservice starting", "port", cfg.GRPCPort)

	if err := grpcServer.Serve(lis); err != nil {
		slog.Error("Failed to serve", "error", err)
		os.Exit(1)
	}
}
