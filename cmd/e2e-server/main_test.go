//go:build integration

package main

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"tangled.org/arabica.social/arabica/tests/integration"
)

func TestE2EAccountAIAccessOptIn(t *testing.T) {
	h, err := integration.StartHarnessRuntime(context.Background(), t.TempDir(), nil)
	require.NoError(t, err)
	defer h.Close()
	control := e2eControlHandler(h)
	for _, body := range []string{"", `{"ai_access":false}`, `{"ai_access":true}`} {
		rec := httptest.NewRecorder()
		control.ServeHTTP(rec, httptest.NewRequest("POST", "/accounts", strings.NewReader(body)))
		require.Equal(t, 201, rec.Code, rec.Body.String())
		var account struct {
			DID string `json:"did"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &account))
		allowed, err := h.AIAccess.Allows(context.Background(), account.DID)
		require.NoError(t, err)
		require.Equal(t, body == `{"ai_access":true}`, allowed)
	}
	allowed, err := h.AIAccess.Allows(context.Background(), h.PrimaryAccount.DID)
	require.NoError(t, err)
	require.False(t, allowed)
}
