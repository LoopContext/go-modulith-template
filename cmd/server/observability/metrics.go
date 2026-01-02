// Package observability provides observability initialization utilities.
package observability

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

var metricsHandler http.Handler

// SetMetricsHandler sets the metrics HTTP handler.
func SetMetricsHandler(h http.Handler) {
	metricsHandler = h
}

// GetMetricsHandler returns the metrics HTTP handler.
func GetMetricsHandler() http.Handler {
	return metricsHandler
}

// InitMetrics initializes Prometheus metrics.
func InitMetrics() (http.Handler, func(), error) {
	reg := prometheus.NewRegistry()

	exporter, err := otelprom.New(otelprom.WithRegisterer(reg))
	if err != nil {
		return nil, func() {}, fmt.Errorf("failed to create prometheus exporter: %w", err)
	}

	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	otel.SetMeterProvider(provider)

	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{}), func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			slog.Error("failed to shutdown meter provider", "error", err)
		}
	}, nil
}

