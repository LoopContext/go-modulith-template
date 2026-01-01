// Package telemetry provides observability helpers for tracing and metrics.
package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	// TracerName is the name of the tracer used by this application.
	TracerName = "modulith"
)

// StartSpan starts a new span with the given name and returns the new context and span.
// The span should be ended by the caller using defer span.End().
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := otel.Tracer(TracerName)
	return tracer.Start(ctx, name, opts...)
}

// StartSpanWithAttributes starts a new span with the given name and attributes.
func StartSpanWithAttributes(ctx context.Context, name string, attrs map[string]string) (context.Context, trace.Span) {
	attributes := make([]attribute.KeyValue, 0, len(attrs))
	for k, v := range attrs {
		attributes = append(attributes, attribute.String(k, v))
	}

	return StartSpan(ctx, name, trace.WithAttributes(attributes...))
}

// AddEvent adds an event to the current span.
func AddEvent(ctx context.Context, name string, attrs map[string]string) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	attributes := make([]trace.EventOption, 0, len(attrs))
	for k, v := range attrs {
		attributes = append(attributes, trace.WithAttributes(attribute.String(k, v)))
	}

	span.AddEvent(name, attributes...)
}

// SetAttributes sets attributes on the current span.
func SetAttributes(ctx context.Context, attrs map[string]string) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	for k, v := range attrs {
		span.SetAttributes(attribute.String(k, v))
	}
}

// SetAttribute sets a single attribute on the current span.
func SetAttribute(ctx context.Context, key, value string) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	span.SetAttributes(attribute.String(key, value))
}

// RecordError records an error on the current span.
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	span.RecordError(err)
}

// ModuleSpan starts a span with module-specific attributes.
func ModuleSpan(ctx context.Context, moduleName, operation string) (context.Context, trace.Span) {
	return StartSpanWithAttributes(ctx, operation, map[string]string{
		"module":    moduleName,
		"operation": operation,
	})
}

// RepositorySpan starts a span for repository operations.
func RepositorySpan(ctx context.Context, moduleName, operation, entity string) (context.Context, trace.Span) {
	return StartSpanWithAttributes(ctx, operation, map[string]string{
		"module":    moduleName,
		"layer":     "repository",
		"operation": operation,
		"entity":    entity,
	})
}

// ServiceSpan starts a span for service operations.
func ServiceSpan(ctx context.Context, moduleName, operation string) (context.Context, trace.Span) {
	return StartSpanWithAttributes(ctx, operation, map[string]string{
		"module":    moduleName,
		"layer":     "service",
		"operation": operation,
	})
}

