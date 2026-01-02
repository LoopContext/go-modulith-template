// Package setup provides server setup and configuration utilities.
package setup

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/cmelgarejo/go-modulith-template/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib" // Register pgx driver for database/sql
)

// InitDB initializes and connects to the database.
func InitDB(cfg *config.AppConfig) (*sql.DB, error) {
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

// CloseDB closes the database connection.
func CloseDB(db *sql.DB) {
	if err := db.Close(); err != nil {
		slog.Error("Failed to close DB", "error", err)
	}
}

