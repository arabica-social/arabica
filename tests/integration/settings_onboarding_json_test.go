//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// settingsJSONResponse mirrors the GET /api/settings JSON envelope.
type settingsJSONResponse struct {
	ProfileStatsVisibility struct {
		BeanAvgRating    string `json:"bean_avg_rating"`
		RoasterAvgRating string `json:"roaster_avg_rating"`
	} `json:"profile_stats_visibility"`
	UserPreferences struct {
		TemperatureUnit string `json:"temperature_unit"`
	} `json:"user_preferences"`
	BlueskyProfile struct {
		HasScopes      bool   `json:"has_scopes"`
		DisplayName    string `json:"display_name,omitempty"`
		AvatarURL      string `json:"avatar_url,omitempty"`
		LoadError      string `json:"load_error,omitempty"`
		NeedsAuthAgain bool   `json:"needs_auth_again,omitempty"`
	} `json:"bluesky_profile"`
}

// settingsSavedJSON mirrors the POST /api/settings/* JSON response.
type settingsSavedJSON struct {
	Saved bool `json:"saved"`
}

func TestHTTP_SettingsJSON(t *testing.T) {
	h := StartHarness(t, nil)

	resp := getJSON(t, h, "/api/settings")
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	var settings settingsJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &settings))
	// The harness feed index has no saved prefs, so defaults are returned.
	// Defaults are "public" for both visibility fields.
	assert.Equal(t, "public", settings.ProfileStatsVisibility.BeanAvgRating)
	assert.Equal(t, "public", settings.ProfileStatsVisibility.RoasterAvgRating)
	// Bluesky profile fields should be present (even if empty/false).
	assert.False(t, settings.BlueskyProfile.HasScopes)
}

func TestHTTP_SettingsJSONUnauth(t *testing.T) {
	h := StartHarness(t, nil)

	req, err := http.NewRequest("GET", h.URL("/api/settings"), nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	body := ReadBody(t, resp)
	assert.Equal(t, 401, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	assert.JSONEq(t, `{"error":"Authentication required","code":"authentication_required"}`, body)
}

func TestHTTP_SettingsPreferencesJSON(t *testing.T) {
	h := StartHarness(t, nil)

	formData := url.Values{}
	formData.Set("temperature_unit", "fahrenheit")
	req, err := http.NewRequest("POST", h.URL("/api/settings/preferences"), strings.NewReader(formData.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := h.Client.Do(req)
	require.NoError(t, err)
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	var result settingsSavedJSON
	require.NoError(t, json.Unmarshal([]byte(body), &result))
	assert.True(t, result.Saved)

	resp2 := getJSON(t, h, "/api/settings")
	body2 := ReadBody(t, resp2)
	require.Equal(t, 200, resp2.StatusCode, statusErr(resp2, body2))
	var settings settingsJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body2), &settings))
	assert.Equal(t, "fahrenheit", settings.UserPreferences.TemperatureUnit)
}

func TestHTTP_SettingsProfileVisibilityJSON(t *testing.T) {
	h := StartHarness(t, nil)

	formData := url.Values{}
	formData.Set("bean_avg_rating", "private")
	formData.Set("roaster_avg_rating", "public")
	req, err := http.NewRequest("POST", h.URL("/api/settings/profile-visibility"), strings.NewReader(formData.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := h.Client.Do(req)
	require.NoError(t, err)
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	var result settingsSavedJSON
	require.NoError(t, json.Unmarshal([]byte(body), &result))
	assert.True(t, result.Saved)

	resp2 := getJSON(t, h, "/api/settings")
	body2 := ReadBody(t, resp2)
	require.Equal(t, 200, resp2.StatusCode, statusErr(resp2, body2))
	var settings settingsJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body2), &settings))
	assert.Equal(t, "private", settings.ProfileStatsVisibility.BeanAvgRating)
	assert.Equal(t, "public", settings.ProfileStatsVisibility.RoasterAvgRating)
}

