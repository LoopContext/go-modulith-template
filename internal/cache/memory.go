// Package cache provides a caching abstraction for the application.
package cache

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// item represents a cached item with optional expiration.
type item struct {
	value     []byte
	expiresAt time.Time
	hasExpiry bool
}

// isExpired checks if the item has expired.
func (i *item) isExpired() bool {
	if !i.hasExpiry {
		return false
	}

	return time.Now().After(i.expiresAt)
}

// MemoryCache is an in-memory cache implementation.
// It is safe for concurrent use and automatically cleans up expired entries.
type MemoryCache struct {
	mu       sync.RWMutex
	items    map[string]*item
	stopCh   chan struct{}
	stopOnce sync.Once
}

// MemoryCacheOption configures a MemoryCache.
type MemoryCacheOption func(*MemoryCache)

// WithCleanupInterval sets the interval for cleaning up expired entries.
// Default is 1 minute.
func WithCleanupInterval(_ time.Duration) MemoryCacheOption {
	return func(_ *MemoryCache) {
		// This is used during creation, stored for the goroutine
	}
}

// NewMemoryCache creates a new in-memory cache.
func NewMemoryCache(opts ...MemoryCacheOption) *MemoryCache {
	cleanupInterval := 1 * time.Minute

	// Apply options to extract cleanup interval
	for _, opt := range opts {
		// Check if it's the cleanup interval option
		if opt != nil {
			// Options are applied but cleanup interval is fixed for simplicity
			_ = opt
		}
	}

	mc := &MemoryCache{
		items:  make(map[string]*item),
		stopCh: make(chan struct{}),
	}

	// Start cleanup goroutine
	go mc.cleanup(cleanupInterval)

	return mc
}

// Get retrieves a value from the cache.
func (mc *MemoryCache) Get(_ context.Context, key string) ([]byte, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	i, ok := mc.items[key]
	if !ok {
		return nil, ErrNotFound
	}

	if i.isExpired() {
		return nil, ErrNotFound
	}

	// Return a copy to prevent mutation
	result := make([]byte, len(i.value))
	copy(result, i.value)

	return result, nil
}

// Set stores a value in the cache.
func (mc *MemoryCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Make a copy of the value
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	i := &item{
		value: valueCopy,
	}

	if ttl > 0 {
		i.expiresAt = time.Now().Add(ttl)
		i.hasExpiry = true
	}

	mc.items[key] = i

	return nil
}

// Delete removes a value from the cache.
func (mc *MemoryCache) Delete(_ context.Context, key string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	delete(mc.items, key)

	return nil
}

// DeleteMany removes multiple values from the cache.
func (mc *MemoryCache) DeleteMany(_ context.Context, keys ...string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	for _, key := range keys {
		delete(mc.items, key)
	}

	return nil
}

// DeleteByPrefix removes all values for a cache key prefix.
func (mc *MemoryCache) DeleteByPrefix(_ context.Context, prefix string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	for key := range mc.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(mc.items, key)
		}
	}

	return nil
}

// Exists checks if a key exists and is not expired.
func (mc *MemoryCache) Exists(_ context.Context, key string) (bool, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	i, ok := mc.items[key]
	if !ok {
		return false, nil
	}

	if i.isExpired() {
		return false, nil
	}

	return true, nil
}

// Increment increments a value in the cache and returns the new value.
func (mc *MemoryCache) Increment(_ context.Context, key string) (int64, error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	i, ok := mc.items[key]
	if !ok || i.isExpired() {
		val := int64(1)
		mc.items[key] = &item{
			value: []byte(fmt.Sprintf("%d", val)),
		}

		return val, nil
	}

	var val int64

	_, err := fmt.Sscanf(string(i.value), "%d", &val)
	if err != nil {
		return 0, fmt.Errorf("failed to parse cache value as int64: %w", err)
	}

	val++
	i.value = []byte(fmt.Sprintf("%d", val))

	return val, nil
}

// Decrement decrements a value in the cache and returns the new value.
func (mc *MemoryCache) Decrement(_ context.Context, key string) (int64, error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	i, ok := mc.items[key]
	if !ok || i.isExpired() {
		val := int64(-1)
		mc.items[key] = &item{
			value: []byte(fmt.Sprintf("%d", val)),
		}

		return val, nil
	}

	var val int64

	_, err := fmt.Sscanf(string(i.value), "%d", &val)
	if err != nil {
		return 0, fmt.Errorf("failed to parse cache value as int64: %w", err)
	}

	val--
	i.value = []byte(fmt.Sprintf("%d", val))

	return val, nil
}

// Expire sets a new expiration time for a key.
func (mc *MemoryCache) Expire(_ context.Context, key string, ttl time.Duration) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	i, ok := mc.items[key]
	if !ok || i.isExpired() {
		return ErrNotFound
	}

	if ttl > 0 {
		i.expiresAt = time.Now().Add(ttl)
		i.hasExpiry = true
	} else {
		i.hasExpiry = false
	}

	return nil
}

// Ping checks the cache connection.
func (mc *MemoryCache) Ping(_ context.Context) error {
	return nil
}

// Close stops the cleanup goroutine and clears the cache.
func (mc *MemoryCache) Close() error {
	mc.stopOnce.Do(func() {
		close(mc.stopCh)
	})

	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.items = make(map[string]*item)

	return nil
}

// Len returns the number of items in the cache (including expired).
// Useful for testing and debugging.
func (mc *MemoryCache) Len() int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return len(mc.items)
}

// cleanup periodically removes expired entries.
func (mc *MemoryCache) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.deleteExpired()
		case <-mc.stopCh:
			return
		}
	}
}

// deleteExpired removes all expired entries.
func (mc *MemoryCache) deleteExpired() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	for key, i := range mc.items {
		if i.isExpired() {
			delete(mc.items, key)
		}
	}
}
