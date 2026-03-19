package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

// NewPrometheusHandler returns an http.Handler that serves Prometheus metrics.
func NewPrometheusHandler() (http.Handler, error) {
	exporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
	}

	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	otel.SetMeterProvider(provider)

	// Refresh the meter instance used by metrics.go
	meterOnce = sync.Once{}

	getMeter()

	return promhttp.Handler(), nil
}

// InitTelemetry initializes both tracing and metrics.
func InitTelemetry(_ context.Context) (func(context.Context) error, error) {
	// Initialize business metrics
	if err := InitBusinessMetrics(); err != nil {
		return nil, err
	}

	// Initialize gRPC metrics
	if err := InitGRPCMetrics(); err != nil {
		return nil, err
	}

	// This is a placeholder for tracing initialization if needed
	return func(_ context.Context) error {
		return nil
	}, nil
}
