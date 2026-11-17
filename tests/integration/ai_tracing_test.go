//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	provider "tangled.org/arabica.social/arabica/internal/ai"
	arabicaai "tangled.org/arabica.social/arabica/internal/arabica/ai"
)

func TestAI_ChatTraceUsage(t *testing.T) {
	for _, finish := range []string{"stop", "length"} {
		t.Run(finish, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			previous := otel.GetTracerProvider()
			otel.SetTracerProvider(tp)
			t.Cleanup(func() { otel.SetTracerProvider(previous); _ = tp.Shutdown(context.Background()) })
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				content := aiChatReply
				if finish == "length" {
					content = ""
				}
				json.NewEncoder(w).Encode(map[string]any{
					"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": content}, "finish_reason": finish}},
					"usage":   map[string]any{"prompt_tokens": 120, "completion_tokens": 30, "prompt_tokens_details": map[string]any{"cached_tokens": 80}},
				})
			}))
			defer upstream.Close()
			client, err := provider.NewClient(provider.Config{DefaultProvider: "test", Providers: map[string]provider.ProviderConfig{"test": {BaseURL: upstream.URL, Model: "trace-model"}}})
			require.NoError(t, err)
			h := StartHarness(t, &HarnessOptions{AI: arabicaai.NewService(client)})
			require.NoError(t, h.AIAccess.Grant(context.Background(), h.PrimaryAccount.DID, "did:plc:staff"))
			resp := h.do(http.MethodPost, "/api/ai/chat", chatBody("Use my previous brew"))
			defer resp.Body.Close()
			if finish == "stop" {
				assert.Equal(t, 200, resp.StatusCode)
			} else {
				assert.Equal(t, 502, resp.StatusCode)
			}
			var root, ai, outbound sdktrace.ReadOnlySpan
			for _, span := range recorder.Ended() {
				if span.Name() == "POST /api/ai/chat" && span.SpanKind() == trace.SpanKindServer {
					root = span
				}
				if span.Name() == "chat trace-model" {
					ai = span
				}
			}
			require.NotNil(t, root, "missing inbound server span")
			require.NotNil(t, ai, "missing model span")
			for _, span := range recorder.Ended() {
				if span.Parent().SpanID() == ai.SpanContext().SpanID() && span.Name() == "HTTP POST" {
					outbound = span
				}
			}
			require.NotNil(t, outbound, "missing outbound HTTP span")
			assert.Equal(t, root.SpanContext().SpanID(), ai.Parent().SpanID())
			assert.Equal(t, root.SpanContext().TraceID(), outbound.SpanContext().TraceID())
			for _, span := range []sdktrace.ReadOnlySpan{root, ai} {
				attrs := map[string]attribute.Value{}
				for _, attr := range span.Attributes() {
					attrs[string(attr.Key)] = attr.Value
				}
				assert.Equal(t, int64(120), attrs["gen_ai.usage.input_tokens"].AsInt64())
				assert.Equal(t, int64(30), attrs["gen_ai.usage.output_tokens"].AsInt64())
				assert.Equal(t, int64(80), attrs["gen_ai.usage.cache_read.input_tokens"].AsInt64())
				assert.Equal(t, "reported", attrs["arabica.ai.usage.state"].AsString())
				assert.Equal(t, []string{finish}, attrs["gen_ai.response.finish_reasons"].AsStringSlice())
				assert.Equal(t, int64(8192), attrs["gen_ai.request.max_tokens"].AsInt64())
				if finish == "length" {
					assert.Equal(t, codes.Error, span.Status().Code)
				}
			}
		})
	}
}
