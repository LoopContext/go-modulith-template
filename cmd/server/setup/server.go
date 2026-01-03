// Package setup provides server setup and configuration utilities.
package setup

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/cmelgarejo/go-modulith-template/internal/authn"
	"github.com/cmelgarejo/go-modulith-template/internal/config"
	"github.com/cmelgarejo/go-modulith-template/internal/i18n"
	"github.com/cmelgarejo/go-modulith-template/internal/middleware"
	"github.com/cmelgarejo/go-modulith-template/internal/registry"
	"github.com/cmelgarejo/go-modulith-template/internal/validation"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// GRPC sets up the gRPC server with interceptors and registers all modules.
func GRPC(cfg *config.AppConfig, reg *registry.Registry) (*grpc.Server, net.Listener, error) {
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

// AndStartServers sets up and starts both gRPC and HTTP servers.
func AndStartServers(ctx context.Context, cfg *config.AppConfig, reg *registry.Registry, stop context.CancelFunc) (*grpc.Server, *http.Server, *grpc.ClientConn) {
	// Setup gRPC server
	grpcServer, lis, err := GRPC(cfg, reg)
	if err != nil {
		slog.Error("Failed to setup gRPC server", "error", err)
		return nil, nil, nil
	}

	// Setup HTTP gateway
	mux, gatewayConn, err := Gateway(ctx, cfg, reg, reg.WebSocketHub())
	if err != nil {
		_ = lis.Close()

		slog.Error("Failed to setup gateway", "error", err)

		return nil, nil, nil
	}

	httpServer := StartHTTPServer(cfg, mux)
	StartGRPCServer(cfg, grpcServer, lis, stop)

	return grpcServer, httpServer, gatewayConn
}

// StartHTTPServer creates and starts the HTTP server.
func StartHTTPServer(cfg *config.AppConfig, mux *http.ServeMux) *http.Server {
	handler := BuildHTTPHandler(cfg, mux)
	server := CreateHTTPServer(cfg, handler)
	StartServerAsync(server)

	return server
}

// BuildHTTPHandler builds the HTTP handler with all middleware.
func BuildHTTPHandler(cfg *config.AppConfig, mux *http.ServeMux) http.Handler {
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

// CreateHTTPServer creates an HTTP server with the given handler.
func CreateHTTPServer(cfg *config.AppConfig, handler http.Handler) *http.Server {
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

// StartServerAsync starts the HTTP server in a goroutine.
func StartServerAsync(server *http.Server) {
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("failed to serve HTTP", "error", err)
		}
	}()
}

// StartGRPCServer starts the gRPC server in a goroutine.
func StartGRPCServer(cfg *config.AppConfig, grpcServer *grpc.Server, lis net.Listener, stop context.CancelFunc) {
	go func() {
		slog.Info("Starting gRPC server", "port", cfg.GRPCPort)

		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("failed to serve gRPC", "error", err)
			stop()
		}
	}()
}

// ShutdownServers gracefully shuts down HTTP and gRPC servers.
func ShutdownServers(cfg *config.AppConfig, httpServer *http.Server, grpcServer *grpc.Server, wsHub interface {
	GetTotalConnections() int
	Stop()
}) {
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
