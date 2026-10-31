package ai

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"tangled.org/arabica.social/arabica/internal/records"
)

func TestChatScreenshotShorthandWithoutConfirmation(t *testing.T) {
	for _, choiceOnly := range []bool{false, true} {
		t.Run(fmt.Sprintf("choices=%v", choiceOnly), func(t *testing.T) {
			patch, choices := `{"bean_rkey":"bean-1","recipe_rkey":"recipe-1"}`, `[]`
			if choiceOnly {
				patch, choices = `{}`, `[{"label":"Confirm coffee","patch":{"bean_rkey":"bean-1"}},{"label":"Confirm recipe","patch":{"recipe_rkey":"recipe-1"}}]`
			}
			first, err := namedBundleChat(t, "brew, el mango, fast 60", patch, choices, nil, namedBundleLibrary())
			if err != nil {
				t.Fatal(err)
			}
			if first.Patch["bean_rkey"] != "mango" || first.Patch["recipe_rkey"] != "fast" || first.Patch["brewer_rkey"] != "v60" || first.Patch["coffee_amount"] != 15.0 || first.Patch["water_amount"] != 250.0 || len(first.Choices) != 0 {
				t.Fatalf("clear first-turn matches still need buttons: %+v", first)
			}
			patch, choices = `{"rating":8,"grinder_rkey":"grinder-1"}`, `[]`
			if choiceOnly {
				patch, choices = `{"rating":8}`, `[{"label":"Confirm grinder","patch":{"grinder_rkey":"grinder-1"}}]`
			}
			second, err := namedBundleChat(t, "8 rating. used zp6 grinder", patch, choices, first.Patch, namedBundleLibrary())
			if err != nil {
				t.Fatal(err)
			}
			if second.Patch["rating"] != float64(8) || second.Patch["grinder_rkey"] != "zp6" || len(second.Choices) != 0 {
				t.Fatalf("explicit grinder still needs a button: %+v", second)
			}
			for _, key := range []string{"bean_rkey", "recipe_rkey", "brewer_rkey", "coffee_amount", "water_amount"} {
				if value, changed := second.Patch[key]; changed && value != first.Patch[key] {
					t.Fatalf("follow-up replaced %s: %+v", key, second)
				}
			}
		})
	}
}

func TestChatRecipeShorthandStillChecksWholeLibrary(t *testing.T) {
	for _, tc := range []struct {
		name, message string
		variants      int
		want          bool
	}{
		{"distinctive recipe word", "use Fast", 0, true},
		{"numeric brewer shorthand distinguishes recipes", "use Fast 60", 1, true},
		{"shared word remains ambiguous", "use Fast", 1, false},
		{"bare equipment is not recipe identity", "use V60", 0, false},
		{"bare number is not recipe identity", "brew in 60 seconds", 0, false},
		{"ambiguity outside twenty recipe cap", "use Fast 60", 24, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := namedBundleLibrary()
			original := store.fetch
			store.fetch = func(ctx context.Context, nsid string) ([]records.RawRecord, error) {
				rows, err := original(ctx, nsid)
				if err != nil || !strings.HasSuffix(nsid, ".recipe") {
					return rows, err
				}
				for i := 0; i < tc.variants; i++ {
					name := "Fast AeroPress"
					if tc.variants > 1 {
						name = fmt.Sprintf("Fast V60 variation %d", i)
					}
					rows = append(rows, records.RawRecord{RKey: fmt.Sprintf("alternative-%d", i), Record: map[string]any{"name": name, "brewerRef": "at://" + ownDID + "/social.arabica.alpha.brewer/v60", "coffeeAmount": 200.0, "waterAmount": 3000.0}})
				}
				return rows, nil
			}
			got, err := namedBundleChat(t, tc.message, `{}`, `[{"label":"recipe","patch":{"recipe_rkey":"recipe-1"}}]`, nil, store)
			if err != nil {
				t.Fatal(err)
			}
			if (got.Patch["recipe_rkey"] == "fast") != tc.want {
				t.Fatalf("recipe authority want=%v response=%+v", tc.want, got)
			}
			if !tc.want && (got.Patch["coffee_amount"] != nil || got.Patch["brewer_rkey"] != nil || len(got.Choices) == 0) {
				t.Fatalf("unconfirmed recipe leaked preparation: %+v", got)
			}
		})
	}
}
