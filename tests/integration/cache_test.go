//go:build integration

package integration

import (
	"encoding/json"
	"testing"

	"tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/pdewey.com/atp"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTP_WitnessCacheFallback verifies that reads fall through to the PDS
// after both cache layers are emptied.
func TestHTTP_WitnessCacheFallback(t *testing.T) {
	h := StartHarness(t, nil)

	createResp := h.PostForm("/api/roasters", form(
		"name", "Cache Fallback Roaster",
		"location", "Portland",
		"website", "https://example.com",
	))
	createBody := ReadBody(t, createResp)
	require.Equal(t, 200, createResp.StatusCode, statusErr(createResp, createBody))

	var created arabica.Roaster
	require.NoError(t, json.Unmarshal([]byte(createBody), &created))
	require.NotEmpty(t, created.RKey)

	preData := fetchData(t, h)
	prePresent := containsRoaster(preData.Roasters, created.RKey)
	require.True(t, prePresent, "roaster should be readable immediately after create")

	h.EvictWitnessRecord(h.PrimaryAccount, arabica.NSIDRoaster, created.RKey)
	h.InvalidateSessionCache(h.PrimaryAccount)

	wr, _ := h.FeedIndex.GetWitnessRecord(t.Context(), atp.BuildATURI(h.PrimaryAccount.DID, arabica.NSIDRoaster, created.RKey))
	require.Nil(t, wr, "witness record should have been evicted")

	postData := fetchData(t, h)
	var found *arabica.Roaster
	for i := range postData.Roasters {
		if postData.Roasters[i].RKey == created.RKey {
			found = &postData.Roasters[i]
			break
		}
	}
	require.NotNil(t, found, "roaster must still be readable via PDS fallback after both caches are empty")

	// The PDS fallback uses a different decode path than the witness cache.
	assert.Equal(t, "Cache Fallback Roaster", found.Name)
	assert.Equal(t, "Portland", found.Location)
	assert.Equal(t, "https://example.com", found.Website)
}

// TestHTTP_WitnessCacheGetByRKeyFallback covers the per-record (not list)
// fallback path: GetRoasterByRKey hits witness cache first, then falls back
// to a single PDS GetRecord call. This path is used by HandleRoasterView and
// other view handlers, so a regression here would surface as "404 not found"
// on detail pages right after creation.
func TestHTTP_WitnessCacheGetByRKeyFallback(t *testing.T) {
	h := StartHarness(t, nil)

	createResp := h.PostForm("/api/roasters", form("name", "Single-Get Fallback"))
	createBody := ReadBody(t, createResp)
	require.Equal(t, 200, createResp.StatusCode, statusErr(createResp, createBody))

	var created arabica.Roaster
	require.NoError(t, json.Unmarshal([]byte(createBody), &created))

	// Evict caches.
	h.EvictWitnessRecord(h.PrimaryAccount, arabica.NSIDRoaster, created.RKey)
	h.InvalidateSessionCache(h.PrimaryAccount)

	// The JSON view endpoint calls GetRoasterRecordByRKey via
	// HandleRoasterViewJSON. With the owner (the primary account's DID) in
	// the path, the handler goes through the public-client path and should
	// hit the PDS fallback after the witness cache eviction above.
	resp := getJSON(t, h, "/api/roasters/"+h.PrimaryAccount.DID+"/"+created.RKey)
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	var view struct {
		Record arabica.Roaster `json:"record"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &view))
	assert.Equal(t, "Single-Get Fallback", view.Record.Name,
		"roaster name should appear in view payload after PDS fallback read")
}

// containsRoaster reports whether a roaster with the given rkey exists in the
// slice. Small helper used by cache tests.
func containsRoaster(roasters []arabica.Roaster, rkey string) bool {
	for _, r := range roasters {
		if r.RKey == rkey {
			return true
		}
	}
	return false
}
