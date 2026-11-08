package ai

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"go.opentelemetry.io/otel/trace"
	provider "tangled.org/arabica.social/arabica/internal/ai"
	"tangled.org/arabica.social/arabica/internal/metrics"
)

func testRequestLog(t *testing.T) (*RequestLogStore, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "requests.db"))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	store, err := NewRequestLogStore(db)
	if err != nil {
		t.Fatal(err)
	}
	return store, db
}

func loggedService(t *testing.T, store *RequestLogStore, handler http.HandlerFunc) *Service {
	t.Helper()
	upstream := httptest.NewServer(handler)
	t.Cleanup(upstream.Close)
	client, err := provider.NewClient(provider.Config{DefaultProvider: "request-log-test", Providers: map[string]provider.ProviderConfig{"request-log-test": {BaseURL: upstream.URL, Model: "configured-model", APIKey: "private-key"}}})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(client)
	service.SetRequestLog(store)
	return service
}

func writeCompletion(w http.ResponseWriter, content, usage string) {
	encoded, _ := json.Marshal(content)
	_, _ = fmt.Fprintf(w, `{"choices":[{"message":{"role":"assistant","content":%s},"finish_reason":"stop"}]%s}`, encoded, usage)
}

func TestChatRequestLog(t *testing.T) {
	for _, tt := range []struct {
		name, usage, state, outcome string
		status                      int
		badOutput                   bool
		wantInput                   sql.NullInt64
	}{
		{name: "reported", usage: `,"usage":{"prompt_tokens":100,"completion_tokens":20,"prompt_tokens_details":{"cached_tokens":40}}`, state: "reported", outcome: "success", status: 200, wantInput: sql.NullInt64{Int64: 100, Valid: true}},
		{name: "zero", usage: `,"usage":{"prompt_tokens":0,"completion_tokens":0}`, state: "reported", outcome: "success", status: 200, wantInput: sql.NullInt64{Valid: true}},
		{name: "missing", state: "missing", outcome: "success", status: 200},
		{name: "invalid usage", usage: `,"usage":{"prompt_tokens":-1,"completion_tokens":2}`, state: "invalid", outcome: "success", status: 200},
		{name: "provider failure", state: "unavailable", outcome: "upstream_http", status: 429},
		{name: "invalid draft still billed", usage: `,"usage":{"prompt_tokens":100,"completion_tokens":20}`, state: "reported", outcome: "success", status: 200, badOutput: true, wantInput: sql.NullInt64{Int64: 100, Valid: true}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			store, db := testRequestLog(t)
			service := loggedService(t, store, func(w http.ResponseWriter, r *http.Request) {
				raw, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				for _, private := range []string{ownDID, "private-request-id"} {
					if strings.Contains(string(raw), private) {
						t.Errorf("sent account metadata upstream: %s", private)
					}
				}
				w.WriteHeader(tt.status)
				content := validOutput
				if tt.badOutput {
					content = "private invalid reply"
				}
				writeCompletion(w, content, tt.usage)
			})
			quota := &fakeUsage{}
			service.SetUsageStore(quota, 1, func(string) bool { return true })
			req := validRequest()
			req.RequestID = "private-request-id"
			traceID := trace.TraceID{1}
			ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: trace.SpanID{2}}))
			_, err := service.Chat(ctx, ownDID, ownLibrary(), req)
			if (err != nil) != (tt.status != 200 || tt.badOutput) {
				t.Fatalf("Chat: %v", err)
			}
			if quota.calls != 0 {
				t.Fatal("exempt caller quota charged")
			}
			var did, requestID, traceValue, providerName, model, state, outcome string
			var status int
			var started, duration int64
			var input, output, cached sql.NullInt64
			err = db.QueryRow(`SELECT did,request_id,trace_id,provider,model,usage_state,outcome,http_status,started_at_ms,duration_ns,input_tokens,output_tokens,cached_input_tokens FROM ai_request_log`).Scan(&did, &requestID, &traceValue, &providerName, &model, &state, &outcome, &status, &started, &duration, &input, &output, &cached)
			if err != nil {
				t.Fatal(err)
			}
			if did != ownDID || requestID != req.RequestID || traceValue != traceID.String() || providerName != "request-log-test" || model != "configured-model" || state != tt.state || outcome != tt.outcome || status != tt.status || started <= 0 || duration <= 0 || input != tt.wantInput {
				t.Fatalf("unexpected log: %s %s %s %s %s %s %s %d %d %d %+v", did, requestID, traceValue, providerName, model, state, outcome, status, started, duration, input)
			}
			if output.Valid != input.Valid || cached.Valid != input.Valid {
				t.Fatal("unknown tokens represented as zero")
			}
			if tt.name == "reported" && (output.Int64 != 20 || cached.Int64 != 40) {
				t.Fatal("token counts changed")
			}
		})
	}
}

