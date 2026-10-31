package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testConfig(endpoint string) Config {
	return Config{DefaultProvider: "test", Providers: map[string]ProviderConfig{"test": {BaseURL: endpoint, Model: "test-model"}}}
}
func testClient(t *testing.T, config Config) *Client {
	t.Helper()
	c, err := NewClient(config)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

var testMessages = []Message{{Role: "user", Content: "test prompt"}}

const goodResponse = `{"choices":[{"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}]}`

func TestCompleteSuccess(t *testing.T) {
	t.Setenv("AI_TEST_KEY", "secret-token")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret-token" || r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing headers")
		}
		var body struct {
			Model          string          `json:"model"`
			Messages       []Message       `json:"messages"`
			Tools          []Tool          `json:"tools"`
			ResponseFormat json.RawMessage `json:"response_format"`
			MaxTokens      int             `json:"max_tokens"`
			Stream         bool            `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body.Model != "test-model" || body.MaxTokens != 2048 || body.Stream || len(body.Messages) != 3 || body.Messages[2].ToolCallID != "call-1" {
			t.Errorf("unexpected request: %+v", body)
		}
		if len(body.Tools) != 1 || !strings.Contains(string(body.Tools[0].Function), `"strict":false`) || string(body.ResponseFormat) != `{"type":"json_object"}` {
			t.Errorf("raw schemas not preserved: %+v", body)
		}
		io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":null,"tool_calls":[{"id":"call-2","type":"function","function":{"name":"read","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}`)
	}))
	defer server.Close()
	cfg := testConfig(server.URL + "/v1/")
	p := cfg.Providers["test"]
	p.APIKey = "${AI_TEST_KEY}"
	cfg.Providers["test"] = p
	client := testClient(t, cfg)
	// Credentials are resolved once, rather than consulting mutable environment per request.
	t.Setenv("AI_TEST_KEY", "changed")
	messages := []Message{testMessages[0], {Role: "assistant", ToolCalls: []ToolCall{{ID: "call-1", Type: "function", Function: FunctionCall{Name: "read", Arguments: "{}"}}}}, {Role: "tool", ToolCallID: "call-1", Content: "result"}}
	got, err := client.Complete(context.Background(), messages, []Tool{{Type: "function", Function: json.RawMessage(`{"name":"read","strict":false,"parameters":{"type":"object"}}`)}}, json.RawMessage(`{"type":"json_object"}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.FinishReason != "tool_calls" || len(got.Message.ToolCalls) != 1 || got.Message.ToolCalls[0].Function.Name != "read" {
		t.Fatalf("unexpected completion: %+v", got)
	}
}

func TestCompleteFailures(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		status     int
		limit      int64
		want       string
	}{
		{"success", goodResponse, 200, 0, ""},
		{"upstream secret", "secret-token test prompt", 401, 0, "HTTP 401"},
		{"malformed", "secret-token test prompt", 200, 0, "invalid provider response"},
		{"trailing", goodResponse + ` {}`, 200, 0, "invalid provider response"},
		{"no choices", `{"choices":[]}`, 200, 0, "invalid provider response"},
		{"wrong role", `{"choices":[{"message":{"role":"user","content":"bad"}}]}`, 200, 0, "invalid assistant message"},
		{"empty", `{"choices":[{"message":{"role":"assistant","content":""}}]}`, 200, 0, "invalid assistant message"},
		{"oversize", strings.Repeat("x", 101), 200, 100, "response exceeds limit"},
		{"empty body", "", 204, 0, "invalid provider response"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "" {
					t.Error("unexpected authorization for unauthenticated provider")
				}
				w.WriteHeader(tt.status)
				io.WriteString(w, tt.body)
			}))
			defer server.Close()
			cfg := testConfig(server.URL)
			p := cfg.Providers["test"]
			p.MaxResponseBytes = tt.limit
			cfg.Providers["test"] = p
			got, err := testClient(t, cfg).Complete(context.Background(), testMessages, nil, nil)
			if tt.want == "" {
				if err != nil || got.Message.Content != "hello" {
					t.Fatalf("got %+v, %v", got, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("wanted %s, got %v", tt.want, err)
			}
			if strings.Contains(err.Error(), "secret-token") || strings.Contains(err.Error(), "test prompt") {
				t.Fatal("error leaks body")
			}
		})
	}
}

func TestRedirectsNeverFollowed(t *testing.T) {
	var calls atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); t.Error("redirect followed") }))
	defer target.Close()
	for _, status := range []int{301, 302, 303, 307, 308} {
		origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, status) }))
		c := testClient(t, testConfig(origin.URL))
		c.key = "secret-token"
		_, err := c.Complete(context.Background(), testMessages, nil, nil)
		origin.Close()
		if err == nil || !strings.Contains(err.Error(), "HTTP ") {
			t.Fatalf("redirect %d: %v", status, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("redirect target was called")
	}
}

func TestCancellationAndTimeout(t *testing.T) {
	for _, phase := range []string{"headers", "body"} {
		t.Run(phase, func(t *testing.T) {
			started := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.Copy(io.Discard, r.Body)
				if phase == "body" {
					w.WriteHeader(200)
					w.(http.Flusher).Flush()
				}
				close(started)
				<-r.Context().Done()
			}))
			defer server.Close()
			client := testClient(t, testConfig(server.URL))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { _, err := client.Complete(ctx, testMessages, nil, nil); done <- err }()
			<-started
			cancel()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("wanted cancellation, got %v", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("cancellation did not stop request")
			}
		})
	}
	t.Run("configured timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.Copy(io.Discard, r.Body); <-r.Context().Done() }))
		defer server.Close()
		cfg := testConfig(server.URL)
		p := cfg.Providers["test"]
		p.TimeoutSeconds = 1
		cfg.Providers["test"] = p
		_, err := testClient(t, cfg).Complete(context.Background(), testMessages, nil, nil)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("wanted deadline, got %v", err)
		}
	})
}

func TestRequestLimitsAndInvalidJSON(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }))
	defer server.Close()
	cfg := testConfig(server.URL)
	p := cfg.Providers["test"]
	p.MaxRequestBytes = 10
	cfg.Providers["test"] = p
	_, err := testClient(t, cfg).Complete(context.Background(), testMessages, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "request exceeds limit") {
		t.Fatal(err)
	}
	client := testClient(t, testConfig(server.URL))
	for _, call := range []func() error{
		func() error {
			_, err := client.Complete(context.Background(), testMessages, nil, json.RawMessage(`invalid`))
			return err
		},
		func() error {
			_, err := client.Complete(context.Background(), testMessages, []Tool{{Type: "function", Function: json.RawMessage(`invalid`)}}, nil)
			return err
		},
		func() error { _, err := client.Complete(context.Background(), nil, nil, nil); return err },
	} {
		if err := call(); err == nil {
			t.Fatal("expected request rejection")
		}
	}
	if calls.Load() != 0 {
		t.Fatal("invalid request reached provider")
	}
}

func TestNetworkErrorRedacted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	endpoint := server.URL
	server.Close()
	_, err := testClient(t, testConfig(endpoint)).Complete(context.Background(), testMessages, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "connection refused") || DescribeFailure(err).Category != "connection_refused" {
		t.Fatalf("unexpected network error: %v", err)
	}
}
