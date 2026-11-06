package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	_ "modernc.org/sqlite"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	provider "tangled.org/arabica.social/arabica/internal/ai"
	arabicaai "tangled.org/arabica.social/arabica/internal/arabica/ai"
	"tangled.org/arabica.social/arabica/internal/records"
)

type aiFakeCompleter func(context.Context, []provider.Message, []provider.Tool, json.RawMessage) (provider.Completion, error)

func (f aiFakeCompleter) Complete(c context.Context, m []provider.Message, t []provider.Tool, r json.RawMessage) (provider.Completion, error) {
	return f(c, m, t, r)
}

const chatRequest = `{"session_id":"session","request_id":"request","revision":0,"messages":[{"role":"user","content":"18 grams coffee"}],"draft":{"kind":"brew","fields":{}}}`
const chatOutput = `{"in_scope":true,"reply":"Which bean?","kind":"brew","patch":{"coffee_amount":18},"choices":[]}`

func setupAI(t *testing.T) (AIHandlers, *TestContext) {
	t.Helper()
	tc := NewTestContext()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "access.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	access, err := arabicaai.NewAccessStore(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := access.Grant(context.Background(), "did:plc:test123456789", "did:plc:admin"); err != nil {
		t.Fatal(err)
	}
	tc.Handler.SetAIAccess(access)
	tc.Handler.SetStoreOverrideForTest(tc.MockStore)
	tc.MockStore.DIDFunc = func() string { return "did:plc:test123456789" }
	service := arabicaai.NewService(aiFakeCompleter(func(context.Context, []provider.Message, []provider.Tool, json.RawMessage) (provider.Completion, error) {
		return provider.Completion{Message: provider.Message{Content: chatOutput}, FinishReason: "stop"}, nil
	}))
	return AIHandlers{Handler: tc.Handler, Service: service}, tc
}
func aiRequest(body string) *http.Request {
	req := newMiddlewareAuthenticatedRequest(http.MethodPost, "/api/ai/chat")
	req.Body = ioNopCloser(body)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestAIStatus(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		h, _ := setupAI(t)
		if !enabled {
			h.Service = nil
		}
		recorder := httptest.NewRecorder()
		h.HandleStatus(recorder, newMiddlewareAuthenticatedRequest("GET", "/api/ai/status"))
		expected := `{"enabled":false,"allowed":true}`
		if enabled {
			expected = `{"enabled":true,"allowed":true}`
		}
		if strings.TrimSpace(recorder.Body.String()) != expected || recorder.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("status: %s", recorder.Body.String())
		}
	}
}

func TestAIChatReadOnly(t *testing.T) {
	h, tc := setupAI(t)
	tc.MockStore.PutRecordFunc = func(context.Context, string, string, any) (string, string, error) {
		t.Fatal("assistant wrote record")
		return "", "", nil
	}
	tc.MockStore.RemoveRecordFunc = func(context.Context, string, string) error { t.Fatal("assistant deleted record"); return nil }
	tc.MockStore.FetchAllRecordsFunc = func(context.Context, string) ([]records.RawRecord, error) { return nil, nil }
	rec := httptest.NewRecorder()
	h.HandleChat(rec, aiRequest(chatRequest))
	if rec.Code != 200 {
		t.Fatalf("chat status %d: %s", rec.Code, rec.Body.String())
	}
	var response arabicaai.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Patch["coffee_amount"] != float64(18) || response.Reply != "Which bean?" || response.Choices == nil {
		t.Fatalf("response %+v", response)
	}
}

