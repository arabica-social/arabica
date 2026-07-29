//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"tangled.org/arabica.social/arabica/internal/arabica/entities"

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

	var created arabica.Roaster
	require.NoError(t, json.Unmarshal([]byte(body), &created))
	assert.Equal(t, "Counter Culture", created.Name)
	assert.Equal(t, "Durham, NC", created.Location)
	assert.NotEmpty(t, created.RKey)
}

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

func TestHTTP_RoasterCreateValidationError(t *testing.T) {
	h := StartHarness(t, nil)

	form := url.Values{}
	form.Set("name", "") // empty: should fail validation

	resp := h.PostForm("/api/roasters", form)
	body := ReadBody(t, resp)

	assert.Equal(t, 400, resp.StatusCode, statusErr(resp, body))
}

// TestHTTP_RoasterUpdateFlow exercises PUT /api/roasters/{id}: create a
// roaster, update it, then verify the change round-tripped through the PDS by
// listing all data via /api/data.
func TestHTTP_RoasterUpdateFlow(t *testing.T) {
	h := StartHarness(t, nil)

	createForm := url.Values{}
	createForm.Set("name", "Sey Coffee")
	createForm.Set("location", "Brooklyn, NY")
	createResp := h.PostForm("/api/roasters", createForm)
	createBody := ReadBody(t, createResp)
	require.Equal(t, 200, createResp.StatusCode, statusErr(createResp, createBody))

	var created arabica.Roaster
	require.NoError(t, json.Unmarshal([]byte(createBody), &created))
	require.NotEmpty(t, created.RKey)

	updateForm := url.Values{}
	updateForm.Set("name", "Sey Coffee Roasters")
	updateForm.Set("location", "Brooklyn, NY")
	updateForm.Set("website", "https://seycoffee.com")
	updateResp := h.PutForm("/api/roasters/"+created.RKey, updateForm)
	updateBody := ReadBody(t, updateResp)
	require.Equal(t, 200, updateResp.StatusCode, statusErr(updateResp, updateBody))

	listResp := h.Get("/api/data")
	listBody := ReadBody(t, listResp)
	require.Equal(t, 200, listResp.StatusCode, statusErr(listResp, listBody))

	var data struct {
		Roasters []arabica.Roaster `json:"roasters"`
	}
	require.NoError(t, json.Unmarshal([]byte(listBody), &data))

	var found *arabica.Roaster
	for i := range data.Roasters {
		if data.Roasters[i].RKey == created.RKey {
			found = &data.Roasters[i]
			break
		}
	}
	require.NotNil(t, found, "updated roaster not found in list")
	assert.Equal(t, "Sey Coffee Roasters", found.Name)
	assert.Equal(t, "https://seycoffee.com", found.Website)
}

// TestHTTP_RoasterDeleteFlow exercises DELETE /api/roasters/{id}: create a
// roaster, delete it, then verify it's gone from /api/data.
func TestHTTP_RoasterDeleteFlow(t *testing.T) {
	h := StartHarness(t, nil)

	createForm := url.Values{}
	createForm.Set("name", "Heart Coffee")
	createResp := h.PostForm("/api/roasters", createForm)
	createBody := ReadBody(t, createResp)
	require.Equal(t, 200, createResp.StatusCode, statusErr(createResp, createBody))

	var created arabica.Roaster
	require.NoError(t, json.Unmarshal([]byte(createBody), &created))
	require.NotEmpty(t, created.RKey)

	delResp := h.Delete("/api/roasters/" + created.RKey)
	delBody := ReadBody(t, delResp)
	require.Equal(t, 200, delResp.StatusCode, statusErr(delResp, delBody))

	listResp := h.Get("/api/data")
	listBody := ReadBody(t, listResp)
	require.Equal(t, 200, listResp.StatusCode, statusErr(listResp, listBody))

	var data struct {
		Roasters []arabica.Roaster `json:"roasters"`
	}
	require.NoError(t, json.Unmarshal([]byte(listBody), &data))
	for _, r := range data.Roasters {
		assert.NotEqual(t, created.RKey, r.RKey, "roaster still present after delete")
	}
}

