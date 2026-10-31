package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	provider "tangled.org/arabica.social/arabica/internal/ai"
	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/records"
	"testing"
)

func recipeAwareLibrary(open int) fakeLibrary {
	return fakeLibrary{did: ownDID, fetch: func(_ context.Context, nsid string) ([]records.RawRecord, error) {
		ref := func(kind, key string) string {
			return "at://" + ownDID + "/" + arabica.NSIDBase + "." + kind + "/" + key
		}
		switch strings.TrimPrefix(nsid, arabica.NSIDBase+".") {
		case "roaster":
			return []records.RawRecord{{RKey: "recluse", Record: map[string]any{"name": "Recluse"}}}, nil
		case "brewer":
			return []records.RawRecord{{RKey: "orea", Record: map[string]any{"name": "Orea 01", "brewerType": "pourover"}}, {RKey: "espresso", Record: map[string]any{"name": "Espresso machine", "brewerType": "espresso"}}}, nil
		case "bean":
			rows := []records.RawRecord{}
			for i := 0; i < open; i++ {
				rows = append(rows, records.RawRecord{RKey: fmt.Sprintf("open-%d", i+1), Record: map[string]any{"name": fmt.Sprintf("October bag %d", i+1), "roasterRef": ref("roaster", "recluse"), "closed": false}})
			}
			return append(rows, records.RawRecord{RKey: "finished", Record: map[string]any{"name": "Finished bag", "roasterRef": ref("roaster", "recluse"), "closed": true}}), nil
		case "recipe":
			return []records.RawRecord{{RKey: "three-pour", Record: map[string]any{"name": "Orea 3 pour", "brewerRef": ref("brewer", "orea"), "brewerType": "pourover", "coffeeAmount": 180.0, "waterAmount": 3000.0, "pours": []any{map[string]any{"waterAmount": 70.0, "timeSeconds": 0.0}, map[string]any{"waterAmount": 130.0, "timeSeconds": 40.0}, map[string]any{"waterAmount": 100.0, "timeSeconds": 90.0}}, "notes": "Use 93°C water. Aim for a three-minute drawdown."}}}, nil
		}
		return nil, nil
	}}
}

// These uncertainty tests need two plausible recipes, not merely a nonliteral name.
func ambiguousRecipeLibrary(open int) fakeLibrary {
	store := recipeAwareLibrary(open)
	original := store.fetch
	store.fetch = func(ctx context.Context, nsid string) ([]records.RawRecord, error) {
		rows, err := original(ctx, nsid)
		if err == nil && strings.HasSuffix(nsid, ".recipe") {
			variant := map[string]any{}
			for key, value := range rows[0].Record {
				variant[key] = value
			}
			variant["name"] = "Orea 3 pour bright"
			rows = append(rows, records.RawRecord{RKey: "three-pour-bright", Record: variant})
		}
		return rows, err
	}
	return store
}

func contextOutput(patch string, choices string) string {
	return `{"in_scope":true,"reply":"Review the draft.","kind":"brew","patch":` + patch + `,"choices":` + choices + `}`
}
func contextRequest() Request {
	r := validRequest()
	r.Messages[0].Content = "brew with bean from recluse, orea 3 pour recipe."
	return r
}

func TestChatUsesOpenRoasterAndSavedRecipeContext(t *testing.T) {
	req := contextRequest()
	client := fakeCompleter(func(ctx context.Context, m []provider.Message, tools []provider.Tool, format json.RawMessage) (provider.Completion, error) {
		var payload struct {
			Library []map[string]any `json:"library"`
		}
		if err := json.Unmarshal([]byte(m[1].Content), &payload); err != nil {
			t.Fatal(err)
		}
		foundBean, foundRecipe := false, false
		for _, item := range payload.Library {
			if item["id"] == "bean-1" {
				foundBean = item["roaster"] == "Recluse" && item["closed"] == false && item["open_roaster_matches"] == 1.0
			}
			if item["id"] == "recipe-1" {
				d := item["defaults"].(map[string]any)
				foundRecipe = d["coffee_amount"] == 18.0 && d["water_amount"] == 300.0 && len(d["pours"].([]any)) == 3 && strings.Contains(item["instructions"].(string), "93°C") && item["brewer"] == "Orea 01"
			}
		}
		if !foundBean || !foundRecipe {
			t.Fatal("missing authoritative roaster/open/recipe facts")
		}
		for _, secret := range []string{ownDID, "three-pour", "open-1", "roasterRef", "brewerRef"} {
			if strings.Contains(m[1].Content, secret) {
				t.Errorf("automatic identity leaked: %s", secret)
			}
		}
		if !strings.Contains(m[0].Content, "recipe instructions") || !strings.Contains(m[0].Content, "untrusted data") {
			t.Fatal("recipe instructions must remain untrusted data")
		}
		return outputClient(contextOutput(`{"bean_rkey":"bean-1","recipe_rkey":"recipe-1"}`, `[]`))(ctx, m, tools, format)
	})
	response, err := NewService(client).Chat(context.Background(), ownDID, recipeAwareLibrary(1), req)
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]any{"bean_rkey": "open-1", "recipe_rkey": "three-pour", "brewer_rkey": "orea", "coffee_amount": 18.0, "water_amount": 300.0} {
		if response.Patch[key] != want {
			t.Errorf("%s=%v want%v", key, response.Patch[key], want)
		}
	}
	if len(response.Choices) != 0 || response.BrewerType != "pourover" || len(response.Patch["pours"].([]any)) != 3 {
		t.Fatalf("not a complete grounded proposal: %+v", response)
	}
	if _, exists := response.Patch["tasting_notes"]; exists {
		t.Fatal("instructions are not brew tasting notes")
	}
	if _, exists := response.Patch["recipe_owner_did"]; exists {
		t.Fatal("own recipe needs no automatic owner identity")
	}
}

