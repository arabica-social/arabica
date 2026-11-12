package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	provider "tangled.org/arabica.social/arabica/internal/ai"
	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/records"
)

type recentPayload struct {
	Recent  []recentBrew `json:"recent_brews"`
	Library catalog      `json:"library"`
}

func recentFixture() map[string][]records.RawRecord {
	rows := map[string][]records.RawRecord{
		"roaster": {contextRow("roast", "Neighbourhood", nil)},
		"brewer":  {contextRow("device", "Dripper", map[string]any{"brewerType": "pourover"})},
		"grinder": {contextRow("mill", "Hand mill", nil)},
		"brew": {
			contextRow("z-old", "", map[string]any{"createdAt": "2026-11-01T12:00:00Z"}),
			contextRow("a-new", "", map[string]any{"createdAt": "2026-11-04T00:30:00+02:00", "beanRef": contextRef("bean", "bag119"), "brewerRef": contextRef("brewer", "device"), "grinderRef": contextRef("grinder", "mill"), "coffeeAmount": 18, "waterAmount": 300, "temperature": 935, "timeSeconds": 185, "grindSize": "22 clicks", "rating": 9, "tastingNotes": "private old tasting results", "sourceRef": "at://did:plc:private/source/key", "espressoParams": map[string]any{"yieldWeight": 385, "pressure": 92, "preInfusionSeconds": 8}, "pouroverParams": map[string]any{"bloomWater": 60, "bloomSeconds": 30, "drawdownSeconds": 40, "bypassWater": 10, "filter": "paper"}, "pours": []any{map[string]any{"waterAmount": 100, "timeSeconds": 30}}}),
			contextRow("middle", "", map[string]any{"createdAt": "2026-11-03T22:00:00Z"}),
			contextRow("third", "", map[string]any{"createdAt": "2026-11-02T12:00:00Z"}),
			contextRow("unknown", "", nil),
			contextRow("bad-date", "", map[string]any{"createdAt": "tomorrow"}),
		},
	}
	for i := 0; i < 120; i++ {
		rows["bean"] = append(rows["bean"], contextRow(fmt.Sprintf("bag%d", i), fmt.Sprintf("Bag %d", i), map[string]any{"closed": true, "roasterRef": contextRef("roaster", "roast")}))
	}
	return rows
}

