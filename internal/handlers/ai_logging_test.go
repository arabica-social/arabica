package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"

	provider "tangled.org/arabica.social/arabica/internal/ai"
	arabicaai "tangled.org/arabica.social/arabica/internal/arabica/ai"
	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/records"
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
			if tc.finish == "length" {
				want["provider_error"] = "output_token_limit"
				if !strings.Contains(rec.Body.String(), "ai_output_limit") {
					t.Fatal("missing output-limit API error")
				}
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
		{"nested preparation", "brew", "unknown_fields", "pourover_params", map[string]any{"pourover_params": map[string]any{"filter": "secret-value"}}, 1},
		{"wrong kind field", "brew", "unknown_fields", "roast_level", map[string]any{"roast_level": "secret-value"}, 1},
		{"invalid known value", "brew", "invalid_value", "temperature", map[string]any{"temperature": float64(999)}, 0},
		{"multiple unknown keys", "brew", "unknown_fields", "recent_brews", map[string]any{"secret-key": "secret-value", "recent_brews": []any{}}, 2},
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
			if tc.field != "" {
				want["field_path"] = "/patch/" + tc.field
			}
			for key, value := range want {
				if entry[key] != value {
					t.Errorf("%s = %v, want %v", key, entry[key], value)
				}
			}
			if _, ok := entry["field_path"]; tc.field == "" && ok {
				t.Error("unexpected field path")
			}
			if strings.Contains(logged.String(), "secret") || strings.Contains(rec.Body.String(), "secret") {
				t.Fatal("diagnostic leaked model content")
			}
		})
	}
}

func TestAITruncatedDraftPreservesRequest(t *testing.T) {
	for _, content := range []string{"", `{"in_scope":true,"reply":"Ready","kind":"brew","patch":{"coffee_amount":20},"choices":[]}`} {
		h, _ := setupAI(t)
		calls := 0
		h.Service = arabicaai.NewService(aiFakeCompleter(func(context.Context, []provider.Message, []provider.Tool, json.RawMessage) (provider.Completion, error) {
			calls++
			return provider.Completion{Message: provider.Message{Role: "assistant", Content: content}, FinishReason: "length"}, nil
		}))
		rec := httptest.NewRecorder()
		h.HandleChat(rec, aiRequest(chatRequest))
		if rec.Code != 502 || !strings.Contains(rec.Body.String(), "ai_output_limit") || !strings.Contains(rec.Body.String(), "max_tokens") || strings.Contains(rec.Body.String(), "coffee_amount") || calls != 1 {
			t.Fatalf("unexpected truncated result: %d %s (%d calls)", rec.Code, rec.Body.String(), calls)
		}
	}
}

