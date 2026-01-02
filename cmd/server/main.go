// Package main is the entry point for the server application.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
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

	"github.com/cmelgarejo/go-modulith-template/internal/admin"
	adminTasks "github.com/cmelgarejo/go-modulith-template/internal/admin/tasks"
	"github.com/cmelgarejo/go-modulith-template/internal/authn"
	"github.com/cmelgarejo/go-modulith-template/internal/config"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/i18n"
	"github.com/cmelgarejo/go-modulith-template/internal/middleware"
	"github.com/cmelgarejo/go-modulith-template/internal/migration"
	"github.com/cmelgarejo/go-modulith-template/internal/notifier"
	"github.com/cmelgarejo/go-modulith-template/internal/registry"
	"github.com/cmelgarejo/go-modulith-template/internal/swagger"
	"github.com/cmelgarejo/go-modulith-template/internal/validation"
	"github.com/cmelgarejo/go-modulith-template/internal/version"
	"github.com/cmelgarejo/go-modulith-template/internal/websocket"
	"github.com/cmelgarejo/go-modulith-template/modules/auth"
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

	cfg := loadConfig()
	if cfg == nil {
		return
	}

	shutdownObs, db := initializeServices(ctx, cfg)
	if db == nil {
		return
	}

	defer shutdownObs()
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

	// Handle special flags (migrate-only, seed-only)
	if handleSpecialFlags(cfg.DBDSN, reg) {
		return
	}

	// Start and run the server
	runServer(ctx, cfg, reg, stop)
}

