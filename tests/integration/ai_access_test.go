//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	provider "tangled.org/arabica.social/arabica/internal/ai"
	"tangled.org/arabica.social/arabica/internal/aiusage"
	arabicaai "tangled.org/arabica.social/arabica/internal/arabica/ai"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const aiChatReply = `{"in_scope":true,"reply":"Which bean?","kind":"brew","patch":{},"choices":[]}`

type cannedCompleter struct {
	calls int
}

func (c *cannedCompleter) Complete(context.Context, []provider.Message, []provider.Tool, json.RawMessage) (provider.Completion, error) {
	c.calls++
	return provider.Completion{Message: provider.Message{Role: "assistant", Content: aiChatReply}, FinishReason: "stop"}, nil
}

func (h *Harness) do(method, path string, body []byte) *http.Response {
	h.T.Helper()
	req, err := http.NewRequest(method, h.URL(path), strings.NewReader(string(body)))
	require.NoError(h.T, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.Client.Do(req)
	require.NoError(h.T, err)
	return resp
}

func chatBody(content string) []byte {
	body, _ := json.Marshal(map[string]any{
		"session_id": "session",
		"request_id": "request",
		"revision":   0,
		"messages":   []map[string]any{{"role": "user", "content": content}},
		"draft":      map[string]any{"kind": "brew", "fields": map[string]any{}},
	})
	return body
}

func TestAI_ChatRequiresGrant(t *testing.T) {
	completer := &cannedCompleter{}
	h := StartHarness(t, &HarnessOptions{AI: arabicaai.NewService(completer)})
	other := h.CreateAccount("bob@test.com", "bob.test", "hunter2")
	otherClient := h.NewClientForAccount(other)

	req, err := http.NewRequest(http.MethodPost, h.URL("/api/ai/chat"), strings.NewReader(string(chatBody("18 grams coffee"))))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := otherClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "an ungranted account must not reach the assistant")
	assert.Equal(t, 0, completer.calls)

	require.NoError(t, h.AIAccess.Grant(context.Background(), h.PrimaryAccount.DID, "did:plc:staff"))
	granted := h.do(http.MethodPost, "/api/ai/chat", chatBody("18 grams coffee"))
	assert.Equal(t, http.StatusOK, granted.StatusCode)
	assert.Equal(t, 1, completer.calls)
}

// TestAI_HeadOnAccessEndpointKeepsGrant guards the moderator management API
// against method confusion: HEAD resolves to the GET pattern, so it must stay a
// read and never take the revoke branch that DELETE owns.
func TestAI_HeadOnAccessEndpointKeepsGrant(t *testing.T) {
	completer := &cannedCompleter{}
	h := StartHarness(t, &HarnessOptions{AI: arabicaai.NewService(completer)})
	target := h.CreateAccount("carol@test.com", "carol.test", "hunter2")
	require.NoError(t, h.SetStaff(nil, []string{h.PrimaryAccount.DID}))

	grant := h.do(http.MethodPut, "/api/_mod/ai-access", []byte(`{"did":"`+target.DID+`"}`))
	require.Equal(t, http.StatusOK, grant.StatusCode)

	head := h.do(http.MethodHead, "/api/_mod/ai-access", []byte(`{"did":"`+target.DID+`"}`))
	require.Equal(t, http.StatusOK, head.StatusCode)

	allowed, err := h.AIAccess.Allows(context.Background(), target.DID)
	require.NoError(t, err)
	assert.True(t, allowed, "HEAD must not revoke a grant")

	post := h.do(http.MethodPost, "/api/_mod/ai-access", []byte(`{"did":"`+target.DID+`"}`))
	assert.NotEqual(t, http.StatusOK, post.StatusCode, "unlisted methods must not mutate grants")
	allowed, err = h.AIAccess.Allows(context.Background(), target.DID)
	require.NoError(t, err)
	assert.True(t, allowed, "unsupported methods must not revoke a grant")
}

