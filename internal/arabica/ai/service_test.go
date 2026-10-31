package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	provider "tangled.org/arabica.social/arabica/internal/ai"
	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/records"
)

type fakeCompleter func(context.Context, []provider.Message, []provider.Tool, json.RawMessage) (provider.Completion, error)

func (f fakeCompleter) Complete(ctx context.Context, m []provider.Message, t []provider.Tool, r json.RawMessage) (provider.Completion, error) {
	return f(ctx, m, t, r)
}

type fakeLibrary struct {
	did   string
	fetch func(context.Context, string) ([]records.RawRecord, error)
}

func (f fakeLibrary) DID() string { return f.did }
func (f fakeLibrary) FetchAllRecords(ctx context.Context, nsid string) ([]records.RawRecord, error) {
	return f.fetch(ctx, nsid)
}

const ownDID = "did:plc:abcdefghijklmnopqrstuvwx"

func ownLibrary() fakeLibrary {
	return fakeLibrary{did: ownDID, fetch: func(ctx context.Context, nsid string) ([]records.RawRecord, error) {
		kind := strings.TrimPrefix(nsid, arabica.NSIDBase+".")
		notes := "do not send full record notes"
		if kind == "recipe" {
			notes = "Use a paper filter."
		}
		return []records.RawRecord{{RKey: "real-" + kind, Record: map[string]any{"name": "Own " + kind, "brewerType": "pourover", "notes": notes}}}, nil
	}}
}
func validRequest() Request {
	return Request{SessionID: "session", RequestID: "request", Messages: []Message{{Role: "user", Content: "18 grams coffee with Own brewer"}}, Draft: Draft{Kind: "brew", Fields: map[string]any{}}}
}

const validOutput = `{"in_scope":true,"reply":"Which bean?","kind":"brew","patch":{"coffee_amount":18,"brewer_rkey":"brewer-1"},"choices":[{"label":"Invented name","patch":{"bean_rkey":"bean-1"}}]}`

func outputClient(body string) fakeCompleter {
	return func(context.Context, []provider.Message, []provider.Tool, json.RawMessage) (provider.Completion, error) {
		return provider.Completion{Message: provider.Message{Role: "assistant", Content: body}, FinishReason: "stop"}, nil
	}
}

func TestChatGroundingAndPrivacy(t *testing.T) {
	req := validRequest()
	req.Draft.Fields["bean_rkey"] = "real-bean"
	req.Messages[0].Content += " @private.example did:plc:secret at://did:plc:secret/social.arabica.alpha.bean/raw"
	before := req.Messages[0].Content
	service := NewService(fakeCompleter(func(ctx context.Context, m []provider.Message, tools []provider.Tool, format json.RawMessage) (provider.Completion, error) {
		if len(m) != 2 || m[0].Role != "system" || len(tools) != 0 || format != nil {
			t.Error("unexpected provider encoding")
		}
		for _, secret := range []string{ownDID, "real-bean", "real-brewer", "do not send full record notes", "session", "request_id"} {
			if strings.Contains(m[1].Content, secret) {
				t.Errorf("unnecessary private data sent: %s", secret)
			}
		}
		if !strings.Contains(m[1].Content, "@private.example did:plc:secret at://did:plc:secret") {
			t.Error("authored facts changed")
		}
		if !strings.Contains(m[1].Content, `"bean_rkey":"bean-1"`) {
			t.Error("draft not converted to opaque references")
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Error("missing whole-operation timeout")
		}
		return outputClient(validOutput)(ctx, m, tools, format)
	}))
	response, err := service.Chat(context.Background(), ownDID, ownLibrary(), req)
	if err != nil {
		t.Fatal(err)
	}
	if response.Kind != "brew" || response.Patch["brewer_rkey"] != "real-brewer" || response.BrewerType != "pourover" || len(response.Choices) != 0 {
		t.Fatalf("invalid grounded result: %+v", response)
	}
	if req.Messages[0].Content != before || req.Draft.Fields["bean_rkey"] != "real-bean" {
		t.Fatal("service mutated caller context")
	}
}