func TestHTTP_SettingsProfileVisibilityInvalidValueDefaults(t *testing.T) {
	h := StartHarness(t, nil)

	formData := url.Values{}
	formData.Set("bean_avg_rating", "bogus_value")
	formData.Set("roaster_avg_rating", "also_bogus")
	req, err := http.NewRequest("POST", h.URL("/api/settings/profile-visibility"), strings.NewReader(formData.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := h.Client.Do(req)
	require.NoError(t, err)
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))

	var result settingsSavedJSON
	require.NoError(t, json.Unmarshal([]byte(body), &result))
	assert.True(t, result.Saved)

	// Invalid values should have defaulted to public.
	resp2 := getJSON(t, h, "/api/settings")
	body2 := ReadBody(t, resp2)
	require.Equal(t, 200, resp2.StatusCode, statusErr(resp2, body2))
	var settings settingsJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body2), &settings))
	assert.Equal(t, "public", settings.ProfileStatsVisibility.BeanAvgRating)
	assert.Equal(t, "public", settings.ProfileStatsVisibility.RoasterAvgRating)
}

func TestHTTP_SettingsPreferencesHTMXStillHTML(t *testing.T) {
	h := StartHarness(t, nil)

	formData := url.Values{}
	formData.Set("temperature_unit", "celsius")
	resp := h.PostForm("/api/settings/preferences", formData)
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.NotEqual(t, "application/json", resp.Header.Get("Content-Type"))
	assert.Contains(t, body, "Saved")
}

// onboardingJSONResponse mirrors the GET /api/onboarding JSON envelope.
type onboardingJSONResponse struct {
	Readiness struct {
		HasBean    bool `json:"HasBean"`
		HasBrewer  bool `json:"HasBrewer"`
		HasRoaster bool `json:"HasRoaster"`
	} `json:"readiness"`
	Beans    []json.RawMessage `json:"beans"`
	Brewers  []json.RawMessage `json:"brewers"`
	Grinders []json.RawMessage `json:"grinders"`
	Roasters []json.RawMessage `json:"roasters"`
}

func TestHTTP_OnboardingJSON(t *testing.T) {
	h := StartHarness(t, nil)

	resp := getJSON(t, h, "/api/onboarding")
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	var onboarding onboardingJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &onboarding))
	// Fresh user has nothing — all readiness flags false.
	assert.False(t, onboarding.Readiness.HasBean)
	assert.False(t, onboarding.Readiness.HasBrewer)
	assert.False(t, onboarding.Readiness.HasRoaster)
	assert.NotNil(t, onboarding.Beans)
	assert.NotNil(t, onboarding.Brewers)
	assert.NotNil(t, onboarding.Grinders)
	assert.NotNil(t, onboarding.Roasters)
}

func TestHTTP_OnboardingJSONAfterCreate(t *testing.T) {
	h := StartHarness(t, nil)

	mustRKey(t, h.PostForm("/api/roasters", form("name", "Onboard Roaster")), "roaster")
	mustRKey(t, h.PostForm("/api/beans", form("name", "Onboard Bean")), "bean")
	mustRKey(t, h.PostForm("/api/brewers", form("name", "Onboard Brewer")), "brewer")

	resp := getJSON(t, h, "/api/onboarding")
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))

	var onboarding onboardingJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &onboarding))
	assert.True(t, onboarding.Readiness.HasBean, "should have a bean")
	assert.True(t, onboarding.Readiness.HasBrewer, "should have a brewer")
	assert.True(t, onboarding.Readiness.HasRoaster, "should have a roaster")
	assert.Len(t, onboarding.Beans, 1)
	assert.Len(t, onboarding.Brewers, 1)
	assert.Len(t, onboarding.Roasters, 1)
}