func TestRequestLogSurvivesCancellation(t *testing.T) {
	store, db := testRequestLog(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service := loggedService(t, store, func(w http.ResponseWriter, r *http.Request) { cancel(); writeCompletion(w, validOutput, "") })
	if _, err := service.Chat(ctx, ownDID, ownLibrary(), validRequest()); err == nil {
		t.Fatal("expected canceled call")
	}
	var outcome, state string
	if err := db.QueryRow(`SELECT outcome,usage_state FROM ai_request_log`).Scan(&outcome, &state); err != nil {
		t.Fatal(err)
	}
	if outcome != "canceled" || state != "unavailable" {
		t.Fatalf("%s %s", outcome, state)
	}
}

func TestRequestLogConcurrentAccountsAndRepeatedIDs(t *testing.T) {
	store, db := testRequestLog(t)
	service := loggedService(t, store, func(w http.ResponseWriter, r *http.Request) {
		writeCompletion(w, validOutput, `,"usage":{"prompt_tokens":10,"completion_tokens":2}`)
	})
	var group sync.WaitGroup
	for i := range 6 {
		group.Go(func() {
			library := ownLibrary()
			library.did = fmt.Sprintf("did:plc:account%d", i)
			for range 2 {
				if _, err := service.Chat(context.Background(), library.did, library, validRequest()); err != nil {
					t.Error(err)
				}
			}
		})
	}
	group.Wait()
	rows, err := db.Query(`SELECT did,COUNT(*),SUM(input_tokens),SUM(output_tokens) FROM ai_request_log GROUP BY did`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var did string
		var requests, input, output int
		if err := rows.Scan(&did, &requests, &input, &output); err != nil {
			t.Fatal(err)
		}
		if requests != 2 || input != 20 || output != 4 {
			t.Fatalf("%s: %d %d %d", did, requests, input, output)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if count != 6 {
		t.Fatalf("accounts: %d", count)
	}
}

func TestRequestLogFailureDoesNotDiscardCompletion(t *testing.T) {
	store, db := testRequestLog(t)
	service := loggedService(t, store, func(w http.ResponseWriter, r *http.Request) { writeCompletion(w, validOutput, "") })
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	var before, after dto.Metric
	_ = metrics.AIRequestLogErrorsTotal.Write(&before)
	if _, err := service.Chat(context.Background(), ownDID, ownLibrary(), validRequest()); err != nil {
		t.Fatal(err)
	}
	_ = metrics.AIRequestLogErrorsTotal.Write(&after)
	if after.GetCounter().GetValue()-before.GetCounter().GetValue() != 1 {
		t.Fatal("lost log was not observable")
	}
	if _, err := NewRequestLogStore(db); err == nil {
		t.Fatal("startup storage failure hidden")
	}
}

func TestRequestLogLocalRejectionsNotRecorded(t *testing.T) {
	store, db := testRequestLog(t)
	service := loggedService(t, store, func(w http.ResponseWriter, r *http.Request) { t.Error("unexpected upstream call") })
	client, err := provider.NewClient(provider.Config{DefaultProvider: "small", Providers: map[string]provider.ProviderConfig{"small": {BaseURL: "http://localhost:1", Model: "small", MaxRequestBytes: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	service.client = client
	if _, err := service.Chat(context.Background(), ownDID, ownLibrary(), validRequest()); !errors.Is(err, ErrProviderRequestTooLarge) {
		t.Fatal(err)
	}
	service.SetUsageStore(&fakeUsage{}, 1, nil)
	if _, err := service.Chat(context.Background(), ownDID, ownLibrary(), validRequest()); !errors.Is(err, ErrDailyLimit) {
		t.Fatal(err)
	}
	if _, err := service.Chat(context.Background(), "did:plc:other", ownLibrary(), validRequest()); !errors.Is(err, ErrLibrary) {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ai_request_log`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("local rejects logged as calls: %d", count)
	}
}

func TestRequestLogRetainsOldRowsAndReopens(t *testing.T) {
	store, db := testRequestLog(t)
	o := provider.Observation{StartedAt: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Provider: "test", Model: "test", UsageStatus: "reported", Usage: &provider.Usage{InputTokens: 10, OutputTokens: 2}}
	for range 2 {
		if err := store.Record(context.Background(), ownDID, "same-id", "", o); err != nil {
			t.Fatal(err)
		}
	}
	var sequence int
	var name, path string
	if err := db.QueryRow(`PRAGMA database_list`).Scan(&sequence, &name, &path); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := NewRequestLogStore(db); err != nil {
		t.Fatal(err)
	}
	quota, err := NewDailyUsageStore(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := quota.PurgeBefore(context.Background(), "2026-01-01"); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ai_request_log`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("historical or repeated calls lost: %d", count)
	}
	if _, err := db.Exec(`INSERT INTO ai_request_log (started_at_ms,did,request_id,trace_id,provider,model,duration_ns,outcome,http_status,usage_state) VALUES (0,'did','request','','provider','model',0,'success',200,'reported')`); err == nil {
		t.Fatal("reported usage without tokens accepted")
	}
}
