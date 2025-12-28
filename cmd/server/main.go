// Package main is the entry point for the server application.
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
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/notifier"
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
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	_ = godotenv.Load()
	ctx := context.Background()

	cfg, err := config.Load("configs/server.yaml")
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		return
	}

	initObservability(ctx, cfg)

	db, err := initDB(cfg.DBDSN)
	if err != nil {
		return
	}

	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("Failed to close DB", "error", err)
		}
	}()

	runMigrations(cfg.DBDSN)

	ebus := events.NewBus()
	initNotifier(ebus)

	grpcServer, lis := setupGRPC(cfg, db, ebus)

	go func() {
		slog.Info("Starting gRPC server", "port", cfg.GRPCPort)

		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("failed to serve gRPC", "error", err)
		}
	}()

	mux := setupGateway(ctx, cfg, db)

	slog.Info("Starting HTTP Gateway", "port", cfg.HTTPPort)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		slog.Error("failed to serve HTTP", "error", err)
	}
}

func initObservability(ctx context.Context, cfg *config.AppConfig) {
	initLogger(cfg.Env)
	initMetrics()

	if cfg.OTLPEndpoint != "" {
		shutdownTracer := initTracer(ctx, cfg.OTLPEndpoint)
		defer shutdownTracer()
	}
}

func initDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		slog.Error("Failed to open DB", "error", err)

		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		slog.Error("Failed to ping DB", "error", err)

		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("Connected to Database")

	return db, nil
}

func initNotifier(ebus *events.Bus) {
	ntf := notifier.NewLogNotifier()
	ns := notifier.NewSubscriber(ntf)
	ns.SubscribeToEvents(ebus)
}

func setupGRPC(cfg *config.AppConfig, db *sql.DB, ebus *events.Bus) (*grpc.Server, net.Listener) {
	lis, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%s", cfg.GRPCPort))
	if err != nil {
		slog.Error("failed to listen gRPC", "error", err)
		return nil, nil // Or handle error better
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	err = auth.Initialize(db, grpcServer, ebus, cfg.Auth)
	if err != nil {
		slog.Error("Failed to initialize auth module", "error", err)
		return nil, nil
	}

	reflection.Register(grpcServer)

	return grpcServer, lis
}

func setupGateway(ctx context.Context, cfg *config.AppConfig, db *sql.DB) *http.ServeMux {
	conn, err := grpc.NewClient(
		fmt.Sprintf("127.0.0.1:%s", cfg.GRPCPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		slog.Error("Failed to dial gRPC server", "error", err)
		return nil
	}

	defer func() {
		if err := conn.Close(); err != nil {
			slog.Error("Failed to close gRPC connection", "error", err)
		}
	}()

	rmux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				EmitUnpopulated: true,
				UseProtoNames:   true,
			},
		}),
	)

	if err := auth.RegisterGateway(ctx, rmux, fmt.Sprintf("127.0.0.1:%s", cfg.GRPCPort), []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}); err != nil {
		slog.Error("Failed to register auth gateway", "error", err)
	}

	mux := http.NewServeMux()
	setupHealthChecks(mux, db)
	mux.Handle("/", rmux)
	mux.Handle("/metrics", promhttp.Handler())

	if cfg.Env == "dev" {
		setupSwagger(mux)
	}

	return mux
}

func setupHealthChecks(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("OK"))
	})

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)

			_, _ = w.Write([]byte("Disconnected"))

			return
		}

		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("Ready"))
	})
}

func setupSwagger(mux *http.ServeMux) {
	slog.Info("Serving Swagger UI", "path", "/swagger-ui/")

	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "gen/openapiv2/proto/auth/v1/auth.swagger.json")
	})

	mux.HandleFunc("/swagger-ui/", func(w http.ResponseWriter, _ *http.Request) {
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

func initTracer(ctx context.Context, endpoint string) func() {
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(endpoint),
	)
	if err != nil {
		slog.Error("failed to create OTLP trace exporter", "error", err)
		return func() {}
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("modulith-server"),
		),
	)
	if err != nil {
		slog.Error("failed to create resource", "error", err)
		return func() {}
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			slog.Error("failed to shutdown tracer provider", "error", err)
		}
	}
}
