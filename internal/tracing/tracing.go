// Package tracing re-exports atp/tracing span helpers and provides
// arabica-specific initialization (zerolog bridge).
package tracing

import (
	"context"

	"github.com/go-logr/zerologr"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	atptracing "tangled.org/pdewey.com/atp/tracing"
)

// Init creates and registers a tracer provider with an OTLP HTTP exporter.
// Bridges OTel's internal logger to zerolog before delegating to atp/tracing.
// serviceName is the OTel service name (typically the app name, e.g.
// "arabica") so traces are attributed to the running app rather than
// hardcoded.
func Init(ctx context.Context, serviceName string) (*sdktrace.TracerProvider, error) {
	otel.SetLogger(zerologr.New(&log.Logger))
	if serviceName == "" {
		serviceName = "arabica"
	}
	return atptracing.Init(ctx, serviceName)
}

// SqliteSpan starts a span for a SQLite operation.
func SqliteSpan(ctx context.Context, op, table string) (context.Context, trace.Span) {
	return atptracing.SqliteSpan(ctx, op, table)
}

// HandlerSpan starts a span for a logical operation within a handler.
func HandlerSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return atptracing.HandlerSpan(ctx, name, attrs...)
}

// EndWithError records an error on a span and sets its status.
func EndWithError(span trace.Span, err error) {
	atptracing.EndWithError(span, err)
}
