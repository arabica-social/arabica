//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"tangled.org/pdewey.com/atp"
)

// feedJSONResponse mirrors the GET /api/feed JSON envelope for assertions.
type feedJSONResponse struct {
	Items           []json.RawMessage `json:"items"`
	NextCursor      string            `json:"next_cursor"`
	IsAuthenticated bool              `json:"is_authenticated"`
	Query           struct {
		Type string `json:"type"`
		Sort string `json:"sort"`
	} `json:"query"`
}

// getJSON fetches a path with an Accept: application/json header so the
// content-negotiating handlers return JSON instead of HTML.
func getJSON(t *testing.T, h *Harness, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("GET", h.URL(path), nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")
	resp, err := h.Client.Do(req)
	require.NoError(t, err)
	return resp
}

// TestHTTP_FeedJSON verifies that GET /api/feed with Accept: application/json
// returns a JSON envelope (not an HTML partial) with the expected shape.
func TestHTTP_FeedJSON(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	// Create a roaster so the feed has at least one item.
	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Feed JSON Roaster")), "roaster")
	uri := atp.BuildATURI(h.PrimaryAccount.DID, "social.arabica.alpha.roaster", rkey)
	h.WaitForRecord(uri, firehoseWait)

	resp := getJSON(t, h, "/api/feed")
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var feed feedJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &feed))
	assert.True(t, feed.IsAuthenticated, "authenticated user should see is_authenticated=true")
	assert.Equal(t, "recent", feed.Query.Sort)
	assert.NotEmpty(t, feed.Items, "feed should contain the created roaster")
}

// TestHTTP_FeedJSONHTMXStillHTML verifies that an HTMX request (no Accept:
// application/json) still gets the HTML partial, confirming content
// negotiation doesn't break existing HTMX clients.
func TestHTTP_FeedJSONHTMXStillHTML(t *testing.T) {
	h := StartHarness(t, nil)

	resp := h.GetHTMX("/api/feed")
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	// HTML partials start with a tag or whitespace+tag, never a JSON object.
	assert.NotEqual(t, "application/json", resp.Header.Get("Content-Type"))
}

// TestHTTP_FeedJSONTypeFilter verifies the type query param is echoed in the
// response and the items match the filter.
func TestHTTP_FeedJSONTypeFilter(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Filter Roaster")), "roaster")
	uri := atp.BuildATURI(h.PrimaryAccount.DID, "social.arabica.alpha.roaster", rkey)
	h.WaitForRecord(uri, firehoseWait)

	resp := getJSON(t, h, "/api/feed?type=roaster&sort=recent")
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))

	var feed feedJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &feed))
	assert.Equal(t, "roaster", feed.Query.Type)
	assert.Equal(t, "recent", feed.Query.Sort)
}
