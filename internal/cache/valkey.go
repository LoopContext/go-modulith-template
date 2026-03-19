// Package cache provides a caching abstraction for the application.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/cmelgarejo/go-modulith-template/internal/telemetry"
	"github.com/redis/go-redis/v9"
)

// ValkeyConfig holds Valkey connection configuration.
type ValkeyConfig struct {
	// Addr is the server address (e.g., "localhost:6379").
	Addr string
	// Password is the password (optional).
	Password string //nolint:gosec
	// DB is the database number (default 0).
	DB int
	// PoolSize is the maximum number of connections.
	PoolSize int
	// MinIdleConns is the minimum number of idle connections.
	MinIdleConns int
	// DialTimeout is the timeout for establishing new connections.
	DialTimeout time.Duration
	// ReadTimeout is the timeout for read operations.
	ReadTimeout time.Duration
	// WriteTimeout is the timeout for write operations.
	WriteTimeout time.Duration
}

// DefaultValkeyConfig returns default Valkey configuration.
func DefaultValkeyConfig() ValkeyConfig {
	return ValkeyConfig{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 2,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}

// ValkeyCache is a Valkey-backed cache implementation (using Redis protocol).
type ValkeyCache struct {
	config ValkeyConfig
	client *redis.Client
}

// NewValkeyCache creates a new Valkey cache.
// Returns an error if connection fails.
func NewValkeyCache(cfg ValkeyConfig) (*ValkeyCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("valkey connection failed: %w", err)
	}

	return &ValkeyCache{
		config: cfg,
		client: client,
	}, nil
}

// Get retrieves a value from the cache.
func (rc *ValkeyCache) Get(ctx context.Context, key string) ([]byte, error) {
	ctx, span := telemetry.ModuleSpan(ctx, "cache", "Get")
	defer span.End()

	val, err := rc.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("valkey get: %w", err)
	}

	return val, nil
}

// Set stores a value in the cache.
func (rc *ValkeyCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	ctx, span := telemetry.ModuleSpan(ctx, "cache", "Set")
	defer span.End()

	if err := rc.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("valkey set: %w", err)
	}

	return nil
}

// Delete removes a value from the cache.
func (rc *ValkeyCache) Delete(ctx context.Context, key string) error {
	ctx, span := telemetry.ModuleSpan(ctx, "cache", "Delete")
	defer span.End()

	if err := rc.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("valkey delete: %w", err)
	}

	return nil
}

// DeleteMany removes multiple values from the cache.
func (rc *ValkeyCache) DeleteMany(ctx context.Context, keys ...string) error {
	ctx, span := telemetry.ModuleSpan(ctx, "cache", "DeleteMany")
	defer span.End()

	if len(keys) == 0 {
		return nil
	}

	if err := rc.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("valkey delete many: %w", err)
	}

	return nil
}

// DeleteByPrefix removes all keys matching a prefix using SCAN to avoid blocking Redis.
func (rc *ValkeyCache) DeleteByPrefix(ctx context.Context, prefix string) error {
	ctx, span := telemetry.ModuleSpan(ctx, "cache", "DeleteByPrefix")
	defer span.End()

	var cursor uint64

	for {
		keys, nextCursor, err := rc.client.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return fmt.Errorf("valkey delete by prefix scan: %w", err)
		}

		if len(keys) > 0 {
			if err := rc.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("valkey delete by prefix delete: %w", err)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			return nil
		}
	}
}

// Exists checks if a key exists in the cache.
func (rc *ValkeyCache) Exists(ctx context.Context, key string) (bool, error) {
	ctx, span := telemetry.ModuleSpan(ctx, "cache", "Exists")
	defer span.End()

	n, err := rc.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("valkey exists: %w", err)
	}

	return n > 0, nil
}

// Increment increments a numeric value in the cache.
func (rc *ValkeyCache) Increment(ctx context.Context, key string) (int64, error) {
	ctx, span := telemetry.ModuleSpan(ctx, "cache", "Increment")
	defer span.End()

	n, err := rc.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("valkey incr: %w", err)
	}

	return n, nil
}

// Decrement decrements a numeric value in the cache.
func (rc *ValkeyCache) Decrement(ctx context.Context, key string) (int64, error) {
	ctx, span := telemetry.ModuleSpan(ctx, "cache", "Decrement")
	defer span.End()

	n, err := rc.client.Decr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("valkey decr: %w", err)
	}

	return n, nil
}

// Expire sets a new expiration time for a key.
func (rc *ValkeyCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	ctx, span := telemetry.ModuleSpan(ctx, "cache", "Expire")
	defer span.End()

	if err := rc.client.Expire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("valkey expire: %w", err)
	}

	return nil
}

// Close closes the connection.
func (rc *ValkeyCache) Close() error {
	if err := rc.client.Close(); err != nil {
		return fmt.Errorf("valkey close: %w", err)
	}

	return nil
}

// Ping checks the connection.
func (rc *ValkeyCache) Ping(ctx context.Context) error {
	if err := rc.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("valkey ping: %w", err)
	}

	return nil
}
