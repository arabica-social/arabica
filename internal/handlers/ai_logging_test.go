package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"net/http/httptest"
	"strings"
	provider "tangled.org/arabica.social/arabica/internal/ai"
	arabicaai "tangled.org/arabica.social/arabica/internal/arabica/ai"
	"testing"
)

func TestAIProviderFailureLogsSafeDiagnostics(t *testing.T) {
	var logged bytes.Buffer
	original := log.Logger
	log.Logger = zerolog.New(&logged)
	t.Cleanup(func() { log.Logger = original })
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte("secret-provider-body"))
	}))
	defer upstream.Close()
	client, err := provider.NewClient(provider.Config{DefaultProvider: "test", Providers: map[string]provider.ProviderConfig{"test": {BaseURL: upstream.URL, Model: "test", APIKey: "secret-key"}}})
	if err != nil {
		t.Fatal(err)
	}
	h, _ := setupAI(t)
	h.Service = arabicaai.NewService(client)
	request := aiRequest(strings.Replace(chatRequest, "18 grams coffee", "secret-prompt", 1))
	tid, _ := trace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
	sid, _ := trace.SpanIDFromHex("0123456789abcdef")
	request = request.WithContext(trace.ContextWithSpanContext(request.Context(), trace.NewSpanContext(trace.SpanContextConfig{TraceID: tid, SpanID: sid})))
	rec := httptest.NewRecorder()
	h.HandleChat(rec, request)
	if rec.Code != 502 {
		t.Fatalf("unexpected HTTP %d", rec.Code)
	}
	var entry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(logged.Bytes()), &entry); err != nil {
		t.Fatal("missing diagnostic JSON")
	}
	if entry["stage"] != "request" || entry["provider_error"] != "upstream_http" || entry["upstream_status"] != float64(401) || entry["trace_id"] != tid.String() {
		t.Fatalf("wrong safe diagnostic: %v", entry)
	}
	if strings.Contains(logged.String(), "secret") || strings.Contains(rec.Body.String(), "secret") {
		t.Fatal("logs or API leaked credentials/content")
	}
	logged.Reset()
	h.Service = arabicaai.NewService(aiFakeCompleter(func(context.Context, []provider.Message, []provider.Tool, json.RawMessage) (provider.Completion, error) {
		return provider.Completion{}, errors.New("secret-credential")
	}))
	h.HandleChat(httptest.NewRecorder(), aiRequest(chatRequest))
	if strings.Contains(logged.String(), "secret") {
		t.Fatal("opaque dependency error leaked into logs")
	}
}