func TestChatRejectsProviderOutput(t *testing.T) {
	for name, body := range map[string]string{
		"malformed": "not JSON", "partial": `{"in_scope":true}`, "trailing": validOutput + ` {}`,
		"unknown action":         strings.TrimSuffix(validOutput, "}") + `,"save":true}`,
		"unknown field":          strings.Replace(validOutput, `"coffee_amount":18`, `"arbitrary":18`, 1),
		"fractional integer":     strings.Replace(validOutput, `"coffee_amount":18`, `"coffee_amount":18.5`, 1),
		"range":                  strings.Replace(validOutput, `"coffee_amount":18`, `"coffee_amount":1001`, 1),
		"raw ref":                strings.Replace(validOutput, `"brewer-1"`, `"real-brewer"`, 1),
		"foreign ref":            strings.Replace(validOutput, `"brewer-1"`, `"brewer-99"`, 1),
		"wrong ref type":         strings.Replace(validOutput, `"brewer-1"`, `"bean-1"`, 1),
		"model metadata":         strings.Replace(validOutput, `"patch":{"bean_rkey":"bean-1"}`, `"patch":{"bean_rkey":"bean-1"},"brewer_type":"espresso"`, 1),
		"choice draft operation": strings.Replace(validOutput, `"bean_rkey":"bean-1"`, `"coffee_amount":20`, 1),
		"null choice":            strings.Replace(validOutput, `"bean_rkey":"bean-1"`, `"bean_rkey":null`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := NewService(outputClient(body)).Chat(context.Background(), ownDID, ownLibrary(), validRequest())
			if !errors.Is(err, ErrProvider) {
				t.Fatalf("wanted provider rejection, got %v", err)
			}
		})
	}
	for _, completion := range []provider.Completion{
		{Message: provider.Message{Content: validOutput}, FinishReason: "length"},
		{Message: provider.Message{Content: validOutput, ToolCalls: []provider.ToolCall{{ID: "execute"}}}, FinishReason: "stop"},
	} {
		svc := NewService(fakeCompleter(func(context.Context, []provider.Message, []provider.Tool, json.RawMessage) (provider.Completion, error) {
			return completion, nil
		}))
		if _, err := svc.Chat(context.Background(), ownDID, ownLibrary(), validRequest()); !errors.Is(err, ErrProvider) {
			t.Fatal(err)
		}
	}
}

func TestScopeAndKindSafety(t *testing.T) {
	req := validRequest()
	unrelated := strings.Replace(validOutput, `"in_scope":true`, `"in_scope":false`, 1)
	got, err := NewService(outputClient(unrelated)).Chat(context.Background(), ownDID, ownLibrary(), req)
	if err != nil || got.Reply != refusal || len(got.Patch) != 0 || len(got.Choices) != 0 {
		t.Fatalf("refusal %+v %v", got, err)
	}
	beanOutput := `{"in_scope":true,"reply":"What roast?","kind":"bean","patch":{"name":"New bag"},"choices":[]}`
	got, err = NewService(outputClient(beanOutput)).Chat(context.Background(), ownDID, ownLibrary(), req)
	if err != nil || got.Kind != "bean" {
		t.Fatalf("empty provisional kind %+v %v", got, err)
	}
	req.Draft.Fields["coffee_amount"] = float64(18)
	got, err = NewService(outputClient(beanOutput)).Chat(context.Background(), ownDID, ownLibrary(), req)
	if err != nil || got.Kind != "brew" || len(got.Patch) != 0 {
		t.Fatalf("nonempty kind safety %+v %v", got, err)
	}
}

