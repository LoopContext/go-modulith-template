// Package telemetry provides observability helpers for tracing and metrics.
package telemetry

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	// MeterName is the name of the meter used by this application.
	MeterName = "modulith"
)

var (
	meter     metric.Meter
	meterOnce sync.Once
)

// getMeter returns the global meter instance.
func getMeter() metric.Meter {
	meterOnce.Do(func() {
		meter = otel.Meter(MeterName)
	})

	return meter
}

// Counter represents a monotonically increasing counter metric.
type Counter struct {
	counter metric.Int64Counter
	attrs   []metric.AddOption
}

// NewCounter creates a new counter metric with the given name and description.
// Example: telemetry.NewCounter("users_created_total", "Total number of users created")
//
//nolint:wrapcheck // OpenTelemetry errors are passed through unchanged
func NewCounter(name, description string) (*Counter, error) {
	counter, err := getMeter().Int64Counter(
		name,
		metric.WithDescription(description),
	)
	if err != nil {
		return nil, err
	}

	return &Counter{counter: counter}, nil
}

// Inc increments the counter by 1.
func (c *Counter) Inc(ctx context.Context) {
	c.counter.Add(ctx, 1, c.attrs...)
}

// Add increments the counter by the given value.
func (c *Counter) Add(ctx context.Context, value int64) {
	c.counter.Add(ctx, value, c.attrs...)
}

// WithAttributes returns a new Counter with additional attributes.
func (c *Counter) WithAttributes(attrs ...metric.AddOption) *Counter {
	return &Counter{
		counter: c.counter,
		attrs:   append(c.attrs, attrs...),
	}
}

// Gauge represents a metric that can go up or down.
type Gauge struct {
	gauge metric.Int64UpDownCounter
	attrs []metric.AddOption
}

// NewGauge creates a new gauge metric with the given name and description.
// Example: telemetry.NewGauge("active_connections", "Number of active connections")
//
//nolint:wrapcheck // OpenTelemetry errors are passed through unchanged
func NewGauge(name, description string) (*Gauge, error) {
	gauge, err := getMeter().Int64UpDownCounter(
		name,
		metric.WithDescription(description),
	)
	if err != nil {
		return nil, err
	}

	return &Gauge{gauge: gauge}, nil
}

// Set adds the delta to the gauge (use negative for decrease).
func (g *Gauge) Set(ctx context.Context, delta int64) {
	g.gauge.Add(ctx, delta, g.attrs...)
}

// Inc increments the gauge by 1.
func (g *Gauge) Inc(ctx context.Context) {
	g.gauge.Add(ctx, 1, g.attrs...)
}

// Dec decrements the gauge by 1.
func (g *Gauge) Dec(ctx context.Context) {
	g.gauge.Add(ctx, -1, g.attrs...)
}

// WithAttributes returns a new Gauge with additional attributes.
func (g *Gauge) WithAttributes(attrs ...metric.AddOption) *Gauge {
	return &Gauge{
		gauge: g.gauge,
		attrs: append(g.attrs, attrs...),
	}
}

// Histogram represents a distribution of values.
type Histogram struct {
	histogram metric.Float64Histogram
	attrs     []metric.RecordOption
}

// NewHistogram creates a new histogram metric with the given name and description.
// Example: telemetry.NewHistogram("request_duration_seconds", "Request duration in seconds")
//
//nolint:wrapcheck // OpenTelemetry errors are passed through unchanged
func NewHistogram(name, description string, unit string) (*Histogram, error) {
	histogram, err := getMeter().Float64Histogram(
		name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		return nil, err
	}

	return &Histogram{histogram: histogram}, nil
}

// Record records a value in the histogram.
func (h *Histogram) Record(ctx context.Context, value float64) {
	h.histogram.Record(ctx, value, h.attrs...)
}

// RecordDuration records a duration in seconds.
func (h *Histogram) RecordDuration(ctx context.Context, d time.Duration) {
	h.histogram.Record(ctx, d.Seconds(), h.attrs...)
}

// WithAttributes returns a new Histogram with additional attributes.
func (h *Histogram) WithAttributes(attrs ...metric.RecordOption) *Histogram {
	return &Histogram{
		histogram: h.histogram,
		attrs:     append(h.attrs, attrs...),
	}
}

// Timer is a helper for measuring operation durations.
type Timer struct {
	start     time.Time
	histogram *Histogram
	ctx       context.Context
}

// NewTimer starts a new timer for the given histogram.
func NewTimer(ctx context.Context, h *Histogram) *Timer {
	return &Timer{
		start:     time.Now(),
		histogram: h,
		ctx:       ctx,
	}
}

// Stop stops the timer and records the duration.
func (t *Timer) Stop() time.Duration {
	d := time.Since(t.start)
	t.histogram.RecordDuration(t.ctx, d)

	return d
}

// Attr creates an attribute for use with counter/gauge metrics.
func Attr(key, value string) metric.AddOption {
	return metric.WithAttributes(attribute.String(key, value))
}

// RecordAttr creates a record attribute for use with histograms.
func RecordAttr(key, value string) metric.RecordOption {
	return metric.WithAttributes(attribute.String(key, value))
}
