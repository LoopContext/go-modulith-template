package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time" // Added for time.Second

	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/cmelgarejo/go-modulith-template/internal/config"
	"github.com/cmelgarejo/go-modulith-template/modules/auth"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	_ = godotenv.Load()

	// 0. Load Configuration
	cfg, err := config.Load("configs/server.yaml")
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// 0.1 Observability (Logger & Metrics)
	initLogger(cfg.Env)
	initMetrics()

	// 1. Database
	db, err := sql.Open("pgx", cfg.DBDSN)
	if err != nil {
		slog.Error("Failed to open DB", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		slog.Error("Failed to ping DB", "error", err)
		os.Exit(1)
	}
	slog.Info("Connected to Database")

	// 1.5 Migrations
	runMigrations(cfg.DBDSN)

	// 2. gRPC Server Listener
	lis, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%s", cfg.GRPCPort))
	if err != nil {
		slog.Error("failed to listen gRPC", "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	// Initialize Modules
	if err := auth.Initialize(db, grpcServer, cfg.Auth); err != nil {
		slog.Error("Failed to initialize auth module", "error", err)
		os.Exit(1)
	}

	reflection.Register(grpcServer)

	// Start gRPC in goroutine
	go func() {
		slog.Info("Starting gRPC server", "port", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("failed to serve gRPC", "error", err)
			os.Exit(1)
		}
	}()

	// 3. HTTP Gateway & Swagger UI
	conn, err := grpc.NewClient(
		fmt.Sprintf("127.0.0.1:%s", cfg.GRPCPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		slog.Error("Failed to dial gRPC server", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	rmux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				EmitUnpopulated: true,
				UseProtoNames:   true,
			},
		}),
	)
	ctx := context.Background()

	// Register Module Gateways
	if err := auth.RegisterGateway(ctx, rmux, fmt.Sprintf("127.0.0.1:%s", cfg.GRPCPort), []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}); err != nil {
		slog.Error("Failed to register auth gateway", "error", err)
		os.Exit(1)
	}

	// Serve Swagger UI if ENV is dev
	mux := http.NewServeMux()
	mux.Handle("/", rmux)
	mux.Handle("/metrics", promhttp.Handler())

	if cfg.Env == "dev" {
		slog.Info("Serving Swagger UI", "path", "/swagger-ui/")
		// Serve swagger.json
		mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "gen/openapiv2/proto/auth/v1/auth.swagger.json")
		})

		// Simple HTML for Swagger UI
		mux.HandleFunc("/swagger-ui/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			if _, err := w.Write([]byte(`
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui.css" />
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-bundle.js"></script>
<script>
window.onload = () => {
  window.ui = SwaggerUIBundle({
    url: '/swagger.json',
    dom_id: '#swagger-ui',
  });
};
</script>
</body>
</html>
			`)); err != nil {
				slog.Error("failed to write swagger-ui response", "error", err)
			}
		})
	}

	slog.Info("Starting HTTP Gateway", "port", cfg.HTTPPort)
	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		slog.Error("failed to serve HTTP", "error", err)
		os.Exit(1)
	}
}

func runMigrations(dbDSN string) {
	m, err := migrate.New(
		"file://modules/auth/resources/db/migration",
		dbDSN,
	)
	if err != nil {
		slog.Error("Migration failed to initialize", "error", err)
		os.Exit(1)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		slog.Error("Migration failed to run", "error", err)
		os.Exit(1)
	}
	slog.Info("Migrations executed successfully")
}

func initLogger(env string) {
	var handler slog.Handler
	if env == "prod" {
		handler = slog.NewJSONHandler(os.Stdout, nil)
	} else {
		handler = slog.NewTextHandler(os.Stdout, nil)
	}
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func initMetrics() {
	exporter, err := prometheus.New()
	if err != nil {
		slog.Error("Failed to create prometheus exporter", "error", err)
		os.Exit(1)
	}
	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	otel.SetMeterProvider(provider)
}
