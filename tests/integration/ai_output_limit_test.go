//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	provider "tangled.org/arabica.social/arabica/internal/ai"
	arabicaai "tangled.org/arabica.social/arabica/internal/arabica/ai"
)

func TestAI_OutputLimit(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content any
	}{{"empty", nil}, {"partial", aiChatReply}} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				var request struct {
					MaxTokens int `json:"max_tokens"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
				}
				assert.Equal(t, 8192, request.MaxTokens)
				json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": tc.content}, "finish_reason": "length"}}})
			}))
			defer upstream.Close()
			client, err := provider.NewClient(provider.Config{DefaultProvider: "test", Providers: map[string]provider.ProviderConfig{"test": {BaseURL: upstream.URL, Model: "test"}}})
			require.NoError(t, err)
			h := StartHarness(t, &HarnessOptions{AI: arabicaai.NewService(client)})
			require.NoError(t, h.AIAccess.Grant(context.Background(), h.PrimaryAccount.DID, "did:plc:staff"))
			resp := h.do(http.MethodPost, "/api/ai/chat", chatBody("Use my previous brew"))
			defer resp.Body.Close()
			assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
			var result map[string]any
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
			assert.Equal(t, "ai_output_limit", result["code"])
			assert.Contains(t, result["error"], "max_tokens")
			assert.NotContains(t, result, "patch")
			assert.Equal(t, int32(1), calls.Load())
		})
	}
}
