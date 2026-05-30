package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"
)

func TestUserDIDSpanMiddlewareAddsAuthenticatedDIDToActiveSpan(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	tracer := provider.Tracer("test")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := atpmiddleware.ContextWithAuth(req.Context(), "did:plc:test123", "test-session")
	ctx, span := tracer.Start(ctx, "GET /")
	req = req.WithContext(ctx)

	UserDIDSpanMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(httptest.NewRecorder(), req)
	span.End()

	ended := recorder.Ended()
	assert.Len(t, ended, 1)
	assert.Contains(t, ended[0].Attributes(), attribute.String("user.did", "did:plc:test123"))
}

func TestUserDIDSpanMiddlewareSkipsAnonymousRequests(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	tracer := provider.Tracer("test")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx, span := tracer.Start(req.Context(), "GET /")
	req = req.WithContext(ctx)

	UserDIDSpanMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(httptest.NewRecorder(), req)
	span.End()

	ended := recorder.Ended()
	assert.Len(t, ended, 1)
	assert.NotContains(t, ended[0].Attributes(), attribute.String("user.did", "did:plc:test123"))
}
