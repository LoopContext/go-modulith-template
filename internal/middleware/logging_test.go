package middleware_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cmelgarejo/go-modulith-template/internal/middleware"
)

func TestLogging_BasicRequest(t *testing.T) {
	// Capture logs
	var buf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&buf, nil))
	slog.SetDefault(logger)

	handler := middleware.LoggingWithDefaults()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "HTTP request") {
		t.Error("expected log to contain 'HTTP request'")
	}

	if !strings.Contains(logOutput, "method=GET") {
		t.Error("expected log to contain method")
	}

	if !strings.Contains(logOutput, "path=/api/users") {
		t.Error("expected log to contain path")
	}
}

func TestLogging_SkipsHealthCheck(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&buf, nil))
	slog.SetDefault(logger)

	handler := middleware.LoggingWithDefaults()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if buf.Len() > 0 {
		t.Error("expected no logs for /healthz path")
	}
}

func TestLogging_ErrorStatus(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&buf, nil))
	slog.SetDefault(logger)

	handler := middleware.LoggingWithDefaults()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/error", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "level=ERROR") {
		t.Error("expected ERROR level for 500 status")
	}
}

func TestLogging_SlowRequest(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&buf, nil))
	slog.SetDefault(logger)

	config := middleware.LoggingConfig{
		SkipPaths:            []string{},
		SlowRequestThreshold: 10 * time.Millisecond,
	}

	handler := middleware.Logging(config)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/slow", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "slow=true") {
		t.Error("expected slow=true for slow request")
	}
}

func TestLogging_WithRequestID(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&buf, nil))
	slog.SetDefault(logger)

	// Chain RequestID middleware with Logging middleware
	handler := middleware.RequestID(
		middleware.LoggingWithDefaults()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "request_id=") {
		t.Error("expected log to contain request_id")
	}
}