func inspectRecent(t *testing.T, m []provider.Message) recentPayload {
	t.Helper()
	var p recentPayload
	if err := json.Unmarshal([]byte(m[1].Content), &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestChatRecentBrewsAlwaysIncludedAndRepeat(t *testing.T) {
	req := validRequest()
	req.Messages[0].Content = "same as last time, but use 16 grams coffee"
	rows := recentFixture()
	service := NewService(fakeCompleter(func(ctx context.Context, m []provider.Message, tools []provider.Tool, format json.RawMessage) (provider.Completion, error) {
		p := inspectRecent(t, m)
		if len(p.Recent) != 3 || p.Recent[0].Date != "2026-11-04T00:30:00+02:00" || p.Recent[1].Date != "2026-11-03T22:00:00Z" || p.Recent[2].Date != "2026-11-02T12:00:00Z" {
			t.Fatalf("dates: %+v", p.Recent)
		}
		b := p.Recent[0]
		if b.ID != "recent-brew-1" || b.Labels["bean"] != "Bag 119" || b.Labels["roaster"] != "Neighbourhood" || b.Labels["brewer"] != "Dripper" || b.Labels["grinder"] != "Hand mill" {
			t.Fatalf("labels: %+v", b)
		}
		for key, want := range map[string]any{"coffee_amount": float64(18), "water_amount": float64(300), "temperature": 93.5, "time_seconds": float64(185), "espresso_params.yield_weight": 38.5, "espresso_params.pressure": 9.2, "espresso_params.pre_infusion_seconds": float64(8), "pourover_params.bloom_water": float64(60), "pourover_params.bloom_seconds": float64(30), "pourover_params.drawdown_seconds": float64(40), "pourover_params.bypass_water": float64(10), "pourover_params.filter": "paper", "grind_size": "22 clicks"} {
			if b.Fields[key] != want {
				t.Errorf("%s=%v want %v", key, b.Fields[key], want)
			}
		}
		if b.Fields["pours"].([]any)[0].(map[string]any)["water_amount"] != float64(100) {
			t.Fatal("pours scaled")
		}
		for _, private := range []string{ownDID, "at://", "bag119", "a-new", "private old tasting results", "sourceRef"} {
			if strings.Contains(m[1].Content, private) {
				t.Errorf("leaked %s", private)
			}
		}
		historyJSON, _ := json.Marshal(p.Recent)
		for _, key := range []string{"rating", "tasting", "source"} {
			if strings.Contains(string(historyJSON), key) {
				t.Errorf("history leaks %s", key)
			}
		}
		patch := map[string]any{"coffee_amount": 16, "bean_rkey": b.Fields["bean_rkey"], "brewer_rkey": b.Fields["brewer_rkey"], "grinder_rkey": b.Fields["grinder_rkey"], "water_amount": b.Fields["water_amount"]}
		body, _ := json.Marshal(map[string]any{"recent_brew_id": b.ID, "in_scope": true, "reply": "Same preparation, with 16 grams.", "kind": "brew", "patch": patch, "choices": []any{}})
		return outputClient(string(body))(ctx, m, tools, format)
	}))
	response, err := service.Chat(context.Background(), ownDID, contextLibrary(rows), req)
	if err != nil {
		t.Fatal(err)
	}
	if response.Patch["coffee_amount"] != float64(16) || response.Patch["bean_rkey"] != "bag119" || response.Patch["brewer_rkey"] != "device" || response.Patch["grinder_rkey"] != "mill" || len(response.Choices) != 0 {
		t.Fatalf("repeat: %+v", response)
	}
	if len(req.Draft.Fields) != 0 {
		t.Fatal("request mutated")
	}
}

func TestChatRecentBrewsDoNotAuthorizeUnrelatedOrManualReplacement(t *testing.T) {
	for _, mode := range []string{"unrelated-turn", "wrong-history-reference", "manual-selection"} {
		t.Run(mode, func(t *testing.T) {
			req := validRequest()
			req.Messages[0].Content = "same as last time"
			if mode == "unrelated-turn" {
				req.Messages[0].Content = "20 grams please"
			}
			if mode == "manual-selection" {
				req.Draft.Fields["bean_rkey"] = "bag0"
			}
			service := NewService(fakeCompleter(func(ctx context.Context, m []provider.Message, tools []provider.Tool, format json.RawMessage) (provider.Completion, error) {
				p := inspectRecent(t, m)
				if len(p.Recent) != 3 {
					t.Fatal("history not always sent")
				}
				id := p.Recent[0].Fields["bean_rkey"]
				if mode == "wrong-history-reference" {
					for _, item := range p.Library {
						if item.Kind == "bean" && item.Label != "Bag 119" {
							id = item.ID
						}
					}
				}
				sourceID := p.Recent[0].ID
				if mode == "unrelated-turn" {
					sourceID = ""
				}
				body, _ := json.Marshal(map[string]any{"recent_brew_id": sourceID, "in_scope": true, "reply": "A proposal.", "kind": "brew", "patch": map[string]any{"bean_rkey": id}, "choices": []any{}})
				return outputClient(string(body))(ctx, m, tools, format)
			}))
			response, err := service.Chat(context.Background(), ownDID, contextLibrary(recentFixture()), req)
			if err != nil {
				t.Fatal(err)
			}
			if _, exists := response.Patch["bean_rkey"]; exists || len(response.Choices) != 1 {
				t.Fatalf("unguarded reference: %+v", response)
			}
			if mode == "manual-selection" && req.Draft.Fields["bean_rkey"] != "bag0" {
				t.Fatal("manual selection mutated")
			}
		})
	}
}

func TestChatRecentBrewsForeignMissingAndMalformedFields(t *testing.T) {
	rows := map[string][]records.RawRecord{
		"bean":   {contextRow("same", "Own bag", map[string]any{"closed": true})},
		"brewer": {contextRow("same", "Own brewer", nil)},
		"brew":   {contextRow("legacy", "", map[string]any{"createdAt": "2026-11-01T00:00:00Z", "beanRef": "at://did:plc:foreign/" + arabica.NSIDBean + "/same", "brewerRef": contextRef("brewer", "deleted"), "recipeRef": "at://did:plc:foreign/" + arabica.NSIDRecipe + "/same", "coffeeAmount": 18.5, "waterAmount": "300", "temperature": -1, "espressoParams": map[string]any{"pressure": 9.2}, "pours": []any{map[string]any{"waterAmount": -1, "timeSeconds": 30}}})},
	}
	service := NewService(fakeCompleter(func(ctx context.Context, m []provider.Message, tools []provider.Tool, format json.RawMessage) (provider.Completion, error) {
		p := inspectRecent(t, m)
		if len(p.Recent) != 1 || len(p.Recent[0].Fields) != 0 || len(p.Recent[0].Labels) != 0 {
			t.Fatalf("unsafe legacy fields: %+v", p.Recent)
		}
		if strings.Contains(m[1].Content, "did:plc:foreign") || strings.Contains(m[1].Content, "deleted") {
			t.Fatal("foreign/dangling reference leaked")
		}
		return outputClient(`{"in_scope":true,"reply":"Which coffee?","kind":"brew","patch":{},"choices":[]}`)(ctx, m, tools, format)
	}))
	if _, err := service.Chat(context.Background(), ownDID, contextLibrary(rows), validRequest()); err != nil {
		t.Fatal(err)
	}
}

func TestChatRecentBrewsEmptyAndLibraryFailures(t *testing.T) {
	for _, mode := range []string{"empty", "foreign-row", "invalid-key", "fetch-error"} {
		t.Run(mode, func(t *testing.T) {
			called := false
			store := fakeLibrary{did: ownDID, fetch: func(_ context.Context, nsid string) ([]records.RawRecord, error) {
				if nsid != arabica.NSIDBrew {
					return nil, nil
				}
				switch mode {
				case "foreign-row":
					return []records.RawRecord{{RKey: "valid", URI: "at://did:plc:foreign/" + nsid + "/valid"}}, nil
				case "invalid-key":
					return []records.RawRecord{{RKey: "../bad"}}, nil
				case "fetch-error":
					return nil, errors.New("unavailable")
				}
				return nil, nil
			}}
			service := NewService(fakeCompleter(func(ctx context.Context, m []provider.Message, tools []provider.Tool, format json.RawMessage) (provider.Completion, error) {
				called = true
				p := inspectRecent(t, m)
				if p.Recent == nil || len(p.Recent) != 0 {
					t.Fatal("empty history must be []")
				}
				return outputClient(`{"in_scope":true,"reply":"What brew?","kind":"brew","patch":{},"choices":[]}`)(ctx, m, tools, format)
			}))
			_, err := service.Chat(context.Background(), ownDID, store, validRequest())
			if mode == "empty" {
				if err != nil || !called {
					t.Fatalf("%v, called %v", err, called)
				}
			} else if !errors.Is(err, ErrLibrary) || called {
				t.Fatalf("%v, called %v", err, called)
			}
		})
	}
}

func TestChatRecentBrewSourceSelection(t *testing.T) {
	for _, sourceID := range []string{"recent-brew-2", "recent-brew-9", "", "recent-brew-1"} {
		t.Run(sourceID, func(t *testing.T) {
			rows := recentFixture()
			rows["brew"][2].Record["beanRef"] = contextRef("bean", "bag118")
			req := validRequest()
			req.Messages[0].Content = "Repeat the brew from 2026-11-03"
			service := NewService(fakeCompleter(func(ctx context.Context, m []provider.Message, tools []provider.Tool, format json.RawMessage) (provider.Completion, error) {
				p := inspectRecent(t, m)
				id := p.Recent[1].Fields["bean_rkey"]
				body, _ := json.Marshal(map[string]any{"recent_brew_id": sourceID, "in_scope": true, "reply": "Repeat it.", "kind": "brew", "patch": map[string]any{"bean_rkey": id}, "choices": []any{}})
				return outputClient(string(body))(ctx, m, tools, format)
			}))
			response, err := service.Chat(context.Background(), ownDID, contextLibrary(rows), req)
			if sourceID == "recent-brew-9" {
				if !errors.Is(err, ErrProvider) {
					t.Fatalf("invalid source: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if sourceID == "recent-brew-2" {
				if response.Patch["bean_rkey"] != "bag118" || len(response.Choices) != 0 {
					t.Fatalf("source not grounded: %+v", response)
				}
			} else if len(response.Choices) != 1 {
				t.Fatalf("unrelated source grounded: %+v", response)
			}
		})
	}
}

func TestChatRecentBrewsWithImages(t *testing.T) {
	req := validRequest()
	req.Messages[0] = Message{Role: "user", Content: "same as last time but 16 grams", Images: []Image{testImage(t, "png", 2, 2)}}
	service := NewService(fakeCompleter(func(ctx context.Context, m []provider.Message, tools []provider.Tool, format json.RawMessage) (provider.Completion, error) {
		p := inspectRecent(t, m)
		if len(p.Recent) != 3 || len(m) != 3 || len(m[2].Parts) != 2 {
			t.Fatalf("history or image missing: %d messages, history %+v", len(m), p.Recent)
		}
		body, _ := json.Marshal(map[string]any{"recent_brew_id": p.Recent[0].ID, "in_scope": true, "reply": "16 grams with the same bean.", "kind": "brew", "patch": map[string]any{"bean_rkey": p.Recent[0].Fields["bean_rkey"], "coffee_amount": 16}, "choices": []any{}})
		return outputClient(string(body))(ctx, m, tools, format)
	}))
	response, err := service.Chat(context.Background(), ownDID, contextLibrary(recentFixture()), req)
	if err != nil {
		t.Fatal(err)
	}
	if response.Patch["bean_rkey"] != "bag119" || response.Patch["coffee_amount"] != float64(16) || len(response.Choices) != 0 {
		t.Fatalf("image adaptation: %+v", response)
	}
}

func TestChatRecentBrewsTimestampTiesAndRecipePins(t *testing.T) {
	rows := recentFixture()
	rows["brew"] = nil
	for i := 2; i >= 0; i-- {
		rows["brew"] = append(rows["brew"], contextRow(fmt.Sprintf("tie%d", i), "", map[string]any{"createdAt": "2026-11-03T22:00:00.123Z", "beanRef": contextRef("bean", fmt.Sprintf("bag%d", 117+i)), "recipeRef": contextRef("recipe", fmt.Sprintf("recipe%d", 22+i))}))
	}
	for i := 0; i < 25; i++ {
		rows["recipe"] = append(rows["recipe"], contextRow(fmt.Sprintf("recipe%d", i), fmt.Sprintf("Recipe %d", i), map[string]any{"coffeeAmount": 180, "waterAmount": 3000}))
	}
	service := NewService(fakeCompleter(func(ctx context.Context, m []provider.Message, tools []provider.Tool, format json.RawMessage) (provider.Completion, error) {
		p := inspectRecent(t, m)
		for i, b := range p.Recent {
			if b.Date != "2026-11-03T22:00:00.123Z" || b.Labels["bean"] != fmt.Sprintf("Bag %d", 117+i) || b.Labels["recipe"] != fmt.Sprintf("Recipe %d", 22+i) || b.Fields["recipe_rkey"] == nil {
				t.Fatalf("unstable tie or lost pinned reference: %+v", p.Recent)
			}
		}
		counts := map[string]int{}
		for _, item := range p.Library {
			counts[item.Kind]++
		}
		if counts["recipe"] > 23 || counts["bean"] > 103 {
			t.Fatalf("unbounded context: %v", counts)
		}
		body, _ := json.Marshal(map[string]any{"recent_brew_id": p.Recent[2].ID, "in_scope": true, "reply": "Same recipe with 16 grams.", "kind": "brew", "patch": map[string]any{"recipe_rkey": p.Recent[2].Fields["recipe_rkey"], "coffee_amount": 16}, "choices": []any{}})
		return outputClient(string(body))(ctx, m, tools, format)
	}))
	response, err := service.Chat(context.Background(), ownDID, contextLibrary(rows), validRequest())
	if err != nil {
		t.Fatal(err)
	}
	if response.Patch["recipe_rkey"] != "recipe24" || response.Patch["coffee_amount"] != float64(16) || response.Patch["water_amount"] != float64(300) || len(response.Choices) != 0 {
		t.Fatalf("recipe repeat: %+v", response)
	}
}

func TestChatRecentSourcePromotesBundledChoices(t *testing.T) {
	for _, layout := range []string{"separate", "combined", "combined-with-main-dose", "manual-recipe", "missing-source"} {
		t.Run(layout, func(t *testing.T) {
			rows := recentFixture()
			rows["recipe"] = []records.RawRecord{contextRow("four-pour", "4-Pour Dripper", map[string]any{"brewerRef": contextRef("brewer", "device"), "coffeeAmount": 150, "waterAmount": 2500}), contextRow("manual", "Manual recipe", nil)}
			rows["brew"][1].Record["recipeRef"] = contextRef("recipe", "four-pour")
			req := validRequest()
			req.Messages[0].Content = "same as last time, but make it 16g"
			if layout == "manual-recipe" {
				req.Draft.Fields["recipe_rkey"] = "manual"
			}
			service := NewService(fakeCompleter(func(ctx context.Context, m []provider.Message, tools []provider.Tool, format json.RawMessage) (provider.Completion, error) {
				if !strings.Contains(m[0].Content, "MUST include") || !strings.Contains(m[0].Content, "Do not append routine saving disclaimers") {
					t.Fatal("missing provider contract")
				}
				p := inspectRecent(t, m)
				b := p.Recent[0]
				source := b.ID
				if layout == "missing-source" {
					source = ""
				}
				patch := map[string]any{}
				recipe := map[string]any{"recipe_rkey": b.Fields["recipe_rkey"], "brewer_rkey": b.Fields["brewer_rkey"], "coffee_amount": 16, "water_amount": 250, "grind_size": "4.7"}
				choices := []Choice{{Label: "The bean", Patch: map[string]any{"bean_rkey": b.Fields["bean_rkey"]}}, {Label: "The recipe", Patch: recipe}}
				if strings.HasPrefix(layout, "combined") {
					recipe["bean_rkey"] = b.Fields["bean_rkey"]
					choices = []Choice{{Label: "The brew", Patch: recipe}}
				}
				if layout == "combined-with-main-dose" {
					patch["coffee_amount"] = 16
					delete(recipe, "coffee_amount")
				}
				body, _ := json.Marshal(map[string]any{"recent_brew_id": source, "in_scope": true, "reply": "Using your most recent brew with 16g coffee.", "kind": "brew", "patch": patch, "choices": choices})
				return outputClient(string(body))(ctx, m, tools, format)
			}))
			response, err := service.Chat(context.Background(), ownDID, contextLibrary(rows), req)
			if err != nil {
				t.Fatal(err)
			}
			if layout == "manual-recipe" || layout == "missing-source" {
				if response.Patch["recipe_rkey"] == "four-pour" || len(response.Choices) == 0 {
					t.Fatalf("unguarded recipe: %+v", response)
				}
				return
			}
			if response.Patch["bean_rkey"] != "bag119" || response.Patch["recipe_rkey"] != "four-pour" || response.Patch["brewer_rkey"] != "device" || response.Patch["coffee_amount"] != float64(16) || response.Patch["water_amount"] != float64(250) || response.Patch["grind_size"] != "4.7" || len(response.Choices) != 0 {
				t.Fatalf("extra clicks or lost coupled preparation: %+v", response)
			}
			if strings.Contains(response.Reply, "Choose a saved recipe") {
				t.Fatalf("unnecessary recipe confirmation: %s", response.Reply)
			}
		})
	}
}