func handleSpecialFlags(dbDSN string, reg *registry.Registry) bool {
	// If migrate-only flag is set, exit after migrations
	if *migrateOnly {
		slog.Info("✅ Migrations completed successfully")
		return true
	}

	// If seed-only flag is set, run seed data and exit
	if *seedOnly {
		if err := runSeedData(dbDSN, reg); err != nil {
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

	grpcServer, httpServer, gatewayConn := setupAndStartServers(ctx, cfg, reg, stop)
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
	shutdownServers(cfg, httpServer, grpcServer, reg.WebSocketHub())
}

func loadConfig() *config.AppConfig {
	initLoggerEarly()

	systemEnvVars := captureSystemEnvVars()
	_ = godotenv.Load()

	cfg, err := config.Load("configs/server.yaml", systemEnvVars)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		return nil
	}

	initLogger(cfg.Env, cfg.LogLevel)

	// Initialize i18n
	if err := i18n.Init(cfg.DefaultLocale); err != nil {
		slog.Error("Failed to initialize i18n", "error", err)
		return nil
	}

	slog.Info("Starting application", "version", version.Info())

	return cfg
}

func initializeServices(ctx context.Context, cfg *config.AppConfig) (func(), *sql.DB) {
	shutdownObs, err := initObservability(ctx, cfg)
	if err != nil {
		slog.Error("Failed to initialize observability", "error", err)
		return func() {}, nil
	}

	db, err := initDB(cfg)
	if err != nil {
		return shutdownObs, nil
	}

	return shutdownObs, db
}

func closeDB(db *sql.DB) {
	if err := db.Close(); err != nil {
		slog.Error("Failed to close DB", "error", err)
	}
}

func createRegistry(cfg *config.AppConfig, db *sql.DB) *registry.Registry {
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

func registerModules(reg *registry.Registry) {
	// Register all modules here
	reg.Register(auth.NewModule())
	// Add more modules as needed:
	// reg.Register(order.NewModule())
	// reg.Register(payment.NewModule())
}

func setupAndStartServers(ctx context.Context, cfg *config.AppConfig, reg *registry.Registry, stop context.CancelFunc) (grpcServer *grpc.Server, httpServer *http.Server, gatewayConn *grpc.ClientConn) {
	// Setup gRPC server
	grpcServer, lis, err := setupGRPC(cfg, reg)
	if err != nil {
		slog.Error("Failed to setup gRPC server", "error", err)
		return nil, nil, nil
	}

	// Setup HTTP gateway
	mux, gatewayConn, err := setupGateway(ctx, cfg, reg, reg.WebSocketHub())
	if err != nil {
		_ = lis.Close()

		slog.Error("Failed to setup gateway", "error", err)

		return nil, nil, nil
	}

	httpServer = startHTTPServer(cfg, mux)
	startGRPCServer(cfg, grpcServer, lis, stop)

	return grpcServer, httpServer, gatewayConn
}

func closeGatewayConn(conn *grpc.ClientConn) {
	if conn != nil {
		if err := conn.Close(); err != nil {
			slog.Error("Failed to close gateway gRPC connection", "error", err)
		}
	}
}

func startHTTPServer(cfg *config.AppConfig, mux *http.ServeMux) *http.Server {
	handler := buildHTTPHandler(cfg, mux)
	server := createHTTPServer(cfg, handler)
	startServerAsync(server)

	return server
}

func buildHTTPHandler(cfg *config.AppConfig, mux *http.ServeMux) http.Handler {
	// Wrap with middleware (innermost first)
	var handler http.Handler = mux

	// Apply CORS middleware
	corsConfig := middleware.DefaultCORSConfig()
	if len(cfg.CORSAllowedOrigins) > 0 {
		corsConfig.AllowedOrigins = cfg.CORSAllowedOrigins
	}

	handler = middleware.CORS(corsConfig)(handler)

	// Apply rate limiting middleware if enabled
	if cfg.RateLimitEnabled {
		rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)
		handler = rateLimiter.Middleware()(handler)

		slog.Info("Rate limiting enabled",
			"rps", cfg.RateLimitRPS,
			"burst", cfg.RateLimitBurst,
		)
	}

	// Apply timeout middleware (enforces maximum request duration)
	requestTimeout, err := time.ParseDuration(cfg.RequestTimeout)
	if err != nil {
		slog.Warn("Invalid REQUEST_TIMEOUT, using default 30s", "value", cfg.RequestTimeout, "error", err)

		requestTimeout = 30 * time.Second
	}

	handler = middleware.Timeout(requestTimeout)(handler)

	// Apply logging middleware (logs requests with method, path, status, duration)
	handler = middleware.LoggingWithDefaults()(handler)

	// Apply request ID middleware (outermost - ensures request_id is available for logging)
	handler = middleware.RequestID(handler)

	return handler
}

func createHTTPServer(cfg *config.AppConfig, handler http.Handler) *http.Server {
	// Parse timeout configurations
	readTimeout, err := time.ParseDuration(cfg.ReadTimeout)
	if err != nil {
		slog.Warn("Invalid READ_TIMEOUT, using default 5s", "value", cfg.ReadTimeout, "error", err)

		readTimeout = 5 * time.Second
	}

	writeTimeout, err := time.ParseDuration(cfg.WriteTimeout)
	if err != nil {
		slog.Warn("Invalid WRITE_TIMEOUT, using default 10s", "value", cfg.WriteTimeout, "error", err)

		writeTimeout = 10 * time.Second
	}

	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler:           handler,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.Info("Starting HTTP Gateway",
		"port", cfg.HTTPPort,
		"read_timeout", readTimeout,
		"write_timeout", writeTimeout,
	)

	return server
}

func startServerAsync(server *http.Server) {
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("failed to serve HTTP", "error", err)
		}
	}()
}

func startGRPCServer(cfg *config.AppConfig, grpcServer *grpc.Server, lis net.Listener, stop context.CancelFunc) {
	go func() {
		slog.Info("Starting gRPC server", "port", cfg.GRPCPort)

		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("failed to serve gRPC", "error", err)
			stop()
		}
	}()
}

func shutdownServers(cfg *config.AppConfig, httpServer *http.Server, grpcServer *grpc.Server, wsHub *websocket.Hub) {
	slog.Info("Shutting down server")

	// Parse shutdown timeout
	shutdownTimeout, err := time.ParseDuration(cfg.ShutdownTimeout)
	if err != nil {
		slog.Warn("Invalid SHUTDOWN_TIMEOUT, using default 30s", "value", cfg.ShutdownTimeout, "error", err)

		shutdownTimeout = 30 * time.Second
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Step 1: Stop accepting new connections
	slog.Info("Stopping new connections")

	// Step 2: Gracefully close WebSocket connections
	if wsHub != nil {
		slog.Info("Closing WebSocket connections", "active_connections", wsHub.GetTotalConnections())
		wsHub.Stop()
		// Give WebSocket hub time to close connections gracefully
		time.Sleep(2 * time.Second)
	}

	// Step 3: Shutdown HTTP server (waits for in-flight requests)
	slog.Info("Shutting down HTTP server", "timeout", shutdownTimeout)

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("Failed to shutdown HTTP server", "error", err)
	} else {
		slog.Info("HTTP server shutdown complete")
	}

	// Step 4: Shutdown gRPC server
	slog.Info("Shutting down gRPC server")
	grpcServer.GracefulStop()
	slog.Info("gRPC server shutdown complete")

	// Step 5: Flush telemetry data (if needed)
	// OpenTelemetry SDK handles this automatically via shutdown hooks
	slog.Info("Server shutdown complete")
}

