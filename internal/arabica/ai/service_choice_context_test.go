package ai

import (
	"context"
	"reflect"
	"strings"
	"tangled.org/arabica.social/arabica/internal/records"
	"testing"
)

func TestChatChoiceContextNormalization(t *testing.T) {
	tests := []struct {
		name           string
		current        map[string]any
		patch, choices string
		labels         []string
		wantPatches    []map[string]any
	}{
		{
			name:    "ambiguous beans omit fixed grounded brewer",
			patch:   `{"brewer_rkey":"brewer-1"}`,
			choices: `[{"label":"a","patch":{"bean_rkey":"bean-1"}},{"label":"b","patch":{"bean_rkey":"bean-2"}},{"label":"fixed","patch":{"brewer_rkey":"brewer-1"}}]`,
			labels:  []string{"October bag 1 · Recluse", "October bag 2 · Recluse"},
		},
		{
			name:        "combined choices label only unresolved bean and retain guard context",
			patch:       `{"brewer_rkey":"brewer-1"}`,
			choices:     `[{"label":"a","patch":{"bean_rkey":"bean-1","brewer_rkey":"brewer-1"}},{"label":"b","patch":{"bean_rkey":"bean-2","brewer_rkey":"brewer-1"}}]`,
			labels:      []string{"October bag 1 · Recluse", "October bag 2 · Recluse"},
			wantPatches: []map[string]any{{"bean_rkey": "open-1", "brewer_rkey": "orea"}, {"bean_rkey": "open-2", "brewer_rkey": "orea"}},
		},
		{
			name:    "unchanged reference pure choice omitted",
			current: map[string]any{"bean_rkey": "open-1"},
			patch:   `{}`,
			choices: `[{"label":"a","patch":{"bean_rkey":"bean-1"}}]`,
			labels:  []string{},
		},
		{
			name:    "single real unconfirmed suggestion retained",
			patch:   `{}`,
			choices: `[{"label":"a","patch":{"bean_rkey":"bean-1"}}]`,
			labels:  []string{"October bag 1 · Recluse"},
		},
		{
			name:    "ungrounded main patch remains a choice",
			patch:   `{"bean_rkey":"bean-1"}`,
			choices: `[]`,
			labels:  []string{"October bag 1 · Recluse"},
		},
		{
			name:        "same recipe changed preparation not dropped",
			current:     map[string]any{"recipe_rkey": "three-pour", "coffee_amount": 18.0},
			patch:       `{}`,
			choices:     `[{"label":"recipe","patch":{"recipe_rkey":"recipe-1","coffee_amount":20}}]`,
			labels:      []string{"Orea 3 pour · Orea 01"},
			wantPatches: []map[string]any{{"recipe_rkey": "three-pour", "coffee_amount": 20.0}},
		},
		{
			name:    "same recipe equal preparation omitted",
			current: map[string]any{"recipe_rkey": "three-pour", "coffee_amount": 18.0},
			patch:   `{}`,
			choices: `[{"label":"recipe","patch":{"recipe_rkey":"recipe-1","coffee_amount":18}}]`,
			labels:  []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validRequest()
			req.Messages[0].Content = "Brew with Orea 01; which bean?"
			if tt.current != nil {
				req.Draft.Fields = tt.current
			}
			got, err := NewService(outputClient(contextOutput(tt.patch, tt.choices))).Chat(context.Background(), ownDID, recipeAwareLibrary(2), req)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Choices) != len(tt.labels) {
				t.Fatalf("choices: %+v; want labels %v", got.Choices, tt.labels)
			}
			for i, label := range tt.labels {
				if got.Choices[i].Label != label {
					t.Errorf("label %q want %q", got.Choices[i].Label, label)
				}
				if tt.wantPatches != nil && !reflect.DeepEqual(got.Choices[i].Patch, tt.wantPatches[i]) {
					t.Errorf("coupled patch changed: %+v want %+v", got.Choices[i].Patch, tt.wantPatches[i])
				}
			}
			if _, proposed := got.Patch["bean_rkey"]; proposed {
				t.Fatal("unconfirmed bean autoapplied")
			}
		})
	}
}

func TestChatGroundedRecipeChoiceContext(t *testing.T) {
	req := contextRequest()
	body := contextOutput(`{"recipe_rkey":"recipe-1"}`, `[{"label":"fixed recipe","patch":{"recipe_rkey":"recipe-1"}},{"label":"bean and fixed recipe","patch":{"bean_rkey":"bean-1","recipe_rkey":"recipe-1"}}]`)
	got, err := NewService(outputClient(body)).Chat(context.Background(), ownDID, recipeAwareLibrary(2), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Choices) != 2 || got.Choices[0].Label != "October bag 1 · Recluse" {
		t.Fatalf("unexpected choices %+v", got.Choices)
	}
	choice := got.Choices[0]
	if choice.Patch["recipe_rkey"] != "three-pour" || choice.Patch["brewer_rkey"] != "orea" || choice.BrewerType != "pourover" || choice.Patch["coffee_amount"] != 18.0 || len(choice.Patch["pours"].([]any)) != 3 {
		t.Fatalf("lost recipe coupling/guard fields: %+v", choice)
	}
}

