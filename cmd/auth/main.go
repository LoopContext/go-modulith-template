// Package main is the entry point for the auth service.
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

	"github.com/cmelgarejo/go-modulith-template/internal/authn"
	"github.com/cmelgarejo/go-modulith-template/internal/config"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/notifier"
	"github.com/cmelgarejo/go-modulith-template/internal/registry"
	"github.com/cmelgarejo/go-modulith-template/modules/auth"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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
	"google.golang.org/grpc/reflection"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := loadConfigAuth()
	if cfg == nil {
		return
	}

	metricsHandler, shutdownObs := initializeObservabilityAuth(ctx, cfg)
	if shutdownObs == nil {
		return
	}

	db := initializeDBAuth(ctx, cfg)
	if db == nil {
		shutdownObs()
		return
	}

	defer closeDBAuth(db)

	if err := runMigrations(cfg.DBDSN); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		shutdownObs()

		return
	}

	httpSrv, grpcServer := setupAndStartServersAuth(ctx, cfg, db, metricsHandler, stop)
	if grpcServer == nil {
		shutdownObs()
		return
	}

	<-ctx.Done()
	shutdownServersAuth(ctx, httpSrv, grpcServer, shutdownObs)
}

func loadConfigAuth() *config.AppConfig {
	initLoggerEarly()

	systemEnvVars := captureSystemEnvVars()
	_ = godotenv.Load()

	cfg, err := config.Load("configs/auth.yaml", systemEnvVars)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		return nil
	}

	initLogger(cfg.Env)

	return cfg
}

func initializeObservabilityAuth(ctx context.Context, cfg *config.AppConfig) (metricsHandler http.Handler, shutdownObs func()) {
	metricsHandler, shutdownObs, err := initObservability(ctx, cfg)
	if err != nil {
		slog.Error("Failed to initialize observability", "error", err)
		return nil, nil
	}

	return
}

func initializeDBAuth(ctx context.Context, cfg *config.AppConfig) *sql.DB {
	db, err := sql.Open("pgx", cfg.DBDSN)
	if err != nil {
		slog.Error("Failed to open DB", "error", err)
		return nil
	}

	if err := db.PingContext(ctx); err != nil {
		slog.Error("Failed to ping DB", "error", err)

		_ = db.Close()

		return nil
	}

	return db
}

func closeDBAuth(db *sql.DB) {
	if err := db.Close(); err != nil {
		slog.Error("Failed to close DB", "error", err)
	}
}

func setupAndStartServersAuth(_ context.Context, cfg *config.AppConfig, db *sql.DB, metricsHandler http.Handler, stop context.CancelFunc) (httpSrv *http.Server, grpcServer *grpc.Server) {
	httpSrv = setupHTTPServer(cfg, db, metricsHandler)
	startHTTPServerAuth(cfg, httpSrv, stop)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		slog.Error("Failed to listen", "error", err)
		return nil, nil
	}

	grpcServer = setupGRPCServerAuth(cfg, db, lis)
	if grpcServer == nil {
		_ = lis.Close()
		return nil, nil
	}

	startGRPCServerAuth(cfg, grpcServer, lis, stop)

	return
}

func startHTTPServerAuth(cfg *config.AppConfig, httpSrv *http.Server, stop context.CancelFunc) {
	go func() {
		slog.Info("HTTP server starting", "port", cfg.HTTPPort)

		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("failed to serve http", "error", err)
			stop()
		}
	}()
}

