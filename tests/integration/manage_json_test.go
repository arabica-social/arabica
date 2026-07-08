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

// manageJSONResponse mirrors the GET /api/manage JSON envelope.
type manageJSONResponse struct {
	DID      string `json:"did"`
	Beans    []json.RawMessage `json:"beans"`
	Roasters []json.RawMessage `json:"roasters"`
	Grinders []json.RawMessage `json:"grinders"`
	Brewers  []json.RawMessage `json:"brewers"`
	Recipes  []json.RawMessage `json:"recipes"`
	Stats    struct {
		BeanBrewCounts        map[string]int     `json:"bean_brew_counts"`
		GrinderBrewCounts     map[string]int     `json:"grinder_brew_counts"`
		BrewerBrewCounts      map[string]int     `json:"brewer_brew_counts"`
		RoasterBeanCounts     map[string]int     `json:"roaster_bean_counts"`
		BeanAvgBrewRatings    map[string]float64 `json:"bean_avg_brew_ratings"`
		RoasterAvgBrewRatings map[string]float64 `json:"roaster_avg_brew_ratings"`
	} `json:"stats"`
}

// TestHTTP_ManageJSON verifies that GET /api/manage with Accept: application/json
// returns records + stats as JSON.
func TestHTTP_ManageJSON(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	// Create some entities so the response is non-empty.
	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Manage JSON Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form("name", "Manage JSON Bean", "roaster_rkey", roasterRKey)), "bean")
	grinderRKey := mustRKey(t, h.PostForm("/api/grinders", form("name", "Manage JSON Grinder")), "grinder")
	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "Manage JSON Brewer")), "brewer")

	resp := getJSON(t, h, "/api/manage")
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var manage manageJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &manage))
	assert.Equal(t, h.PrimaryAccount.DID, manage.DID)
	assert.NotEmpty(t, manage.Roasters, "should have at least one roaster")
	assert.NotEmpty(t, manage.Beans, "should have at least one bean")
	assert.NotEmpty(t, manage.Grinders, "should have at least one grinder")
	assert.NotEmpty(t, manage.Brewers, "should have at least one brewer")
	// Stats maps should be non-nil even if empty.
	assert.NotNil(t, manage.Stats.BeanBrewCounts)
	assert.NotNil(t, manage.Stats.RoasterBeanCounts)

	// Verify the roaster name round-trips.
	var firstRoaster map[string]any
	require.NoError(t, json.Unmarshal(manage.Roasters[0], &firstRoaster))
	assert.Equal(t, "Manage JSON Roaster", firstRoaster["name"])

	_ = beanRKey
	_ = grinderRKey
	_ = brewerRKey
}

// TestHTTP_ManageJSONStats verifies that brew counts and avg ratings appear in
// the stats when a brew references a bean.
func TestHTTP_ManageJSONStats(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Stats Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form("name", "Stats Bean", "roaster_rkey", roasterRKey)), "bean")
	grinderRKey := mustRKey(t, h.PostForm("/api/grinders", form("name", "Stats Grinder")), "grinder")
	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "Stats Brewer")), "brewer")

	// Create a brew referencing the bean so usage counts are populated.
	brewForm := form(
		"bean_rkey", beanRKey,
		"grinder_rkey", grinderRKey,
		"brewer_rkey", brewerRKey,
		"method", "Pour Over",
		"water_amount", "300",
		"coffee_amount", "18",
		"rating", "8",
	)
	brewResp := h.PostForm("/brews", brewForm)
	require.Equal(t, 200, brewResp.StatusCode, statusErr(brewResp, ReadBody(t, brewResp)))

	// Wait for the brew to be indexed.
	beanURI := atp.BuildATURI(h.PrimaryAccount.DID, "social.arabica.alpha.bean", beanRKey)
	h.WaitForRecord(beanURI, firehoseWait)

	resp := getJSON(t, h, "/api/manage")
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))

	var manage manageJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &manage))
	assert.Equal(t, 1, manage.Stats.BeanBrewCounts[beanURI], "bean should have 1 brew")
}

// TestHTTP_ManageJSONHTMXStillHTML verifies HTMX clients still get HTML.
func TestHTTP_ManageJSONHTMXStillHTML(t *testing.T) {
	h := StartHarness(t, nil)

	resp := h.GetHTMX("/api/manage")
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.NotEqual(t, "application/json", resp.Header.Get("Content-Type"))
}

// TestHTTP_ManageJSONUnauth verifies unauthenticated requests get 401.
func TestHTTP_ManageJSONUnauth(t *testing.T) {
	h := StartHarness(t, nil)

	req, err := http.NewRequest("GET", h.URL("/api/manage"), nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 401, resp.StatusCode)
}
