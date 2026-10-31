package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	provider "tangled.org/arabica.social/arabica/internal/ai"
	arabicaai "tangled.org/arabica.social/arabica/internal/arabica/ai"
	"tangled.org/arabica.social/arabica/internal/records"
)

// This is a real HTTP provider round trip through the authenticated handler,
// app validation and generic transport. It needs no external credentials.
func TestAIChatProviderHTTPRoundTrip(t *testing.T) {
	t.Setenv("AI_TRANSPORT_TEST_KEY", "test-secret")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Error("invalid provider request")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if strings.Contains(string(body), "did:plc:test123456789") || strings.Contains(string(body), "real-bean") {
			t.Error("automatic identity/reference data forwarded")
		}
		if !strings.Contains(string(body), "bean-1") {
			t.Error("missing opaque own library")
		}
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": `{"in_scope":true,"reply":"Which bean?","kind":"brew","patch":{"coffee_amount":18},"choices":[{"label":"Use this","patch":{"bean_rkey":"bean-1"}}]}`}, "finish_reason": "stop"}}})
	}))
	defer upstream.Close()
	client, err := provider.NewClient(provider.Config{DefaultProvider: "test", Providers: map[string]provider.ProviderConfig{"test": {BaseURL: upstream.URL + "/v1", Model: "test", APIKey: "${AI_TRANSPORT_TEST_KEY}"}}})
	if err != nil {
		t.Fatal(err)
	}
	h, tc := setupAI(t)
	h.Service = arabicaai.NewService(client)
	tc.MockStore.FetchAllRecordsFunc = func(ctx context.Context, nsid string) ([]records.RawRecord, error) {
		if strings.HasSuffix(nsid, ".bean") {
			// Keep this a real unresolved choice: a sole open bag now fills a new draft.
			return []records.RawRecord{
				{RKey: "real-bean", Record: map[string]any{"name": "My beans"}},
				{RKey: "other-bean", Record: map[string]any{"name": "Other beans"}},
			}, nil
		}
		return nil, nil
	}
	rec := httptest.NewRecorder()
	h.HandleChat(rec, aiRequest(chatRequest))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"bean_rkey":"real-bean"`) || !strings.Contains(rec.Body.String(), `"label":"My beans"`) {
		t.Fatalf("HTTP grounding failed: %d %s", rec.Code, rec.Body.String())
	}
}