func TestChatOpenRoasterAmbiguityAndClosedExclusion(t *testing.T) {
	for _, count := range []int{0, 2, 105} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			response, err := NewService(outputClient(contextOutput(`{"bean_rkey":"bean-1"}`, `[]`))).Chat(context.Background(), ownDID, recipeAwareLibrary(count), contextRequest())
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := response.Patch["bean_rkey"]; ok {
				t.Fatal("ambiguous/closed bean auto-applied")
			}
			want := count
			if want > 8 {
				want = 8
			}
			if len(response.Choices) != want {
				t.Fatalf("got%d choices want%d", len(response.Choices), want)
			}
			for _, choice := range response.Choices {
				if choice.Patch["bean_rkey"] == "finished" {
					t.Fatal("closed bag offered for open-roaster request")
				}
			}
			if count == 0 && !strings.Contains(response.Reply, "No open beans") {
				t.Fatal(response.Reply)
			}
			if count == 2 && !strings.Contains(response.Reply, "2 open beans") {
				t.Fatal(response.Reply)
			}
			if count == 105 && !strings.Contains(response.Reply, "8 of 105") {
				t.Fatal(response.Reply)
			}
		})
	}
}

func TestChatAmbiguousRecipeKeepsAllDetailsBehindChoice(t *testing.T) {
	for _, choiceOnly := range []bool{false, true} {
		t.Run(fmt.Sprint(choiceOnly), func(t *testing.T) {
			req := contextRequest()
			req.Messages[0].Content = "brew from Recluse with the Orea three-pour recipe"
			patch := `{"bean_rkey":"bean-1","recipe_rkey":"recipe-1","coffee_amount":25,"temperature":93,"brewer_rkey":"brewer-1","pourover_params.filter":"paper"}`
			choices := `[]`
			if choiceOnly {
				patch = `{"bean_rkey":"bean-1","coffee_amount":25,"temperature":93,"brewer_rkey":"brewer-1","pourover_params.filter":"paper"}`
				choices = `[{"label":"untrusted label","patch":{"recipe_rkey":"recipe-1"}}]`
			}
			response, err := NewService(outputClient(contextOutput(patch, choices))).Chat(context.Background(), ownDID, ambiguousRecipeLibrary(1), req)
			if err != nil {
				t.Fatal(err)
			}
			if len(response.Patch) != 1 || response.Patch["bean_rkey"] != "open-1" {
				t.Fatalf("recipe details leaked into main patch: %+v", response.Patch)
			}
			if len(response.Choices) != 1 {
				t.Fatalf("unexpected choices: %+v", response.Choices)
			}
			pick := response.Choices[0]
			for key, want := range map[string]any{"recipe_rkey": "three-pour", "brewer_rkey": "orea", "coffee_amount": 25.0, "water_amount": 300.0, "temperature": 93.0, "pourover_params.filter": "paper"} {
				if pick.Patch[key] != want {
					t.Errorf("choice%s=%v want%v", key, pick.Patch[key], want)
				}
			}
			if pick.BrewerType != "pourover" || len(pick.Patch["pours"].([]any)) != 3 || strings.Contains(pick.Label, "untrusted") {
				t.Fatalf("invalid grounded recipe choice: %+v", pick)
			}
		})
	}
}

