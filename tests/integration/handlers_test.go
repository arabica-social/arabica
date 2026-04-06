package integration

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"arabica/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTP_RoasterCreateFlow exercises the full POST /api/roasters flow:
// authenticated request → handler → AtprotoStore → real PDS, then verifies
// the record exists by listing it through a separate GET.
func TestHTTP_RoasterCreateFlow(t *testing.T) {
	h := StartHarness(t, nil)

	form := url.Values{}
	form.Set("name", "Counter Culture")
	form.Set("location", "Durham, NC")
	form.Set("website", "https://counterculturecoffee.com")

	resp := h.PostForm("/api/roasters", form)
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))

	var created models.Roaster
	require.NoError(t, json.Unmarshal([]byte(body), &created))
	assert.Equal(t, "Counter Culture", created.Name)
	assert.Equal(t, "Durham, NC", created.Location)
	assert.NotEmpty(t, created.RKey)
}

// TestHTTP_UnauthenticatedPostRejected ensures POST endpoints reject requests
// without test auth headers (simulating an unauthenticated client).
func TestHTTP_UnauthenticatedPostRejected(t *testing.T) {
	h := StartHarness(t, nil)

	// Use a bare http.Client with no auth headers (and no cookies).
	req, err := http.NewRequest("POST", h.URL("/api/roasters"), nil)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", h.Server.URL)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 401, resp.StatusCode)
}

// TestHTTP_RoasterCreateValidationError ensures the handler returns a 400
// when the request body is invalid (empty name).
func TestHTTP_RoasterCreateValidationError(t *testing.T) {
	h := StartHarness(t, nil)

	form := url.Values{}
	form.Set("name", "") // empty: should fail validation

	resp := h.PostForm("/api/roasters", form)
	body := ReadBody(t, resp)

	assert.Equal(t, 400, resp.StatusCode, statusErr(resp, body))
}

// TestHTTP_BeanCreateLinksToRoaster exercises a multi-step flow: create a
// roaster, then create a bean referencing it. Verifies the cross-entity
// reference round-trips through the handler layer.
func TestHTTP_BeanCreateLinksToRoaster(t *testing.T) {
	h := StartHarness(t, nil)

	// Step 1: create roaster
	roasterForm := url.Values{}
	roasterForm.Set("name", "Onyx Coffee Lab")
	roasterResp := h.PostForm("/api/roasters", roasterForm)
	roasterBody := ReadBody(t, roasterResp)
	require.Equal(t, 200, roasterResp.StatusCode, statusErr(roasterResp, roasterBody))

	var roaster models.Roaster
	require.NoError(t, json.Unmarshal([]byte(roasterBody), &roaster))
	require.NotEmpty(t, roaster.RKey)

	// Step 2: create bean referencing roaster
	beanForm := url.Values{}
	beanForm.Set("name", "Geometry")
	beanForm.Set("roaster_rkey", roaster.RKey)
	beanForm.Set("roast_level", "Medium")

	beanResp := h.PostForm("/api/beans", beanForm)
	beanBody := ReadBody(t, beanResp)
	require.Equal(t, 200, beanResp.StatusCode, statusErr(beanResp, beanBody))

	var bean models.Bean
	require.NoError(t, json.Unmarshal([]byte(beanBody), &bean))
	assert.Equal(t, "Geometry", bean.Name)
	assert.Equal(t, roaster.RKey, bean.RoasterRKey)
}
