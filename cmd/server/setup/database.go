// Package setup provides server setup and configuration utilities.
package setup

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/cmelgarejo/go-modulith-template/internal/config"
)

// InitDB initializes and connects to the database pool.
func InitDB(cfg *config.AppConfig) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DBDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database DSN: %w", err)
	}

	// Configure connection pool
	poolConfig.MaxConns = int32(cfg.DBMaxOpenConns) // #nosec G115
	poolConfig.MinConns = int32(cfg.DBMaxIdleConns) // #nosec G115

	// Parse lifetime duration
	if cfg.DBConnMaxLifetime != "" {
		lifetime, err := time.ParseDuration(cfg.DBConnMaxLifetime)
		if err != nil {
			slog.Warn("Invalid DB_CONN_MAX_LIFETIME, using default", "value", cfg.DBConnMaxLifetime, "error", err)
		} else {
			poolConfig.MaxConnLifetime = lifetime
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

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("Connected to Database (pgxpool)",
		"max_conns", poolConfig.MaxConns,
		"min_conns", poolConfig.MinConns,
		"conn_max_lifetime", poolConfig.MaxConnLifetime,
		"connect_timeout", connectTimeout,
	)

	return pool, nil
}

// CloseDB closes the database connection pool.
func CloseDB(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
	}
}