func initObservability(ctx context.Context, cfg *config.AppConfig) (func(), error) {
	// Logger already initialized in main() before config loading
	metricsHandler, metricsShutdown, err := initMetrics()
	if err != nil {
		return func() {}, fmt.Errorf("failed to init metrics: %w", err)
	}

	var tracerShutdown func()
	if cfg.OTLPEndpoint != "" {
		tracerShutdown = initTracer(ctx, cfg.OTLPEndpoint, cfg.ServiceName)
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

func initDB(cfg *config.AppConfig) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DBDSN)
	if err != nil {
		slog.Error("Failed to open DB", "error", err)

		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)

	// Parse lifetime duration
	if cfg.DBConnMaxLifetime != "" {
		lifetime, err := time.ParseDuration(cfg.DBConnMaxLifetime)
		if err != nil {
			slog.Warn("Invalid DB_CONN_MAX_LIFETIME, using default", "value", cfg.DBConnMaxLifetime, "error", err)
		} else {
			db.SetConnMaxLifetime(lifetime)
		}
	}

	// Parse connect timeout and ping with context
	connectTimeout := 10 * time.Second // default

	if cfg.DBConnectTimeout != "" {
		if parsed, err := time.ParseDuration(cfg.DBConnectTimeout); err != nil {
			slog.Warn("Invalid DB_CONNECT_TIMEOUT, using default 10s", "value", cfg.DBConnectTimeout, "error", err)
		} else {
			connectTimeout = parsed
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		slog.Error("Failed to ping DB", "error", err, "timeout", connectTimeout)

		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("Connected to Database",
		"max_open_conns", cfg.DBMaxOpenConns,
		"max_idle_conns", cfg.DBMaxIdleConns,
		"conn_max_lifetime", cfg.DBConnMaxLifetime,
		"connect_timeout", connectTimeout,
	)

	return db, nil
}

func setupGRPC(cfg *config.AppConfig, reg *registry.Registry) (*grpc.Server, net.Listener, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to listen gRPC: %w", err)
	}

	verifier, err := authn.NewJWTVerifier(cfg.Auth.JWTSecret)
	if err != nil {
		_ = lis.Close()
		return nil, nil, fmt.Errorf("failed to init jwt verifier: %w", err)
	}

	// Get public endpoints from all modules
	public := reg.GetPublicEndpoints()

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			i18n.UnaryServerInterceptor(cfg.DefaultLocale), // Detect locale first
			validation.UnaryServerInterceptor(),            // Validate requests
			authn.UnaryServerInterceptor(authn.InterceptorConfig{
				Verifier:      verifier,
				PublicMethods: public,
			}),
		),
	)

	// Register all modules with gRPC server
	reg.RegisterGRPCAll(grpcServer)

	reflection.Register(grpcServer)

	return grpcServer, lis, nil
}

func setupGateway(ctx context.Context, cfg *config.AppConfig, reg *registry.Registry, wsHub *websocket.Hub) (*http.ServeMux, *grpc.ClientConn, error) {
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

	// Register all modules with gateway
	if err := reg.RegisterGatewayAll(ctx, rmux, conn); err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("failed to register gateway: %w", err)
	}

	mux := http.NewServeMux()
	setupHealthChecks(mux, reg.DB(), wsHub, reg)
	mux.Handle("/", rmux)

	// Setup WebSocket endpoint with security
	verifier, err := authn.NewJWTVerifier(cfg.Auth.JWTSecret)
	if err != nil {
		slog.Warn("Failed to create JWT verifier for WebSocket, connections will be unauthenticated",
			"error", err)

		verifier = nil
	}

	wsHandler := websocket.NewHandler(websocket.HandlerConfig{
		Hub:            wsHub,
		Verifier:       verifier,
		AllowedOrigins: cfg.CORSAllowedOrigins,
		Env:            cfg.Env,
	})
	mux.Handle("/ws", wsHandler)
	slog.Info("WebSocket endpoint registered", "path", "/ws", "auth_enabled", verifier != nil)

	if h := getMetricsHandler(); h != nil {
		mux.Handle("/metrics", h)
	}

	if cfg.Env == "dev" {
		swagger.Setup(mux)
	}

	return mux, conn, nil
}

