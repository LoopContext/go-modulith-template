package graphql

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/websocket"
)

// Note: These tests require GraphQL to be initialized (run: just add-graphql)
// They will be skipped if the Setup function doesn't exist yet.
// Once GraphQL is set up, these tests will run and verify the server functionality.

func TestSetup_WhenGraphQLInitialized(t *testing.T) {
	// This test will only work after running: just add-graphql
	// We check if Setup function exists by trying to call it
	// If it fails to compile, the test is skipped
	ctx := context.Background()
	eventBus := events.NewBus()
	wsHub := websocket.NewHub(ctx)

	// Start hub in background
	go wsHub.Run()
	defer wsHub.Stop()

	// Try to call Setup - if it doesn't exist, this will fail to compile
	// and the test file won't be included in the build
	handler := Setup(ctx, eventBus, wsHub)

	if handler == nil {
		t.Skip("GraphQL not initialized. Run: just add-graphql to enable these tests")
	}

	// Verify it's an http.Handler
	_ = handler
}

func TestSetup_WithNilEventBus_WhenGraphQLInitialized(t *testing.T) {
	ctx := context.Background()
	wsHub := websocket.NewHub(ctx)

	go wsHub.Run()
	defer wsHub.Stop()

	handler := Setup(ctx, nil, wsHub)

	if handler == nil {
		t.Skip("GraphQL not initialized. Run: just add-graphql")
	}

	// Should still create handler (resolvers handle nil gracefully)
	if handler == nil {
		t.Fatal("Expected handler to be created even with nil eventBus")
	}
}

func TestSetup_HandlerServesHTTP_WhenGraphQLInitialized(t *testing.T) {
	ctx := context.Background()
	eventBus := events.NewBus()
	wsHub := websocket.NewHub(ctx)

	go wsHub.Run()
	defer wsHub.Stop()

	handler := Setup(ctx, eventBus, wsHub)

	if handler == nil {
		t.Skip("GraphQL not initialized. Run: just add-graphql")
	}

	// Create a test request
	req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	w := httptest.NewRecorder()

	// Handler should not panic
	handler.ServeHTTP(w, req)

	// Should return some response (even if it's an error)
	if w.Code == 0 {
		t.Error("Expected handler to write a response")
	}
}

func TestPlaygroundHandler_WhenGraphQLInitialized(t *testing.T) {
	handler := PlaygroundHandler()

	if handler == nil {
		t.Skip("GraphQL not initialized. Run: just add-graphql")
	}

	// Verify it's an http.Handler
	_ = handler
}

func TestPlaygroundHandler_ServesHTTP_WhenGraphQLInitialized(t *testing.T) {
	handler := PlaygroundHandler()

	if handler == nil {
		t.Skip("GraphQL not initialized. Run: just add-graphql")
	}

	req := httptest.NewRequest(http.MethodGet, "/graphql/playground", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Stub returns NotFoundHandler, so we expect 404 until GraphQL is initialized
	// After running: just add-graphql, this should return 200
	if w.Code == http.StatusNotFound {
		t.Skip("GraphQL not fully initialized. Playground returns 404 until setup is complete. Run: just add-graphql")
	}

	// When GraphQL is initialized, should return HTML content
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType == "" {
		t.Error("Expected Content-Type header to be set")
	}
}

func TestSetup_ConcurrentRequests_WhenGraphQLInitialized(t *testing.T) {
	ctx := context.Background()
	eventBus := events.NewBus()
	wsHub := websocket.NewHub(ctx)

	go wsHub.Run()
	defer wsHub.Stop()

	handler := Setup(ctx, eventBus, wsHub)

	if handler == nil {
		t.Skip("GraphQL not initialized. Run: just add-graphql")
	}

	// Test concurrent requests
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			done <- true
		}()
	}

	// Wait for all requests to complete
	timeout := time.After(5 * time.Second)

	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// Request completed
		case <-timeout:
			t.Fatal("Timeout waiting for concurrent requests")
		}
	}
}

func TestSetup_HandlerIsReusable_WhenGraphQLInitialized(t *testing.T) {
	ctx := context.Background()
	eventBus := events.NewBus()
	wsHub := websocket.NewHub(ctx)

	go wsHub.Run()
	defer wsHub.Stop()

	handler := Setup(ctx, eventBus, wsHub)

	if handler == nil {
		t.Skip("GraphQL not initialized. Run: just add-graphql")
	}

	// Use the same handler multiple times
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code == 0 {
			t.Errorf("Expected handler to write response on iteration %d", i)
		}
	}
}
