//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// brewListJSONResponse mirrors the GET /api/brews JSON envelope.
type brewListJSONResponse struct {
	Brews      []json.RawMessage `json:"brews"`
	HasMore    bool              `json:"has_more"`
	NextOffset int               `json:"next_offset"`
}

func TestHTTP_BrewListJSON(t *testing.T) {
	h := StartHarness(t, nil)

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Brew List Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form("name", "Brew List Bean", "roaster_rkey", roasterRKey)), "bean")
	grinderRKey := mustRKey(t, h.PostForm("/api/grinders", form("name", "Brew List Grinder")), "grinder")
	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "Brew List Brewer")), "brewer")

	for i := range 3 {
		brewForm := form(
			"bean_rkey", beanRKey,
			"grinder_rkey", grinderRKey,
			"brewer_rkey", brewerRKey,
			"method", "Pour Over",
			"water_amount", "300",
			"coffee_amount", "18",
			"rating", "7",
		)
		resp := h.PostForm("/brews", brewForm)
		require.Equal(t, 200, resp.StatusCode, "brew %d: %s", i, statusErr(resp, ReadBody(t, resp)))
	}

	// Fetch with a limit of 2 to test pagination.
	resp := getJSON(t, h, "/api/brews?limit=2")
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	var list brewListJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &list))
	assert.Len(t, list.Brews, 2, "should return exactly 2 brews with limit=2")
	assert.True(t, list.HasMore, "should have more results")
	assert.Equal(t, 2, list.NextOffset, "next offset should be 2")

	resp2 := getJSON(t, h, "/api/brews?limit=2&offset=2")
	body2 := ReadBody(t, resp2)
	require.Equal(t, 200, resp2.StatusCode, statusErr(resp2, body2))

	var list2 brewListJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body2), &list2))
	assert.Len(t, list2.Brews, 1, "should return 1 brew on the second page")
	assert.False(t, list2.HasMore, "should not have more results")
}

func TestHTTP_BrewListJSONHTMXStillHTML(t *testing.T) {
	h := StartHarness(t, nil)

	resp := h.GetHTMX("/api/brews")
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.NotEqual(t, "application/json", resp.Header.Get("Content-Type"))
}

func TestHTTP_BrewListJSONUnauth(t *testing.T) {
	h := StartHarness(t, nil)

	req, err := http.NewRequest("GET", h.URL("/api/brews"), nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 401, resp.StatusCode)
}
