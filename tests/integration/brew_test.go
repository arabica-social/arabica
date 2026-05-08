package integration

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"tangled.org/arabica.social/arabica/internal/entities/arabica"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// form is a small ergonomic helper that returns url.Values from an alternating
// key/value list. Panics on odd-length input — only used in tests.
func form(kv ...string) url.Values {
	if len(kv)%2 != 0 {
		panic("form: odd number of arguments")
	}
	v := url.Values{}
	for i := 0; i < len(kv); i += 2 {
		v.Set(kv[i], kv[i+1])
	}
	return v
}

// mustRKey decodes a JSON entity-create response and returns its rkey, failing
// the test if the request did not succeed or the response is unparseable.
func mustRKey(t *testing.T, resp *http.Response, label string) string {
	t.Helper()
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, "%s create: %s", label, statusErr(resp, body))
	var generic struct {
		RKey string `json:"rkey"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &generic))
	require.NotEmpty(t, generic.RKey, "%s create: empty rkey in %s", label, body)
	return generic.RKey
}

// TestHTTP_BrewCreatePourover exercises the most complex record marshaling
// path: a pour-over brew with bean/grinder/brewer references, multiple pours,
// and pourover params. After creation, /api/data is fetched and the brew is
// inspected to confirm pours and pourover params round-tripped through the
// PDS write -> witness cache read path.
func TestHTTP_BrewCreatePourover(t *testing.T) {
	h := StartHarness(t, nil)

	// Set up the entities the brew references.
	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Brew Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans",
		form("name", "Brew Bean", "roaster_rkey", roasterRKey)), "bean")
	grinderRKey := mustRKey(t, h.PostForm("/api/grinders", form("name", "Brew Grinder")), "grinder")
	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "V60")), "brewer")

	brewForm := url.Values{}
	brewForm.Set("bean_rkey", beanRKey)
	brewForm.Set("grinder_rkey", grinderRKey)
	brewForm.Set("brewer_rkey", brewerRKey)
	brewForm.Set("method", "Pour Over")
	brewForm.Set("temperature", "94")
	brewForm.Set("water_amount", "300")
	brewForm.Set("coffee_amount", "18")
	brewForm.Set("time_seconds", "210")
	brewForm.Set("rating", "8")
	brewForm.Set("grind_size", "Medium")
	brewForm.Set("tasting_notes", "bright, floral")

	// Three pours.
	brewForm.Set("pour_water_0", "60")
	brewForm.Set("pour_time_0", "0")
	brewForm.Set("pour_water_1", "120")
	brewForm.Set("pour_time_1", "45")
	brewForm.Set("pour_water_2", "120")
	brewForm.Set("pour_time_2", "90")

	// Pourover params.
	brewForm.Set("pourover_bloom_water", "60")
	brewForm.Set("pourover_bloom_seconds", "30")
	brewForm.Set("pourover_drawdown_seconds", "45")
	brewForm.Set("pourover_filter", "Hario tabbed")

	resp := h.PostForm("/brews", brewForm)
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "/my-coffee", resp.Header.Get("HX-Redirect"))

	// Verify the brew round-tripped by listing all data.
	listResp := h.Get("/api/data")
	listBody := ReadBody(t, listResp)
	require.Equal(t, 200, listResp.StatusCode, statusErr(listResp, listBody))

	var data struct {
		Brews []arabica.Brew `json:"brews"`
	}
	require.NoError(t, json.Unmarshal([]byte(listBody), &data))
	require.Len(t, data.Brews, 1, "expected exactly one brew")

	brew := data.Brews[0]
	assert.Equal(t, beanRKey, brew.BeanRKey)
	assert.Equal(t, grinderRKey, brew.GrinderRKey)
	assert.Equal(t, brewerRKey, brew.BrewerRKey)
	assert.Equal(t, "Pour Over", brew.Method)
	assert.Equal(t, 18, brew.CoffeeAmount)
	assert.Equal(t, 300, brew.WaterAmount)
	assert.Equal(t, 8, brew.Rating)
	assert.Len(t, brew.Pours, 3, "expected three pours to round-trip")

	require.NotNil(t, brew.PouroverParams, "pourover params should be present")
	assert.Equal(t, 60, brew.PouroverParams.BloomWater)
	assert.Equal(t, 30, brew.PouroverParams.BloomSeconds)
	assert.Equal(t, 45, brew.PouroverParams.DrawdownSeconds)
	assert.Equal(t, "Hario tabbed", brew.PouroverParams.Filter)
}
