package integration

import (
	"encoding/json"
	"testing"

	"tangled.org/arabica.social/arabica/internal/entities/arabica"
	"tangled.org/pdewey.com/atp"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTP_WitnessCacheFallback verifies that reads succeed even when both
// cache layers (session cache + witness cache) are empty, by falling through
// to a real PDS XRPC call.
//
// This is the riskiest architectural piece in the codebase: if write-through
// ever drifts from PDS reads, or if the fallback path silently breaks, users
// would see "missing" data right after creating it. This test exercises:
//
//  1. Create a roaster (write-through to witness cache happens here).
//  2. Confirm a normal read returns it (witness-cache hit path).
//  3. Evict the witness cache entry + invalidate the session cache.
//  4. Read again — must still return the same data, this time via the
//     real-PDS fallback inside AtprotoStore.GetRoasterByRKey/ListRoasters.
func TestHTTP_WitnessCacheFallback(t *testing.T) {
	h := StartHarness(t, nil)

	// Step 1: create a roaster.
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

	// Step 2: read via /api/data — this should hit the witness cache (or
	// session cache, populated by ListRoasters).
	preData := fetchData(t, h)
	prePresent := containsRoaster(preData.Roasters, created.RKey)
	require.True(t, prePresent, "roaster should be readable immediately after create")

	// Step 3: evict the witness record and clear the session cache. After
	// this, both fast paths in AtprotoStore.ListRoasters will miss and the
	// store has to fall through to s.client.ListAllRecords (real PDS read).
	h.EvictWitnessRecord(h.PrimaryAccount, arabica.NSIDRoaster, created.RKey)
	h.InvalidateSessionCache(h.PrimaryAccount)

	// Sanity check: confirm the witness cache really is empty for that record.
	wr, _ := h.FeedIndex.GetWitnessRecord(t.Context(), atp.BuildATURI(h.PrimaryAccount.DID, arabica.NSIDRoaster, created.RKey))
	require.Nil(t, wr, "witness record should have been evicted")

	// Step 4: read again — must still return the roaster, this time via the
	// PDS fallback path.
	postData := fetchData(t, h)
	var found *arabica.Roaster
	for i := range postData.Roasters {
		if postData.Roasters[i].RKey == created.RKey {
			found = &postData.Roasters[i]
			break
		}
	}
	require.NotNil(t, found, "roaster must still be readable via PDS fallback after both caches are empty")

	// Field-level: the fallback path goes through a different decode path
	// (RecordToRoaster on a fresh PDS payload, not WitnessRecordToMap on
	// cached JSON). Verify the round-trip preserves all the fields we set.
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

	// The view page calls GetRoasterRecordByRKey via HandleRoasterView. With
	// the owner (the primary account's DID) in the path, the handler goes
	// through the public-client path and should hit the PDS fallback after
	// the witness cache eviction above.
	resp := h.Get("/roasters/" + h.PrimaryAccount.DID + "/" + created.RKey)
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.Contains(t, body, "Single-Get Fallback",
		"roaster name should appear on view page after PDS fallback read")
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
