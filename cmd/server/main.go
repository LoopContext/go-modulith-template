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
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/cmelgarejo/go-modulith-template/internal/authn"
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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	_ = godotenv.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load("configs/server.yaml")
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		return
	}

	shutdownObs, err := initObservability(ctx, cfg)
	if err != nil {
		slog.Error("Failed to initialize observability", "error", err)
		return
	}

	defer shutdownObs()

	db, err := initDB(cfg.DBDSN)
	if err != nil {
		return
	}

	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("Failed to close DB", "error", err)
		}
	}()

	if err := runMigrations(cfg.DBDSN); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		return
	}

	ebus := events.NewBus()
	initNotifier(ebus)

	grpcServer, lis, err := setupGRPC(cfg, db, ebus)
	if err != nil {
		slog.Error("Failed to setup gRPC server", "error", err)
		return
	}

	go func() {
		slog.Info("Starting gRPC server", "port", cfg.GRPCPort)

		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("failed to serve gRPC", "error", err)
			stop()
		}
	}()

	mux, gatewayConn, err := setupGateway(ctx, cfg, db)
	if err != nil {
		slog.Error("Failed to setup gateway", "error", err)
		return
	}

	defer func() {
		if gatewayConn != nil {
			if err := gatewayConn.Close(); err != nil {
				slog.Error("Failed to close gateway gRPC connection", "error", err)
			}
		}
	}()

	slog.Info("Starting HTTP Gateway", "port", cfg.HTTPPort)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("failed to serve HTTP", "error", err)
			stop()
		}
	}()

	<-ctx.Done()

	slog.Info("Shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shutdown http server", "error", err)
	}

	grpcServer.GracefulStop()
}

func initObservability(ctx context.Context, cfg *config.AppConfig) (func(), error) {
	initLogger(cfg.Env)
	metricsHandler, metricsShutdown, err := initMetrics()
	if err != nil {
		return func() {}, fmt.Errorf("failed to init metrics: %w", err)
	}

	var tracerShutdown func()
	if cfg.OTLPEndpoint != "" {
		tracerShutdown = initTracer(ctx, cfg.OTLPEndpoint)
	} else {
		tracerShutdown = func() {}
	}

	// Expose metrics handler via global var to the HTTP mux setup.
	setMetricsHandler(metricsHandler)

	return func() {
		tracerShutdown()
		metricsShutdown()
	}, nil
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

func setupGRPC(cfg *config.AppConfig, db *sql.DB, ebus *events.Bus) (*grpc.Server, net.Listener, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to listen gRPC: %w", err)
	}

	verifier, err := authn.NewJWTVerifier(cfg.Auth.JWTSecret)
	if err != nil {
		_ = lis.Close()
		return nil, nil, fmt.Errorf("failed to init jwt verifier: %w", err)
	}

	public := map[string]struct{}{
		"/auth.v1.AuthService/RequestLogin":  {},
		"/auth.v1.AuthService/CompleteLogin": {},
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(authn.UnaryServerInterceptor(authn.InterceptorConfig{
			Verifier:      verifier,
			PublicMethods: public,
		})),
	)

	err = auth.Initialize(db, grpcServer, ebus, cfg.Auth)
	if err != nil {
		_ = lis.Close()
		return nil, nil, fmt.Errorf("failed to initialize auth module: %w", err)
	}

	reflection.Register(grpcServer)

	return grpcServer, lis, nil
}

func setupGateway(ctx context.Context, cfg *config.AppConfig, db *sql.DB) (*http.ServeMux, *grpc.ClientConn, error) {
	rmux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				EmitUnpopulated: true,
				UseProtoNames:   true,
			},
		}),
	)

	// Create gRPC connection explicitly to manage its lifecycle
	conn, err := grpc.NewClient(
		fmt.Sprintf("127.0.0.1:%s", cfg.GRPCPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial gRPC server: %w", err)
	}

	// Register gateway handlers using the explicit connection
	if err := auth.RegisterGatewayWithConn(ctx, rmux, conn); err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("failed to register auth gateway: %w", err)
	}

	mux := http.NewServeMux()
	setupHealthChecks(mux, db)
	mux.Handle("/", rmux)
	if h := getMetricsHandler(); h != nil {
		mux.Handle("/metrics", h)
	}

	if cfg.Env == "dev" {
		setupSwagger(mux)
	}

	return mux, conn, nil
}

func setupHealthChecks(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("OK"))
	})

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
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

func runMigrations(dbDSN string) error {
	m, err := migrate.New(
		"file://modules/auth/resources/db/migration",
		dbDSN,
	)
	if err != nil {
		return fmt.Errorf("migration failed to initialize: %w", err)
	}

	defer func() {
		sourceErr, dbErr := m.Close()
		if sourceErr != nil {
			slog.Error("Failed to close migration source", "error", sourceErr)
		}
		if dbErr != nil {
			slog.Error("Failed to close migration database connection", "error", dbErr)
		}
	}()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed to run: %w", err)
	}

	slog.Info("Migrations executed successfully")

	return nil
}

func initLogger(env string) {
	var handler slog.Handler
	if env == "prod" {
		handler = slog.NewJSONHandler(os.Stdout, nil)
	} else {
		handler = slog.NewTextHandler(os.Stdout, nil)
	}

	logger := slog.New(traceContextHandler{next: handler})
	slog.SetDefault(logger)
}

type traceContextHandler struct {
	next slog.Handler
}

func (h traceContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h traceContextHandler) Handle(ctx context.Context, r slog.Record) error {
	span := oteltrace.SpanFromContext(ctx)
	sc := span.SpanContext()
	if sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}

	return h.next.Handle(ctx, r)
}

func (h traceContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return traceContextHandler{next: h.next.WithAttrs(attrs)}
}

func (h traceContextHandler) WithGroup(name string) slog.Handler {
	return traceContextHandler{next: h.next.WithGroup(name)}
}

var metricsHandler http.Handler

func setMetricsHandler(h http.Handler) {
	metricsHandler = h
}

func getMetricsHandler() http.Handler {
	return metricsHandler
}

func initMetrics() (http.Handler, func(), error) {
	reg := prometheus.NewRegistry()

	exporter, err := otelprom.New(otelprom.WithRegisterer(reg))
	if err != nil {
		return nil, func() {}, fmt.Errorf("failed to create prometheus exporter: %w", err)
	}

	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	otel.SetMeterProvider(provider)

	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{}), func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			slog.Error("failed to shutdown meter provider", "error", err)
		}
	}, nil
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
