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

// withClient swaps the harness client temporarily so calls run as a different
// account. Returns a restore function the test should defer.
func withClient(h *Harness, c *http.Client) func() {
	prev := h.Client
	h.Client = c
	return func() { h.Client = prev }
}

// TestHTTP_CrossUserView creates a roaster as Alice and renders the view page
// as Bob via ?owner=did:alice. Verifies the witness-cache-backed cross-user
// read path that single-user tests skip entirely.
func TestHTTP_CrossUserView(t *testing.T) {
	h := StartHarness(t, nil)

	createResp := h.PostForm("/api/roasters", form("name", "Alice Roaster", "location", "Seattle"))
	createBody := ReadBody(t, createResp)
	require.Equal(t, 200, createResp.StatusCode, statusErr(createResp, createBody))

	var roaster arabica.Roaster
	require.NoError(t, json.Unmarshal([]byte(createBody), &roaster))
	require.NotEmpty(t, roaster.RKey)

	bob := h.CreateAccount("bob@test.com", "bob.test", "hunter2")
	bobClient := h.NewClientForAccount(bob)
	defer withClient(h, bobClient)()

	viewURL := "/api/roasters/" + url.PathEscape(h.PrimaryAccount.DID) + "/" + roaster.RKey
	resp := getJSON(t, h, viewURL)
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	var view struct {
		Record arabica.Roaster `json:"record"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &view))
	assert.Equal(t, "Alice Roaster", view.Record.Name,
		"Bob should see Alice's roaster name in the view payload")
}

// TestHTTP_CrossUserDeleteIsolation verifies that DELETE /api/roasters/{rkey}
// only operates on the calling user's PDS — Bob attempting to delete a record
// owned by Alice cannot affect Alice's data.
//
// This is the catastrophic-class authz check: if it ever regresses, one user
// can wipe another user's records by guessing their rkeys.
func TestHTTP_CrossUserDeleteIsolation(t *testing.T) {
	h := StartHarness(t, nil)

	createResp := h.PostForm("/api/roasters", form("name", "Alice Owned"))
	createBody := ReadBody(t, createResp)
	require.Equal(t, 200, createResp.StatusCode, statusErr(createResp, createBody))

	var alicesRoaster arabica.Roaster
	require.NoError(t, json.Unmarshal([]byte(createBody), &alicesRoaster))

	bob := h.CreateAccount("bob@test.com", "bob.test", "hunter2")
	bobClient := h.NewClientForAccount(bob)

	bobAttempt := func() *http.Response {
		restore := withClient(h, bobClient)
		defer restore()
		return h.Delete("/api/roasters/" + alicesRoaster.RKey)
	}()
	bobBody := ReadBody(t, bobAttempt)
	// We don't pin the exact response — the PDS may return success-as-noop or
	// a not-found error. What matters is that Alice's data survives.
	t.Logf("bob delete attempt: status=%d body=%s", bobAttempt.StatusCode, bobBody)

	data := fetchData(t, h)
	var found bool
	for _, r := range data.Roasters {
		if r.RKey == alicesRoaster.RKey {
			found = true
			assert.Equal(t, "Alice Owned", r.Name)
		}
	}
	assert.True(t, found, "Alice's roaster must still exist after Bob's delete attempt")
}

// TestHTTP_CrossUserUpdateIsolation verifies the PUT path is similarly
// isolated: Bob attempting to update a record at Alice's rkey cannot mutate
// Alice's record.
func TestHTTP_CrossUserUpdateIsolation(t *testing.T) {
	h := StartHarness(t, nil)

	createResp := h.PostForm("/api/roasters", form(
		"name", "Untouchable", "location", "Original Location",
	))
	createBody := ReadBody(t, createResp)
	require.Equal(t, 200, createResp.StatusCode, statusErr(createResp, createBody))

	var alicesRoaster arabica.Roaster
	require.NoError(t, json.Unmarshal([]byte(createBody), &alicesRoaster))

	bob := h.CreateAccount("bob@test.com", "bob.test", "hunter2")
	bobClient := h.NewClientForAccount(bob)

	bobAttempt := func() *http.Response {
		restore := withClient(h, bobClient)
		defer restore()
		return h.PutForm("/api/roasters/"+alicesRoaster.RKey, form(
			"name", "PWNED", "location", "Hacker House",
		))
	}()
	bobBody := ReadBody(t, bobAttempt)
	t.Logf("bob update attempt: status=%d body=%s", bobAttempt.StatusCode, bobBody)

	data := fetchData(t, h)
	var found *arabica.Roaster
	for i := range data.Roasters {
		if data.Roasters[i].RKey == alicesRoaster.RKey {
			found = &data.Roasters[i]
		}
	}
	require.NotNil(t, found, "Alice's roaster missing after Bob's update attempt")
	assert.Equal(t, "Untouchable", found.Name, "Alice's name must not be overwritten")
	assert.Equal(t, "Original Location", found.Location, "Alice's location must not be overwritten")
}
