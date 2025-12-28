package auth

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	authv1 "github.com/cmelgarejo/go-modulith-template/gen/go/proto/auth/v1"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/repository"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/service"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/token"
)

// Config holds the Auth module settings
type Config struct {
	JWTSecret string
}

// Initialize registers the Auth module with the gRPC server
func Initialize(db *sql.DB, grpcServer *grpc.Server, cfg Config) error {
	tokenService, err := token.NewTokenService(cfg.JWTSecret)
	if err != nil {
		return fmt.Errorf("failed to init token service: %w", err)
	}

	repo := repository.NewSQLRepository(db)
	svc := service.NewAuthService(repo, tokenService)

	authv1.RegisterAuthServiceServer(grpcServer, svc)
	return nil
}

// RegisterGateway registers the gRPC-Gateway handler
func RegisterGateway(ctx context.Context, mux *runtime.ServeMux, grpcEndpoint string, opts []grpc.DialOption) error {
	return authv1.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, grpcEndpoint, opts)
}
