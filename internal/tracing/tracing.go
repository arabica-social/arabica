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
func Init(ctx context.Context) (*sdktrace.TracerProvider, error) {
	otel.SetLogger(zerologr.New(&log.Logger))
	return atptracing.Init(ctx, "arabica")
}

// BoltSpan starts a span for a BoltDB operation.
func BoltSpan(ctx context.Context, op, bucket string) (context.Context, trace.Span) {
	return atptracing.BoltSpan(ctx, op, bucket)
}

// SqliteSpan starts a span for a SQLite operation.
func SqliteSpan(ctx context.Context, op, table string) (context.Context, trace.Span) {
	return atptracing.SqliteSpan(ctx, op, table)
}

// PdsSpan starts a span for a PDS XRPC operation.
func PdsSpan(ctx context.Context, method, collection, did string) (context.Context, trace.Span) {
	return atptracing.PdsSpan(ctx, method, collection, did)
}

// HandlerSpan starts a span for a logical operation within a handler.
func HandlerSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return atptracing.HandlerSpan(ctx, name, attrs...)
}

// EndWithError records an error on a span and sets its status.
func EndWithError(span trace.Span, err error) {
	atptracing.EndWithError(span, err)
}
