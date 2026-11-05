package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	provider "tangled.org/arabica.social/arabica/internal/ai"
	arabicaai "tangled.org/arabica.social/arabica/internal/arabica/ai"
	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/records"
)

// Exercise the real provider transport, authenticated store boundary, context
// projection and server-derived recipe proposal without external credentials.
func TestAIChatCatalogContextHTTPRoundTrip(t *testing.T) {
	const did = "did:plc:test123456789"
	ownRef := func(kind, key string) string { return "at://" + did + "/" + arabica.NSIDBase + "." + kind + "/" + key }
	for _, tc := range []struct {
		name, recipeID string
		status         int
		choiceOnly     bool
		bundleRefs     bool
	}{{"own recipe", "recipe-1", http.StatusOK, false, false}, {"own recipe emitted as redundant choice", "recipe-1", http.StatusOK, true, false}, {"known bundled bean grinder and recipe", "recipe-1", http.StatusOK, true, true}, {"foreign recipe alias", "recipe-foreign", http.StatusBadGateway, false, false}} {
		t.Run(tc.name, func(t *testing.T) {
			var calls, mutations atomic.Int32
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
					http.Error(w, "read", 500)
					return
				}
				for _, private := range []string{did, "private-open-bean", "private-closed-bean", "private-roaster", "private-brewer", "private-grinder", "private-recipe", "at://", "sourceRef", "roasterRef", "brewerRef", "did:plc:foreign"} {
					if strings.Contains(string(body), private) {
						t.Errorf("automatic private reference forwarded: %s", private)
					}
				}
				var envelope struct {
					Messages []provider.Message `json:"messages"`
				}
				if err := json.Unmarshal(body, &envelope); err != nil || len(envelope.Messages) != 2 {
					t.Errorf("invalid provider envelope: %v", err)
					http.Error(w, "decode", 500)
					return
				}
				var payload struct {
					Library []struct {
						ID           string         `json:"id"`
						Kind         string         `json:"kind"`
						Label        string         `json:"label"`
						Roaster      string         `json:"roaster"`
						Closed       *bool          `json:"closed"`
						Matches      int            `json:"open_roaster_matches"`
						Brewer       string         `json:"brewer"`
						Defaults     map[string]any `json:"defaults"`
						Instructions string         `json:"instructions"`
					} `json:"library"`
				}
				if err := json.Unmarshal([]byte(envelope.Messages[1].Content), &payload); err != nil {
					t.Error(err)
					http.Error(w, "payload", 500)
					return
				}
				seenOpen, seenClosed, seenRecipe := false, false, false
				for _, item := range payload.Library {
					switch item.ID {
					case "bean-1":
						seenOpen = true
						if item.Kind != "bean" || item.Roaster != "Recluse" || item.Closed == nil || *item.Closed || item.Matches != 1 {
							t.Errorf("open bean context: %+v", item)
						}
					case "bean-2":
						seenClosed = true
						if item.Closed == nil || !*item.Closed || item.Roaster != "Recluse" || item.Matches != 1 {
							t.Errorf("closed bean context: %+v", item)
						}
					case "recipe-1":
						seenRecipe = true
						if item.Kind != "recipe" || item.Brewer != "Orea" || item.Defaults["coffee_amount"] != float64(18) || item.Defaults["water_amount"] != float64(300) || item.Defaults["brewer_rkey"] != "brewer-1" || item.Instructions != "93°C, medium grind, three pours" {
							t.Errorf("recipe context: %+v", item)
						}
						pours, ok := item.Defaults["pours"].([]any)
						if !ok || len(pours) != 3 {
							t.Errorf("recipe pours %v", item.Defaults["pours"])
						} else if pours[0].(map[string]any)["water_amount"] != float64(100) || pours[0].(map[string]any)["time_seconds"] != float64(0) {
							t.Error("pour units changed")
						}
					}
				}
				if !seenOpen || !seenClosed || !seenRecipe {
					t.Errorf("missing context: open=%t closed=%t recipe=%t", seenOpen, seenClosed, seenRecipe)
				}
				output := `{"in_scope":true,"reply":"Using your saved recipe.","kind":"brew","patch":{"bean_rkey":"bean-1","recipe_rkey":"` + tc.recipeID + `"},"choices":[]}`
				if tc.choiceOnly {
					output = `{"in_scope":true,"reply":"Using your saved recipe.","kind":"brew","patch":{"bean_rkey":"bean-1"},"choices":[{"label":"Use Orea 3 pour","patch":{"recipe_rkey":"` + tc.recipeID + `"}}]}`
				}
				if tc.bundleRefs {
					output = `{"in_scope":true,"reply":"Using Ethiopia, Orea 3 pour and 1ZPresso ZP6.","kind":"brew","patch":{},"choices":[{"label":"Use named equipment","patch":{"bean_rkey":"bean-1","recipe_rkey":"recipe-1","grinder_rkey":"grinder-1"}}]}`
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": output}, "finish_reason": "stop"}}})
			}))
			defer upstream.Close()
			client, err := provider.NewClient(provider.Config{DefaultProvider: "test", Providers: map[string]provider.ProviderConfig{"test": {BaseURL: upstream.URL + "/v1", Model: "context-test"}}})
			if err != nil {
				t.Fatal(err)
			}
			h, ctx := setupAI(t)
			h.Service = arabicaai.NewService(client)
			ctx.MockStore.PutRecordFunc = func(context.Context, string, string, any) (string, string, error) {
				mutations.Add(1)
				return "", "", nil
			}
			ctx.MockStore.RemoveRecordFunc = func(context.Context, string, string) error { mutations.Add(1); return nil }
			ctx.MockStore.FetchAllRecordsFunc = func(_ context.Context, nsid string) ([]records.RawRecord, error) {
				row := func(kind, key, name string, extra map[string]any) records.RawRecord {
					record := map[string]any{"name": name}
					for k, v := range extra {
						record[k] = v
					}
					return records.RawRecord{RKey: key, URI: ownRef(kind, key), Record: record}
				}
				switch strings.TrimPrefix(nsid, arabica.NSIDBase+".") {
				case "roaster":
					return []records.RawRecord{row("roaster", "private-roaster", "Recluse", nil)}, nil
				case "bean":
					return []records.RawRecord{row("bean", "private-open-bean", "Ethiopia", map[string]any{"closed": false, "roasterRef": ownRef("roaster", "private-roaster")}), row("bean", "private-closed-bean", "Colombia", map[string]any{"closed": true, "roasterRef": ownRef("roaster", "private-roaster")})}, nil
				case "grinder":
					return []records.RawRecord{row("grinder", "private-grinder", "1ZPresso ZP6", nil)}, nil
				case "brewer":
					return []records.RawRecord{row("brewer", "private-brewer", "Orea", map[string]any{"brewerType": "pourover"})}, nil
				case "recipe":
					return []records.RawRecord{row("recipe", "private-recipe", "Orea 3 pour", map[string]any{"brewerRef": ownRef("brewer", "private-brewer"), "coffeeAmount": int64(180), "waterAmount": int64(3000), "notes": "93°C, medium grind, three pours", "sourceRef": "at://did:plc:foreign/" + arabica.NSIDBase + ".recipe/foreign-recipe", "pours": []any{map[string]any{"waterAmount": int64(100), "timeSeconds": int64(0)}, map[string]any{"waterAmount": int64(100), "timeSeconds": int64(30)}, map[string]any{"waterAmount": int64(100), "timeSeconds": int64(60)}}})}, nil
				}
				return nil, nil
			}
			request := strings.Replace(chatRequest, "18 grams coffee", "Brew Ethiopia from Recluse using Orea 3 pour", 1)
			if tc.bundleRefs {
				request = strings.Replace(request, "using Orea 3 pour", "using Orea 3 pour and 1ZPresso ZP6", 1)
			}
			recorder := httptest.NewRecorder()
			h.HandleChat(recorder, aiRequest(request))
			if recorder.Code != tc.status {
				t.Fatalf("status %d want %d: %s", recorder.Code, tc.status, recorder.Body.String())
			}
			if calls.Load() != 1 || mutations.Load() != 0 {
				t.Fatalf("provider calls=%d mutations=%d", calls.Load(), mutations.Load())
			}
			if tc.status != http.StatusOK {
				if strings.Contains(recorder.Body.String(), "private-recipe") {
					t.Error("foreign alias error exposed record identity")
				}
				return
			}
			var response arabicaai.Response
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if tc.bundleRefs && response.Patch["grinder_rkey"] != "private-grinder" {
				t.Fatalf("named grinder still requires a choice: %+v", response)
			}
			if response.Patch["bean_rkey"] != "private-open-bean" || response.Patch["recipe_rkey"] != "private-recipe" || response.Patch["brewer_rkey"] != "private-brewer" || response.Patch["coffee_amount"] != float64(18) || response.Patch["water_amount"] != float64(300) || response.BrewerType != "pourover" || len(response.Choices) != 0 {
				t.Fatalf("ungrounded response: %+v", response)
			}
			if pours, ok := response.Patch["pours"].([]any); !ok || len(pours) != 3 {
				t.Fatalf("missing pours: %v", response.Patch)
			}
			if _, ok := response.Patch["recipe_owner_did"]; ok {
				t.Error("automatic recipe owner disclosed")
			}
			if _, ok := response.Patch["tasting_notes"]; ok {
				t.Error("recipe instructions copied into tasting notes")
			}
		})
	}
}