func TestChatRecipeDefaultsDoNotResetExistingManualValues(t *testing.T) {
	req := contextRequest()
	req.Draft.Fields = map[string]any{"recipe_rkey": "three-pour", "brewer_rkey": "orea", "coffee_amount": 22.0, "water_amount": 340.0}
	response, err := NewService(outputClient(contextOutput(`{"recipe_rkey":"recipe-1","temperature":94}`, `[]`))).Chat(context.Background(), ownDID, recipeAwareLibrary(1), req)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"coffee_amount", "water_amount", "pours"} {
		if _, ok := response.Patch[key]; ok {
			t.Fatalf("unchanged recipe reset %s", key)
		}
	}
	if req.Draft.Fields["coffee_amount"] != 22.0 {
		t.Fatal("caller draft mutated")
	}
}
func TestChatRecipeNeverRoundsOrCopiesIncompatiblePours(t *testing.T) {
	lib := recipeAwareLibrary(1)
	fetch := lib.fetch
	lib.fetch = func(ctx context.Context, nsid string) ([]records.RawRecord, error) {
		rows, err := fetch(ctx, nsid)
		if strings.HasSuffix(nsid, ".recipe") {
			rows[0].Record["coffeeAmount"] = 185.0
		}
		return rows, err
	}
	for _, espresso := range []bool{false, true} {
		t.Run(fmt.Sprint(espresso), func(t *testing.T) {
			req := contextRequest()
			patch := `{"recipe_rkey":"recipe-1"}`
			if espresso {
				req.Messages[0].Content += " Use Espresso machine."
				patch = `{"recipe_rkey":"recipe-1","brewer_rkey":"brewer-2"}`
			}
			response, err := NewService(outputClient(contextOutput(patch, `[]`))).Chat(context.Background(), ownDID, lib, req)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := response.Patch["coffee_amount"]; ok {
				t.Fatal("fractional recipe dose silently rounded")
			}
			if response.Patch["water_amount"] != 300.0 {
				t.Fatal("safe partial defaults missing")
			}
			_, hasPours := response.Patch["pours"]
			if hasPours == espresso {
				t.Fatal("recipe pours must follow known compatible brewer")
			}
		})
	}
}
func TestChatRecipeChoicesRejectUnrecognizedOrUnrelatedOperations(t *testing.T) {
	for _, body := range []string{contextOutput(`{"recipe_rkey":"not-a-library-id"}`, `[]`), contextOutput(`{}`, `[{"label":"bad","patch":{"bean_rkey":"bean-1","coffee_amount":25}}]`), contextOutput(`{}`, `[{"label":"bad","patch":{"recipe_rkey":"recipe-1","rating":10}}]`), contextOutput(`{}`, `[{"label":"bad","patch":{"recipe_rkey":"recipe-1","tasting_notes":"invented"}}]`)} {
		_, err := NewService(outputClient(body)).Chat(context.Background(), ownDID, recipeAwareLibrary(1), contextRequest())
		if !errors.Is(err, ErrProvider) {
			t.Fatalf("untrusted reference/choice accepted: %v", err)
		}
	}
}

func TestShorthandRecipeIsRetainedButNotAutoSelected(t *testing.T) {
	lib := recipeAwareLibrary(1)
	fetch := lib.fetch
	lib.fetch = func(ctx context.Context, nsid string) ([]records.RawRecord, error) {
		rows, err := fetch(ctx, nsid)
		if !strings.HasSuffix(nsid, ".recipe") {
			return rows, err
		}
		target := rows[0]
		target.RKey = "target-recipe"
		target.Record["name"] = "Three-pour"
		target.Record["notes"] = strings.Repeat("あ", 900)
		result := []records.RawRecord{}
		for i := 0; i < 30; i++ {
			result = append(result, records.RawRecord{RKey: fmt.Sprintf("other-%d", i), Record: map[string]any{"name": fmt.Sprintf("Unrelated %d", i)}})
		}
		return append(result, target), nil
	}
	c, err := loadCatalog(context.Background(), lib, contextRequest().Draft, contextRequest().Messages[0].Content)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range c {
		if item.Kind == "recipe" && item.rkey == "target-recipe" {
			found = true
			if len(item.Instructions) > 2000 || !item.InstructionsTruncated {
				t.Fatal("incomplete instructions must be marked, bounded and UTF-8 safe")
			}
		}
	}
	if !found {
		t.Fatal("relevant shorthand recipe lost beyond context cap")
	}
	if c.explicitlyNamed("recipe_rkey", "target-recipe", contextRequest().Messages[0].Content) {
		t.Fatal("context ranking granted fuzzy auto-selection")
	}
}

func TestChatRecipeChoiceKeepsExplicitInstructionFactsWithSource(t *testing.T) {
	req := contextRequest()
	req.Messages[0].Content = "Recluse bean and the Orea three-pour recipe"
	response, err := NewService(outputClient(contextOutput(`{"bean_rkey":"bean-1"}`, `[{"label":"invented","patch":{"recipe_rkey":"recipe-1","temperature":93,"time_seconds":180}}]`))).Chat(context.Background(), ownDID, ambiguousRecipeLibrary(1), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Patch) != 1 || response.Patch["bean_rkey"] != "open-1" {
		t.Fatalf("instruction facts applied before recipe choice: %+v", response.Patch)
	}
	if len(response.Choices) != 1 {
		t.Fatal(response.Choices)
	}
	choice := response.Choices[0]
	for key, want := range map[string]any{"recipe_rkey": "three-pour", "coffee_amount": 18.0, "water_amount": 300.0, "temperature": 93.0, "time_seconds": 180.0} {
		if choice.Patch[key] != want {
			t.Fatalf("missing recipe/instruction fact %s: %+v", key, choice)
		}
	}
}