func TestHTTP_RoasterJSONLifecycle(t *testing.T) {
	h := StartHarness(t, nil)

	createResp := h.PostJSON("/api/roasters", map[string]string{
		"name": "JSON Lifecycle Roaster", "location": "Portland, OR", "website": "https://example.com",
	})
	createBody := ReadBody(t, createResp)
	require.Equal(t, http.StatusOK, createResp.StatusCode, statusErr(createResp, createBody))
	assert.Equal(t, "application/json; charset=utf-8", createResp.Header.Get("Content-Type"))
	var created arabica.Roaster
	require.NoError(t, json.Unmarshal([]byte(createBody), &created))
	require.NotEmpty(t, created.RKey)

	readResp := h.Get("/api/roasters/" + h.PrimaryAccount.DID + "/" + created.RKey)
	readBody := ReadBody(t, readResp)
	require.Equal(t, http.StatusOK, readResp.StatusCode, statusErr(readResp, readBody))
	assert.Equal(t, "application/json; charset=utf-8", readResp.Header.Get("Content-Type"))
	assert.Contains(t, readBody, "JSON Lifecycle Roaster")

	updateResp := h.PutJSON("/api/roasters/"+created.RKey, map[string]string{
		"name": "JSON Lifecycle Roaster", "location": "Seattle, WA", "website": "https://updated.example",
	})
	updateBody := ReadBody(t, updateResp)
	require.Equal(t, http.StatusOK, updateResp.StatusCode, statusErr(updateResp, updateBody))
	assert.Equal(t, "application/json; charset=utf-8", updateResp.Header.Get("Content-Type"))
	var updated arabica.Roaster
	require.NoError(t, json.Unmarshal([]byte(updateBody), &updated))
	assert.Equal(t, "Seattle, WA", updated.Location)
	assert.Equal(t, "https://updated.example", updated.Website)

	persisted := h.PDSGetRecord(h.PrimaryAccount, arabica.NSIDRoaster, created.RKey)
	assert.Equal(t, "Seattle, WA", persisted["location"])
	assert.Equal(t, "https://updated.example", persisted["website"])

	deleteResp := h.DeleteJSON("/api/roasters/" + created.RKey)
	deleteBody := ReadBody(t, deleteResp)
	require.Equal(t, http.StatusOK, deleteResp.StatusCode, statusErr(deleteResp, deleteBody))
	assert.Equal(t, "application/json; charset=utf-8", deleteResp.Header.Get("Content-Type"))
	assert.JSONEq(t, `{"deleted":true}`, deleteBody)

	validationResp := h.PostJSON("/api/roasters", map[string]string{"name": ""})
	validationBody := ReadBody(t, validationResp)
	require.Equal(t, http.StatusBadRequest, validationResp.StatusCode, statusErr(validationResp, validationBody))
	assert.Equal(t, "application/json; charset=utf-8", validationResp.Header.Get("Content-Type"))
	assert.JSONEq(t, `{"error":"name is required","code":"validation_failed"}`, validationBody)
}

// TestHTTP_BeanCreateLinksToRoaster exercises a multi-step flow: create a
// roaster, then create a bean referencing it. Verifies the cross-entity
// reference round-trips through the handler layer.
func TestHTTP_BeanCreateLinksToRoaster(t *testing.T) {
	h := StartHarness(t, nil)

	roasterForm := url.Values{}
	roasterForm.Set("name", "Onyx Coffee Lab")
	roasterResp := h.PostForm("/api/roasters", roasterForm)
	roasterBody := ReadBody(t, roasterResp)
	require.Equal(t, 200, roasterResp.StatusCode, statusErr(roasterResp, roasterBody))

	var roaster arabica.Roaster
	require.NoError(t, json.Unmarshal([]byte(roasterBody), &roaster))
	require.NotEmpty(t, roaster.RKey)

	beanForm := url.Values{}
	beanForm.Set("name", "Geometry")
	beanForm.Set("roaster_rkey", roaster.RKey)
	beanForm.Set("roast_level", "Medium")

	beanResp := h.PostForm("/api/beans", beanForm)
	beanBody := ReadBody(t, beanResp)
	require.Equal(t, 200, beanResp.StatusCode, statusErr(beanResp, beanBody))

	var bean arabica.Bean
	require.NoError(t, json.Unmarshal([]byte(beanBody), &bean))
	assert.Equal(t, "Geometry", bean.Name)
	assert.Equal(t, roaster.RKey, bean.RoasterRKey)
}