func TestAI_DailyAllowanceEnforcedPerAccount(t *testing.T) {
	completer := &cannedCompleter{}
	service := arabicaai.NewService(completer)
	h := StartHarness(t, &HarnessOptions{AI: service})
	require.NoError(t, h.AIAccess.Grant(context.Background(), h.PrimaryAccount.DID, "did:plc:staff"))

	usage, err := aiusage.NewStore(h.FeedIndex.DB())
	require.NoError(t, err)
	// "18 grams coffee" charges 15 characters, so a 20-character day admits one request.
	service.SetUsageStore(usage, 20, nil)

	first := h.do(http.MethodPost, "/api/ai/chat", chatBody("18 grams coffee"))
	require.Equal(t, http.StatusOK, first.StatusCode)
	require.Equal(t, 1, completer.calls)

	second := h.do(http.MethodPost, "/api/ai/chat", chatBody("18 grams coffee"))
	require.Equal(t, http.StatusTooManyRequests, second.StatusCode)
	body, err := io.ReadAll(second.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "ai_daily_limit")
	retry, err := strconv.Atoi(second.Header.Get("Retry-After"))
	require.NoError(t, err, "daily limit must publish a Retry-After window")
	assert.Positive(t, retry)
	assert.LessOrEqual(t, retry, 24*60*60)
	assert.Equal(t, 1, completer.calls, "a refused request must not reach the provider")

	used, err := usage.Used(context.Background(), h.PrimaryAccount.DID, aiusage.Day(time.Now()))
	require.NoError(t, err)
	assert.EqualValues(t, 15, used, "one charged request must be persisted for the UTC day")
}

func TestAI_DailyAllowanceExemptsAdmins(t *testing.T) {
	completer := &cannedCompleter{}
	service := arabicaai.NewService(completer)
	h := StartHarness(t, &HarnessOptions{AI: service})
	require.NoError(t, h.AIAccess.Grant(context.Background(), h.PrimaryAccount.DID, "did:plc:staff"))
	usage, err := aiusage.NewStore(h.FeedIndex.DB())
	require.NoError(t, err)
	service.SetUsageStore(usage, 1, func(did string) bool { return did == h.PrimaryAccount.DID })

	for i := 0; i < 2; i++ {
		resp := h.do(http.MethodPost, "/api/ai/chat", chatBody("18 grams coffee"))
		require.Equal(t, http.StatusOK, resp.StatusCode, "exempt accounts are not charged")
	}
	assert.Equal(t, 2, completer.calls)
	used, err := usage.Used(context.Background(), h.PrimaryAccount.DID, aiusage.Day(time.Now()))
	require.NoError(t, err)
	assert.Zero(t, used)
}

// TestAI_RefusedRequestsDoNotSpendAllowance pins the accounting boundary: a
// request the server rejects before the provider call costs the account nothing.
func TestAI_RefusedRequestsDoNotSpendAllowance(t *testing.T) {
	completer := &cannedCompleter{}
	service := arabicaai.NewService(completer)
	h := StartHarness(t, &HarnessOptions{AI: service})
	require.NoError(t, h.AIAccess.Grant(context.Background(), h.PrimaryAccount.DID, "did:plc:staff"))
	usage, err := aiusage.NewStore(h.FeedIndex.DB())
	require.NoError(t, err)
	service.SetUsageStore(usage, 20, nil)

	invalid := h.do(http.MethodPost, "/api/ai/chat", []byte(`{"session_id":"s","request_id":"r","revision":0,"messages":[],"draft":{"kind":"brew","fields":{}}}`))
	require.Equal(t, http.StatusBadRequest, invalid.StatusCode)

	used, err := usage.Used(context.Background(), h.PrimaryAccount.DID, aiusage.Day(time.Now()))
	require.NoError(t, err)
	assert.Zero(t, used)
	assert.Zero(t, completer.calls)
}