func TestChoiceContextJSONEquality(t *testing.T) {
	choices := []Choice{{Label: "Recipe", Patch: map[string]any{"recipe_rkey": "same", "coffee_amount": 18}}}
	got := (catalog{}).unresolvedChoices(choices, map[string]any{"recipe_rkey": "same", "coffee_amount": 18.0}, nil)
	if len(got) != 0 {
		t.Fatalf("equal JSON numbers are not a choice: %+v", got)
	}
	// An explicit clear must not be mistaken for an absent value.
	choices = []Choice{{Label: "Recipe", Patch: map[string]any{"recipe_rkey": "same", "pours": nil}}}
	got = (catalog{}).unresolvedChoices(choices, map[string]any{"recipe_rkey": "same"}, nil)
	if len(got) != 1 {
		t.Fatal("explicit clear was dropped")
	}
}

func TestChatPromotesGroundedNamedChoices(t *testing.T) {
	tests := []struct {
		name, message, patch, choices string
		wantBrewer, wantRecipe        string
		wantChoices                   int
	}{
		{"exact brewer with ambiguous beans", "Use Orea 01 with one of my beans", `{}`, `[{"label":"brewer","patch":{"brewer_rkey":"brewer-1"}},{"label":"bean","patch":{"bean_rkey":"bean-1"}},{"label":"bean","patch":{"bean_rkey":"bean-2"}}]`, "orea", "", 2},
		{"exact recipe enriches owned defaults", "Use Orea 3 pour", `{}`, `[{"label":"recipe","patch":{"recipe_rkey":"recipe-1"}}]`, "orea", "three-pour", 0},
		{"exact recipe owner null pair", "Use Orea 3 pour", `{}`, `[{"label":"recipe","patch":{"recipe_rkey":"recipe-1","recipe_owner_did":null}}]`, "orea", "three-pour", 0},
		{"distinctive recipe shorthand", "Use Orea three pour", `{}`, `[{"label":"recipe","patch":{"recipe_rkey":"recipe-1"}}]`, "orea", "three-pour", 0},
		{"main patch conflict not overwritten", "Use Orea 01 or Espresso machine", `{"brewer_rkey":"brewer-2"}`, `[{"label":"brewer","patch":{"brewer_rkey":"brewer-1"}}]`, "espresso", "", 1},
		{"two named brewer candidates conflict", "Use Orea 01 or Espresso machine", `{}`, `[{"label":"brewer","patch":{"brewer_rkey":"brewer-1"}},{"label":"brewer","patch":{"brewer_rkey":"brewer-2"}}]`, "", "", 2},
		{"named conflict inside combined bundle", "Use Orea 01 or Espresso machine", `{}`, `[{"label":"brewer","patch":{"brewer_rkey":"brewer-1"}},{"label":"combined","patch":{"bean_rkey":"bean-1","brewer_rkey":"brewer-2"}}]`, "", "", 2},
		{"named recipe preparation promoted", "Use Orea 3 pour", `{}`, `[{"label":"recipe","patch":{"recipe_rkey":"recipe-1","coffee_amount":20}}]`, "orea", "three-pour", 0},
		{"named recipe retains main preparation", "Use Orea 3 pour", `{"coffee_amount":20}`, `[{"label":"recipe","patch":{"recipe_rkey":"recipe-1"}}]`, "orea", "three-pour", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validRequest()
			req.Messages[0].Content = tt.message
			got, err := NewService(outputClient(contextOutput(tt.patch, tt.choices))).Chat(context.Background(), ownDID, recipeAwareLibrary(2), req)
			if err != nil {
				t.Fatal(err)
			}
			brewer, _ := got.Patch["brewer_rkey"].(string)
			recipe, _ := got.Patch["recipe_rkey"].(string)
			if brewer != tt.wantBrewer || recipe != tt.wantRecipe || len(got.Choices) != tt.wantChoices {
				t.Fatalf("unexpected promotion: %+v", got)
			}
			if brewer == "orea" && got.BrewerType != "pourover" {
				t.Fatal("lost authoritative brewer type")
			}
			if recipe == "three-pour" {
				wantCoffee := 18.0
				if strings.HasPrefix(tt.name, "named recipe") {
					wantCoffee = 20.0
				}
				if got.Patch["coffee_amount"] != wantCoffee || got.Patch["water_amount"] != 300.0 || len(got.Patch["pours"].([]any)) != 3 {
					t.Fatalf("recipe defaults missing: %+v", got)
				}
			}
			if tt.name == "exact brewer with ambiguous beans" {
				for _, choice := range got.Choices {
					if _, ok := choice.Patch["bean_rkey"]; !ok {
						t.Fatalf("fixed brewer shown as option: %+v", choice)
					}
				}
			}
		})
	}
}

func TestChatDuplicateNamedBrewerChoiceIsNotPromoted(t *testing.T) {
	library := recipeAwareLibrary(2)
	original := library.fetch
	library.fetch = func(ctx context.Context, nsid string) ([]records.RawRecord, error) {
		rows, err := original(ctx, nsid)
		if strings.HasSuffix(nsid, ".brewer") {
			rows = append(rows, records.RawRecord{RKey: "duplicate-orea", Record: map[string]any{"name": "Orea 01", "brewerType": "pourover"}})
		}
		return rows, err
	}
	req := validRequest()
	req.Messages[0].Content = "Use Orea 01"
	got, err := NewService(outputClient(contextOutput(`{}`, `[{"label":"brewer","patch":{"brewer_rkey":"brewer-1"}}]`))).Chat(context.Background(), ownDID, library, req)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := got.Patch["brewer_rkey"]; exists || len(got.Choices) != 1 {
		t.Fatalf("duplicate name was autoaccepted: %+v", got)
	}
}