func TestAIChatErrorContract(t *testing.T) {
	for _, tt := range []struct {
		name, body, contentType string
		status                  int
		code                    string
	}{
		{"malformed", "bad", "application/json", 400, "invalid_request"},
		{"unknown request", strings.TrimSuffix(chatRequest, "}") + `,"library":[]}`, "application/json", 400, "invalid_request"},
		{"trailing", chatRequest + ` {}`, "application/json", 400, "invalid_request"},
		{"invalid messages", strings.Replace(chatRequest, `"role":"user"`, `"role":"system"`, 1), "application/json", 400, "invalid_request"},
		{"missing fields", `{}`, "application/json", 400, "invalid_request"},
		{"oversize", `{"session_id":"` + strings.Repeat("x", 6*1024*1024+1) + `"}`, "application/json", 413, "request_too_large"},
		{"oversize trailing", chatRequest + strings.Repeat(" ", 6*1024*1024+1), "application/json", 413, "request_too_large"},
		{"wrong content type", chatRequest, "text/plain", 415, "invalid_content_type"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := setupAI(t)
			req := aiRequest(tt.body)
			req.Header.Set("Content-Type", tt.contentType)
			rec := httptest.NewRecorder()
			h.HandleChat(rec, req)
			if rec.Code != tt.status || !strings.Contains(rec.Body.String(), tt.code) {
				t.Fatalf("%d %s", rec.Code, rec.Body.String())
			}
		})
	}
	t.Run("unauthenticated", func(t *testing.T) {
		h, _ := setupAI(t)
		rec := httptest.NewRecorder()
		h.HandleChat(rec, httptest.NewRequest("POST", "/api/ai/chat", nil))
		if rec.Code != 404 {
			t.Fatalf("%d %s", rec.Code, rec.Body.String())
		}
	})
	t.Run("disabled", func(t *testing.T) {
		h, _ := setupAI(t)
		h.Service = nil
		rec := httptest.NewRecorder()
		h.HandleChat(rec, aiRequest(chatRequest))
		if rec.Code != 503 || !strings.Contains(rec.Body.String(), "ai_disabled") {
			t.Fatalf("%d %s", rec.Code, rec.Body.String())
		}
	})
	t.Run("wrong account", func(t *testing.T) {
		h, tc := setupAI(t)
		tc.MockStore.DIDFunc = func() string { return "did:plc:other" }
		rec := httptest.NewRecorder()
		h.HandleChat(rec, aiRequest(chatRequest))
		if rec.Code != 401 {
			t.Fatalf("%d %s", rec.Code, rec.Body.String())
		}
	})
	t.Run("library failure", func(t *testing.T) {
		h, tc := setupAI(t)
		tc.MockStore.FetchAllRecordsFunc = func(context.Context, string) ([]records.RawRecord, error) {
			return nil, errors.New("secret store error")
		}
		rec := httptest.NewRecorder()
		h.HandleChat(rec, aiRequest(chatRequest))
		if rec.Code != 503 || !strings.Contains(rec.Body.String(), "ai_library_unavailable") || strings.Contains(rec.Body.String(), "secret") {
			t.Fatalf("%d %s", rec.Code, rec.Body.String())
		}
	})
	t.Run("provider failure", func(t *testing.T) {
		h, _ := setupAI(t)
		h.Service = arabicaai.NewService(aiFakeCompleter(func(context.Context, []provider.Message, []provider.Tool, json.RawMessage) (provider.Completion, error) {
			return provider.Completion{}, errors.New("secret credential")
		}))
		rec := httptest.NewRecorder()
		h.HandleChat(rec, aiRequest(chatRequest))
		if rec.Code != 502 || !strings.Contains(rec.Body.String(), "ai_provider_error") || strings.Contains(rec.Body.String(), "secret") {
			t.Fatalf("%d %s", rec.Code, rec.Body.String())
		}
	})
	t.Run("rate", func(t *testing.T) {
		h, _ := setupAI(t)
		for i := 0; i < 10; i++ {
			rec := httptest.NewRecorder()
			h.HandleChat(rec, aiRequest(chatRequest))
			if rec.Code != 200 {
				t.Fatal(rec.Code)
			}
		}
		rec := httptest.NewRecorder()
		h.HandleChat(rec, aiRequest(chatRequest))
		if rec.Code != 429 || !strings.Contains(rec.Body.String(), "ai_rate_limited") || rec.Header().Get("Retry-After") != "60" {
			t.Fatalf("%d %s", rec.Code, rec.Body.String())
		}
	})
}

func TestAIChatCrossOriginProtection(t *testing.T) {
	h, _ := setupAI(t)
	mux := http.NewServeMux()
	mux.Handle("POST /api/ai/chat", http.NewCrossOriginProtection().Handler(http.HandlerFunc(h.HandleChat)))
	req := aiRequest(chatRequest)
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Fatalf("cross-origin chat permitted: %d", rec.Code)
	}
	req = aiRequest(chatRequest)
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("same-origin failed: %d %s", rec.Code, rec.Body.String())
	}
}

