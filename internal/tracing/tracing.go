package tracing

import (
	"context"
	"os"

	"github.com/go-logr/zerologr"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// tracer returns the package tracer. This must be a function (not a package-level var)
// because the global TracerProvider isn't set until Init() runs.
func tracer() trace.Tracer {
	return otel.Tracer("arabica")
}

// Init creates and registers a tracer provider with an OTLP HTTP exporter.
// It reads OTEL_EXPORTER_OTLP_ENDPOINT (default: localhost:4318).
// Returns the provider so the caller can defer Shutdown.
func Init(ctx context.Context) (*sdktrace.TracerProvider, error) {
	// Bridge OTel's internal logger to zerolog
	otel.SetLogger(zerologr.New(&log.Logger))

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4318"
	}

	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("arabica"),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp, nil
}

// PdsSpan starts a span for a PDS operation with standard attributes.
func PdsSpan(ctx context.Context, method, collection, did string) (context.Context, trace.Span) {
	return tracer().Start(ctx, "pds."+method,
		trace.WithAttributes(
			attribute.String("pds.method", method),
			attribute.String("pds.collection", collection),
			attribute.String("pds.did", did),
		),
	)
}

// EndWithError records an error on a span and sets its status.
// If err is nil, this is a no-op.
func EndWithError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}
