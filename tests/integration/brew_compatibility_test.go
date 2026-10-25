//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/pdewey.com/atp"
)

func TestHTTP_BrewCreateRequiresBrewer(t *testing.T) {
	for _, transport := range []string{"form", "json"} {
		t.Run(transport, func(t *testing.T) {
			h := StartHarness(t, nil)
			beanRKey := mustRKey(t, h.PostForm("/api/beans", form("name", "Required brewer bean")), "bean")
			brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "Required brewer")), "brewer")

			for _, selection := range []struct {
				name  string
				value string
				omit  bool
			}{
				{name: "omitted", omit: true},
				{name: "empty"},
				{name: "invalid", value: "invalid/rkey"},
			} {
				t.Run(selection.name, func(t *testing.T) {
					var resp *http.Response
					if transport == "json" {
						body := map[string]any{"bean_rkey": beanRKey}
						if !selection.omit {
							body["brewer_rkey"] = selection.value
						}
						resp = h.PostJSON("/api/brews", body)
					} else {
						body := form("bean_rkey", beanRKey)
						if !selection.omit {
							body.Set("brewer_rkey", selection.value)
						}
						resp = postFormJSON(t, h, "/brews", body.Encode())
					}
					body := ReadBody(t, resp)
					require.Equal(t, http.StatusBadRequest, resp.StatusCode, body)
					assert.Contains(t, body, "Brewer selection")
					assert.Empty(t, h.PDSListRecords(h.PrimaryAccount, arabica.NSIDBrew), "rejected creates must not write PDS records")
				})
			}

			// New brews still need no rating (or method, grinder, or measurements).
			var resp *http.Response
			if transport == "json" {
				resp = h.PostJSON("/api/brews", map[string]any{"bean_rkey": beanRKey, "brewer_rkey": brewerRKey})
			} else {
				resp = postFormJSON(t, h, "/brews", form("bean_rkey", beanRKey, "brewer_rkey", brewerRKey).Encode())
			}
			body := ReadBody(t, resp)
			require.Equal(t, http.StatusOK, resp.StatusCode, body)
			var created struct {
				Brew arabica.Brew `json:"brew"`
			}
			require.NoError(t, json.Unmarshal([]byte(body), &created))
			assert.Equal(t, brewerRKey, created.Brew.BrewerRKey)
			assert.Zero(t, created.Brew.Rating)
			record := h.PDSGetRecord(h.PrimaryAccount, arabica.NSIDBrew, created.Brew.RKey)
			assert.NotContains(t, record, "rating")
			assert.NotContains(t, record, "method")
		})
	}
}

func TestHTTP_BrewLegacyUpdateRoundTrip(t *testing.T) {
	for _, transport := range []string{"form", "json"} {
		for _, legacy := range []struct {
			name         string
			grind        string
			explicitZero bool
		}{
			{name: "descriptive_grind_omitted_rating", grind: "Medium-fine (legacy)"},
			{name: "numeric_grind_zero_rating", grind: "18.5", explicitZero: true},
		} {
			t.Run(transport+"/"+legacy.name, func(t *testing.T) {
				h := StartHarness(t, &HarnessOptions{EnableFirehose: true})
				beanRKey := mustRKey(t, h.PostForm("/api/beans", form("name", "Legacy brew bean")), "bean")
				const rkey = "legacy-brew"
				// Seed an existing PDS record directly: old brews need not have a
				// brewer, and may have both method-parameter blocks and pours.
				record := map[string]any{
					"$type":          arabica.NSIDBrew,
					"beanRef":        atp.BuildATURI(h.PrimaryAccount.DID, arabica.NSIDBean, beanRKey),
					"createdAt":      "2025-01-02T03:04:05Z",
					"method":         "Legacy hybrid method",
					"grindSize":      legacy.grind,
					"rating":         8,
					"tastingNotes":   "Before edit",
					"espressoParams": map[string]any{"yieldWeight": 365, "pressure": 90, "preInfusionSeconds": 5},
					"pouroverParams": map[string]any{"bloomWater": 50, "bloomSeconds": 30, "drawdownSeconds": 45, "bypassWater": 20, "filter": "paper"},
					"pours":          []map[string]any{{"waterAmount": 50, "timeSeconds": 0}, {"waterAmount": 200, "timeSeconds": 30}},
				}
				uri, _, err := h.atpClients[h.PrimaryAccount.DID].PutRecord(context.Background(), arabica.NSIDBrew, rkey, record)
				require.NoError(t, err)
				h.WaitForRecord(uri, 5*time.Second)
				original := h.PDSGetRecord(h.PrimaryAccount, arabica.NSIDBrew, rkey)

				resp := getJSON(t, h, "/api/brews")
				body := ReadBody(t, resp)
				require.Equal(t, http.StatusOK, resp.StatusCode, body)
				var list struct {
					Brews []arabica.Brew `json:"brews"`
				}
				require.NoError(t, json.Unmarshal([]byte(body), &list))
				require.Len(t, list.Brews, 1)
				brew := list.Brews[0]
				assert.Empty(t, brew.BrewerRKey)
				assert.Equal(t, legacy.grind, brew.GrindSize)
				assert.Equal(t, "Legacy hybrid method", brew.Method)
				require.NotNil(t, brew.EspressoParams)
				require.NotNil(t, brew.PouroverParams)
				require.Len(t, brew.Pours, 2)

				if transport == "json" {
					update := map[string]any{
						"bean_rkey": brew.BeanRKey, "method": brew.Method, "grind_size": brew.GrindSize,
						"espresso_params": brew.EspressoParams, "pourover_params": brew.PouroverParams,
						"pours": []arabica.CreatePourData{
							{WaterAmount: brew.Pours[0].WaterAmount, TimeSeconds: brew.Pours[0].TimeSeconds},
							{WaterAmount: brew.Pours[1].WaterAmount, TimeSeconds: brew.Pours[1].TimeSeconds},
						},
						"tasting_notes": "After edit",
					}
					if legacy.explicitZero {
						update["rating"] = 0
					}
					resp = h.PutJSON("/api/brews/"+rkey, update)
				} else {
					update := form(
						"bean_rkey", brew.BeanRKey, "method", brew.Method, "grind_size", brew.GrindSize,
						"espresso_yield_weight", "36.5", "espresso_pressure", "9", "espresso_pre_infusion_seconds", "5",
						"pourover_bloom_water", "50", "pourover_bloom_seconds", "30", "pourover_drawdown_seconds", "45",
						"pourover_bypass_water", "20", "pourover_filter", "paper",
						"pour_water_0", "50", "pour_time_0", "0", "pour_water_1", "200", "pour_time_1", "30",
						"tasting_notes", "After edit",
					)
					if legacy.explicitZero {
						update.Set("rating", "0")
					} else {
						update.Set("rating", "")
					}
					resp = putFormJSON(t, h, "/brews/"+rkey, update.Encode())
				}
				body = ReadBody(t, resp)
				require.Equal(t, http.StatusOK, resp.StatusCode, body)
				var updated struct {
					Brew arabica.Brew `json:"brew"`
				}
				require.NoError(t, json.Unmarshal([]byte(body), &updated))
				assert.Empty(t, updated.Brew.BrewerRKey)
				assert.Zero(t, updated.Brew.Rating)
				assert.Equal(t, brew.Method, updated.Brew.Method)
				assert.Equal(t, brew.GrindSize, updated.Brew.GrindSize)
				assert.Equal(t, brew.EspressoParams, updated.Brew.EspressoParams)
				assert.Equal(t, brew.PouroverParams, updated.Brew.PouroverParams)

				// Compare the authoritative record, not just a cached response.
				delete(original, "rating")
				original["tastingNotes"] = "After edit"
				assert.Equal(t, original, h.PDSGetRecord(h.PrimaryAccount, arabica.NSIDBrew, rkey))
			})
		}
	}
}

