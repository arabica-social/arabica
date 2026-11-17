package ai

import (
	"context"
	"io"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func startCompletionSpan(ctx context.Context, provider string, config ProviderConfig) (context.Context, trace.Span) {
	return otel.Tracer("tangled.org/arabica.social/arabica/internal/ai").Start(ctx, "chat "+config.Model,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("gen_ai.operation.name", "chat"),
			attribute.String("gen_ai.provider.name", "openai"),
			attribute.String("arabica.ai.provider", provider),
			attribute.String("gen_ai.request.model", config.Model),
			attribute.Int("gen_ai.request.max_tokens", config.MaxTokens),
		))
}

// RecordSpan adds content-free provider metadata without ending the caller's span.
// Missing or invalid usage is identified explicitly, never recorded as zero tokens.
func (o Observation) RecordSpan(span trace.Span) {
	span.SetAttributes(
		attribute.String("gen_ai.operation.name", "chat"),
		attribute.String("gen_ai.provider.name", "openai"),
		attribute.String("arabica.ai.provider", o.Provider),
		attribute.String("gen_ai.request.model", o.Model),
		attribute.Int("gen_ai.request.max_tokens", o.MaxTokens),
		attribute.String("arabica.ai.usage.state", o.UsageStatus),
	)
	if o.HTTPStatus != 0 {
		span.SetAttributes(attribute.Int("arabica.ai.upstream.status_code", o.HTTPStatus))
	}
	if o.FinishReason != "" {
		span.SetAttributes(attribute.StringSlice("gen_ai.response.finish_reasons", []string{o.FinishReason}))
	}
	if o.Usage != nil {
		span.SetAttributes(
			attribute.Int64("gen_ai.usage.input_tokens", o.Usage.InputTokens),
			attribute.Int64("gen_ai.usage.output_tokens", o.Usage.OutputTokens),
			attribute.Int64("gen_ai.usage.cache_read.input_tokens", o.Usage.CachedInputTokens),
		)
	}
	if o.Failure.Category != "" {
		span.SetAttributes(attribute.String("error.type", o.Failure.Category))
		span.SetStatus(codes.Error, o.Failure.Category)
	}
}

func tracedTransport(base http.RoundTripper) http.RoundTripper {
	return otelhttp.NewTransport(safeTransport{base}, otelhttp.WithPropagators(propagation.TraceContext{}))
}

// otelhttp records transport and body-read error strings. Sanitize them before
// instrumentation sees them, while preserving their types for failure classification.
type safeTransport struct{ base http.RoundTripper }

func (t safeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(r)
	if err != nil {
		return resp, safeHTTPError{cause: err, category: DescribeFailure(networkFailure(err)).Category}
	}
	if resp != nil && resp.Body != nil {
		resp.Body = safeResponseBody{resp.Body}
	}
	return resp, nil
}

type safeHTTPError struct {
	cause    error
	category string
}

func (e safeHTTPError) Error() string { return "ai: " + e.category }
func (e safeHTTPError) Unwrap() error { return e.cause }

type safeResponseBody struct{ io.ReadCloser }

func (b safeResponseBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if err != nil && err != io.EOF {
		return n, safeHTTPError{cause: err, category: "response_read"}
	}
	return n, err
}