func TestAIChatCancellationAndProviderTimeout(t *testing.T) {
	for _, mode := range []string{"cancelled", "deadline", "provider timeout"} {
		t.Run(mode, func(t *testing.T) {
			h, _ := setupAI(t)
			var logs bytes.Buffer
			oldLogger := log.Logger
			log.Logger = zerolog.New(&logs)
			defer func() { log.Logger = oldLogger }()
			req := aiRequest(chatRequest)
			if mode == "provider timeout" {
				release := make(chan struct{})
				upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-release }))
				defer upstream.Close()
				defer close(release)
				client, err := provider.NewClient(provider.Config{DefaultProvider: "test", Providers: map[string]provider.ProviderConfig{"test": {BaseURL: upstream.URL, Model: "test", TimeoutSeconds: 1}}})
				if err != nil {
					t.Fatal(err)
				}
				h.Service = arabicaai.NewService(client)
			} else {
				ctx, cancel := context.WithCancel(req.Context())
				if mode == "deadline" {
					ctx, cancel = context.WithTimeout(req.Context(), 20*time.Millisecond)
				}
				defer cancel()
				req = req.WithContext(ctx)
				h.Service = arabicaai.NewService(aiFakeCompleter(func(ctx context.Context, _ []provider.Message, _ []provider.Tool, _ json.RawMessage) (provider.Completion, error) {
					if mode == "cancelled" {
						cancel()
					}
					<-ctx.Done()
					return provider.Completion{}, ctx.Err()
				}))
			}
			rec := httptest.NewRecorder()
			h.HandleChat(rec, req)
			if mode == "provider timeout" {
				if rec.Code != 502 || !strings.Contains(logs.String(), "AI chat provider failure") || req.Context().Err() != nil {
					t.Fatalf("provider timeout %d %s %s", rec.Code, rec.Body.String(), logs.String())
				}
			} else if rec.Code != 408 || !strings.Contains(rec.Body.String(), "request_cancelled") || strings.Contains(logs.String(), "AI chat provider failure") {
				t.Fatalf("request abort %d %s %s", rec.Code, rec.Body.String(), logs.String())
			}
		})
	}
}

func TestAIStatusDenied(t *testing.T) {
	h, _ := setupAI(t)
	for _, anonymous := range []bool{false, true} {
		req := newMiddlewareAuthenticatedRequest("GET", "/api/ai/status")
		if anonymous {
			req = httptest.NewRequest("GET", "/api/ai/status", nil)
		} else {
			h.SetAIAccess(nil)
		}
		rec := httptest.NewRecorder()
		h.HandleStatus(rec, req)
		if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != `{"enabled":false,"allowed":false}` {
			t.Fatalf("denied status %d %s", rec.Code, rec.Body.String())
		}
	}
}

type aiUsageStub struct {
	allowed bool
	err     error
	calls   int
}

func (s *aiUsageStub) Consume(context.Context, string, string, int64, int64) (int64, bool, error) {
	s.calls++
	if s.err != nil {
		return 0, false, s.err
	}
	return 0, s.allowed, nil
}

func (s *aiUsageStub) PurgeBefore(context.Context, string) error { return nil }

func TestAIChatDailyLimitResponses(t *testing.T) {
	for _, tc := range []struct {
		name    string
		usage   *aiUsageStub
		status  int
		code    string
		retry   bool
		charged bool
	}{
		{"allowance exhausted", &aiUsageStub{allowed: false}, 429, "ai_daily_limit", true, true},
		{"accounting unavailable", &aiUsageStub{allowed: true, err: errors.New("database is locked")}, 503, "ai_usage_unavailable", false, true},
		{"within allowance", &aiUsageStub{allowed: true}, 200, "", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := setupAI(t)
			providerCalls := 0
			h.Service = arabicaai.NewService(aiFakeCompleter(func(context.Context, []provider.Message, []provider.Tool, json.RawMessage) (provider.Completion, error) {
				providerCalls++
				return provider.Completion{Message: provider.Message{Content: chatOutput}, FinishReason: "stop"}, nil
			}))
			h.Service.SetUsageStore(tc.usage, 10000, nil)
			rec := httptest.NewRecorder()
			h.HandleChat(rec, aiRequest(chatRequest))
			if rec.Code != tc.status {
				t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
			}
			if tc.usage.calls != 1 {
				t.Fatalf("usage accounting not consulted: %d", tc.usage.calls)
			}
			if tc.code != "" && !strings.Contains(rec.Body.String(), tc.code) {
				t.Fatalf("missing %s: %s", tc.code, rec.Body.String())
			}
			retry := rec.Header().Get("Retry-After")
			if tc.retry && (retry == "" || retry == "0") {
				t.Fatalf("missing Retry-After: %q", retry)
			}
			wantProvider := 0
			if tc.status == 200 {
				wantProvider = 1
			}
			if providerCalls != wantProvider {
				t.Fatalf("provider calls %d, want %d", providerCalls, wantProvider)
			}
		})
	}
}
