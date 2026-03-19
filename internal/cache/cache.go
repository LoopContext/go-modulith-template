// Package cache provides a caching abstraction for the application.
// It supports multiple backends (memory, Valkey) and can be used for
// session storage, rate limiting data, or general-purpose caching.
package cache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Common errors returned by cache implementations.
var (
	// ErrNotFound is returned when a key is not found in the cache.
	ErrNotFound = errors.New("cache: key not found")
	// ErrExpired is returned when a key has expired.
	ErrExpired = errors.New("cache: key expired")
)

// Cache defines the interface for cache operations.
// All implementations must be safe for concurrent use.
type Cache interface {
	// Get retrieves a value from the cache.
	// Returns ErrNotFound if the key doesn't exist.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value in the cache with an optional TTL.
	// If ttl is 0, the value never expires.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a value from the cache.
	// Returns nil if the key doesn't exist.
	Delete(ctx context.Context, key string) error

	// DeleteMany removes multiple values from the cache.
	// Returns nil if no keys exist.
	DeleteMany(ctx context.Context, keys ...string) error

	// DeleteByPrefix removes all values that share a common key prefix.
	// Implementations should treat this as a best-effort bulk invalidation helper.
	DeleteByPrefix(ctx context.Context, prefix string) error

	// Exists checks if a key exists in the cache.
	Exists(ctx context.Context, key string) (bool, error)

	// Increment increments a numeric value in the cache.
	// Returns the new value.
	Increment(ctx context.Context, key string) (int64, error)

	// Decrement decrements a numeric value in the cache.
	// Returns the new value.
	Decrement(ctx context.Context, key string) (int64, error)

	// Expire sets a new expiration time for a key.
	Expire(ctx context.Context, key string, ttl time.Duration) error

	// Ping checks the cache connection.
	Ping(ctx context.Context) error

	// Close closes the cache connection.
	Close() error
}

// StringCache provides string-typed convenience methods on top of Cache.
// This is a thin wrapper that delegates all operations to the underlying Cache.
//
//nolint:wrapcheck // This is a thin wrapper that intentionally passes through errors
type StringCache struct {
	cache Cache
}

// Key builds a standardized cache key using ":" separators while skipping empty parts.
func Key(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}

		filtered = append(filtered, strings.Trim(part, ":"))
	}

	return strings.Join(filtered, ":")
}

// NewStringCache wraps a Cache with string convenience methods.
func NewStringCache(c Cache) *StringCache {
	return &StringCache{cache: c}
}

// Get retrieves a string value from the cache.
//
//nolint:wrapcheck // Thin wrapper passes through errors
func (sc *StringCache) Get(ctx context.Context, key string) (string, error) {
	data, err := sc.cache.Get(ctx, key)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// Set stores a string value in the cache.
//
//nolint:wrapcheck // Thin wrapper passes through errors
func (sc *StringCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return sc.cache.Set(ctx, key, []byte(value), ttl)
}

// Delete removes a value from the cache.
//
//nolint:wrapcheck // Thin wrapper passes through errors
func (sc *StringCache) Delete(ctx context.Context, key string) error {
	return sc.cache.Delete(ctx, key)
}

// DeleteMany removes multiple values from the cache.
//
//nolint:wrapcheck // Thin wrapper passes through errors
func (sc *StringCache) DeleteMany(ctx context.Context, keys ...string) error {
	return sc.cache.DeleteMany(ctx, keys...)
}

// DeleteByPrefix removes all values for a cache key prefix.
//
//nolint:wrapcheck // Thin wrapper passes through errors
func (sc *StringCache) DeleteByPrefix(ctx context.Context, prefix string) error {
	return sc.cache.DeleteByPrefix(ctx, prefix)
}

// Exists checks if a key exists in the cache.
//
//nolint:wrapcheck // Thin wrapper passes through errors
func (sc *StringCache) Exists(ctx context.Context, key string) (bool, error) {
	return sc.cache.Exists(ctx, key)
}

// Increment increments a value in the cache and returns the new value.
//
//nolint:wrapcheck // Thin wrapper passes through errors
func (sc *StringCache) Increment(ctx context.Context, key string) (int64, error) {
	return sc.cache.Increment(ctx, key)
}

// Decrement decrements a value in the cache and returns the new value.
//
//nolint:wrapcheck // Thin wrapper passes through errors
func (sc *StringCache) Decrement(ctx context.Context, key string) (int64, error) {
	return sc.cache.Decrement(ctx, key)
}

// Expire sets a new expiration time for a key.
//
//nolint:wrapcheck // Thin wrapper passes through errors
func (sc *StringCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return sc.cache.Expire(ctx, key, ttl)
}

// Close closes the underlying cache.
//
//nolint:wrapcheck // Thin wrapper passes through errors
func (sc *StringCache) Close() error {
	return sc.cache.Close()
}

// Ping checks the underlying cache.
func (sc *StringCache) Ping(ctx context.Context) error {
	if err := sc.cache.Ping(ctx); err != nil {
		return fmt.Errorf("cache ping: %w", err)
	}

	return nil
}
