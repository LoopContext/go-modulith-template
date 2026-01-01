package websocket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewHandler(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)
	handler := NewHandler(hub)

	if handler == nil {
		t.Fatal("Expected handler to be created")
	}

	if handler.hub != hub {
		t.Error("Expected handler to have the same hub reference")
	}
}

func TestGetUserIDFromContext(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "with user_id parameter",
			query:    "user_id=test-user-123",
			expected: "test-user-123",
		},
		{
			name:     "without user_id parameter",
			query:    "",
			expected: "",
		},
		{
			name:     "empty user_id",
			query:    "user_id=",
			expected: "",
		},
		{
			name:     "other parameters",
			query:    "token=abc&session=xyz",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/ws"
			if tt.query != "" {
				url += "?" + tt.query
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			result := getUserIDFromContext(req)

			if result != tt.expected {
				t.Errorf("Expected user_id %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestHandler_ServeHTTP_InvalidRequest(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)
	handler := NewHandler(hub)

	// Create a regular HTTP request (not WebSocket upgrade)
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should return an error (not a valid WebSocket upgrade)
	if w.Code == http.StatusOK {
		t.Error("Expected error for non-WebSocket request")
	}
}

func TestHandler_ServeHTTP_PostRequest(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)
	handler := NewHandler(hub)

	// POST requests should also fail
	req := httptest.NewRequest(http.MethodPost, "/ws", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Error("Expected error for POST request to WebSocket endpoint")
	}
}