func setupGRPCServerAuth(cfg *config.AppConfig, db *sql.DB, _ net.Listener) *grpc.Server {
	verifier, err := authn.NewJWTVerifier(cfg.Auth.JWTSecret)
	if err != nil {
		slog.Error("Failed to initialize jwt verifier", "error", err)
		return nil
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

	ebus := events.NewBus()
	ntf := notifier.NewLogNotifier()
	ns := notifier.NewSubscriber(ntf, cfg.DefaultLocale)
	ns.SubscribeToEvents(ebus)

	// Create registry with dependencies
	reg := registry.New(
		registry.WithConfig(cfg),
		registry.WithDatabase(db),
		registry.WithEventBus(ebus),
		registry.WithNotifier(ntf),
	)

	// Register and initialize auth module
	authModule := auth.NewModule()
	reg.Register(authModule)

	if err := reg.InitializeAll(); err != nil {
		slog.Error("Failed to initialize auth module", "error", err)
		return nil
	}

	// Register gRPC services
	reg.RegisterGRPCAll(grpcServer)

	reflection.Register(grpcServer)

	return grpcServer
}

func startGRPCServerAuth(cfg *config.AppConfig, grpcServer *grpc.Server, lis net.Listener, stop context.CancelFunc) {
	slog.Info("Auth Microservice starting", "port", cfg.GRPCPort)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("Failed to serve", "error", err)
			stop()
		}
	}()
}

func shutdownServersAuth(_ context.Context, httpSrv *http.Server, grpcServer *grpc.Server, shutdownObs func()) {
	slog.Info("Shutting down auth microservice")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shutdown http server", "error", err)
	}

	grpcServer.GracefulStop()
	shutdownObs()
}

func initObservability(ctx context.Context, cfg *config.AppConfig) (http.Handler, func(), error) {
	// Logger already initialized in main() before config loading
	metricsHandler, metricsShutdown, err := initMetrics()
	if err != nil {
		return nil, func() {}, fmt.Errorf("failed to init metrics: %w", err)
	}

	var tracerShutdown func()
	if cfg.OTLPEndpoint != "" {
		tracerShutdown = initTracer(ctx, cfg.OTLPEndpoint, "auth")
	} else {
		tracerShutdown = func() {}
	}

	return metricsHandler, func() {
		tracerShutdown()
		metricsShutdown()
	}, nil
}

// initLoggerEarly initializes a basic logger with debug enabled before config is loaded
func initLoggerEarly() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug, // Enable debug logs
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func initLogger(env string) {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug, // Enable debug logs
	}
	if env == "prod" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(traceContextHandler{next: handler})
	slog.SetDefault(logger)
}

// captureSystemEnvVars captures system environment variables before .env is loaded
func captureSystemEnvVars() map[string]string {
	systemEnvVars := make(map[string]string)
	if env := os.Getenv("ENV"); env != "" {
		systemEnvVars["ENV"] = env
	}

	if port := os.Getenv("HTTP_PORT"); port != "" {
		systemEnvVars["HTTP_PORT"] = port
	}

	if port := os.Getenv("GRPC_PORT"); port != "" {
		systemEnvVars["GRPC_PORT"] = port
	}

	if dsn := os.Getenv("DB_DSN"); dsn != "" {
		systemEnvVars["DB_DSN"] = dsn
	}

	if endpoint := os.Getenv("OTLP_ENDPOINT"); endpoint != "" {
		systemEnvVars["OTLP_ENDPOINT"] = endpoint
	}

	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		systemEnvVars["JWT_SECRET"] = secret
	}

	return systemEnvVars
}

type traceContextHandler struct {
	next slog.Handler
}

func (h traceContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

//nolint:gocritic // slog.Record is a standard library type, cannot change signature
func (h traceContextHandler) Handle(ctx context.Context, r slog.Record) error {
	span := oteltrace.SpanFromContext(ctx)

	sc := span.SpanContext()
	if sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}

	if err := h.next.Handle(ctx, r); err != nil {
		return fmt.Errorf("failed to handle log record: %w", err)
	}

	return nil
}

func (h traceContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return traceContextHandler{next: h.next.WithAttrs(attrs)}
}

func (h traceContextHandler) WithGroup(name string) slog.Handler {
	return traceContextHandler{next: h.next.WithGroup(name)}
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
			slog.Error("Failed to shutdown meter provider", "error", err)
		}
	}, nil
}

func initTracer(ctx context.Context, endpoint, serviceName string) func() {
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
			semconv.ServiceName(serviceName),
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

func setupHTTPServer(cfg *config.AppConfig, db *sql.DB, metricsHandler http.Handler) *http.Server {
	mux := http.NewServeMux()

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

	if metricsHandler != nil {
		mux.Handle("/metrics", metricsHandler)
	}

	return &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
