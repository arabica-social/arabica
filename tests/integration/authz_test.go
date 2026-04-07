package integration

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// authzCase describes one entity's mutating-endpoint surface plus how to find
// the test's record in /api/data after the cross-user attempts.
type authzCase struct {
	name string

	// HTTP surface.
	createPath string // POST target
	mutatePath string // template like "/api/beans/%s" — also used for DELETE

	// Forms.
	createForm func(refs entityRefs) url.Values
	attackForm url.Values // what Bob will PUT

	// extract returns (name, location-or-extra, found) for the given rkey from
	// /api/data so the test can verify nothing changed. The "extra" string is
	// entity-specific (location for roaster, origin for bean, etc.) and lets
	// us assert two fields without writing one extractor per entity.
	extract func(data listAllResponse, rkey string) (name, extra string, found bool)
}

// entityRefs holds dependency rkeys created by the harness fixture so each
// case's createForm closure can reference them (e.g. bean needs a roaster).
type entityRefs struct {
	roasterRKey string
	brewerRKey  string
}

// TestHTTP_CrossUserMutationIsolation walks every mutating entity surface and
// verifies that Bob cannot affect Alice's records by guessing her rkeys.
//
// For each entity:
//  1. Alice creates a record (captures original name + secondary field).
//  2. Bob attempts PUT and DELETE on Alice's rkey.
//  3. Alice re-reads — name and secondary field must be unchanged and the
//     record must still exist.
//
// This is the catastrophic-class authz check, expanded from the roaster-only
// version to cover beans, grinders, brewers, recipes, and brews.
func TestHTTP_CrossUserMutationIsolation(t *testing.T) {
	cases := []authzCase{
		{
			name:       "roaster",
			createPath: "/api/roasters",
			mutatePath: "/api/roasters/%s",
			createForm: func(_ entityRefs) url.Values {
				return form("name", "Alice Roaster", "location", "Seattle")
			},
			attackForm: form("name", "PWNED", "location", "Hacker House"),
			extract: func(data listAllResponse, rkey string) (string, string, bool) {
				for _, r := range data.Roasters {
					if r.RKey == rkey {
						return r.Name, r.Location, true
					}
				}
				return "", "", false
			},
		},
		{
			name:       "bean",
			createPath: "/api/beans",
			mutatePath: "/api/beans/%s",
			createForm: func(refs entityRefs) url.Values {
				return form(
					"name", "Alice Bean",
					"origin", "Ethiopia",
					"roaster_rkey", refs.roasterRKey,
					"roast_level", "Light",
				)
			},
			attackForm: form("name", "PWNED", "origin", "Hacker Origin"),
			extract: func(data listAllResponse, rkey string) (string, string, bool) {
				for _, b := range data.Beans {
					if b.RKey == rkey {
						return b.Name, b.Origin, true
					}
				}
				return "", "", false
			},
		},
		{
			name:       "grinder",
			createPath: "/api/grinders",
			mutatePath: "/api/grinders/%s",
			createForm: func(_ entityRefs) url.Values {
				return form("name", "Alice Grinder", "grinder_type", "Manual")
			},
			attackForm: form("name", "PWNED", "grinder_type", "Hacker"),
			extract: func(data listAllResponse, rkey string) (string, string, bool) {
				for _, g := range data.Grinders {
					if g.RKey == rkey {
						return g.Name, g.GrinderType, true
					}
				}
				return "", "", false
			},
		},
		{
			name:       "brewer",
			createPath: "/api/brewers",
			mutatePath: "/api/brewers/%s",
			createForm: func(_ entityRefs) url.Values {
				return form("name", "Alice Brewer", "brewer_type", "Pour Over")
			},
			attackForm: form("name", "PWNED", "brewer_type", "Hacker"),
			extract: func(data listAllResponse, rkey string) (string, string, bool) {
				for _, b := range data.Brewers {
					if b.RKey == rkey {
						return b.Name, b.BrewerType, true
					}
				}
				return "", "", false
			},
		},
		{
			name:       "recipe",
			createPath: "/api/recipes",
			mutatePath: "/api/recipes/%s",
			createForm: func(refs entityRefs) url.Values {
				return form(
					"name", "Alice Recipe",
					"brewer_rkey", refs.brewerRKey,
					"brewer_type", "Pour Over",
					"coffee_amount", "18",
					"water_amount", "300",
					"notes", "original notes",
				)
			},
			attackForm: form(
				"name", "PWNED",
				"brewer_type", "Pour Over",
				"notes", "hacker notes",
			),
			extract: func(data listAllResponse, rkey string) (string, string, bool) {
				for _, r := range data.Recipes {
					if r.RKey == rkey {
						return r.Name, r.Notes, true
					}
				}
				return "", "", false
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := StartHarness(t, nil)

			// Alice's per-test fixture: a roaster + brewer the create forms can reference.
			refs := entityRefs{
				roasterRKey: mustRKey(t, h.PostForm("/api/roasters", form("name", "Refs Roaster")), "roaster"),
				brewerRKey:  mustRKey(t, h.PostForm("/api/brewers", form("name", "Refs Brewer", "brewer_type", "Pour Over")), "brewer"),
			}

			// Alice creates the entity under test.
			createResp := h.PostForm(tc.createPath, tc.createForm(refs))
			rkey := mustRKey(t, createResp, tc.name)

			// Capture the original (name, extra) for later comparison.
			origData := fetchData(t, h)
			origName, origExtra, ok := tc.extract(origData, rkey)
			require.True(t, ok, "%s not found right after create", tc.name)
			require.NotEmpty(t, origName)

			// Bob signs in.
			bob := h.CreateAccount("bob@test.com", "bob.test", "hunter2")
			bobClient := h.NewClientForAccount(bob)

			// Bob attempts PUT and DELETE while masquerading as the harness client.
			func() {
				restore := withClient(h, bobClient)
				defer restore()

				putResp := h.PutForm(fmt.Sprintf(tc.mutatePath, rkey), tc.attackForm)
				putBody := ReadBody(t, putResp)
				t.Logf("bob PUT %s: status=%d body=%s", tc.name, putResp.StatusCode, truncate(putBody, 200))

				delResp := h.Delete(fmt.Sprintf(tc.mutatePath, rkey))
				delBody := ReadBody(t, delResp)
				t.Logf("bob DELETE %s: status=%d body=%s", tc.name, delResp.StatusCode, truncate(delBody, 200))
			}()

			// Back as Alice — verify the record is intact and unchanged.
			//
			// Reads go through the session cache; Alice's session was never
			// invalidated by Bob's writes (those went to Bob's PDS, not
			// Alice's), so cached state would be stale-but-correct here. To be
			// extra safe and detect actual data mutation, we evict Alice's
			// session cache to force a witness/PDS re-read.
			h.InvalidateSessionCache(h.PrimaryAccount)

			data := fetchData(t, h)
			gotName, gotExtra, found := tc.extract(data, rkey)
			require.True(t, found, "Alice's %s must still exist after Bob's attempts", tc.name)
			assert.Equal(t, origName, gotName, "Alice's %s name must be unchanged", tc.name)
			assert.Equal(t, origExtra, gotExtra, "Alice's %s secondary field must be unchanged", tc.name)
		})
	}
}