func TestAIChatMisplacedRecentBrewMarker(t *testing.T) {
	const did = "did:plc:test123456789"
	const topPath, patchPath = "/recent_brew_id", "/patch/recent_brew_id"
	const choicePath = "/choices/0/patch/recent_brew_id"
	for _, tc := range []struct {
		name, kind, topLevel, patch, choices, reason, issue, path string
		outOfScope                                                bool
	}{
		{name: "recover linked bean", patch: `{"recent_brew_id":"recent-brew-1","bean_rkey":"bean-1","coffee_amount":16}`},
		{name: "unrecognized source", patch: `{"recent_brew_id":"secret-id","bean_rkey":"bean-1"}`, reason: "unrecognized_recent_brew", issue: "unrecognized_source", path: patchPath},
		{name: "canonical unrecognized source", topLevel: `"recent_brew_id":"secret-id",`, patch: `{}`, reason: "unrecognized_recent_brew", issue: "unrecognized_source", path: topPath},
		{name: "duplicate placement", topLevel: `"recent_brew_id":"recent-brew-1",`, patch: `{"recent_brew_id":"recent-brew-1"}`, reason: "invalid_draft_fields", issue: "multiple_field_locations", path: patchPath},
		{name: "explicit null is still a duplicate", topLevel: `"recent_brew_id":null,`, patch: `{"recent_brew_id":"recent-brew-1"}`, reason: "invalid_draft_fields", issue: "multiple_field_locations", path: patchPath},
		{name: "malformed source", patch: `{"recent_brew_id":{"id":"secret-id"}}`, reason: "invalid_draft_fields", issue: "invalid_value", path: patchPath},
		{name: "empty source", patch: `{"recent_brew_id":""}`, reason: "invalid_draft_fields", issue: "invalid_value", path: patchPath},
		{name: "non-brew", kind: "bean", patch: `{"recent_brew_id":"recent-brew-1"}`, reason: "invalid_draft_fields", issue: "invalid_kind", path: patchPath},
		{name: "out of scope", outOfScope: true, patch: `{"recent_brew_id":"recent-brew-1"}`, reason: "invalid_draft_fields", issue: "invalid_scope", path: patchPath},
		{name: "invalid other field", patch: `{"recent_brew_id":"recent-brew-1","temperature":999}`, reason: "invalid_draft_fields", issue: "invalid_value", path: "/patch/temperature"},
		{name: "choice placement is rejected", patch: `{}`, choices: `[{"label":"Brew","patch":{"recent_brew_id":"recent-brew-1","bean_rkey":"bean-1"}}]`, reason: "invalid_choice_fields", issue: "misplaced_field", path: choicePath},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logged bytes.Buffer
			original := log.Logger
			log.Logger = zerolog.New(&logged)
			t.Cleanup(func() { log.Logger = original })
			h, ctx := setupAI(t)
			beanURI := "at://" + did + "/" + arabica.NSIDBean + "/old-bag"
			ctx.MockStore.FetchAllRecordsFunc = func(_ context.Context, nsid string) ([]records.RawRecord, error) {
				switch nsid {
				case arabica.NSIDBrew:
					return []records.RawRecord{{RKey: "old-brew", Record: map[string]any{"createdAt": "2026-11-01T00:00:00Z", "beanRef": beanURI}}}, nil
				case arabica.NSIDBean:
					return []records.RawRecord{{RKey: "old-bag", URI: beanURI, Record: map[string]any{"name": "Old bag", "closed": true}}}, nil
				}
				return nil, nil
			}
			kind := tc.kind
			if kind == "" {
				kind = "brew"
			}
			choices := tc.choices
			if choices == "" {
				choices = "[]"
			}
			calls := 0
			h.Service = arabicaai.NewService(aiFakeCompleter(func(context.Context, []provider.Message, []provider.Tool, json.RawMessage) (provider.Completion, error) {
				calls++
				body := `{` + tc.topLevel + `"kind":"` + kind + `","in_scope":true,"reply":"private reply","patch":` + tc.patch + `,"choices":` + choices + `}`
				if tc.outOfScope {
					body = strings.Replace(body, `"in_scope":true`, `"in_scope":false`, 1)
				}
				return provider.Completion{Message: provider.Message{Role: "assistant", Content: body}, FinishReason: "stop"}, nil
			}))
			request := aiRequest(strings.Replace(chatRequest, "18 grams coffee", "same as last time but 16 grams", 1))
			tid, _ := trace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
			sid, _ := trace.SpanIDFromHex("0123456789abcdef")
			request = request.WithContext(trace.ContextWithSpanContext(request.Context(), trace.NewSpanContext(trace.SpanContextConfig{TraceID: tid, SpanID: sid})))
			rec := httptest.NewRecorder()
			h.HandleChat(rec, request)
			if calls != 1 {
				t.Fatalf("provider calls = %d, want 1", calls)
			}
			var entry map[string]any
			if err := json.Unmarshal(logged.Bytes(), &entry); err != nil {
				t.Fatalf("missing or invalid diagnostic event: %v", err)
			}
			want := map[string]any{"stage": "response", "trace_id": tid.String()}
			if tc.reason == "" {
				if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "recent_brew_id") {
					t.Fatalf("expected provider-only marker to be removed in a 200: %d %s", rec.Code, rec.Body)
				}
				var response arabicaai.Response
				if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil || response.Patch["bean_rkey"] != "old-bag" || response.Patch["coffee_amount"] != float64(16) || len(response.Choices) != 0 {
					t.Fatalf("expected grounded repeat with explicit override: %+v (%v)", response, err)
				}
				want["reason"], want["field_path"], want["expected_field_path"] = "misplaced_field", patchPath, topPath
				want["message"] = "AI chat provider response repaired"
			} else {
				if rec.Code != http.StatusBadGateway || !strings.Contains(rec.Body.String(), "ai_provider_error") || strings.Contains(rec.Body.String(), "private reply") {
					t.Fatalf("expected redacted 502 without proposal: %d %s", rec.Code, rec.Body)
				}
				want["reason"], want["validation_issue"] = tc.reason, tc.issue
				if tc.path != "" {
					want["field_path"] = tc.path
				}
				if tc.path == patchPath || tc.path == choicePath {
					want["expected_field_path"] = topPath
				}
				want["message"] = "AI chat provider failure"
			}
			for key, value := range want {
				if entry[key] != value {
					t.Errorf("%s = %v, want %v", key, entry[key], value)
				}
			}
			for _, omitted := range []string{"upstream_status", "max_tokens", "provider_error"} {
				if _, ok := entry[omitted]; ok {
					t.Errorf("unexpected transport field %s", omitted)
				}
			}
			for _, secret := range []string{did, "secret-id", "recent-brew-1", "private reply", "old-bag"} {
				if strings.Contains(logged.String(), secret) {
					t.Errorf("diagnostic leaked %q", secret)
				}
			}
		})
	}
}
