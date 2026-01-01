package telemetry

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func setupTestTracer(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()

	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)

	return exporter
}

func TestStartSpan(t *testing.T) {
	exporter := setupTestTracer(t)
	ctx := context.Background()

	_, span := StartSpan(ctx, "test-operation")
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	if spans[0].Name != "test-operation" {
		t.Errorf("expected span name 'test-operation', got '%s'", spans[0].Name)
	}
}

func TestStartSpanWithAttributes(t *testing.T) {
	exporter := setupTestTracer(t)
	ctx := context.Background()

	attrs := map[string]string{
		"module": "auth",
		"action": "login",
	}

	_, span := StartSpanWithAttributes(ctx, "test-operation", attrs)
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	// Check attributes
	foundModule := false
	foundAction := false

	for _, attr := range spans[0].Attributes {
		if attr.Key == "module" && attr.Value.AsString() == "auth" {
			foundModule = true
		}

		if attr.Key == "action" && attr.Value.AsString() == "login" {
			foundAction = true
		}
	}

	if !foundModule {
		t.Error("expected module attribute not found")
	}

	if !foundAction {
		t.Error("expected action attribute not found")
	}
}

func TestModuleSpan(t *testing.T) {
	exporter := setupTestTracer(t)
	ctx := context.Background()

	_, span := ModuleSpan(ctx, "auth", "CreateUser")
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	// Verify module and operation attributes
	foundModule := false
	foundOperation := false

	for _, attr := range spans[0].Attributes {
		if attr.Key == "module" && attr.Value.AsString() == "auth" {
			foundModule = true
		}

		if attr.Key == "operation" && attr.Value.AsString() == "CreateUser" {
			foundOperation = true
		}
	}

	if !foundModule || !foundOperation {
		t.Error("expected module and operation attributes not found")
	}
}

func TestRecordError(t *testing.T) {
	exporter := setupTestTracer(t)
	ctx := context.Background()

	ctx, span := StartSpan(ctx, "test-operation")
	testErr := errors.New("test error")
	RecordError(ctx, testErr)
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	if len(spans[0].Events) == 0 {
		t.Error("expected error event to be recorded")
	}
}

func TestSetAttribute(t *testing.T) {
	exporter := setupTestTracer(t)
	ctx := context.Background()

	ctx, span := StartSpan(ctx, "test-operation")
	SetAttribute(ctx, "user_id", "user_123")
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	found := false

	for _, attr := range spans[0].Attributes {
		if attr.Key == "user_id" && attr.Value.AsString() == "user_123" {
			found = true

			break
		}
	}

	if !found {
		t.Error("expected user_id attribute not found")
	}
}

func TestAddEvent(t *testing.T) {
	exporter := setupTestTracer(t)
	ctx := context.Background()

	ctx, span := StartSpan(ctx, "test-operation")
	AddEvent(ctx, "user.created", map[string]string{"user_id": "user_123"})
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	if len(spans[0].Events) == 0 {
		t.Fatal("expected event to be recorded")
	}

	if spans[0].Events[0].Name != "user.created" {
		t.Errorf("expected event name 'user.created', got '%s'", spans[0].Events[0].Name)
	}
}

func TestRepositorySpan(t *testing.T) {
	exporter := setupTestTracer(t)
	ctx := context.Background()

	_, span := RepositorySpan(ctx, "auth", "GetUser", "user")
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	// Verify layer attribute
	foundLayer := false

	for _, attr := range spans[0].Attributes {
		if attr.Key == "layer" && attr.Value.AsString() == "repository" {
			foundLayer = true

			break
		}
	}

	if !foundLayer {
		t.Error("expected layer=repository attribute not found")
	}
}

func TestServiceSpan(t *testing.T) {
	exporter := setupTestTracer(t)
	ctx := context.Background()

	_, span := ServiceSpan(ctx, "auth", "Login")
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	// Verify layer attribute
	foundLayer := false

	for _, attr := range spans[0].Attributes {
		if attr.Key == "layer" && attr.Value.AsString() == "service" {
			foundLayer = true

			break
		}
	}

	if !foundLayer {
		t.Error("expected layer=service attribute not found")
	}
}

