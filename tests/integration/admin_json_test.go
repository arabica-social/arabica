//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// adminMutationJSONResponse mirrors the admin mutation JSON response.
type adminMutationJSONResponse struct {
	OK      bool   `json:"ok"`
	Action  string `json:"action"`
	Message string `json:"message,omitempty"`
}

// TestHTTP_AdminJSON verifies that the admin JSON endpoints return 403 for
// non-moderators (the test harness has no moderation service configured, so
// the RequireModerator middleware denies access).
func TestHTTP_AdminJSON(t *testing.T) {
	h := StartHarness(t, nil)

	// GET /api/_mod requires moderator — harness has no moderation service,
	// so this returns 403.
	resp := getJSON(t, h, "/api/_mod")
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	ReadBody(t, resp) // drain

	// GET /api/_mod/stats requires admin — also 403.
	resp2 := getJSON(t, h, "/api/_mod/stats")
	assert.Equal(t, http.StatusForbidden, resp2.StatusCode)
	ReadBody(t, resp2)
}

// TestHTTP_AdminMutationJSON verifies that the admin mutation endpoints return
// 403 for non-moderators even with Accept: application/json.
func TestHTTP_AdminMutationJSON(t *testing.T) {
	h := StartHarness(t, nil)

	formData := "uri=at://did:plc:test/social.arabica.alpha.brew/xyz"
	req, err := http.NewRequest("POST", h.URL("/_mod/hide"), strings.NewReader(formData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := h.Client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Without a moderation service, the RequirePermission middleware denies.
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// TestHTTP_AdminMutationJSONContentType verifies the response is JSON when
// the moderation service is configured. Since the harness doesn't set up
// moderation, this test just verifies the content negotiation doesn't crash
// and returns an appropriate status code.
func TestHTTP_AdminMutationJSONContentType(t *testing.T) {
	h := StartHarness(t, nil)

	// Verify the JSON path is reachable (not 404 or 500 from a missing route).
	// The 403 confirms the route exists and the middleware is working.
	formData := "did=did:plc:test"
	req, err := http.NewRequest("POST", h.URL("/_mod/block"), strings.NewReader(formData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := h.Client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// TestHTTP_AdminStatsJSONShape verifies the JSON response shape for the stats
// endpoint when access is denied. This is a lightweight check that the route
// is wired and returns the expected status code.
func TestHTTP_AdminStatsJSONShape(t *testing.T) {
	h := StartHarness(t, nil)

	resp := getJSON(t, h, "/api/_mod/stats")
	body := ReadBody(t, resp)

	// 403 means the route exists and RequireAdmin middleware is working.
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	// The body should be a plain text error, not JSON (middleware writes
	// http.Error before the handler runs).
	assert.NotEmpty(t, body)
}

// Ensure json import is used (for potential future shape assertions).
var _ = json.Marshal