func TestHTTP_OnboardingJSONUnauth(t *testing.T) {
	h := StartHarness(t, nil)

	req, err := http.NewRequest("GET", h.URL("/api/onboarding"), nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 401, resp.StatusCode)
}

// incompleteRecordsJSONResponse mirrors the GET /api/incomplete-records
// JSON envelope.
type incompleteRecordsJSONResponse struct {
	Records []struct {
		EntityType    string   `json:"EntityType"`
		RKey          string   `json:"RKey"`
		Name          string   `json:"Name"`
		MissingFields []string `json:"MissingFields"`
	} `json:"records"`
}

func TestHTTP_IncompleteRecordsJSON(t *testing.T) {
	h := StartHarness(t, nil)

	mustRKey(t, h.PostForm("/api/beans", form("name", "Incomplete Bean")), "bean")

	resp := getJSON(t, h, "/api/incomplete-records")
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	var result incompleteRecordsJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &result))
	require.NotEmpty(t, result.Records, "should find at least one incomplete record")
	var foundBean bool
	for _, rec := range result.Records {
		if rec.EntityType == "bean" && rec.Name == "Incomplete Bean" {
			foundBean = true
			assert.NotEmpty(t, rec.RKey)
			assert.NotEmpty(t, rec.MissingFields)
		}
	}
	assert.True(t, foundBean, "incomplete bean should be in the list")
}

func TestHTTP_IncompleteRecordsJSONUnauth(t *testing.T) {
	h := StartHarness(t, nil)

	req, err := http.NewRequest("GET", h.URL("/api/incomplete-records"), nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 401, resp.StatusCode)
}

func TestHTTP_PopularRecipesJSON(t *testing.T) {
	h := StartHarness(t, nil)

	mustRKey(t, h.PostForm("/api/brewers", form("name", "Popular Recipe Brewer")), "brewer")
	mustRKey(t, h.PostForm("/api/recipes", form("name", "Popular Recipe", "brewer_rkey", "unused")), "recipe")

	resp := getJSON(t, h, "/api/popular-recipes")
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	// The response is a bare JSON array of Recipe objects (not an envelope).
	var recipes []map[string]any
	require.NoError(t, json.Unmarshal([]byte(body), &recipes))
	// Popular recipes may be empty if the recipe isn't indexed yet, but the
	// shape (array) must be correct.
	assert.NotNil(t, recipes)
}

func TestHTTP_PopularRecipesJSONUnauth(t *testing.T) {
	h := StartHarness(t, nil)

	req, err := http.NewRequest("GET", h.URL("/api/popular-recipes"), nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 401, resp.StatusCode)
}

// signupCategoriesJSONResponse mirrors the GET /api/signup/categories envelope.
type signupCategoriesJSONResponse struct {
	Categories []struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Providers   []struct {
			URL         string `json:"url"`
			Name        string `json:"name"`
			Domain      string `json:"domain"`
			Description string `json:"description"`
			Location    string `json:"location"`
			Badge       string `json:"badge"`
			BadgeColor  string `json:"badge_color"`
		} `json:"providers"`
		DevOnly bool `json:"dev_only"`
	} `json:"categories"`
}

func TestHTTP_SignupCategoriesJSON(t *testing.T) {
	h := StartHarness(t, nil)

	resp := getJSON(t, h, "/api/signup/categories")
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	var result signupCategoriesJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &result))
	require.NotEmpty(t, result.Categories, "should return at least one category")

	// Each category should have a non-empty title and a providers array.
	for _, cat := range result.Categories {
		assert.NotEmpty(t, cat.Title, "category should have a title")
		assert.NotNil(t, cat.Providers, "category should have a providers array")
	}
}

func TestHTTP_SignupCategoriesJSONUnauth(t *testing.T) {
	h := StartHarness(t, nil)

	req, err := http.NewRequest("GET", h.URL("/api/signup/categories"), nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	var result signupCategoriesJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &result))
	assert.NotEmpty(t, result.Categories)
}
