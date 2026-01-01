package telemetry_test

import (
	"context"
	"testing"
	"time"

	"github.com/cmelgarejo/go-modulith-template/internal/telemetry"
)

func TestNewCounter(t *testing.T) {
	counter, err := telemetry.NewCounter("test_counter", "A test counter")
	if err != nil {
		t.Fatalf("failed to create counter: %v", err)
	}

	if counter == nil {
		t.Fatal("expected counter to be non-nil")
	}

	// Should not panic
	counter.Inc(context.Background())
	counter.Add(context.Background(), 5)
}

func TestNewGauge(t *testing.T) {
	gauge, err := telemetry.NewGauge("test_gauge", "A test gauge")
	if err != nil {
		t.Fatalf("failed to create gauge: %v", err)
	}

	if gauge == nil {
		t.Fatal("expected gauge to be non-nil")
	}

	// Should not panic
	gauge.Inc(context.Background())
	gauge.Dec(context.Background())
	gauge.Set(context.Background(), 10)
}

func TestNewHistogram(t *testing.T) {
	histogram, err := telemetry.NewHistogram("test_duration", "A test histogram", "s")
	if err != nil {
		t.Fatalf("failed to create histogram: %v", err)
	}

	if histogram == nil {
		t.Fatal("expected histogram to be non-nil")
	}

	// Should not panic
	histogram.Record(context.Background(), 0.5)
	histogram.RecordDuration(context.Background(), 100*time.Millisecond)
}

func TestTimer(t *testing.T) {
	histogram, err := telemetry.NewHistogram("timer_test", "Timer test", "s")
	if err != nil {
		t.Fatalf("failed to create histogram: %v", err)
	}

	timer := telemetry.NewTimer(context.Background(), histogram)

	// Simulate some work
	time.Sleep(10 * time.Millisecond)

	duration := timer.Stop()
	if duration < 10*time.Millisecond {
		t.Errorf("expected duration >= 10ms, got %v", duration)
	}
}

func TestCounterWithAttributes(t *testing.T) {
	counter, err := telemetry.NewCounter("test_counter_attrs", "Counter with attrs")
	if err != nil {
		t.Fatalf("failed to create counter: %v", err)
	}

	// Create counter with attributes
	counterWithAttrs := counter.WithAttributes(telemetry.Attr("module", "auth"))

	// Should not panic
	counterWithAttrs.Inc(context.Background())
}

