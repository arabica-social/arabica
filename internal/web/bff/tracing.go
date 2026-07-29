package bff

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/trace"
)

// Traceparent returns the W3C traceparent header value for the current span,
// or an empty string if no active trace exists.
func Traceparent(ctx context.Context) string {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.HasTraceID() || !sc.HasSpanID() {
		return ""
	}

	flags := "00"
	if sc.IsSampled() {
		flags = "01"
	}

	return fmt.Sprintf("00-%s-%s-%s", sc.TraceID(), sc.SpanID(), flags)
}
