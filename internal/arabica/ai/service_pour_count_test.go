package ai

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"tangled.org/arabica.social/arabica/internal/records"
)

func pourCountLibrary(extraThree bool) fakeLibrary {
	store := recipeAwareLibrary(1)
	original := store.fetch
	store.fetch = func(ctx context.Context, nsid string) ([]records.RawRecord, error) {
		rows, err := original(ctx, nsid)
		if err != nil {
			return nil, err
		}
		if strings.HasSuffix(nsid, ".bean") {
			rows[0].Record["name"] = "The Fix – October 2026"
		}
		if !strings.HasSuffix(nsid, ".recipe") {
			return rows, nil
		}
		three := rows[0]
		three.Record["name"] = "3-Pour (1 bloom) Orea 01"
		three.Record["coffeeAmount"] = 150.0
		three.Record["waterAmount"] = 2500.0
		three.Record["pours"] = []any{map[string]any{"waterAmount": 40.0, "timeSeconds": 20.0}, map[string]any{"waterAmount": 120.0, "timeSeconds": 45.0}, map[string]any{"waterAmount": 90.0, "timeSeconds": 0.0}}
		four := records.RawRecord{RKey: "four-pour", Record: map[string]any{}}
		for key, value := range three.Record {
			four.Record[key] = value
		}
		four.Record["name"] = "4-Pour Orea 01"
		four.Record["coffeeAmount"] = 200.0
		four.Record["pours"] = []any{map[string]any{"waterAmount": 50.0, "timeSeconds": 0.0}, map[string]any{"waterAmount": 50.0, "timeSeconds": 20.0}, map[string]any{"waterAmount": 50.0, "timeSeconds": 40.0}, map[string]any{"waterAmount": 100.0, "timeSeconds": 60.0}}
		rows = []records.RawRecord{four, three}
		if extraThree {
			variant := records.RawRecord{RKey: "three-bright", Record: map[string]any{}}
			for key, value := range three.Record {
				variant.Record[key] = value
			}
			variant.Record["name"] = "3-Pour bright Orea 01"
			rows = append(rows, variant)
		}
		return rows, nil
	}
	return store
}

func TestChatPourCountDisambiguatesRecipe(t *testing.T) {
	for _, wording := range []string{"3 pour", "3 pours", "three-pour"} {
		for _, reverse := range []bool{false, true} {
			for _, main := range []string{"none", "correct", "incorrect"} {
				t.Run(fmt.Sprintf("%s/reverse=%v/main=%s", wording, reverse, main), func(t *testing.T) {
					choices := `[{"label":"four","patch":{"recipe_rkey":"recipe-1"}},{"label":"three","patch":{"recipe_rkey":"recipe-2"}}]`
					if reverse {
						choices = `[{"label":"three","patch":{"recipe_rkey":"recipe-2"}},{"label":"four","patch":{"recipe_rkey":"recipe-1"}}]`
					}
					patch := `{"bean_rkey":"bean-1"}`
					if main == "correct" {
						patch = `{"bean_rkey":"bean-1","recipe_rkey":"recipe-2"}`
					}
					if main == "incorrect" {
						patch = `{"bean_rkey":"bean-1","recipe_rkey":"recipe-1","coffee_amount":20}`
					}
					got, err := namedBundleChat(t, "new brew with the fix from this month, "+wording+" orea recipe", patch, choices, nil, pourCountLibrary(false))
					if err != nil {
						t.Fatal(err)
					}
					if got.Patch["recipe_rkey"] != "three-pour" || got.Patch["brewer_rkey"] != "orea" || got.Patch["coffee_amount"] != 15.0 || got.Patch["water_amount"] != 250.0 || len(got.Choices) != 0 {
						t.Fatalf("unambiguous pour count still needs a choice: %+v", got)
					}
					pours, ok := got.Patch["pours"].([]any)
					if !ok || len(pours) != 3 || pours[1].(map[string]any)["water_amount"] != 120.0 {
						t.Fatalf("wrong recipe preparation: %+v", got)
					}
				})
			}
		}
	}
}

func TestChatSamePourCountRemainsAmbiguous(t *testing.T) {
	got, err := namedBundleChat(t, "3 pour orea recipe", `{}`, `[{"label":"first","patch":{"recipe_rkey":"recipe-2"}},{"label":"second","patch":{"recipe_rkey":"recipe-3"}}]`, nil, pourCountLibrary(true))
	if err != nil {
		t.Fatal(err)
	}
	if got.Patch["recipe_rkey"] != nil || got.Patch["pours"] != nil || len(got.Choices) != 2 {
		t.Fatalf("same-count recipes guessed: %+v", got)
	}
}

func TestSpecifiedPourCount(t *testing.T) {
	for _, tc := range []struct {
		text string
		want int
	}{
		{"3-Pour (1 bloom) Orea 01", 3}, {"three pours on Orea 01", 3},
		{"15g coffee, 250g water, V60", 0}, {"1 bloom, Orea 01", 0},
		{"3 pour or 4 pour", 0}, {"3 pour, three pours", 3},
	} {
		if got := specifiedPourCount(tc.text); got != tc.want {
			t.Errorf("%q: got %d want %d", tc.text, got, tc.want)
		}
	}
}

func TestChatPourCountCanUseStoredPours(t *testing.T) {
	store := pourCountLibrary(false)
	original := store.fetch
	store.fetch = func(ctx context.Context, nsid string) ([]records.RawRecord, error) {
		rows, err := original(ctx, nsid)
		if strings.HasSuffix(nsid, ".recipe") {
			rows[0].Record["name"] = "Evening Orea"
			rows[1].Record["name"] = "Morning Orea"
		}
		return rows, err
	}
	got, err := namedBundleChat(t, "3 pours orea recipe", `{}`, `[{"label":"evening","patch":{"recipe_rkey":"recipe-1"}},{"label":"morning","patch":{"recipe_rkey":"recipe-2"}}]`, nil, store)
	if err != nil {
		t.Fatal(err)
	}
	if got.Patch["recipe_rkey"] != "three-pour" || len(got.Choices) != 0 {
		t.Fatalf("stored count not used: %+v", got)
	}
}

func TestPruningWrongPourRecipeKeepsIndependentChoice(t *testing.T) {
	got, err := namedBundleChat(t, "3 pour orea recipe", `{}`, `[{"label":"wrong recipe plus bean","patch":{"recipe_rkey":"recipe-1","bean_rkey":"bean-2"}},{"label":"three","patch":{"recipe_rkey":"recipe-2"}}]`, nil, pourCountLibrary(false))
	if err != nil {
		t.Fatal(err)
	}
	if got.Patch["recipe_rkey"] != "three-pour" || len(got.Choices) != 1 {
		t.Fatalf("lost independent choice: %+v", got)
	}
	pick := got.Choices[0]
	if len(pick.Patch) != 1 || pick.Patch["bean_rkey"] != "finished" {
		t.Fatalf("contradicted source leaked into bean choice: %+v", pick)
	}
}

func TestWrongPourCandidateCannotBypassReferenceValidation(t *testing.T) {
	_, err := namedBundleChat(t, "3 pour orea recipe", `{}`, `[{"label":"wrong recipe plus foreign ref","patch":{"recipe_rkey":"recipe-1","bean_rkey":"foreign-record"}},{"label":"three","patch":{"recipe_rkey":"recipe-2"}}]`, nil, pourCountLibrary(false))
	if err == nil {
		t.Fatal("invalid reference discarded instead of rejecting complete provider envelope")
	}
}
