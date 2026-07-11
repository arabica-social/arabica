//go:build integration

package integration

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"tangled.org/pdewey.com/atp"
)

// profileJSONResponse mirrors the GET /api/profile/{actor} JSON envelope.
type profileJSONResponse struct {
	Profile            json.RawMessage    `json:"profile"`
	DID                string             `json:"did"`
	IsOwnProfile       bool               `json:"is_own_profile"`
	IsAuthenticated    bool               `json:"is_authenticated"`
	IsArabicaUser      bool               `json:"is_app_user"`
	Brews              []json.RawMessage  `json:"brews"`
	TotalBrews         int                `json:"total_brews"`
	BrewsHasMore       bool               `json:"brews_has_more"`
	BrewsNextOffset    int                `json:"brews_next_offset"`
	Beans              []json.RawMessage  `json:"beans"`
	Roasters           []json.RawMessage  `json:"roasters"`
	Grinders           []json.RawMessage  `json:"grinders"`
	Brewers            []json.RawMessage  `json:"brewers"`
	BrewLikeCounts     map[string]int     `json:"brew_like_counts"`
	BrewCommentCounts  map[string]int     `json:"brew_comment_counts"`
	BeanBrewCounts     map[string]int     `json:"bean_brew_counts"`
	RoasterBeanCounts  map[string]int     `json:"roaster_bean_counts"`
	BeanAvgBrewRatings map[string]float64 `json:"bean_avg_brew_ratings"`
}

// TestHTTP_ProfileJSON verifies that GET /api/profile/{actor} with Accept:
// application/json returns the full profile data bundle as JSON.
func TestHTTP_ProfileJSON(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	// Create some entities so the profile is non-empty.
	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Profile JSON Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form("name", "Profile JSON Bean", "roaster_rkey", roasterRKey)), "bean")
	grinderRKey := mustRKey(t, h.PostForm("/api/grinders", form("name", "Profile JSON Grinder")), "grinder")
	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "Profile JSON Brewer")), "brewer")

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

	// Wait for indexing so the profile is recognized as an arabica user.
	beanURI := atp.BuildATURI(h.PrimaryAccount.DID, "social.arabica.alpha.bean", beanRKey)
	h.WaitForRecord(beanURI, firehoseWait)

	// Fetch own profile via DID.
	actor := h.PrimaryAccount.DID
	resp := getJSON(t, h, "/api/profile/"+actor)
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	var profile profileJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &profile))
	assert.Equal(t, actor, profile.DID)
	assert.True(t, profile.IsOwnProfile)
	assert.True(t, profile.IsAuthenticated)
	assert.True(t, profile.IsArabicaUser)
	assert.NotEmpty(t, profile.Roasters, "should have at least one roaster")
	assert.NotEmpty(t, profile.Beans, "should have at least one bean")
	assert.GreaterOrEqual(t, profile.TotalBrews, 1)

	// Verify the profile object has a handle.
	var profileObj map[string]any
	require.NoError(t, json.Unmarshal(profile.Profile, &profileObj))
	assert.NotEmpty(t, profileObj["handle"])
}

// TestHTTP_ProfileJSONNotFound verifies that a non-existent user returns 404.
func TestHTTP_ProfileJSONNotFound(t *testing.T) {
	h := StartHarness(t, nil)

	resp := getJSON(t, h, "/api/profile/did:plc:nonexistentuser12345")
	assert.Equal(t, 404, resp.StatusCode)
}

// TestHTTP_ProfileJSONHTMXStillHTML verifies HTMX clients still get HTML.
func TestHTTP_ProfileJSONHTMXStillHTML(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	// Create a record so the user is an arabica user.
	mustRKey(t, h.PostForm("/api/roasters", form("name", "HTMX Profile Roaster")), "roaster")
	roasterURI := atp.BuildATURI(h.PrimaryAccount.DID, "social.arabica.alpha.roaster", "")
	_ = roasterURI

	actor := h.PrimaryAccount.DID
	resp := h.GetHTMX("/api/profile/" + actor)
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.NotEqual(t, "application/json", resp.Header.Get("Content-Type"))
}
