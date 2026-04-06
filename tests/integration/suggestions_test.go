package integration

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"arabica/internal/models"
	"arabica/internal/suggestions"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// suggestionResult mirrors suggestions.EntitySuggestion to keep this test
// independent of the internal package import path.
type suggestionResult struct {
	Name      string            `json:"name"`
	SourceURI string            `json:"source_uri"`
	Fields    map[string]string `json:"fields"`
	Count     int               `json:"count"`
}

// postRoasterAs is a helper that creates a roaster on behalf of the given client
// and returns the created entity. If sourceRef is non-empty, it sets the
// source_ref field (used to track community adoption).
func postRoasterAs(t *testing.T, h *Harness, client *http.Client, name, location, sourceRef string) models.Roaster {
	t.Helper()
	form := url.Values{}
	form.Set("name", name)
	if location != "" {
		form.Set("location", location)
	}
	if sourceRef != "" {
		form.Set("source_ref", sourceRef)
	}
	req, err := http.NewRequest("POST", h.URL("/api/roasters"), strings.NewReader(form.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	require.NoError(t, err)
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	var roaster models.Roaster
	require.NoError(t, json.Unmarshal([]byte(body), &roaster))
	return roaster
}

// fetchSuggestions hits GET /api/suggestions/{entity}?q=... as the given client.
func fetchSuggestions(t *testing.T, h *Harness, client *http.Client, entity, query string) []suggestionResult {
	t.Helper()
	req, err := http.NewRequest("GET", h.URL("/api/suggestions/"+entity+"?q="+url.QueryEscape(query)), nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	var results []suggestionResult
	require.NoError(t, json.Unmarshal([]byte(body), &results))
	return results
}

// roasterURI builds the AT-URI for a roaster owned by the given DID.
func roasterURI(did, rkey string) string {
	return "at://" + did + "/social.arabica.alpha.roaster/" + rkey
}

// TestHTTP_SuggestionScoring_PrefersReferenced verifies the sourceRef selection
// logic end-to-end: when a roaster has been "adopted" by other users (their
// records carry source_ref pointing back at the original), that original URI
// should win as the canonical sourceRef in the suggestions response.
//
// This is the original motivating scenario for the scoring work — exercises
// witness cache writes, the json_extract reference query, and the composite
// score function all together.
func TestHTTP_SuggestionScoring_PrefersReferenced(t *testing.T) {
	h := StartHarness(t, nil)

	// Four users total: alice/bob/carol contribute, dave queries.
	bob := h.CreateAccount("bob@test.com", "bob.test", "hunter2")
	carol := h.CreateAccount("carol@test.com", "carol.test", "hunter2")
	dave := h.CreateAccount("dave@test.com", "dave.test", "hunter2")

	aliceClient := h.Client // primary == alice
	bobClient := h.NewClientForAccount(bob)
	carolClient := h.NewClientForAccount(carol)
	daveClient := h.NewClientForAccount(dave)
	alice := h.PrimaryAccount

	// Alice creates the canonical "Counter Culture" roaster (no sourceRef).
	aliceRoaster := postRoasterAs(t, h, aliceClient, "Counter Culture", "Durham, NC", "")
	aliceURI := roasterURI(alice.DID, aliceRoaster.RKey)

	// Bob and Carol both adopt Alice's roaster — i.e. they create their own
	// records with source_ref pointing at her URI. Each adoption gives Alice
	// +2 points in the score function (refCount * 2).
	postRoasterAs(t, h, bobClient, "Counter Culture Coffee", "Durham, NC", aliceURI)
	postRoasterAs(t, h, carolClient, "Counter Culture", "Durham, NC", aliceURI)

	// Dave queries suggestions. The three contributing roasters dedupe into
	// one candidate, and Alice's URI should win because two records reference it.
	results := fetchSuggestions(t, h, daveClient, "roasters", "counter")
	require.NotEmpty(t, results, "expected at least one suggestion")

	var cc *suggestionResult
	for i := range results {
		if strings.Contains(strings.ToLower(results[i].Name), "counter culture") {
			cc = &results[i]
			break
		}
	}
	require.NotNil(t, cc, "expected a Counter Culture suggestion in results")

	assert.Equal(t, 3, cc.Count, "all three contributing users should be counted")
	assert.Equal(t, aliceURI, cc.SourceURI,
		"alice's roaster (referenced by 2 others) should win as the canonical sourceRef")
}

// TestHTTP_SuggestionScoring_PreferredDIDOverride verifies that a DID added to
// suggestions.PreferredDIDs wins over a record with more references.
func TestHTTP_SuggestionScoring_PreferredDIDOverride(t *testing.T) {
	h := StartHarness(t, nil)

	bob := h.CreateAccount("bob@test.com", "bob.test", "hunter2")
	carol := h.CreateAccount("carol@test.com", "carol.test", "hunter2")
	dave := h.CreateAccount("dave@test.com", "dave.test", "hunter2")

	aliceClient := h.Client
	bobClient := h.NewClientForAccount(bob)
	carolClient := h.NewClientForAccount(carol)
	daveClient := h.NewClientForAccount(dave)
	alice := h.PrimaryAccount

	// Alice creates the canonical roaster, Bob and Carol adopt it.
	aliceRoaster := postRoasterAs(t, h, aliceClient, "Counter Culture", "Durham, NC", "")
	aliceURI := roasterURI(alice.DID, aliceRoaster.RKey)
	bobRoaster := postRoasterAs(t, h, bobClient, "Counter Culture Coffee", "Durham, NC", aliceURI)
	postRoasterAs(t, h, carolClient, "Counter Culture", "Durham, NC", aliceURI)

	// Mark Bob as preferred. The +10 bonus should overcome Alice's +4 from refs.
	suggestions.PreferredDIDs[bob.DID] = struct{}{}
	t.Cleanup(func() { delete(suggestions.PreferredDIDs, bob.DID) })

	results := fetchSuggestions(t, h, daveClient, "roasters", "counter")
	var cc *suggestionResult
	for i := range results {
		if strings.Contains(strings.ToLower(results[i].Name), "counter culture") {
			cc = &results[i]
			break
		}
	}
	require.NotNil(t, cc)

	bobURI := roasterURI(bob.DID, bobRoaster.RKey)
	assert.Equal(t, bobURI, cc.SourceURI,
		"bob's roaster should win because his DID is in PreferredDIDs")
}

// TestHTTP_SuggestionDedupe verifies that when multiple users post a roaster
// with the same fuzzy-name, the suggestion endpoint dedupes them into a
// single result and counts all contributing DIDs.
func TestHTTP_SuggestionDedupe(t *testing.T) {
	h := StartHarness(t, nil)

	// Four users total: alice/bob/carol contribute roasters, dave queries.
	// (The suggestion handler excludes the requester's own records.)
	bob := h.CreateAccount("bob@test.com", "bob.test", "hunter2")
	carol := h.CreateAccount("carol@test.com", "carol.test", "hunter2")
	dave := h.CreateAccount("dave@test.com", "dave.test", "hunter2")

	aliceClient := h.Client // primary == alice
	bobClient := h.NewClientForAccount(bob)
	carolClient := h.NewClientForAccount(carol)
	daveClient := h.NewClientForAccount(dave)

	// All three create a roaster that should fuzzy-match into a single
	// dedupe group ("Counter Culture" / "Counter Culture Coffee").
	postRoasterAs(t, h, aliceClient, "Counter Culture", "Durham, NC", "")
	postRoasterAs(t, h, bobClient, "Counter Culture Coffee", "Durham, NC", "")
	postRoasterAs(t, h, carolClient, "Counter Culture", "Durham, NC", "")

	results := fetchSuggestions(t, h, daveClient, "roasters", "counter")
	require.NotEmpty(t, results, "expected at least one suggestion")

	// Find the Counter Culture entry.
	var cc *suggestionResult
	for i := range results {
		if strings.Contains(strings.ToLower(results[i].Name), "counter culture") {
			cc = &results[i]
			break
		}
	}
	require.NotNil(t, cc, "expected a Counter Culture suggestion in results")

	// All three contributing users should be counted in a single dedupe group.
	assert.Equal(t, 3, cc.Count, "all three contributing users should be counted")
}

// TestHTTP_SuggestionScoring_ExcludesRequester verifies that the suggestion
// handler hides the requesting user's own records (so users only see community
// data, not their own data echoed back).
func TestHTTP_SuggestionScoring_ExcludesRequester(t *testing.T) {
	h := StartHarness(t, nil)

	bob := h.CreateAccount("bob@test.com", "bob.test", "hunter2")
	bobClient := h.NewClientForAccount(bob)

	// Alice creates a roaster.
	postRoasterAs(t, h, h.Client, "Onyx Coffee Lab", "Rogers, AR", "")

	// Alice queries — should see nothing (her own roaster is excluded).
	aliceResults := fetchSuggestions(t, h, h.Client, "roasters", "onyx")
	assert.Empty(t, aliceResults, "querying user's own records should be excluded")

	// Bob queries — should see Alice's roaster.
	bobResults := fetchSuggestions(t, h, bobClient, "roasters", "onyx")
	require.Len(t, bobResults, 1)
	assert.Equal(t, "Onyx Coffee Lab", bobResults[0].Name)
}
