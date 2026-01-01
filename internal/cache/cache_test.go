package cache_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cmelgarejo/go-modulith-template/internal/cache"
)

func TestMemoryCache_SetGet(t *testing.T) {
	mc := cache.NewMemoryCache()

	defer func() {
		if err := mc.Close(); err != nil {
			t.Errorf("failed to close cache: %v", err)
		}
	}()

	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")

	// Set value
	if err := mc.Set(ctx, key, value, 0); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get value
	got, err := mc.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(got) != string(value) {
		t.Errorf("expected %q, got %q", value, got)
	}
}

func TestMemoryCache_GetNotFound(t *testing.T) {
	mc := cache.NewMemoryCache()

	defer func() {
		if err := mc.Close(); err != nil {
			t.Errorf("failed to close cache: %v", err)
		}
	}()

	ctx := context.Background()

	_, err := mc.Get(ctx, "non-existent")
	if !errors.Is(err, cache.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryCache_Expiration(t *testing.T) {
	mc := cache.NewMemoryCache()

	defer func() {
		if err := mc.Close(); err != nil {
			t.Errorf("failed to close cache: %v", err)
		}
	}()

	ctx := context.Background()
	key := "expiring-key"
	value := []byte("expiring-value")

	// Set with very short TTL
	if err := mc.Set(ctx, key, value, 50*time.Millisecond); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Should exist initially
	got, err := mc.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(got) != string(value) {
		t.Errorf("expected %q, got %q", value, got)
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired now
	_, err = mc.Get(ctx, key)
	if !errors.Is(err, cache.ErrNotFound) {
		t.Errorf("expected ErrNotFound after expiration, got %v", err)
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	mc := cache.NewMemoryCache()

	defer func() {
		if err := mc.Close(); err != nil {
			t.Errorf("failed to close cache: %v", err)
		}
	}()

	ctx := context.Background()
	key := "delete-key"
	value := []byte("delete-value")

	// Set value
	if err := mc.Set(ctx, key, value, 0); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Delete value
	if err := mc.Delete(ctx, key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Should not exist
	_, err := mc.Get(ctx, key)
	if !errors.Is(err, cache.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMemoryCache_Exists(t *testing.T) {
	mc := cache.NewMemoryCache()

	defer func() {
		if err := mc.Close(); err != nil {
			t.Errorf("failed to close cache: %v", err)
		}
	}()

	ctx := context.Background()
	key := "exists-key"
	value := []byte("exists-value")

	// Should not exist initially
	exists, err := mc.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	if exists {
		t.Error("expected key to not exist")
	}

	// Set value
	if err := mc.Set(ctx, key, value, 0); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Should exist now
	exists, err = mc.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	if !exists {
		t.Error("expected key to exist")
	}
}

func TestStringCache(t *testing.T) {
	mc := cache.NewMemoryCache()

	defer func() {
		if err := mc.Close(); err != nil {
			t.Errorf("failed to close cache: %v", err)
		}
	}()

	sc := cache.NewStringCache(mc)
	ctx := context.Background()
	key := "string-key"
	value := "string-value"

	// Set string value
	if err := sc.Set(ctx, key, value, 0); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get string value
	got, err := sc.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got != value {
		t.Errorf("expected %q, got %q", value, got)
	}
}

func TestMemoryCache_ValueIsolation(t *testing.T) {
	mc := cache.NewMemoryCache()

	defer func() {
		if err := mc.Close(); err != nil {
			t.Errorf("failed to close cache: %v", err)
		}
	}()

	ctx := context.Background()
	key := "isolation-key"
	original := []byte("original-value")

	// Set value
	if err := mc.Set(ctx, key, original, 0); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get value and modify it
	got, err := mc.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	got[0] = 'X' // Modify the returned slice

	// Get again - should be unchanged
	got2, err := mc.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(got2) != string(original) {
		t.Errorf("cache value was modified: expected %q, got %q", original, got2)
	}
}

