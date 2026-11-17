package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func traceRecorder(t *testing.T) (*sdktrace.TracerProvider, *tracetest.SpanRecorder) {
	t.Helper()
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() { otel.SetTracerProvider(previous); _ = provider.Shutdown(context.Background()) })
	return provider, recorder
}

func spanAttributes(span sdktrace.ReadOnlySpan) map[string]attribute.Value {
	attrs := make(map[string]attribute.Value)
	for _, attr := range span.Attributes() {
		attrs[string(attr.Key)] = attr.Value
	}
	return attrs
}

func assertContentFreeSpans(t *testing.T, spans []sdktrace.ReadOnlySpan) {
	t.Helper()
	for _, span := range spans {
		dump := fmt.Sprintf("%s %v %v %v", span.Name(), span.Attributes(), span.Status(), span.Events())
		if strings.Contains(dump, "secret-") {
			t.Fatalf("sensitive data in span: %s", dump)
		}
	}
}

func TestCompletionTraces(t *testing.T) {
	for _, tc := range []struct {
		name, response, state, failure string
		status                         int
		tokens                         bool
	}{
		{"success", `{"choices":[{"message":{"role":"assistant","content":"secret-response"},"finish_reason":"stop"}],"usage":{"prompt_tokens":120,"completion_tokens":30,"prompt_tokens_details":{"cached_tokens":80}}}`, "reported", "", 200, true},
		{"missing usage", goodResponse, "missing", "", 200, false},
		{"invalid usage", `{"choices":[{"message":{"role":"assistant","content":"secret-response"},"finish_reason":"stop"}],"usage":{"prompt_tokens":-1,"completion_tokens":3}}`, "invalid", "", 200, false},
		{"empty reasoning exhaustion", `{"choices":[{"message":{"role":"assistant","content":null},"finish_reason":"length"}],"usage":{"prompt_tokens":120,"completion_tokens":30,"prompt_tokens_details":{"cached_tokens":80}}}`, "reported", "output_token_limit", 200, true},
		{"wrong role billed", `{"choices":[{"message":{"role":"user","content":"secret-response"},"finish_reason":"secret-finish"}],"usage":{"prompt_tokens":120,"completion_tokens":30,"prompt_tokens_details":{"cached_tokens":80}}}`, "reported", "invalid_assistant_message", 200, true},
		{"upstream rejection", "secret-provider-body", "unavailable", "upstream_http", 401, false},
		{"malformed response", "secret-provider-body", "unavailable", "invalid_provider_response", 200, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider, recorder := traceRecorder(t)
			var upstreamContext trace.SpanContext
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				upstreamContext = trace.SpanContextFromContext(propagation.TraceContext{}.Extract(r.Context(), propagation.HeaderCarrier(r.Header)))
				if r.Header.Get("Authorization") != "Bearer secret-key" {
					t.Error("authorization not preserved")
				}
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.response)
			}))
			defer server.Close()
			cfg := testConfig(server.URL + "/v1")
			p := cfg.Providers["test"]
			p.APIKey = "secret-key"
			cfg.Providers["test"] = p
			client := testClient(t, cfg)
			ctx, root := provider.Tracer("test").Start(context.Background(), "POST /api/ai/chat", trace.WithSpanKind(trace.SpanKindServer))
			_, err := client.Complete(ctx, []Message{{Role: "user", Content: "secret-prompt"}}, nil, nil)
			root.End()
			if (err != nil) != (tc.failure != "") {
				t.Fatalf("unexpected error: %v", err)
			}
			spans := recorder.Ended()
			if len(spans) != 3 {
				t.Fatalf("spans = %d", len(spans))
			}
			var ai, httpSpan sdktrace.ReadOnlySpan
			for _, span := range spans {
				if span.Name() == "chat test-model" {
					ai = span
				}
				if span.Name() == "HTTP POST" {
					httpSpan = span
				}
			}
			if ai == nil || httpSpan == nil {
				t.Fatal("missing model or HTTP span")
			}
			if ai.SpanKind() != trace.SpanKindClient || ai.Parent().SpanID() != root.SpanContext().SpanID() || httpSpan.Parent().SpanID() != ai.SpanContext().SpanID() || ai.SpanContext().TraceID() != root.SpanContext().TraceID() {
				t.Fatal("broken span hierarchy")
			}
			if upstreamContext.SpanID() != httpSpan.SpanContext().SpanID() || upstreamContext.TraceID() != root.SpanContext().TraceID() {
				t.Fatal("outbound trace context not propagated")
			}
			attrs := spanAttributes(ai)
			if attrs["gen_ai.request.max_tokens"].AsInt64() != 8192 || attrs["gen_ai.request.model"].AsString() != "test-model" || attrs["arabica.ai.provider"].AsString() != "test" || attrs["arabica.ai.usage.state"].AsString() != tc.state {
				t.Fatalf("wrong model metadata: %v", attrs)
			}
			if tc.tokens {
				if attrs["gen_ai.usage.input_tokens"].AsInt64() != 120 || attrs["gen_ai.usage.output_tokens"].AsInt64() != 30 || attrs["gen_ai.usage.cache_read.input_tokens"].AsInt64() != 80 {
					t.Fatalf("lost usage: %v", attrs)
				}
			} else if _, ok := attrs["gen_ai.usage.input_tokens"]; ok {
				t.Fatal("missing/invalid usage became zero tokens")
			}
			if tc.failure != "" && (ai.Status().Code != codes.Error || attrs["error.type"].AsString() != tc.failure) {
				t.Fatalf("wrong failure: %v %v", ai.Status(), attrs)
			}
			if tc.failure == "" && ai.Status().Code == codes.Error {
				t.Fatal("success marked as error")
			}
			httpAttrs := spanAttributes(httpSpan)
			if httpAttrs["http.response.status_code"].AsInt64() != int64(tc.status) || httpAttrs["url.full"].AsString() != server.URL+"/v1/chat/completions" {
				t.Fatalf("missing HTTP metadata: %v", httpAttrs)
			}
			if tc.failure == "output_token_limit" && httpSpan.Status().Code == codes.Error {
				t.Fatal("HTTP200 truncation is not an HTTP error")
			}
			assertContentFreeSpans(t, spans)
		})
	}
}

