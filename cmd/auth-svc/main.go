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
	_ = godotenv.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Load Configuration
	cfg, err := config.Load("configs/auth-svc.yaml")
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	metricsHandler, shutdownObs, err := initObservability(ctx, cfg)
	if err != nil {
		slog.Error("Failed to initialize observability", "error", err)
		os.Exit(1)
	}

	var shutdownCalled bool
	fatalExit := func() {
		if !shutdownCalled {
			shutdownCalled = true
			shutdownObs()
		}
		os.Exit(1)
	}

	db, err := sql.Open("pgx", cfg.DBDSN)
	if err != nil {
		slog.Error("Failed to open DB", "error", err)
		fatalExit()
	}

	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("Failed to close DB", "error", err)
		}
	}()

	if err := db.PingContext(ctx); err != nil {
		slog.Error("Failed to ping DB", "error", err)
		fatalExit()
	}

	if err := runMigrations(cfg.DBDSN); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		fatalExit()
	}

	httpSrv := setupHTTPServer(cfg, db, metricsHandler)
	go func() {
		slog.Info("HTTP server starting", "port", cfg.HTTPPort)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("failed to serve http", "error", err)
			stop()
		}
	}()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		slog.Error("Failed to listen", "error", err)
		fatalExit()
	}

	verifier, err := authn.NewJWTVerifier(cfg.Auth.JWTSecret)
	if err != nil {
		slog.Error("Failed to initialize jwt verifier", "error", err)
		_ = lis.Close()
		fatalExit()
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

	// Initialize Event Bus
	ebus := events.NewBus()

	// Initialize Notifier & Subscriber (Asynchronous delivery)
	ntf := notifier.NewLogNotifier()
	ns := notifier.NewSubscriber(ntf)
	ns.SubscribeToEvents(ebus)

	// Initialize ONLY the Auth module
	if err := auth.Initialize(db, grpcServer, ebus, cfg.Auth); err != nil {
		slog.Error("Failed to initialize auth module", "error", err)
		_ = lis.Close()
		fatalExit()
	}

	reflection.Register(grpcServer)

	slog.Info("Auth Microservice starting", "port", cfg.GRPCPort)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("Failed to serve", "error", err)
			stop()
		}
	}()

	<-ctx.Done()

	slog.Info("Shutting down auth microservice")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shutdown http server", "error", err)
	}

	grpcServer.GracefulStop()

	if !shutdownCalled {
		shutdownObs()
	}
}

func initObservability(ctx context.Context, cfg *config.AppConfig) (http.Handler, func(), error) {
	initLogger(cfg.Env)
	metricsHandler, metricsShutdown, err := initMetrics()
	if err != nil {
		return nil, func() {}, fmt.Errorf("failed to init metrics: %w", err)
	}

	var tracerShutdown func()
	if cfg.OTLPEndpoint != "" {
		tracerShutdown = initTracer(ctx, cfg.OTLPEndpoint, "auth-svc")
	} else {
		tracerShutdown = func() {}
	}

	return metricsHandler, func() {
		tracerShutdown()
		metricsShutdown()
	}, nil
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