// TestHTTP_CrossUserBrewIsolation is the brew-specific version of the authz
// matrix above. Brew uses different routes (/brews/{id}) and a much wider
// form, so it's split out rather than wedged into the table.
func TestHTTP_CrossUserBrewIsolation(t *testing.T) {
	h := StartHarness(t, nil)

	// Alice creates a brew + its dependencies.
	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Alice Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form(
		"name", "Alice Bean",
		"roaster_rkey", roasterRKey,
		"roast_level", "Medium",
	)), "bean")
	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "Alice V60", "brewer_type", "Pour Over")), "brewer")

	createForm := url.Values{}
	createForm.Set("bean_rkey", beanRKey)
	createForm.Set("brewer_rkey", brewerRKey)
	createForm.Set("method", "Pour Over")
	createForm.Set("water_amount", "300")
	createForm.Set("coffee_amount", "18")
	createForm.Set("rating", "8")
	createForm.Set("tasting_notes", "original notes")
	createResp := h.PostForm("/brews", createForm)
	require.Equal(t, 200, createResp.StatusCode, statusErr(createResp, ReadBody(t, createResp)))

	data := fetchData(t, h)
	require.Len(t, data.Brews, 1)
	brewRKey := data.Brews[0].RKey

	// Bob signs in and attacks.
	bob := h.CreateAccount("bob@test.com", "bob.test", "hunter2")
	bobClient := h.NewClientForAccount(bob)

	func() {
		restore := withClient(h, bobClient)
		defer restore()

		// Bob's PUT requires a valid bean_rkey from his own context. Use a fake
		// (well-formed) rkey — handler should reject because it doesn't exist
		// in Bob's PDS, and even if it doesn't, Alice's record must be safe.
		attack := url.Values{}
		attack.Set("bean_rkey", beanRKey) // Alice's rkey — handler treats it as Bob's
		attack.Set("brewer_rkey", brewerRKey)
		attack.Set("method", "PWNED METHOD")
		attack.Set("water_amount", "1")
		attack.Set("coffee_amount", "1")
		attack.Set("rating", "1")
		attack.Set("tasting_notes", "hacker notes")

		putResp := h.PutForm("/brews/"+brewRKey, attack)
		putBody := ReadBody(t, putResp)
		t.Logf("bob PUT brew: status=%d body=%s", putResp.StatusCode, truncate(putBody, 200))

		delResp := h.Delete("/brews/" + brewRKey)
		delBody := ReadBody(t, delResp)
		t.Logf("bob DELETE brew: status=%d body=%s", delResp.StatusCode, truncate(delBody, 200))
	}()

	// Back as Alice — verify her brew is intact.
	h.InvalidateSessionCache(h.PrimaryAccount)
	data = fetchData(t, h)
	require.Len(t, data.Brews, 1, "Alice's brew must still exist after Bob's attempts")
	brew := data.Brews[0]
	assert.Equal(t, brewRKey, brew.RKey)
	assert.Equal(t, "Pour Over", brew.Method, "method must not be overwritten")
	assert.Equal(t, "original notes", brew.TastingNotes, "tasting notes must not be overwritten")
	assert.Equal(t, 8, brew.Rating, "rating must not be overwritten")
	assert.Equal(t, 300, brew.WaterAmount, "water amount must not be overwritten")
}

// truncate returns s shortened to max chars with an ellipsis suffix when
// truncated. Used to keep test logs from drowning in HTML error pages.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
