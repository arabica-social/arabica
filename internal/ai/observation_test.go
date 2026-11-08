package ai

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestObservedCompletion(t *testing.T) {
	for _, tt := range []struct {
		name, usage, state, outcome string
		status                      int
		invalidChoice               bool
		want                        *Usage
	}{
		{name: "tokens", usage: `,"usage":{"prompt_tokens":120,"completion_tokens":30,"prompt_tokens_details":{"cached_tokens":80},"completion_tokens_details":{"reasoning_tokens":10}}`, state: "reported", status: 200, want: &Usage{120, 30, 80}},
		{name: "zero", usage: `,"usage":{"prompt_tokens":0,"completion_tokens":0}`, state: "reported", status: 200, want: &Usage{}},
		{name: "missing", state: "missing", status: 200},
		{name: "null", usage: `,"usage":null`, state: "missing", status: 200},
		{name: "negative", usage: `,"usage":{"prompt_tokens":-1,"completion_tokens":2}`, state: "invalid", status: 200},
		{name: "partial", usage: `,"usage":{"prompt_tokens":2}`, state: "invalid", status: 200},
		{name: "fraction", usage: `,"usage":{"prompt_tokens":1.5,"completion_tokens":2}`, state: "invalid", status: 200},
		{name: "wrong type", usage: `,"usage":"secret"`, state: "invalid", status: 200},
		{name: "excess cache", usage: `,"usage":{"prompt_tokens":2,"completion_tokens":1,"prompt_tokens_details":{"cached_tokens":3}}`, state: "invalid", status: 200},
		{name: "billed invalid choice", usage: `,"usage":{"prompt_tokens":12,"completion_tokens":3}`, state: "reported", status: 200, invalidChoice: true, outcome: "invalid_assistant_message", want: &Usage{12, 3, 0}},
		{name: "rate limited", state: "unavailable", status: 429, outcome: "upstream_http"},
		{name: "provider failure", state: "unavailable", status: 503, outcome: "upstream_http"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				role := "assistant"
				if tt.invalidChoice {
					role = "user"
				}
				_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"` + role + `","content":"hello"}}]` + tt.usage + `}`))
			}))
			defer server.Close()
			var seen []Observation
			client, err := NewClientWithObserver(testConfig(server.URL), func(o Observation) { seen = append(seen, o) })
			if err != nil {
				t.Fatal(err)
			}
			var scoped []Observation
			ctx := WithRequestObserver(context.Background(), func(o Observation) { scoped = append(scoped, o) })
			got, err := client.Complete(ctx, testMessages, nil, nil)
			if (err != nil) != (tt.outcome != "") {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(seen) != 1 {
				t.Fatalf("observations: %d", len(seen))
			}
			o := seen[0]
			if len(scoped) != 1 || scoped[0] != o || o.StartedAt.IsZero() {
				t.Fatalf("scoped observation mismatch: %+v", scoped)
			}
			if o.Provider != "test" || o.Model != "test-model" || o.Duration <= 0 || o.HTTPStatus != tt.status || o.Failure.Category != tt.outcome || o.UsageStatus != tt.state {
				t.Fatalf("unexpected observation: %+v", o)
			}
			if (o.Usage == nil) != (tt.want == nil) || (tt.want != nil && *o.Usage != *tt.want) {
				t.Fatalf("usage: %+v", o.Usage)
			}
			if err == nil && got.Usage != o.Usage {
				t.Fatal("completion usage differs from observation")
			}
		})
	}
}

func TestObservedCompletionBoundaries(t *testing.T) {
	var seen []Observation
	client, err := NewClientWithObserver(testConfig("http://localhost:1"), func(o Observation) { seen = append(seen, o) })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Complete(context.Background(), nil, nil, nil); err == nil {
		t.Fatal("expected validation error")
	}
	client.provider.MaxRequestBytes = 1
	if _, err := client.Complete(context.Background(), testMessages, nil, nil); !errors.Is(err, ErrRequestTooLarge) {
		t.Fatal(err)
	}
	if len(seen) != 0 {
		t.Fatal("local rejection counted as upstream call")
	}
	client.provider.MaxRequestBytes = 10000
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.Complete(ctx, testMessages, nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if len(seen) != 1 || seen[0].Failure.Category != "canceled" || seen[0].HTTPStatus != 0 || seen[0].UsageStatus != "unavailable" {
		t.Fatalf("observations: %+v", seen)
	}
}

func TestConcurrentObserver(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(goodResponse)) }))
	defer server.Close()
	var mu sync.Mutex
	count := 0
	client, err := NewClientWithObserver(testConfig(server.URL), func(Observation) { mu.Lock(); count++; mu.Unlock() })
	if err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for range 10 {
		group.Go(func() {
			if _, err := client.Complete(context.Background(), testMessages, nil, nil); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	if count != 10 {
		t.Fatalf("observed %d calls", count)
	}
}
