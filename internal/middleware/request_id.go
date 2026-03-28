package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

// RequestIDMiddleware extracts the OTel trace ID from the current span (set by
// otelhttp) and injects it into the zerolog logger on the request context. If
// no active trace exists, it generates a random 8-byte hex ID as a fallback.
//
// Every downstream handler that uses zerolog.Ctx(r.Context()) will
// automatically include the trace_id field in its log output, making it easy
// to correlate all log lines from a single request.
//
// The trace ID is also set as the X-Trace-ID response header so that it can be
// correlated with client-side error reports.
func RequestIDMiddleware(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID := extractTraceID(r)

			// Create a sub-logger with trace_id and inject into context
			subLogger := logger.With().Str("trace_id", traceID).Logger()
			ctx := subLogger.WithContext(r.Context())

			// Set response header for client-side correlation
			w.Header().Set("X-Trace-ID", traceID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractTraceID returns the OTel trace ID if an active span exists,
// otherwise generates a random fallback ID.
func extractTraceID(r *http.Request) string {
	sc := trace.SpanFromContext(r.Context()).SpanContext()
	if sc.HasTraceID() {
		return sc.TraceID().String()
	}

	// Fallback: generate a random ID (e.g. for requests filtered out of tracing)
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	return hex.EncodeToString(buf[:])
}
