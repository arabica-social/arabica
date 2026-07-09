//go:build integration

package integration

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"tangled.org/pdewey.com/atp"
)

// entityViewJSONResponse mirrors the GET /api/{entity}/{actor}/{id} envelope.
type entityViewJSONResponse struct {
	Record          json.RawMessage `json:"record"`
	SubjectURI      string          `json:"subject_uri"`
	SubjectCID      string          `json:"subject_cid"`
	Author          json.RawMessage `json:"author"`
	Social          json.RawMessage `json:"social"`
	Backlinks       json.RawMessage `json:"backlinks"`
	IsOwnProfile    bool            `json:"is_own_profile"`
	IsAuthenticated bool            `json:"is_authenticated"`
	ShareURL        string          `json:"share_url"`
	EntityType      string          `json:"entity_type"`
	EntityCount     int             `json:"entity_count"`
}

// TestHTTP_EntityViewJSON verifies that GET /api/roasters/{actor}/{id} with
// Accept: application/json returns a JSON envelope with the record and social
// context, not an HTML page.
func TestHTTP_EntityViewJSON(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "JSON Roaster", "location", "Seattle, WA")), "roaster")

	// The entity view JSON route uses the owner's DID as the actor segment.
	actor := h.PrimaryAccount.DID
	resp := getJSON(t, h, "/api/roasters/"+actor+"/"+rkey)
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	var view entityViewJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &view))
	assert.Equal(t, "roaster", view.EntityType)
	assert.True(t, view.IsAuthenticated)
	assert.True(t, view.IsOwnProfile)
	assert.NotEmpty(t, view.SubjectURI)
	assert.NotEmpty(t, view.ShareURL)
	assert.Contains(t, view.ShareURL, "/roasters/"+actor+"/"+rkey)

	// The record field should contain the roaster name.
	var record map[string]any
	require.NoError(t, json.Unmarshal(view.Record, &record))
	assert.Equal(t, "JSON Roaster", record["name"])
	assert.Equal(t, "Seattle, WA", record["location"])
}

// TestHTTP_EntityViewJSONBean verifies the bean JSON view includes ref
// resolution (roaster hydration) and the entity count.
func TestHTTP_EntityViewJSONBean(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Bean JSON Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form("name", "Bean JSON Bean", "roaster_rkey", roasterRKey)), "bean")

	// Wait for indexing so backlinks/counts are available.
	roasterURI := atp.BuildATURI(h.PrimaryAccount.DID, "social.arabica.alpha.roaster", roasterRKey)
	h.WaitForRecord(roasterURI, firehoseWait)

	actor := h.PrimaryAccount.DID
	resp := getJSON(t, h, "/api/beans/"+actor+"/"+beanRKey)
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))

	var view entityViewJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &view))
	assert.Equal(t, "bean", view.EntityType)

	var bean map[string]any
	require.NoError(t, json.Unmarshal(view.Record, &bean))
	assert.Equal(t, "Bean JSON Bean", bean["name"])
	assert.Equal(t, roasterRKey, bean["roaster_rkey"])
}

// TestHTTP_EntityViewJSONBrew verifies the brew JSON view round-trips with
// all its references.
func TestHTTP_EntityViewJSONBrew(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Brew JSON Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form("name", "Brew JSON Bean", "roaster_rkey", roasterRKey)), "bean")
	grinderRKey := mustRKey(t, h.PostForm("/api/grinders", form("name", "Brew JSON Grinder")), "grinder")
	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "Brew JSON Brewer")), "brewer")

	brewForm := form(
		"bean_rkey", beanRKey,
		"grinder_rkey", grinderRKey,
		"brewer_rkey", brewerRKey,
		"method", "Pour Over",
		"water_amount", "300",
		"coffee_amount", "18",
		"rating", "7",
	)
	// Brew create returns HX-Redirect, not JSON — read the redirect then
	// fetch the brew list to get the rkey.
	brewResp := h.PostForm("/brews", brewForm)
	require.Equal(t, 200, brewResp.StatusCode, statusErr(brewResp, ReadBody(t, brewResp)))

	listResp := h.Get("/api/data")
	listBody := ReadBody(t, listResp)
	require.Equal(t, 200, listResp.StatusCode, statusErr(listResp, listBody))
	var data struct {
		Brews []struct {
			RKey string `json:"rkey"`
		} `json:"brews"`
	}
	require.NoError(t, json.Unmarshal([]byte(listBody), &data))
	require.Len(t, data.Brews, 1, "expected one brew")
	brewRKey := data.Brews[0].RKey
	require.NotEmpty(t, brewRKey)

	actor := h.PrimaryAccount.DID
	resp := getJSON(t, h, "/api/brews/"+actor+"/"+brewRKey)
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))

	var view entityViewJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &view))
	assert.Equal(t, "brew", view.EntityType)

	var brew map[string]any
	require.NoError(t, json.Unmarshal(view.Record, &brew))
	assert.Equal(t, "Pour Over", brew["method"])
	assert.Equal(t, beanRKey, brew["bean_rkey"])
}

// TestHTTP_EntityViewJSONNotFound verifies that a non-existent record returns 404.
func TestHTTP_EntityViewJSONNotFound(t *testing.T) {
	h := StartHarness(t, nil)

	actor := h.PrimaryAccount.DID
	resp := getJSON(t, h, "/api/roasters/"+actor+"/nonexistentrkey")
	body := ReadBody(t, resp)
	assert.Equal(t, 404, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	assert.JSONEq(t, `{"error":"Roaster not found","code":"not_found"}`, body)
}
