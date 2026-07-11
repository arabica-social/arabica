//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTP_OolongAPIDataContract(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{App: "oolong"})

	resp := h.Get("/api/data")
	body := ReadBody(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode, statusErr(resp, body))
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

	var data struct {
		DID      string            `json:"did"`
		Teas     []json.RawMessage `json:"teas"`
		Vendors  []json.RawMessage `json:"vendors"`
		Vessels  []json.RawMessage `json:"vessels"`
		Infusers []json.RawMessage `json:"infusers"`
		Brews    []json.RawMessage `json:"brews"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &data))
	assert.Equal(t, h.PrimaryAccount.DID, data.DID)
	assert.NotNil(t, data.Teas)
	assert.NotNil(t, data.Vendors)
	assert.NotNil(t, data.Vessels)
	assert.NotNil(t, data.Infusers)
	assert.NotNil(t, data.Brews)
}

func TestHTTP_OolongVendorJSONViewContract(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{App: "oolong"})
	rkey := mustRKey(t, h.PostForm("/api/vendors", form(
		"name", "Oolong Vendor",
		"location", "Portland, OR",
	)), "vendor")

	resp := h.Get("/api/vendors/" + h.PrimaryAccount.DID + "/" + rkey)
	body := ReadBody(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode, statusErr(resp, body))
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

	var view struct {
		Record struct {
			RKey string `json:"rkey"`
			Name string `json:"name"`
		} `json:"record"`
		EntityType string `json:"entity_type"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &view))
	assert.Equal(t, rkey, view.Record.RKey)
	assert.Equal(t, "Oolong Vendor", view.Record.Name)
	assert.Equal(t, "vendor", view.EntityType)
}

func TestHTTP_OolongLegacyPagesRemainLegacyWhenSPAIsEnabled(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{App: "oolong", EnableSPA: true})

	resp := h.Get("/my-tea")
	body := ReadBody(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode, statusErr(resp, body))
	assert.Contains(t, body, `data-app="oolong"`)
	assert.NotContains(t, body, `data-frontend="sveltekit"`)
}
