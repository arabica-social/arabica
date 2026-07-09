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

// brewMutationJSONResponse mirrors the POST /brews and PUT /brews/{id} JSON
// envelope returned when Accept: application/json is sent.
type brewMutationJSONResponse struct {
	Brew struct {
		RKey        string `json:"rkey"`
		Method      string `json:"method"`
		BeanRKey    string `json:"bean_rkey"`
		WaterAmount int    `json:"water_amount"`
		Rating      int    `json:"rating"`
	} `json:"brew"`
	IncompleteNudge *struct {
		EntityType    string `json:"entity_type"`
		RKey          string `json:"rkey"`
		Name          string `json:"name"`
		MissingFields string `json:"missing"`
	} `json:"incomplete_nudge,omitempty"`
}

// postFormJSON posts a urlencoded form with Accept: application/json so the
// content-negotiating handlers return JSON instead of HX-Redirect.
func postFormJSON(t *testing.T, h *Harness, path string, form string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("POST", h.URL(path), strings.NewReader(form))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := h.Client.Do(req)
	require.NoError(t, err)
	return resp
}

// putFormJSON sends a urlencoded form via PUT with Accept: application/json.
func putFormJSON(t *testing.T, h *Harness, path string, form string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("PUT", h.URL(path), strings.NewReader(form))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := h.Client.Do(req)
	require.NoError(t, err)
	return resp
}

// TestHTTP_BrewCreateJSON verifies that POST /brews with Accept: application/json
// returns the created brew as JSON (not an HX-Redirect).
func TestHTTP_BrewCreateJSON(t *testing.T) {
	h := StartHarness(t, nil)

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "JSON Brew Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form("name", "JSON Brew Bean", "roaster_rkey", roasterRKey)), "bean")
	grinderRKey := mustRKey(t, h.PostForm("/api/grinders", form("name", "JSON Brew Grinder")), "grinder")
	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "JSON Brew Brewer")), "brewer")

	brewForm := form(
		"bean_rkey", beanRKey,
		"grinder_rkey", grinderRKey,
		"brewer_rkey", brewerRKey,
		"method", "Espresso",
		"water_amount", "36",
		"coffee_amount", "18",
		"rating", "9",
	)
	resp := postFormJSON(t, h, "/brews", brewForm.Encode())
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	// Must NOT have HX-Redirect in the JSON path.
	assert.Empty(t, resp.Header.Get("HX-Redirect"))

	var result brewMutationJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &result))
	assert.Equal(t, "Espresso", result.Brew.Method)
	assert.Equal(t, beanRKey, result.Brew.BeanRKey)
	assert.Equal(t, 36, result.Brew.WaterAmount)
	assert.Equal(t, 9, result.Brew.Rating)
	assert.NotEmpty(t, result.Brew.RKey, "created brew should have an rkey")
}

// TestHTTP_BrewUpdateJSON verifies that PUT /brews/{id} with Accept:
// application/json returns the updated brew as JSON.
func TestHTTP_BrewUpdateJSON(t *testing.T) {
	h := StartHarness(t, nil)

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Update JSON Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form("name", "Update JSON Bean", "roaster_rkey", roasterRKey)), "bean")
	grinderRKey := mustRKey(t, h.PostForm("/api/grinders", form("name", "Update JSON Grinder")), "grinder")
	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "Update JSON Brewer")), "brewer")

	// Create a brew first (via JSON path so we get the rkey back).
	createForm := form(
		"bean_rkey", beanRKey,
		"grinder_rkey", grinderRKey,
		"brewer_rkey", brewerRKey,
		"method", "Pour Over",
		"water_amount", "300",
		"coffee_amount", "18",
		"rating", "6",
	)
	createResp := postFormJSON(t, h, "/brews", createForm.Encode())
	createBody := ReadBody(t, createResp)
	require.Equal(t, 200, createResp.StatusCode, statusErr(createResp, createBody))
	var created brewMutationJSONResponse
	require.NoError(t, json.Unmarshal([]byte(createBody), &created))
	brewRKey := created.Brew.RKey
	require.NotEmpty(t, brewRKey)

	// Update via JSON path.
	updateForm := form(
		"bean_rkey", beanRKey,
		"grinder_rkey", grinderRKey,
		"brewer_rkey", brewerRKey,
		"method", "Pour Over",
		"water_amount", "350",
		"coffee_amount", "20",
		"rating", "8",
	)
	updateResp := putFormJSON(t, h, "/brews/"+brewRKey, updateForm.Encode())
	updateBody := ReadBody(t, updateResp)
	require.Equal(t, 200, updateResp.StatusCode, statusErr(updateResp, updateBody))
	assert.Equal(t, "application/json; charset=utf-8", updateResp.Header.Get("Content-Type"))
	assert.Empty(t, updateResp.Header.Get("HX-Redirect"))

	var updated brewMutationJSONResponse
	require.NoError(t, json.Unmarshal([]byte(updateBody), &updated))
	assert.Equal(t, brewRKey, updated.Brew.RKey)
	assert.Equal(t, 350, updated.Brew.WaterAmount)
	assert.Equal(t, 8, updated.Brew.Rating)
}

// TestHTTP_BrewCreateJSONIncompleteNudge verifies that when the referenced bean
// is incomplete (missing fields), the JSON response includes the nudge.
func TestHTTP_BrewCreateJSONIncompleteNudge(t *testing.T) {
	h := StartHarness(t, nil)

	// Create a bean with only a name — no origin, roast_level, etc. This
	// makes it "incomplete" per Bean.IsIncomplete().
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form("name", "Incomplete Bean")), "bean")
	grinderRKey := mustRKey(t, h.PostForm("/api/grinders", form("name", "Nudge Grinder")), "grinder")
	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "Nudge Brewer")), "brewer")

	brewForm := form(
		"bean_rkey", beanRKey,
		"grinder_rkey", grinderRKey,
		"brewer_rkey", brewerRKey,
		"method", "Pour Over",
		"water_amount", "250",
		"coffee_amount", "15",
		"rating", "5",
	)
	resp := postFormJSON(t, h, "/brews", brewForm.Encode())
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))

	var result brewMutationJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &result))
	require.NotNil(t, result.IncompleteNudge, "incomplete bean should trigger a nudge")
	assert.Equal(t, "bean", result.IncompleteNudge.EntityType)
	assert.Equal(t, beanRKey, result.IncompleteNudge.RKey)
	assert.Equal(t, "Incomplete Bean", result.IncompleteNudge.Name)
}

// TestHTTP_BrewCreateHTMXStillRedirects verifies that a form POST without
// Accept: application/json still gets the HX-Redirect (existing HTMX behavior).
func TestHTTP_BrewCreateHTMXStillRedirects(t *testing.T) {
	h := StartHarness(t, nil)

	beanRKey := mustRKey(t, h.PostForm("/api/beans", form("name", "HTMX Redirect Bean")), "bean")
	grinderRKey := mustRKey(t, h.PostForm("/api/grinders", form("name", "HTMX Grinder")), "grinder")
	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "HTMX Brewer")), "brewer")

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
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, ReadBody(t, resp)))
	assert.Equal(t, "/my-coffee", resp.Header.Get("HX-Redirect"))
}
