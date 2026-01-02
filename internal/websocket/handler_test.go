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
	handler := NewHandler(HandlerConfig{
		Hub:            hub,
		Verifier:       nil,
		AllowedOrigins: []string{"*"},
		Env:             "dev",
	})

	if handler == nil {
		t.Fatal("Expected handler to be created")
	}

	if handler.hub != hub {
		t.Error("Expected handler to have the same hub reference")
	}
}

func TestHandler_AuthenticateRequest_DevMode(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)
	handler := NewHandler(HandlerConfig{
		Hub:            hub,
		Verifier:       nil,
		AllowedOrigins: []string{"*"},
		Env:             "dev",
	})

	tests := getDevModeAuthTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/ws"
			if tt.query != "" {
				url += "?" + tt.query
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			userID, err := handler.authenticateRequest(req)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if userID != tt.expectedUserID {
				t.Errorf("Expected user_id %q, got %q", tt.expectedUserID, userID)
			}
		})
	}
}

type devModeAuthTestCase struct {
	name          string
	query         string
	expectedUserID string
	expectError   bool
}

func getDevModeAuthTestCases() []devModeAuthTestCase {
	return []devModeAuthTestCase{
		{
			name:          "with user_id parameter in dev",
			query:         "user_id=test-user-123",
			expectedUserID: "test-user-123",
			expectError:   false,
		},
		{
			name:          "without user_id parameter in dev",
			query:         "",
			expectedUserID: "",
			expectError:   false, // Dev mode allows anonymous
		},
		{
			name:          "empty user_id",
			query:         "user_id=",
			expectedUserID: "",
			expectError:   false,
		},
	}
}

func TestHandler_ServeHTTP_InvalidRequest(t *testing.T) {
	ctx := context.Background()
	hub := NewHub(ctx)
	handler := NewHandler(HandlerConfig{
		Hub:            hub,
		Verifier:       nil,
		AllowedOrigins: []string{"*"},
		Env:             "dev",
	})

	// Create a regular HTTP request (not WebSocket upgrade)
	req := httptest.NewRequest(http.MethodGet, "/ws?user_id=test", nil)
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
	handler := NewHandler(HandlerConfig{
		Hub:            hub,
		Verifier:       nil,
		AllowedOrigins: []string{"*"},
		Env:             "dev",
	})

	// POST requests should also fail
	req := httptest.NewRequest(http.MethodPost, "/ws?user_id=test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Error("Expected error for POST request to WebSocket endpoint")
	}
}

func TestCreateOriginChecker(t *testing.T) {
	testCases := make([]originTestCase, 0, 8)
	testCases = append(testCases, getDevModeTestCases())
	testCases = append(testCases, getProdModeTestCases())
	testCases = append(testCases, getWildcardTestCases()...)
	testCases = append(testCases, getMatchingTestCases()...)
	testCases = append(testCases, getEmptyOriginTestCases()...)

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			checker := createOriginChecker(tt.allowedOrigins, tt.env)

			req := httptest.NewRequest(http.MethodGet, "/ws", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			result := checker(req)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for origin %q", tt.expected, result, tt.origin)
			}
		})
	}
}

type originTestCase struct {
	name           string
	allowedOrigins []string
	env            string
	origin         string
	expected       bool
}

func getDevModeTestCases() originTestCase {
	return originTestCase{
		name:           "dev mode allows all when no origins configured",
		allowedOrigins: []string{},
		env:            "dev",
		origin:         "https://evil.com",
		expected:       true,
	}
}

func getProdModeTestCases() originTestCase {
	return originTestCase{
		name:           "prod mode denies all when no origins configured",
		allowedOrigins: []string{},
		env:            "prod",
		origin:         "https://example.com",
		expected:       false,
	}
}

func getWildcardTestCases() []originTestCase {
	return []originTestCase{
		{
			name:           "wildcard allows all",
			allowedOrigins: []string{"*"},
			env:            "prod",
			origin:         "https://any-origin.com",
			expected:       true,
		},
	}
}

func getMatchingTestCases() []originTestCase {
	return []originTestCase{
		{
			name:           "exact match allows",
			allowedOrigins: []string{"https://example.com"},
			env:            "prod",
			origin:         "https://example.com",
			expected:       true,
		},
		{
			name:           "case insensitive match",
			allowedOrigins: []string{"https://Example.com"},
			env:            "prod",
			origin:         "https://example.com",
			expected:       true,
		},
		{
			name:           "no match denies",
			allowedOrigins: []string{"https://example.com"},
			env:            "prod",
			origin:         "https://evil.com",
			expected:       false,
		},
	}
}

func getEmptyOriginTestCases() []originTestCase {
	return []originTestCase{
		{
			name:           "empty origin denied when no wildcard",
			allowedOrigins: []string{"https://example.com"},
			env:            "prod",
			origin:         "",
			expected:       false,
		},
		{
			name:           "empty origin allowed with wildcard",
			allowedOrigins: []string{"*"},
			env:            "prod",
			origin:         "",
			expected:       true,
		},
	}
}