type tracingRoundTripper func(*http.Request) (*http.Response, error)

func (f tracingRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type failingTraceBody struct{}

func (failingTraceBody) Read([]byte) (int, error) { return 0, errors.New("secret-body-error") }
func (failingTraceBody) Close() error             { return nil }

func TestTransportTraceErrorsAreSanitized(t *testing.T) {
	for _, phase := range []string{"connect", "read"} {
		t.Run(phase, func(t *testing.T) {
			_, recorder := traceRecorder(t)
			client := testClient(t, testConfig("http://provider.invalid/v1"))
			client.http.Transport = tracedTransport(tracingRoundTripper(func(r *http.Request) (*http.Response, error) {
				if phase == "connect" {
					r.Body.Close()
					return nil, errors.New("secret-network-error")
				}
				r.Body.Close()
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: failingTraceBody{}, Request: r}, nil
			}))
			_, err := client.Complete(context.Background(), testMessages, nil, nil)
			if err == nil {
				t.Fatal("missing failure")
			}
			spans := recorder.Ended()
			if len(spans) != 2 {
				t.Fatalf("spans = %d", len(spans))
			}
			assertContentFreeSpans(t, spans)
			for _, span := range spans {
				if span.Status().Code != codes.Error {
					t.Fatalf("error not recorded on %s", span.Name())
				}
			}
		})
	}
}

func TestNoCompletionTraceBeforeUpstreamAdmission(t *testing.T) {
	_, recorder := traceRecorder(t)
	cfg := testConfig("http://provider.invalid/v1")
	p := cfg.Providers["test"]
	p.MaxRequestBytes = 1
	cfg.Providers["test"] = p
	_, err := testClient(t, cfg).Complete(context.Background(), testMessages, nil, nil)
	if !errors.Is(err, ErrRequestTooLarge) || len(recorder.Ended()) != 0 {
		t.Fatal("local rejection produced an upstream span")
	}
}