func TestHTTP_BrewLongSteepRoundTrip(t *testing.T) {
	for _, transport := range []string{"form", "json"} {
		t.Run(transport, func(t *testing.T) {
			h := StartHarness(t, nil)
			bean := mustRKey(t, h.PostForm("/api/beans", form("name", "Cold brew bean")), "bean")
			brewer := mustRKey(t, h.PostForm("/api/brewers", form("name", "Cold brewer")), "brewer")
			var resp *http.Response
			if transport == "json" {
				resp = h.PostJSON("/api/brews", map[string]any{"bean_rkey": bean, "brewer_rkey": brewer, "time_seconds": 45000})
			} else {
				resp = postFormJSON(t, h, "/brews", form("bean_rkey", bean, "brewer_rkey", brewer, "time_seconds", "45000").Encode())
			}
			body := ReadBody(t, resp)
			require.Equal(t, http.StatusOK, resp.StatusCode, body)
			var result struct {
				Brew arabica.Brew `json:"brew"`
			}
			require.NoError(t, json.Unmarshal([]byte(body), &result))
			rkey := result.Brew.RKey
			require.NotEmpty(t, rkey)
			assert.Equal(t, 45000, result.Brew.TimeSeconds)
			original := h.PDSGetRecord(h.PrimaryAccount, arabica.NSIDBrew, rkey)
			assert.EqualValues(t, 45000, original["timeSeconds"])

			if transport == "json" {
				resp = h.PutJSON("/api/brews/"+rkey, map[string]any{"bean_rkey": bean, "brewer_rkey": brewer, "time_seconds": 45000, "tasting_notes": "After steep"})
			} else {
				resp = putFormJSON(t, h, "/brews/"+rkey, form("bean_rkey", bean, "brewer_rkey", brewer, "time_seconds", "45000", "tasting_notes", "After steep").Encode())
			}
			body = ReadBody(t, resp)
			require.Equal(t, http.StatusOK, resp.StatusCode, body)
			require.NoError(t, json.Unmarshal([]byte(body), &result))
			assert.Equal(t, 45000, result.Brew.TimeSeconds)
			persisted := h.PDSGetRecord(h.PrimaryAccount, arabica.NSIDBrew, rkey)
			assert.Equal(t, original["timeSeconds"], persisted["timeSeconds"])
			assert.Equal(t, "After steep", persisted["tastingNotes"])

			resp = getJSON(t, h, "/api/brews")
			body = ReadBody(t, resp)
			require.Equal(t, http.StatusOK, resp.StatusCode, body)
			var list struct {
				Brews []arabica.Brew `json:"brews"`
			}
			require.NoError(t, json.Unmarshal([]byte(body), &list))
			require.Len(t, list.Brews, 1)
			assert.Equal(t, rkey, list.Brews[0].RKey)
			assert.Equal(t, 45000, list.Brews[0].TimeSeconds)
		})
	}
}
