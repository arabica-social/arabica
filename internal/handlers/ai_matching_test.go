package handlers

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestAIChatNonExactMatchingHTTP(t *testing.T) {
	const did = "did:plc:test123456789"
	for _, tc := range []struct {
		name, message, status                           string
		openCount                                       int
		equipment, choiceOnly, selected, emptySelection bool
		wantBean                                        string
		wantChoices                                     int
	}{
		{name: "short names and historical duplicate", message: "Brew Mango on V60 with ZP6, 15g to 250g", openCount: 1, equipment: true, wantBean: "private-mango"},
		{name: "short names inside a combined choice", message: "Brew Mango on V60 with ZP6, 15g to 250g", openCount: 1, equipment: true, choiceOnly: true, wantBean: "private-mango"},
		{name: "two plausible open beans", message: "Brew Mango on V60 with ZP6", openCount: 2, equipment: true, wantChoices: 2},
		{name: "processing qualifier disambiguates shorthand", message: "Brew Mango Natural", openCount: 2, wantBean: "private-mango"},
		{name: "sole open bag defaults in a new brew", message: "Draft a brew, 15g to 250g", openCount: 1, wantBean: "private-mango"},
		{name: "empty-string reference is still unselected", message: "Draft a brew, 15g to 250g", openCount: 1, emptySelection: true, wantBean: "private-mango"},
		{name: "unknown status is not open", message: "Draft a brew with Mango", status: "unknown", wantChoices: 1},
		{name: "closed shorthand is not an open default", message: "Draft a brew with Mango", status: "closed", wantChoices: 1},
		{name: "unrelated followup keeps selected bean", message: "Change water to 260g", openCount: 1, selected: true, wantBean: "private-selected", wantChoices: -1},
		{name: "ambiguity is not hidden by the provider cap", message: "Brew with Mango", openCount: 102, wantChoices: 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls, mutations atomic.Int32
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				var request struct {
					Messages []provider.Message `json:"messages"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil || len(request.Messages) != 2 {
					t.Errorf("invalid provider request: %v", err)
					http.Error(w, "bad request", 500)
					return
				}
				for _, text := range []string{did, "private-mango", "private-selected", "private-v60", "private-zp6", "at://"} {
					if strings.Contains(request.Messages[1].Content, text) {
						t.Errorf("private identity forwarded: %s", text)
					}
				}
				if strings.Contains(request.Messages[0].Content, "Only put changed references in patch if a unique exact library label") {
					t.Error("provider still instructed to reject all non-exact matches")
				}
				var payload struct {
					Library []struct{ ID, Kind, Label string } `json:"library"`
				}
				if err := json.Unmarshal([]byte(request.Messages[1].Content), &payload); err != nil {
					t.Error(err)
					http.Error(w, "bad payload", 500)
					return
				}
				ids := map[string]string{}
				beans := 0
				for _, item := range payload.Library {
					if item.Kind == "bean" {
						beans++
					}
					key := item.Kind + ":" + item.Label
					if ids[key] == "" {
						ids[key] = item.ID
					}
				}
				if beans > 100 {
					t.Errorf("provider bean projection unbounded: %d", beans)
				}
				refs := map[string]any{"bean_rkey": ids["bean:El Mango Natural"]}
				if tc.equipment {
					refs["brewer_rkey"] = ids["brewer:Hario V60"]
					refs["grinder_rkey"] = ids["grinder:1ZPresso ZP6"]
				}
				patch := map[string]any{"coffee_amount": 15, "water_amount": 250}
				choices := []any{}
				if tc.choiceOnly {
					choices = append(choices, map[string]any{"label": "Use your equipment", "patch": refs})
				} else {
					for key, value := range refs {
						patch[key] = value
					}
				}
				output, err := json.Marshal(map[string]any{"in_scope": true, "reply": "Review your brew.", "kind": "brew", "patch": patch, "choices": choices})
				if err != nil {
					t.Error(err)
					http.Error(w, "bad fixture", 500)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": string(output)}, "finish_reason": "stop"}}})
			}))
			defer upstream.Close()
			client, err := provider.NewClient(provider.Config{DefaultProvider: "test", Providers: map[string]provider.ProviderConfig{"test": {BaseURL: upstream.URL + "/v1", Model: "matching-test"}}})
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
			row := func(kind, key, name string, extra map[string]any) records.RawRecord {
				data := map[string]any{"name": name}
				for key, value := range extra {
					data[key] = value
				}
				return records.RawRecord{RKey: key, URI: "at://" + did + "/" + arabica.NSIDBase + "." + kind + "/" + key, Record: data}
			}
			ctx.MockStore.FetchAllRecordsFunc = func(_ context.Context, nsid string) ([]records.RawRecord, error) {
				switch strings.TrimPrefix(nsid, arabica.NSIDBase+".") {
				case "bean":
					status := any(false)
					if tc.status == "closed" {
						status = true
					} else if tc.status == "unknown" {
						status = "malformed"
					}
					rows := []records.RawRecord{row("bean", "private-mango", "El Mango Natural", map[string]any{"closed": status})}
					if tc.openCount > 0 {
						rows = append(rows, row("bean", "private-historical", "El Mango Natural", map[string]any{"closed": true}))
					}
					for i := 1; i < tc.openCount; i++ {
						name := fmt.Sprintf("El Mango lot %d", i)
						if tc.openCount == 2 {
							name = "El Mango Washed"
						}
						rows = append(rows, row("bean", fmt.Sprintf("private-mango-%d", i), name, map[string]any{"closed": false}))
					}
					if tc.selected {
						rows = append(rows, row("bean", "private-selected", "Previous Colombia", map[string]any{"closed": true}))
					}
					return rows, nil
				case "brewer":
					return []records.RawRecord{row("brewer", "private-v60", "Hario V60", map[string]any{"brewerType": "pourover"})}, nil
				case "grinder":
					return []records.RawRecord{row("grinder", "private-zp6", "1ZPresso ZP6", nil)}, nil
				}
				return nil, nil
			}
			var body map[string]any
			_ = json.Unmarshal([]byte(chatRequest), &body)
			body["messages"] = []any{map[string]any{"role": "user", "content": tc.message}}
			current := map[string]any{}
			if tc.emptySelection {
				current["bean_rkey"] = ""
			}
			if tc.selected {
				current["bean_rkey"] = "private-selected"
			}
			body["draft"] = map[string]any{"kind": "brew", "fields": current}
			raw, _ := json.Marshal(body)
			recorder := httptest.NewRecorder()
			h.HandleChat(recorder, aiRequest(string(raw)))
			if recorder.Code != http.StatusOK {
				t.Fatalf("status %d: %s", recorder.Code, recorder.Body.String())
			}
			var response arabicaai.Response
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			effective := current["bean_rkey"]
			if value, ok := response.Patch["bean_rkey"]; ok {
				effective = value
			}
			if tc.wantBean != "" {
				if effective != tc.wantBean {
					t.Errorf("effective bean=%v want %s; response=%+v", effective, tc.wantBean, response)
				}
			} else if effective != nil {
				t.Errorf("uncertain bean filled automatically: %+v", response)
			}
			if tc.wantChoices >= 0 && len(response.Choices) != tc.wantChoices {
				t.Errorf("choices=%d want %d: %+v", len(response.Choices), tc.wantChoices, response)
			}
			if tc.equipment && (response.Patch["grinder_rkey"] != "private-zp6" || response.Patch["brewer_rkey"] != "private-v60" || response.BrewerType != "pourover") {
				t.Errorf("unambiguous shorthand equipment not filled: %+v", response)
			}
			if tc.wantChoices > 1 {
				seen := map[any]bool{}
				for _, choice := range response.Choices {
					key := choice.Patch["bean_rkey"]
					if key == nil || seen[key] {
						t.Errorf("not distinct bean alternatives: %+v", response.Choices)
					}
					seen[key] = true
				}
			}
			if calls.Load() != 1 || mutations.Load() != 0 {
				t.Errorf("provider calls=%d record mutations=%d", calls.Load(), mutations.Load())
			}
		})
	}
}