const healthStatusHealthy = "healthy"

func setupHealthChecks(mux *http.ServeMux, db *sql.DB, wsHub *websocket.Hub, reg *registry.Registry) {
	setupLivenessProbe(mux)
	setupReadinessProbe(mux, db, wsHub, reg)
	setupWebSocketHealthCheck(mux, wsHub)
}

func setupLivenessProbe(mux *http.ServeMux) {
	// Liveness probe - always returns 200 if process is alive
	mux.HandleFunc("/livez", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Legacy healthz endpoint (same as livez for backward compatibility)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
}

func setupReadinessProbe(mux *http.ServeMux, db *sql.DB, wsHub *websocket.Hub, reg *registry.Registry) {
	// Readiness probe - checks all dependencies
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		status := map[string]interface{}{
			"status": "ready",
			"checks": make(map[string]string),
		}

		checks := status["checks"].(map[string]string)
		allHealthy := checkReadinessDependencies(r.Context(), checks, db, wsHub, reg)

		if !allHealthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		// Write JSON response
		jsonData, _ := json.Marshal(status)
		_, _ = w.Write(jsonData)
	})
}

func checkReadinessDependencies(ctx context.Context, checks map[string]string, db *sql.DB, wsHub *websocket.Hub, reg *registry.Registry) bool {
	allHealthy := true

	// Check module health
	if err := reg.HealthCheckAll(ctx); err != nil {
		checks["modules"] = fmt.Sprintf("unhealthy: %v", err)
		allHealthy = false
	} else {
		checks["modules"] = healthStatusHealthy
	}

	// Check database connectivity
	if err := db.PingContext(ctx); err != nil {
		checks["database"] = fmt.Sprintf("unhealthy: %v", err)
		allHealthy = false
	} else {
		checks["database"] = healthStatusHealthy
	}

	// Check event bus (basic check - if it exists, it's healthy)
	if reg.EventBus() != nil {
		checks["event_bus"] = healthStatusHealthy
	} else {
		checks["event_bus"] = "unhealthy: not initialized"
		allHealthy = false
	}

	// Check WebSocket hub
	if wsHub != nil {
		checks["websocket"] = healthStatusHealthy
	} else {
		checks["websocket"] = "unhealthy: not initialized"
		allHealthy = false
	}

	return allHealthy
}

func setupWebSocketHealthCheck(mux *http.ServeMux, wsHub *websocket.Hub) {
	// WebSocket connections health check
	mux.HandleFunc("/healthz/ws", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := fmt.Sprintf(`{"status":"ok","connections":%d,"users":%d}`,
			wsHub.GetTotalConnections(),
			wsHub.GetConnectedUsers())

		_, _ = w.Write([]byte(response))
	})
}

