package integration

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTP_ValidationErrors covers the per-entity Validate() error matrix.
// Each case asserts the create handler returns 4xx for clearly invalid input.
// We don't pin the exact status (some validations land in the decode path,
// some in Validate(), some in field-specific guards) — only that the request
// is rejected and no record was persisted.
func TestHTTP_ValidationErrors(t *testing.T) {
	h := StartHarness(t, nil)

	// Build long strings once. Limits per entities/arabica/models.go: name=200,
	// location=200, website=500. Use 1000 to comfortably exceed any boundary.
	tooLong := strings.Repeat("a", 1000)

	cases := []struct {
		name     string
		path     string
		form     url.Values
		wantCode int
	}{
		// Roaster
		{
			name:     "roaster_empty_name",
			path:     "/api/roasters",
			form:     form("name", ""),
			wantCode: 400,
		},
		{
			name:     "roaster_name_too_long",
			path:     "/api/roasters",
			form:     form("name", tooLong),
			wantCode: 400,
		},
		{
			name:     "roaster_location_too_long",
			path:     "/api/roasters",
			form:     form("name", "OK", "location", tooLong),
			wantCode: 400,
		},
		{
			name:     "roaster_website_too_long",
			path:     "/api/roasters",
			form:     form("name", "OK", "website", tooLong),
			wantCode: 400,
		},

		// Bean
		{
			name:     "bean_empty_name",
			path:     "/api/beans",
			form:     form("name", "", "origin", "Ethiopia"),
			wantCode: 400,
		},
		{
			name:     "bean_name_too_long",
			path:     "/api/beans",
			form:     form("name", tooLong),
			wantCode: 400,
		},
		{
			name:     "bean_origin_too_long",
			path:     "/api/beans",
			form:     form("name", "OK", "origin", tooLong),
			wantCode: 400,
		},

		// Grinder
		{
			name:     "grinder_empty_name",
			path:     "/api/grinders",
			form:     form("name", ""),
			wantCode: 400,
		},
		{
			name:     "grinder_name_too_long",
			path:     "/api/grinders",
			form:     form("name", tooLong),
			wantCode: 400,
		},

		// Brewer
		{
			name:     "brewer_empty_name",
			path:     "/api/brewers",
			form:     form("name", ""),
			wantCode: 400,
		},
		{
			name:     "brewer_name_too_long",
			path:     "/api/brewers",
			form:     form("name", tooLong),
			wantCode: 400,
		},

		// Recipe
		{
			name:     "recipe_empty_name",
			path:     "/api/recipes",
			form:     form("name", ""),
			wantCode: 400,
		},
		{
			name:     "recipe_name_too_long",
			path:     "/api/recipes",
			form:     form("name", tooLong),
			wantCode: 400,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := h.PostForm(tc.path, tc.form)
			body := ReadBody(t, resp)
			assert.Equal(t, tc.wantCode, resp.StatusCode, statusErr(resp, body))
		})
	}

	// Sanity check: nothing was persisted by any failing case.
	data := fetchData(t, h)
	assert.Empty(t, data.Roasters, "no roasters should have been created by validation cases")
	assert.Empty(t, data.Beans, "no beans should have been created by validation cases")
	assert.Empty(t, data.Grinders, "no grinders should have been created by validation cases")
	assert.Empty(t, data.Brewers, "no brewers should have been created by validation cases")
	assert.Empty(t, data.Recipes, "no recipes should have been created by validation cases")
}

// TestHTTP_BrewValidationErrors covers brew-specific input rejection: missing
// bean reference, malformed rkey, and out-of-range numeric fields validated by
// validateBrewRequest.
func TestHTTP_BrewValidationErrors(t *testing.T) {
	h := StartHarness(t, nil)

	// Set up a valid bean so we can isolate other field failures.
	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Val Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form(
		"name", "Val Bean", "roaster_rkey", roasterRKey, "roast_level", "Medium",
	)), "bean")

	cases := []struct {
		name string
		form url.Values
	}{
		{
			name: "missing_bean_rkey",
			form: form("method", "Pour Over", "water_amount", "300"),
		},
		{
			name: "invalid_bean_rkey_format",
			form: form("bean_rkey", "not a valid rkey!"),
		},
		{
			name: "invalid_grinder_rkey_format",
			form: form("bean_rkey", beanRKey, "grinder_rkey", "bad rkey"),
		},
		{
			name: "temperature_out_of_range",
			form: form("bean_rkey", beanRKey, "temperature", "999"),
		},
		{
			name: "water_amount_out_of_range",
			form: form("bean_rkey", beanRKey, "water_amount", "999999"),
		},
		{
			name: "rating_out_of_range",
			form: form("bean_rkey", beanRKey, "rating", "42"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := h.PostForm("/brews", tc.form)
			body := ReadBody(t, resp)
			require.Equal(t, 400, resp.StatusCode, statusErr(resp, body))
		})
	}

	// Sanity: no brew was persisted.
	data := fetchData(t, h)
	assert.Empty(t, data.Brews, "no brew should have been created by failing validation cases")
}