func TestInvalidRequestAndLibraryNeverCallProvider(t *testing.T) {
	var calls atomic.Int32
	service := NewService(fakeCompleter(func(context.Context, []provider.Message, []provider.Tool, json.RawMessage) (provider.Completion, error) {
		calls.Add(1)
		return provider.Completion{}, nil
	}))
	for name, mutate := range map[string]func(*Request){
		"session": func(r *Request) { r.SessionID = "" }, "revision": func(r *Request) { r.Revision = -1 }, "message role": func(r *Request) { r.Messages[0].Role = "system" }, "empty": func(r *Request) { r.Messages = nil }, "long": func(r *Request) { r.Messages[0].Content = strings.Repeat("x", 4001) }, "many": func(r *Request) {
			for len(r.Messages) < 41 {
				r.Messages = append(r.Messages, r.Messages[0])
			}
		}, "kind": func(r *Request) { r.Draft.Kind = "roaster" }, "unknown field": func(r *Request) { r.Draft.Fields["evil"] = true }, "source": func(r *Request) { r.Draft.Fields["recipe_owner_did"] = ownDID }, "null fields": func(r *Request) { r.Draft.Fields = nil },
	} {
		t.Run(name, func(t *testing.T) {
			req := validRequest()
			mutate(&req)
			if _, err := service.Chat(context.Background(), ownDID, ownLibrary(), req); !errors.Is(err, ErrInvalidRequest) {
				t.Fatal(err)
			}
		})
	}
	if _, err := service.Chat(context.Background(), "did:plc:other", ownLibrary(), validRequest()); !errors.Is(err, ErrLibrary) {
		t.Fatal(err)
	}
	broken := ownLibrary()
	broken.fetch = func(context.Context, string) ([]records.RawRecord, error) { return nil, errors.New("secret upstream") }
	if _, err := service.Chat(context.Background(), ownDID, broken, validRequest()); !errors.Is(err, ErrLibrary) {
		t.Fatal(err)
	}
	req := validRequest()
	req.Draft.Fields["bean_rkey"] = "foreign"
	if _, err := service.Chat(context.Background(), ownDID, ownLibrary(), req); !errors.Is(err, ErrLibrary) {
		t.Fatal(err)
	}
	broken.fetch = func(context.Context, string) ([]records.RawRecord, error) {
		return []records.RawRecord{{RKey: "valid", URI: "at://did:plc:other/social.arabica.alpha.bean/valid"}}, nil
	}
	if _, err := service.Chat(context.Background(), ownDID, broken, validRequest()); !errors.Is(err, ErrLibrary) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("invalid input called provider")
	}
}

func TestLargeCatalogIncludesCurrentSelection(t *testing.T) {
	lib := ownLibrary()
	lib.fetch = func(ctx context.Context, nsid string) ([]records.RawRecord, error) {
		result := make([]records.RawRecord, 150)
		for i := range result {
			result[i] = records.RawRecord{RKey: fmt.Sprintf("key-%d", i), Record: map[string]any{"name": strings.Repeat("x", 500)}}
		}
		return result, nil
	}
	req := validRequest()
	req.Draft.Fields["bean_rkey"] = "key-149"
	cat, err := loadCatalog(context.Background(), lib, req.Draft)
	if err != nil {
		t.Fatal(err)
	}
	if len(cat) != 420 {
		t.Fatalf("catalog not bounded: %d", len(cat))
	}
	mapped, _, err := cat.convert(req.Draft.Fields, true)
	wantAlias := ""
	for _, item := range cat {
		if item.Kind == "bean" && item.rkey == "key-149" {
			wantAlias = item.ID
		}
	}
	if err != nil || wantAlias == "" || mapped["bean_rkey"] != wantAlias {
		t.Fatalf("selected record omitted: %+v %v", mapped, err)
	}
	for _, item := range cat {
		if len(item.Label) > 200 {
			t.Fatal("label exceeds limit")
		}
	}
}

func TestFieldTypesAndRanges(t *testing.T) {
	for _, tt := range []struct {
		kind, key string
		value     any
		valid     bool
	}{
		{"brew", "coffee_amount", 18.0, true}, {"brew", "coffee_amount", 18.5, false}, {"brew", "temperature", 93.5, true}, {"brew", "temperature", 213.0, false}, {"brew", "coffee_amount", "18", false},
		{"brew", "espresso_params.yield_weight", 37.5, true}, {"brew", "espresso_params.pre_infusion_seconds", 2.5, false}, {"brew", "pourover_params.filter", "paper", true},
		{"bean", "rating", 0.0, false}, {"brew", "rating", 0.0, true}, {"bean", "rating", nil, true}, {"bean", "closed", false, true}, {"bean", "closed", "false", false}, {"bean", "roast_date", "2026-99-99", false}, {"bean", "roast_date", "2026-10-31", true},
		{"bean", "link", "javascript:alert(1)", false}, {"bean", "link", "https://example.com", true}, {"bean", "source_ref", "at://foreign", false},
		{"brew", "pours", []any{map[string]any{"water_amount": 50.0, "time_seconds": 30.0}}, true}, {"brew", "pours", []any{map[string]any{"water_amount": 50.5, "time_seconds": 30.0}}, false},
	} {
		t.Run(tt.kind+"/"+tt.key+fmt.Sprint(tt.value), func(t *testing.T) {
			err := validateFields(tt.kind, map[string]any{tt.key: tt.value})
			if (err == nil) != tt.valid {
				t.Fatalf("valid=%v err=%v", tt.valid, err)
			}
		})
	}
}

