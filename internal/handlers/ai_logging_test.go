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

func TestAIInvalidAssistantMessageDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name, role, content, finish, wantRole, wantFinish string
		tools                                             int
	}{
		{"empty truncated", "assistant", "", "length", "assistant", "length", 0},
		{"empty stopped", "assistant", "", "stop", "assistant", "stop", 0},
		{"wrong role", "user", "secret-response", "stop", "user", "stop", 0},
		{"missing role", "", "secret-response", "", "", "", 0},
		{"unknown values", "secret-role", "secret-response", "secret-finish", "other", "other", 0},
		{"wrong role with tools", "user", "", "tool_calls", "user", "tool_calls", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logged bytes.Buffer
			original := log.Logger
			log.Logger = zerolog.New(&logged)
			t.Cleanup(func() { log.Logger = original })
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var content any = tc.content
				if tc.content == "" {
					content = nil
				}
				message := map[string]any{"role": tc.role, "content": content}
				if tc.tools > 0 {
					message["tool_calls"] = []any{map[string]any{"id": "secret-tool", "type": "function", "function": map[string]any{"name": "secret-function", "arguments": "secret-arguments"}}}
				}
				json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": message, "finish_reason": tc.finish}}})
			}))
			defer upstream.Close()
			client, err := provider.NewClient(provider.Config{DefaultProvider: "test", Providers: map[string]provider.ProviderConfig{"test": {BaseURL: upstream.URL, Model: "test", APIKey: "secret-key"}}})
			if err != nil {
				t.Fatal(err)
			}
			h, _ := setupAI(t)
			h.Service = arabicaai.NewService(client)
			rec := httptest.NewRecorder()
			h.HandleChat(rec, aiRequest(chatRequest))
			if rec.Code != 502 {
				t.Fatalf("unexpected HTTP %d", rec.Code)
			}
			var entry map[string]any
			if err := json.Unmarshal(bytes.TrimSpace(logged.Bytes()), &entry); err != nil {
				t.Fatal(err)
			}
			want := map[string]any{
				"stage": "request", "reason": "upstream_failure", "provider_error": "invalid_assistant_message",
				"upstream_status": float64(200), "finish_reason": tc.wantFinish, "message_role": tc.wantRole,
				"response_bytes": float64(len(tc.content)), "tool_calls": float64(tc.tools),
			}
			for key, value := range want {
				if entry[key] != value {
					t.Errorf("%s = %v, want %v", key, entry[key], value)
				}
			}
			if strings.Contains(logged.String(), "secret") || strings.Contains(rec.Body.String(), "secret") {
				t.Fatal("diagnostic leaked upstream content")
			}
		})
	}
}

func TestAIInvalidDraftFieldsDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name, kind, issue, field string
		patch                    map[string]any
		unknown                  int
	}{
		{"unsupported kind", "secret-kind", "invalid_kind", "", map[string]any{}, 0},
		{"unknown key", "brew", "unknown_fields", "", map[string]any{"secret-key": "secret-value"}, 1},
		{"misplaced recent source", "brew", "unknown_fields", "recent_brew_id", map[string]any{"recent_brew_id": "recent-brew-1"}, 1},
		{"nested preparation", "brew", "unknown_fields", "pourover_params", map[string]any{"pourover_params": map[string]any{"filter": "secret-value"}}, 1},
		{"wrong kind field", "brew", "unknown_fields", "roast_level", map[string]any{"roast_level": "secret-value"}, 1},
		{"invalid known value", "brew", "invalid_value", "temperature", map[string]any{"temperature": float64(999)}, 0},
		{"multiple unknown keys", "brew", "unknown_fields", "recent_brew_id", map[string]any{"secret-key": "secret-value", "recent_brew_id": "recent-brew-1"}, 2},
		{"deterministic invalid field", "brew", "invalid_value", "coffee_amount", map[string]any{"temperature": float64(999), "coffee_amount": "secret-value"}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logged bytes.Buffer
			original := log.Logger
			log.Logger = zerolog.New(&logged)
			t.Cleanup(func() { log.Logger = original })
			body, err := json.Marshal(map[string]any{"in_scope": true, "reply": "secret-response", "kind": tc.kind, "patch": tc.patch, "choices": []any{}})
			if err != nil {
				t.Fatal(err)
			}
			h, _ := setupAI(t)
			h.Service = arabicaai.NewService(aiFakeCompleter(func(context.Context, []provider.Message, []provider.Tool, json.RawMessage) (provider.Completion, error) {
				return provider.Completion{Message: provider.Message{Role: "assistant", Content: string(body)}, FinishReason: "stop"}, nil
			}))
			rec := httptest.NewRecorder()
			h.HandleChat(rec, aiRequest(chatRequest))
			if rec.Code != 502 {
				t.Fatalf("unexpected HTTP %d", rec.Code)
			}
			var entry map[string]any
			if err := json.Unmarshal(bytes.TrimSpace(logged.Bytes()), &entry); err != nil {
				t.Fatal(err)
			}
			want := map[string]any{
				"stage": "response", "reason": "invalid_draft_fields", "finish_reason": "stop",
				"validation_issue": tc.issue, "field": tc.field, "unknown_fields": float64(tc.unknown),
			}
			for key, value := range want {
				if entry[key] != value {
					t.Errorf("%s = %v, want %v", key, entry[key], value)
				}
			}
			if strings.Contains(logged.String(), "secret") || strings.Contains(rec.Body.String(), "secret") {
				t.Fatal("diagnostic leaked model content")
			}
		})
	}
}
