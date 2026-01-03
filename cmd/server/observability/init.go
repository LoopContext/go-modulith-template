// Package observability provides observability initialization utilities.
package observability

import (
	"context"
	"fmt"

	"github.com/cmelgarejo/go-modulith-template/internal/config"
)

// InitObservability initializes all observability components (metrics, tracing).
func InitObservability(ctx context.Context, cfg *config.AppConfig) (func(), error) {
	// Logger already initialized in main() before config loading
	metricsHandler, metricsShutdown, err := InitMetrics()
	if err != nil {
		return func() {}, fmt.Errorf("failed to init metrics: %w", err)
	}

	var tracerShutdown func()
	if cfg.OTLPEndpoint != "" {
		tracerShutdown = InitTracer(ctx, cfg.OTLPEndpoint, cfg.ServiceName)
	} else {
		tracerShutdown = func() {}
	}

	// Expose metrics handler via global var to the HTTP mux setup.
	SetMetricsHandler(metricsHandler)

	return func() {
		tracerShutdown()
		metricsShutdown()
	}, nil
}
