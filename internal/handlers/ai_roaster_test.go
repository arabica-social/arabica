package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	provider "tangled.org/arabica.social/arabica/internal/ai"
	arabicaai "tangled.org/arabica.social/arabica/internal/arabica/ai"
	"tangled.org/arabica.social/arabica/internal/records"
)

func TestAIChatRoasterDraft(t *testing.T) {
	for _, tt := range []struct {
		name, draft, patch string
		status             int
	}{
		{"new", `{"kind":"brew","fields":{}}`, `{"name":"North Star","location":"Leeds","website":"https://example.com"}`, http.StatusOK},
		{"follow up", `{"kind":"roaster","fields":{"name":"North Star","website":"https://example.com"}}`, `{"location":"Leeds","website":null}`, http.StatusOK},
		{"name only", `{"kind":"brew","fields":{}}`, `{"name":"North Star"}`, http.StatusOK},
		{"invalid website", `{"kind":"roaster","fields":{}}`, `{"website":"javascript:alert(1)"}`, http.StatusBadGateway},
		{"website credentials", `{"kind":"roaster","fields":{}}`, `{"website":"https://user:pass@example.com"}`, http.StatusBadGateway},
		{"invented provenance", `{"kind":"roaster","fields":{}}`, `{"source_ref":"at://did:plc:other/social.arabica.alpha.roaster/example"}`, http.StatusBadGateway},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h, tc := setupAI(t)
			tc.MockStore.PutRecordFunc = func(context.Context, string, string, any) (string, string, error) {
				t.Fatal("assistant wrote a record")
				return "", "", nil
			}
			tc.MockStore.FetchAllRecordsFunc = func(context.Context, string) ([]records.RawRecord, error) { return nil, nil }
			h.Service = arabicaai.NewService(aiFakeCompleter(func(_ context.Context, _ []provider.Message, tools []provider.Tool, _ json.RawMessage) (provider.Completion, error) {
				if len(tools) != 0 {
					t.Fatal("assistant exposed executable tools")
				}
				return provider.Completion{
					Message:      provider.Message{Content: fmt.Sprintf(`{"in_scope":true,"reply":"Roaster details","kind":"roaster","patch":%s,"choices":[]}`, tt.patch)},
					FinishReason: "stop",
				}, nil
			}))
			rec := httptest.NewRecorder()
			h.HandleChat(rec, aiRequest(fmt.Sprintf(`{"session_id":"session","request_id":"request","revision":0,"messages":[{"role":"user","content":"Add North Star Coffee in Leeds"}],"draft":%s}`, tt.draft)))
			if rec.Code != tt.status {
				t.Fatalf("status %d, want %d: %s", rec.Code, tt.status, rec.Body.String())
			}
			if rec.Code != http.StatusOK {
				return
			}
			var response struct {
				Kind  string          `json:"kind"`
				Patch json.RawMessage `json:"patch"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			require.Equal(t, "roaster", response.Kind)
			require.JSONEq(t, tt.patch, string(response.Patch))
		})
	}
}
