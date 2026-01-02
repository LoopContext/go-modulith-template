// Package health provides health check endpoints for the server.
package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cmelgarejo/go-modulith-template/internal/registry"
	"github.com/cmelgarejo/go-modulith-template/internal/websocket"
)

const healthStatusHealthy = "healthy"

// SetupHealthChecks registers all health check endpoints.
func SetupHealthChecks(mux *http.ServeMux, db *sql.DB, wsHub *websocket.Hub, reg *registry.Registry) {
	SetupLivenessProbe(mux)
	SetupReadinessProbe(mux, db, wsHub, reg)
	SetupWebSocketHealthCheck(mux, wsHub)
}

// SetupLivenessProbe registers the liveness probe endpoint.
func SetupLivenessProbe(mux *http.ServeMux) {
	// Liveness probe - always returns 200 if process is alive
	mux.HandleFunc("/livez", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Legacy healthz endpoint (same as livez for backward compatibility)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
}

// SetupReadinessProbe registers the readiness probe endpoint.
func SetupReadinessProbe(mux *http.ServeMux, db *sql.DB, wsHub *websocket.Hub, reg *registry.Registry) {
	// Readiness probe - checks all dependencies
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		status := map[string]interface{}{
			"status": "ready",
			"checks": make(map[string]string),
		}

		checks := status["checks"].(map[string]string)
		allHealthy := CheckReadinessDependencies(r.Context(), checks, db, wsHub, reg)

		if !allHealthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		// Write JSON response
		jsonData, _ := json.Marshal(status)
		_, _ = w.Write(jsonData)
	})
}

// CheckReadinessDependencies checks all dependencies and updates the checks map.
func CheckReadinessDependencies(ctx context.Context, checks map[string]string, db *sql.DB, wsHub *websocket.Hub, reg *registry.Registry) bool {
	allHealthy := true

	// Check module health
	if err := reg.HealthCheckAll(ctx); err != nil {
		checks["modules"] = fmt.Sprintf("unhealthy: %v", err)
		allHealthy = false
	} else {
		checks["modules"] = healthStatusHealthy
	}

	// Check database connectivity
	if err := db.PingContext(ctx); err != nil {
		checks["database"] = fmt.Sprintf("unhealthy: %v", err)
		allHealthy = false
	} else {
		checks["database"] = healthStatusHealthy
	}

	// Check event bus (basic check - if it exists, it's healthy)
	if reg.EventBus() != nil {
		checks["event_bus"] = healthStatusHealthy
	} else {
		checks["event_bus"] = "unhealthy: not initialized"
		allHealthy = false
	}

	// Check WebSocket hub
	if wsHub != nil {
		checks["websocket"] = healthStatusHealthy
	} else {
		checks["websocket"] = "unhealthy: not initialized"
		allHealthy = false
	}

	return allHealthy
}

// SetupWebSocketHealthCheck registers the WebSocket health check endpoint.
func SetupWebSocketHealthCheck(mux *http.ServeMux, wsHub *websocket.Hub) {
	// WebSocket connections health check
	mux.HandleFunc("/healthz/ws", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := fmt.Sprintf(`{"status":"ok","connections":%d,"users":%d}`,
			wsHub.GetTotalConnections(),
			wsHub.GetConnectedUsers())

		_, _ = w.Write([]byte(response))
	})
}