func TestLimiterConcurrencyRateAndBoundedGrowth(t *testing.T) {
	var l limiter
	now := time.Now()
	release, err := l.acquire("account", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.acquire("account", now); !errors.Is(err, ErrLimit) {
		t.Fatal("concurrent request admitted")
	}
	release()
	for i := 0; i < 9; i++ {
		release, err := l.acquire("account", now)
		if err != nil {
			t.Fatal(err)
		}
		release()
	}
	if _, err := l.acquire("account", now); !errors.Is(err, ErrLimit) {
		t.Fatal("rate limit bypassed")
	}
	if release, err := l.acquire("account", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	} else {
		release()
	}
	var bounded limiter
	for i := 0; i < 1024; i++ {
		release, err := bounded.acquire(fmt.Sprint(i), now)
		if err != nil {
			t.Fatal(err)
		}
		release()
	}
	if _, err := bounded.acquire("extra", now.Add(30*time.Second)); !errors.Is(err, ErrLimit) {
		t.Fatal("account-state capacity exceeded")
	}
	if len(bounded.accounts) != 1024 {
		t.Fatal("limiter growth exceeded cap")
	}
	if release, err := bounded.acquire("extra", now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	} else {
		release()
	}
	if len(bounded.accounts) != 1 {
		t.Fatal("idle keys not pruned")
	}
}

func TestServiceConcurrencyAndProviderFailure(t *testing.T) {
	started := make(chan struct{})
	finish := make(chan struct{})
	service := NewService(fakeCompleter(func(ctx context.Context, m []provider.Message, t []provider.Tool, r json.RawMessage) (provider.Completion, error) {
		close(started)
		<-finish
		return provider.Completion{}, errors.New("secret key provider failure")
	}))
	done := make(chan error, 1)
	go func() {
		_, err := service.Chat(context.Background(), ownDID, ownLibrary(), validRequest())
		done <- err
	}()
	<-started
	if _, err := service.Chat(context.Background(), ownDID, ownLibrary(), validRequest()); !errors.Is(err, ErrLimit) {
		t.Fatal(err)
	}
	close(finish)
	if err := <-done; !errors.Is(err, ErrProvider) || strings.Contains(err.Error(), "secret") {
		t.Fatal(err)
	}
}

func TestBootstrapEnvironment(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("ARABICA_AI_CONFIG", "")
	if s, err := FromEnv(); err != nil || s.Enabled() {
		t.Fatalf("unset feature: %v %v", s, err)
	}
	t.Setenv("ARABICA_AI_CONFIG", filepath.Join(t.TempDir(), "missing"))
	if _, err := FromEnv(); err == nil {
		t.Fatal("explicit missing configuration accepted")
	}
	path := filepath.Join(t.TempDir(), "ai.json")
	config := `{"default_provider":"local","providers":{"local":{"base_url":"http://localhost:9999/v1","model":"local"}}}`
	if err := os.WriteFile(path, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARABICA_AI_CONFIG", path)
	if s, err := FromEnv(); err != nil || !s.Enabled() {
		t.Fatalf("valid local configuration: %v %v", s, err)
	}
	if err := os.WriteFile(path, []byte(`bad secret`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := FromEnv(); err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatal(err)
	}
}

func TestBootstrapDefaultConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("ARABICA_AI_CONFIG", "")
	config := `{"default_provider":"local","providers":{"local":{"base_url":"http://127.0.0.1:1234/v1","model":"test"}}}`
	if err := os.WriteFile("ai.json", []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	if service, err := FromEnv(); err != nil || !service.Enabled() {
		t.Fatalf("default ai.json not loaded: %v", err)
	}
	if err := os.WriteFile("ai.json", []byte("invalid default"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := FromEnv(); err == nil {
		t.Fatal("invalid default configuration accepted")
	}
	if err := os.WriteFile("override.json", []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARABICA_AI_CONFIG", "override.json")
	if service, err := FromEnv(); err != nil || !service.Enabled() {
		t.Fatalf("explicit override did not take precedence: %v", err)
	}
	t.Setenv("ARABICA_AI_CONFIG", "missing.json")
	if _, err := FromEnv(); err == nil {
		t.Fatal("missing explicit path fell back to default")
	}
}
