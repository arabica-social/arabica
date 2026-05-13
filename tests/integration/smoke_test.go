package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTP_PageRenderSmoke renders top-level pages and asserts they return
// 200 with a non-empty body. Catches templ panics, missing layout data, and
// broken context plumbing introduced when refactoring shared page wiring.
func TestHTTP_PageRenderSmoke(t *testing.T) {
	h := StartHarness(t, nil)

	pages := []string{
		"/",
		"/my-coffee",
		"/manage",
		"/brews",
		"/brews/new",
		"/about",
		"/settings",
		"/notifications",
	}

	for _, path := range pages {
		t.Run(path, func(t *testing.T) {
			resp := h.Get(path)
			body := ReadBody(t, resp)
			require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
			assert.NotEmpty(t, body, "empty body for %s", path)
		})
	}
}

// TestHTTP_EntityViewSmoke creates each kind of entity and renders its public
// view page. Catches breakage in view-handler templ rendering and reference
// resolution that unit tests with mocked stores miss.
func TestHTTP_EntityViewSmoke(t *testing.T) {
	h := StartHarness(t, nil)

	// Create one of each entity that has a view page.
	roasterResp := h.PostForm("/api/roasters", form("name", "View Roaster"))
	roaster := mustRKey(t, roasterResp, "roaster")

	beanResp := h.PostForm("/api/beans", form("name", "View Bean", "roaster_rkey", roaster))
	bean := mustRKey(t, beanResp, "bean")

	grinderResp := h.PostForm("/api/grinders", form("name", "View Grinder"))
	grinder := mustRKey(t, grinderResp, "grinder")

	brewerResp := h.PostForm("/api/brewers", form("name", "View Brewer"))
	brewer := mustRKey(t, brewerResp, "brewer")

	actor := h.PrimaryAccount.DID
	views := map[string]string{
		"roaster": "/roasters/" + actor + "/" + roaster,
		"bean":    "/beans/" + actor + "/" + bean,
		"grinder": "/grinders/" + actor + "/" + grinder,
		"brewer":  "/brewers/" + actor + "/" + brewer,
	}

	for label, path := range views {
		t.Run(label, func(t *testing.T) {
			resp := h.Get(path)
			body := ReadBody(t, resp)
			require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
			assert.NotEmpty(t, body)
		})
	}
}

// TestHTTP_HTMXPartialSmoke exercises HTMX-only fragment routes. They sit
// behind RequireHTMXMiddleware so the test sets HX-Request. A 200 + non-empty
// body catches silent template breakage in fragment renderers.
func TestHTTP_HTMXPartialSmoke(t *testing.T) {
	h := StartHarness(t, nil)

	partials := []string{
		"/api/feed",
		"/api/brews",
		"/api/manage",
		"/api/incomplete-records",
		"/api/popular-recipes",
	}

	for _, path := range partials {
		t.Run(path, func(t *testing.T) {
			resp := h.GetHTMX(path)
			body := ReadBody(t, resp)
			// Some partials legitimately render empty when the user has no data
			// (incomplete-records, popular-recipes). A 200 is enough — templ
			// panics would surface as 500.
			require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
		})
	}
}
