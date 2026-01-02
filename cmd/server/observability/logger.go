// Package observability provides observability initialization utilities.
package observability

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	oteltrace "go.opentelemetry.io/otel/trace"
)

// InitLoggerEarly initializes a basic logger with debug enabled before config is loaded.
func InitLoggerEarly() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug, // Enable debug logs
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// InitLogger initializes the logger with the given environment and log level.
func InitLogger(env string, logLevel string) {
	var handler slog.Handler

	// Parse log level
	var level slog.Level

	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}
	if env == "prod" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(traceContextHandler{next: handler})
	slog.SetDefault(logger)
}

type traceContextHandler struct {
	next slog.Handler
}

func (h traceContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

//nolint:gocritic // slog.Record is a standard library type, cannot change signature
func (h traceContextHandler) Handle(ctx context.Context, r slog.Record) error {
	span := oteltrace.SpanFromContext(ctx)

	sc := span.SpanContext()
	if sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}

	if err := h.next.Handle(ctx, r); err != nil {
		return fmt.Errorf("failed to handle log record: %w", err)
	}

	return nil
}

func (h traceContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return traceContextHandler{next: h.next.WithAttrs(attrs)}
}

func (h traceContextHandler) WithGroup(name string) slog.Handler {
	return traceContextHandler{next: h.next.WithGroup(name)}
}

