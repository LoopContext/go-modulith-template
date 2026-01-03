// Package setup provides server setup and configuration utilities.
package setup

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/cmelgarejo/go-modulith-template/internal/authn"
	"github.com/cmelgarejo/go-modulith-template/internal/config"
	"github.com/cmelgarejo/go-modulith-template/internal/registry"
	"github.com/cmelgarejo/go-modulith-template/internal/swagger"
	"github.com/cmelgarejo/go-modulith-template/internal/websocket"
	graphqlServer "github.com/cmelgarejo/go-modulith-template/internal/graphql"
	"github.com/cmelgarejo/go-modulith-template/cmd/server/health"
	"github.com/cmelgarejo/go-modulith-template/cmd/server/observability"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
)

// Gateway sets up the gRPC-Gateway HTTP server.
func Gateway(ctx context.Context, cfg *config.AppConfig, reg *registry.Registry, wsHub *websocket.Hub) (*http.ServeMux, *grpc.ClientConn, error) {
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
	health.SetupHealthChecks(mux, reg.DB(), wsHub, reg)
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

	// Setup GraphQL endpoint
	if graphqlHandler := graphqlServer.Setup(ctx, reg.EventBus(), wsHub); graphqlHandler != nil {
		mux.Handle("/graphql", graphqlHandler)

		if cfg.Env == "dev" {
			playgroundHandler := graphqlServer.PlaygroundHandler()
			mux.Handle("/graphql/playground", playgroundHandler)
			slog.Info("GraphQL playground enabled", "path", "/graphql/playground")
		}

		slog.Info("GraphQL endpoint enabled", "path", "/graphql")
	}

	if h := observability.GetMetricsHandler(); h != nil {
		mux.Handle("/metrics", h)
	}

	if cfg.Env == "dev" {
		swagger.Setup(mux, cfg.SwaggerAPITitle)
	}

	return mux, conn, nil
}