func runMigrations(dbDSN string, reg *registry.Registry) error {
	runner := migration.NewRunner(dbDSN, reg)
	if err := runner.RunAll(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func runSeedData(dbDSN string, reg *registry.Registry) error {
	// Create adapter for the registry to match migration.ModuleRegistry interface
	adapter := &registryAdapter{reg: reg}

	seeder, err := migration.NewSeeder(dbDSN, adapter)
	if err != nil {
		return fmt.Errorf("failed to create seeder: %w", err)
	}

	defer func() {
		if err := seeder.Close(); err != nil {
			slog.Error("Failed to close seeder connection", "error", err)
		}
	}()

	if err := seeder.SeedAll(context.Background()); err != nil {
		return fmt.Errorf("failed to run seed data: %w", err)
	}

	return nil
}

// registryAdapter adapts registry.Registry to migration.ModuleRegistry.
type registryAdapter struct {
	reg *registry.Registry
}

func (r *registryAdapter) Modules() []interface{} {
	modules := r.reg.Modules()
	result := make([]interface{}, len(modules))

	for i, mod := range modules {
		result[i] = mod
	}

	return result
}

func handleSubcommand(args []string) {
	command := args[0]

	switch command {
	case "migrate":
		runMigrateCommand()
	case "seed":
		runSeedCommand()
	case "admin":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: %s admin <task_name>\n", os.Args[0])
			os.Exit(1)
		}

		runAdminCommand(args[1])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		fmt.Fprintf(os.Stderr, "Available commands: migrate, seed, admin\n")
		os.Exit(1)
	}
}

func runMigrateCommand() {
	initLoggerEarly()

	systemEnvVars := captureSystemEnvVars()
	_ = godotenv.Load()

	cfg, err := config.Load("configs/server.yaml", systemEnvVars)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	initLogger(cfg.Env, cfg.LogLevel)

	db, err := initDB(cfg)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	reg := createRegistry(cfg, db)
	registerModules(reg)

	if err := reg.InitializeAll(); err != nil {
		closeDB(db)
		slog.Error("Failed to initialize modules", "error", err)
		os.Exit(1)
	}

	if err := runMigrations(cfg.DBDSN, reg); err != nil {
		closeDB(db)
		slog.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	closeDB(db)
	slog.Info("✅ Migrations completed successfully")
}

func runSeedCommand() {
	initLoggerEarly()

	systemEnvVars := captureSystemEnvVars()
	_ = godotenv.Load()

	cfg, err := config.Load("configs/server.yaml", systemEnvVars)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	initLogger(cfg.Env, cfg.LogLevel)

	db, err := initDB(cfg)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	reg := createRegistry(cfg, db)
	registerModules(reg)

	if err := reg.InitializeAll(); err != nil {
		closeDB(db)
		slog.Error("Failed to initialize modules", "error", err)
		os.Exit(1)
	}

	if err := runSeedData(cfg.DBDSN, reg); err != nil {
		closeDB(db)
		slog.Error("Failed to run seed data", "error", err)
		os.Exit(1)
	}

	closeDB(db)
	slog.Info("✅ Seed data completed successfully")
}

func runAdminCommand(taskName string) {
	initLoggerEarly()

	systemEnvVars := captureSystemEnvVars()
	_ = godotenv.Load()

	cfg, err := config.Load("configs/server.yaml", systemEnvVars)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	initLogger(cfg.Env, cfg.LogLevel)

	db, err := initDB(cfg)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	reg := createRegistry(cfg, db)
	registerModules(reg)

	if err := reg.InitializeAll(); err != nil {
		closeDB(db)
		slog.Error("Failed to initialize modules", "error", err)
		os.Exit(1)
	}

	runner := admin.NewRunner()

	// Register example admin tasks
	adminTasks.RegisterExampleTasks(runner, db)

	// TODO: Modules can register admin tasks here via an interface
	// For now, show available tasks
	if !runner.Has(taskName) {
		closeDB(db)
		slog.Error("Unknown admin task", "task", taskName)

		tasks := runner.List()
		if len(tasks) == 0 {
			slog.Info("No admin tasks registered")
		} else {
			slog.Info("Available admin tasks:")

			for _, t := range tasks {
				slog.Info("  " + t.Name() + " - " + t.Description())
			}
		}

		os.Exit(1)
	}

	if err := runner.Run(context.Background(), taskName); err != nil {
		closeDB(db)
		slog.Error("Admin task failed", "task", taskName, "error", err)
		os.Exit(1)
	}

	closeDB(db)
	slog.Info("✅ Admin task completed successfully", "task", taskName)
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

func initLogger(env string, logLevel string) {
	var handler slog.Handler

	// Parse log level
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

	logger := slog.New(traceContextHandler{next: handler})
	slog.SetDefault(logger)
}

// captureSystemEnvVars captures system environment variables before .env is loaded
func captureSystemEnvVars() map[string]string {
	systemEnvVars := make(map[string]string)
	if env := os.Getenv("ENV"); env != "" {
		systemEnvVars["ENV"] = env
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		systemEnvVars["LOG_LEVEL"] = logLevel
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
