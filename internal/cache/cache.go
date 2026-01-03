// Package cache provides a caching abstraction for the application.
// It supports multiple backends (memory, Redis) and can be used for
// session storage, rate limiting data, or general-purpose caching.
package cache

import (
	"context"
	"errors"
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

	// Exists checks if a key exists in the cache.
	Exists(ctx context.Context, key string) (bool, error)

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

// Exists checks if a key exists in the cache.
//
//nolint:wrapcheck // Thin wrapper passes through errors
func (sc *StringCache) Exists(ctx context.Context, key string) (bool, error) {
	return sc.cache.Exists(ctx, key)
}

// Close closes the underlying cache.
//
//nolint:wrapcheck // Thin wrapper passes through errors
func (sc *StringCache) Close() error {
	return sc.cache.Close()
}
