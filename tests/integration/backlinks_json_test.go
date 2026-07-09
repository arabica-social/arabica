//go:build integration

package integration

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"tangled.org/pdewey.com/atp"
)

// backlinksJSONResponse mirrors the GET /api/{entity}/{actor}/{id}/backlinks
// JSON envelope.
type backlinksJSONResponse struct {
	EntityNoun string          `json:"entity_noun"`
	EntityName string          `json:"entity_name"`
	BackURL    string          `json:"back_url"`
	DetailURL  string          `json:"detail_url"`
	Result     json.RawMessage `json:"result"`
}

// TestHTTP_BacklinksJSON verifies that GET /api/roasters/{actor}/{id}/backlinks
// with Accept: application/json returns the backlinks envelope.
func TestHTTP_BacklinksJSON(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Backlinks JSON Roaster")), "roaster")

	// Wait for indexing so the backlinks lookup has data.
	roasterURI := atp.BuildATURI(h.PrimaryAccount.DID, "social.arabica.alpha.roaster", rkey)
	h.WaitForRecord(roasterURI, firehoseWait)

	actor := h.PrimaryAccount.DID
	resp := getJSON(t, h, "/api/roasters/"+actor+"/"+rkey+"/backlinks")
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	var view backlinksJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &view))
	assert.Equal(t, "roaster", view.EntityNoun)
	assert.Equal(t, "Backlinks JSON Roaster", view.EntityName)
	assert.NotEmpty(t, view.BackURL)
	assert.Contains(t, view.BackURL, "/roasters/"+actor+"/"+rkey)
	assert.NotEmpty(t, view.DetailURL)
	assert.NotNil(t, view.Result, "result should be present (even if empty backlinks)")
}

// TestHTTP_BacklinksJSONBean verifies the bean backlinks endpoint includes
// usage data when a brew references the bean.
func TestHTTP_BacklinksJSONBean(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Bean BL Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form("name", "Bean BL Bean", "roaster_rkey", roasterRKey)), "bean")
	grinderRKey := mustRKey(t, h.PostForm("/api/grinders", form("name", "Bean BL Grinder")), "grinder")
	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "Bean BL Brewer")), "brewer")

	brewForm := form(
		"bean_rkey", beanRKey,
		"grinder_rkey", grinderRKey,
		"brewer_rkey", brewerRKey,
		"method", "Pour Over",
		"water_amount", "300",
		"coffee_amount", "18",
		"rating", "7",
	)
	brewResp := h.PostForm("/brews", brewForm)
	require.Equal(t, 200, brewResp.StatusCode, statusErr(brewResp, ReadBody(t, brewResp)))

	// Wait for the brew to be indexed so the bean backlinks includes usage.
	beanURI := atp.BuildATURI(h.PrimaryAccount.DID, "social.arabica.alpha.bean", beanRKey)
	h.WaitForRecord(beanURI, firehoseWait)

	actor := h.PrimaryAccount.DID
	resp := getJSON(t, h, "/api/beans/"+actor+"/"+beanRKey+"/backlinks")
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))

	var view backlinksJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &view))
	assert.Equal(t, "bean", view.EntityNoun)
	assert.Equal(t, "Bean BL Bean", view.EntityName)

	// The result should be a backlinks.Result object with the expected fields.
	var result struct {
		LibraryEntries []json.RawMessage `json:"LibraryEntries"`
		LibraryCount   int               `json:"LibraryCount"`
		Usage          []json.RawMessage `json:"Usage"`
		UsageCount     int               `json:"UsageCount"`
		RatingAverage  float64           `json:"RatingAverage"`
		RatingCount    int               `json:"RatingCount"`
	}
	require.NoError(t, json.Unmarshal(view.Result, &result))
	assert.NotNil(t, result.LibraryEntries)
	assert.NotNil(t, result.Usage)
}

// TestHTTP_BacklinksJSONNotFound verifies that a non-existent record returns 404.
func TestHTTP_BacklinksJSONNotFound(t *testing.T) {
	h := StartHarness(t, nil)

	actor := h.PrimaryAccount.DID
	resp := getJSON(t, h, "/api/roasters/"+actor+"/nonexistentrkey/backlinks")
	assert.Equal(t, 404, resp.StatusCode)
}

// TestHTTP_BacklinksJSONHTMXStillHTML verifies that an HTMX request still
// gets the HTML backlinks page (not JSON).
func TestHTTP_BacklinksJSONHTMXStillHTML(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "HTMX BL Roaster")), "roaster")
	roasterURI := atp.BuildATURI(h.PrimaryAccount.DID, "social.arabica.alpha.roaster", rkey)
	h.WaitForRecord(roasterURI, firehoseWait)

	actor := h.PrimaryAccount.DID
	resp := h.GetHTMX("/roasters/" + actor + "/" + rkey + "/backlinks")
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.NotEqual(t, "application/json", resp.Header.Get("Content-Type"))
	assert.NotEmpty(t, body)
}
