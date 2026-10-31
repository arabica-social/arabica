package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/records"
)

const namedBrewUser = "Draft a brew with El Mango Natural, my Fast V60 saved recipe on the Hario V60, and the 1ZPresso ZP6."
const namedBrewAssistant = "Drafting a brew with El Mango Natural, your Fast V60 saved recipe on the Hario V60, and the 1ZPresso ZP6. From the recipe: 15 g coffee, 250 g water total — pours of 40 g at 0:20, 120 g at 0:45, then 90 g, aiming for a 1:15–1:45 drawdown. I didn't set a grind size or temperature since neither was specified. Nothing is written until you press Save."

func namedBundleLibrary() fakeLibrary {
	return fakeLibrary{did: ownDID, fetch: func(_ context.Context, nsid string) ([]records.RawRecord, error) {
		switch strings.TrimPrefix(nsid, arabica.NSIDBase+".") {
		case "bean":
			return []records.RawRecord{{RKey: "mango", Record: map[string]any{"name": "El Mango Natural"}}}, nil
		case "grinder":
			return []records.RawRecord{{RKey: "zp6", Record: map[string]any{"name": "1ZPresso ZP6"}}}, nil
		case "brewer":
			return []records.RawRecord{{RKey: "v60", Record: map[string]any{"name": "Hario V60", "brewerType": "pourover"}}}, nil
		case "recipe":
			return []records.RawRecord{{RKey: "fast", Record: map[string]any{"name": "Fast V60", "brewerRef": "at://" + ownDID + "/" + arabica.NSIDBase + ".brewer/v60", "brewerType": "pourover", "coffeeAmount": 150.0, "waterAmount": 2500.0, "notes": "40g at 0:20, 120g at 0:45, then 90g. Aim for 1:15–1:45 drawdown."}}}, nil
		}
		return nil, nil
	}}
}

func namedBundleChat(t *testing.T, message, patch, choices string, current map[string]any, store fakeLibrary) (Response, error) {
	t.Helper()
	req := validRequest()
	req.Messages = []Message{{Role: "user", Content: "Help draft a brew."}, {Role: "assistant", Content: namedBrewAssistant}, {Role: "user", Content: message}}
	if current != nil {
		req.Draft.Fields = current
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(contextOutput(patch, choices)), &body); err != nil {
		t.Fatal(err)
	}
	body["reply"] = namedBrewAssistant
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return NewService(outputClient(string(raw))).Chat(context.Background(), ownDID, store, req)
}

func TestChatNamedBrewBundles(t *testing.T) {
	tests := []struct{ name, patch, choices string }{
		{"bean grinder bundle and main recipe", `{"recipe_rkey":"recipe-1"}`, `[{"label":"equipment","patch":{"bean_rkey":"bean-1","grinder_rkey":"grinder-1"}}]`},
		{"full recipe bundle", `{}`, `[{"label":"all","patch":{"bean_rkey":"bean-1","grinder_rkey":"grinder-1","recipe_rkey":"recipe-1","brewer_rkey":"brewer-1"}}]`},
		{"separate recipe and combined refs", `{}`, `[{"label":"recipe","patch":{"recipe_rkey":"recipe-1"}},{"label":"equipment","patch":{"bean_rkey":"bean-1","grinder_rkey":"grinder-1"}}]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := namedBundleChat(t, namedBrewUser, tt.patch, tt.choices, nil, namedBundleLibrary())
			if err != nil {
				t.Fatal(err)
			}
			want := map[string]any{"bean_rkey": "mango", "grinder_rkey": "zp6", "recipe_rkey": "fast", "brewer_rkey": "v60", "coffee_amount": 15.0, "water_amount": 250.0}
			if !reflect.DeepEqual(got.Patch, want) || len(got.Choices) != 0 || got.BrewerType != "pourover" {
				t.Fatalf("unnecessary click or invented value: %+v", got)
			}
			// No structured third-pour time, temperature, grind, bloom duration or
			// numeric drawdown exists in the source: prose never invents those fields.
		})
	}
}

func TestChatNamedBundleGroundingBoundaries(t *testing.T) {
	const bundle = `[{"label":"all","patch":{"bean_rkey":"bean-1","grinder_rkey":"grinder-1","recipe_rkey":"recipe-1","brewer_rkey":"brewer-1","temperature":93}}]`
	tests := []struct {
		name, message string
		current       map[string]any
		want          map[string]any
		choices       int
	}{
		{"assistant names do not grant lexical authority; sole open bean default", "Please draft it", nil, map[string]any{"bean_rkey": "mango"}, 1},
		{"only bean and grinder named", "Use El Mango Natural with 1ZPresso ZP6", nil, map[string]any{"bean_rkey": "mango", "grinder_rkey": "zp6"}, 1},
		{"grinder not named", "Use El Mango Natural and Fast V60 on Hario V60", nil, map[string]any{"bean_rkey": "mango", "recipe_rkey": "fast", "brewer_rkey": "v60", "coffee_amount": 15.0, "water_amount": 250.0}, 1},
		{"current grounded context", "Use Fast V60", map[string]any{"bean_rkey": "mango", "grinder_rkey": "zp6", "brewer_rkey": "v60"}, map[string]any{"recipe_rkey": "fast", "brewer_rkey": "v60", "temperature": 93.0, "coffee_amount": 15.0, "water_amount": 250.0}, 0},
		{"manual preparation wins", namedBrewUser, map[string]any{"coffee_amount": 17.0, "temperature": 91.0}, map[string]any{"bean_rkey": "mango", "grinder_rkey": "zp6", "brewer_rkey": "v60", "recipe_rkey": "fast", "coffee_amount": 17.0, "water_amount": 250.0, "temperature": 91.0}, 1},
		{"unique meaningful shorthand", "Use El Mango, Fast V60, Hario V60 and ZP6", nil, map[string]any{"bean_rkey": "mango", "grinder_rkey": "zp6", "recipe_rkey": "fast", "brewer_rkey": "v60", "coffee_amount": 15.0, "water_amount": 250.0, "temperature": 93.0}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := namedBundleChat(t, tt.message, `{}`, bundle, tt.current, namedBundleLibrary())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.Patch, tt.want) || len(got.Choices) != tt.choices {
				t.Fatalf("got %+v want patch %+v choices %d", got, tt.want, tt.choices)
			}
			if len(got.Choices) > 0 {
				choice := got.Choices[0]
				if choice.Patch["bean_rkey"] != "mango" || choice.Patch["grinder_rkey"] != "zp6" || choice.Patch["recipe_rkey"] != "fast" || choice.Patch["temperature"] != 93.0 {
					t.Fatalf("lost full guard context: %+v", choice)
				}
			}
		})
	}
}

func TestChatNamedBundleConflictsAreOrderIndependent(t *testing.T) {
	a := `{"label":"a","patch":{"bean_rkey":"bean-1","grinder_rkey":"grinder-1","recipe_rkey":"recipe-1","coffee_amount":16}}`
	b := `{"label":"b","patch":{"bean_rkey":"bean-1","grinder_rkey":"grinder-1","recipe_rkey":"recipe-1","coffee_amount":18}}`
	for _, choices := range []string{"[" + a + "," + b + "]", "[" + b + "," + a + "]"} {
		got, err := namedBundleChat(t, namedBrewUser, `{}`, choices, nil, namedBundleLibrary())
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got.Patch, map[string]any{"bean_rkey": "mango", "grinder_rkey": "zp6"}) || len(got.Choices) != 2 {
			t.Fatalf("conflict merged: %+v", got)
		}
	}
	// An omitted amount still conflicts with another bundle's departure from the
	// saved amount: order must not promote either recipe source/preparation.
	b = `{"label":"default","patch":{"bean_rkey":"bean-1","grinder_rkey":"grinder-1","recipe_rkey":"recipe-1"}}`
	for _, choices := range []string{"[" + a + "," + b + "]", "[" + b + "," + a + "]"} {
		got, err := namedBundleChat(t, namedBrewUser, `{}`, choices, nil, namedBundleLibrary())
		if err != nil {
			t.Fatal(err)
		}
		if got.Patch["recipe_rkey"] != nil || got.Patch["coffee_amount"] != nil || len(got.Choices) != 2 {
			t.Fatalf("default conflict merged: %+v", got)
		}
	}
}

func TestChatNamedBundleValidationAndDuplicateNames(t *testing.T) {
	for _, choices := range []string{
		`[{"label":"foreign","patch":{"bean_rkey":"bean-1","grinder_rkey":"foreign"}}]`,
		`[{"label":"illegal result","patch":{"bean_rkey":"bean-1","recipe_rkey":"recipe-1","rating":5}}]`,
		`[{"label":"legal","patch":{"bean_rkey":"bean-1"}},{"label":"illegal","patch":{"grinder_rkey":"foreign"}}]`,
	} {
		_, err := namedBundleChat(t, namedBrewUser, `{}`, choices, nil, namedBundleLibrary())
		if !errors.Is(err, ErrProvider) {
			t.Fatalf("invalid envelope accepted: %v", err)
		}
	}
	store := namedBundleLibrary()
	original := store.fetch
	store.fetch = func(ctx context.Context, nsid string) ([]records.RawRecord, error) {
		rows, err := original(ctx, nsid)
		if strings.HasSuffix(nsid, ".bean") {
			rows = append(rows, records.RawRecord{RKey: "duplicate", Record: map[string]any{"name": "El Mango Natural"}})
		}
		return rows, err
	}
	got, err := namedBundleChat(t, namedBrewUser, `{}`, `[{"label":"both","patch":{"bean_rkey":"bean-1","grinder_rkey":"grinder-1"}}]`, nil, store)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Patch, map[string]any{"grinder_rkey": "zp6"}) || len(got.Choices) != 2 {
		t.Fatalf("duplicate autoaccepted: %+v", got)
	}
}

func TestChatNamedBundleRecipeFactsStayFaithful(t *testing.T) {
	for _, dose := range []float64{150, 155} {
		t.Run(fmt.Sprint(dose), func(t *testing.T) {
			store := namedBundleLibrary()
			original := store.fetch
			store.fetch = func(ctx context.Context, nsid string) ([]records.RawRecord, error) {
				rows, err := original(ctx, nsid)
				if strings.HasSuffix(nsid, ".recipe") {
					rows[0].Record["coffeeAmount"] = dose
					rows[0].Record["pours"] = []any{map[string]any{"waterAmount": 40.0, "timeSeconds": 20.0}, map[string]any{"waterAmount": 120.0, "timeSeconds": 45.0}}
				}
				return rows, err
			}
			// No brewer name in the request: the saved recipe's authoritative own
			// brewer link is sufficient, unlike an unmentioned model equipment guess.
			got, err := namedBundleChat(t, "Use El Mango Natural, Fast V60 and 1ZPresso ZP6", `{}`, `[{"label":"all","patch":{"bean_rkey":"bean-1","grinder_rkey":"grinder-1","recipe_rkey":"recipe-1","brewer_rkey":"brewer-1"}}]`, nil, store)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Choices) != 0 || got.Patch["brewer_rkey"] != "v60" {
				t.Fatalf("grounded source still needs click: %+v", got)
			}
			if dose == 150 && got.Patch["coffee_amount"] != 15.0 {
				t.Fatalf("wrong dose: %+v", got)
			}
			if dose == 155 && got.Patch["coffee_amount"] != nil {
				t.Fatalf("fraction silently rounded: %+v", got)
			}
			want := []any{map[string]any{"water_amount": 40.0, "time_seconds": 20.0}, map[string]any{"water_amount": 120.0, "time_seconds": 45.0}}
			if !reflect.DeepEqual(got.Patch["pours"], want) {
				t.Fatalf("pours rescaled or invented: %+v", got)
			}
			for _, field := range []string{"temperature", "grind_size", "time_seconds", "pourover_params.bloom_time"} {
				if _, exists := got.Patch[field]; exists {
					t.Fatalf("invented %s: %+v", field, got)
				}
			}
		})
	}
}

func TestChatNamedBundleMainPreparationWins(t *testing.T) {
	got, err := namedBundleChat(t, namedBrewUser, `{"coffee_amount":17}`, `[{"label":"all","patch":{"bean_rkey":"bean-1","grinder_rkey":"grinder-1","recipe_rkey":"recipe-1","coffee_amount":19}}]`, nil, namedBundleLibrary())
	if err != nil {
		t.Fatal(err)
	}
	if got.Patch["coffee_amount"] != 17.0 || got.Patch["recipe_rkey"] != "fast" || len(got.Choices) != 1 || got.Choices[0].Patch["coffee_amount"] != 19.0 {
		t.Fatalf("overrode main proposal or lost conflict choice: %+v", got)
	}
}

func TestChatNamedBundleConflictingCurrentReferenceHoldsPreparation(t *testing.T) {
	store := namedBundleLibrary()
	original := store.fetch
	store.fetch = func(ctx context.Context, nsid string) ([]records.RawRecord, error) {
		rows, err := original(ctx, nsid)
		if strings.HasSuffix(nsid, ".grinder") {
			rows = append(rows, records.RawRecord{RKey: "other", Record: map[string]any{"name": "Other grinder"}})
		}
		return rows, err
	}
	got, err := namedBundleChat(t, namedBrewUser, `{}`, `[{"label":"all","patch":{"bean_rkey":"bean-1","grinder_rkey":"grinder-1","recipe_rkey":"recipe-1","temperature":93}}]`, map[string]any{"grinder_rkey": "other"}, store)
	if err != nil {
		t.Fatal(err)
	}
	if got.Patch["grinder_rkey"] != nil || got.Patch["temperature"] != nil || got.Patch["recipe_rkey"] != nil || len(got.Choices) != 1 {
		t.Fatalf("preparation crossed manual reference: %+v", got)
	}
}

func TestChatNamedCandidatesBeatUnmentionedAlternatives(t *testing.T) {
	store := namedBundleLibrary()
	original := store.fetch
	store.fetch = func(ctx context.Context, nsid string) ([]records.RawRecord, error) {
		rows, err := original(ctx, nsid)
		if strings.HasSuffix(nsid, ".bean") {
			rows = append(rows, records.RawRecord{RKey: "other-bean", Record: map[string]any{"name": "Other bean"}})
		}
		if strings.HasSuffix(nsid, ".recipe") {
			rows = append(rows, records.RawRecord{RKey: "slow", Record: map[string]any{"name": "Slow V60", "brewerRef": "at://" + ownDID + "/" + arabica.NSIDBase + ".brewer/v60", "brewerType": "pourover", "coffeeAmount": 220.0, "waterAmount": 3300.0}})
		}
		return rows, err
	}
	tests := []struct {
		name, message, a, b string
		want                map[string]any
		choices             int
	}{
		{"named bean versus unmentioned bean", "Use El Mango Natural", `{"label":"known","patch":{"bean_rkey":"bean-1"}}`, `{"label":"guess","patch":{"bean_rkey":"bean-2"}}`, map[string]any{"bean_rkey": "mango"}, 1},
		{"named recipe versus unmentioned recipe defaults", "Use Fast V60", `{"label":"known","patch":{"recipe_rkey":"recipe-1"}}`, `{"label":"guess","patch":{"recipe_rkey":"recipe-2","temperature":97}}`, map[string]any{"recipe_rkey": "fast", "brewer_rkey": "v60", "coffee_amount": 15.0, "water_amount": 250.0}, 1},
		{"named recipe preparation versus unmentioned recipe preparation", "Use Fast V60", `{"label":"known","patch":{"recipe_rkey":"recipe-1","temperature":93}}`, `{"label":"guess","patch":{"recipe_rkey":"recipe-2","temperature":97}}`, map[string]any{"recipe_rkey": "fast", "brewer_rkey": "v60", "coffee_amount": 15.0, "water_amount": 250.0, "temperature": 93.0}, 1},
		{"two literally named beans remain choices", "Use El Mango Natural or Other bean", `{"label":"a","patch":{"bean_rkey":"bean-1"}}`, `{"label":"b","patch":{"bean_rkey":"bean-2"}}`, map[string]any{}, 2},
		{"two literally named recipes remain choices", "Use Fast V60 or Slow V60", `{"label":"a","patch":{"recipe_rkey":"recipe-1"}}`, `{"label":"b","patch":{"recipe_rkey":"recipe-2"}}`, map[string]any{}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, choices := range []string{"[" + tt.a + "," + tt.b + "]", "[" + tt.b + "," + tt.a + "]"} {
				got, err := namedBundleChat(t, tt.message, `{}`, choices, nil, store)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got.Patch, tt.want) || len(got.Choices) != tt.choices {
					t.Fatalf("unmentioned guess vetoed grounded selection: %+v want %+v", got, tt.want)
				}
			}
		})
	}
}

func TestChatNamedSourceLeavesUnconfirmedBundlePreparationReviewable(t *testing.T) {
	store := namedBundleLibrary()
	original := store.fetch
	store.fetch = func(ctx context.Context, nsid string) ([]records.RawRecord, error) {
		rows, err := original(ctx, nsid)
		if strings.HasSuffix(nsid, ".bean") {
			rows[0].Record["closed"] = true
		}
		return rows, err
	}
	got, err := namedBundleChat(t, "Use Fast V60", `{}`, `[{"label":"bundle","patch":{"recipe_rkey":"recipe-1","bean_rkey":"bean-1","temperature":93,"coffee_amount":19}}]`, nil, store)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"recipe_rkey": "fast", "brewer_rkey": "v60", "coffee_amount": 15.0, "water_amount": 250.0}
	if !reflect.DeepEqual(got.Patch, want) || len(got.Choices) != 1 {
		t.Fatalf("known source needs redundant click or accepted unconfirmed preparation: %+v", got)
	}
	choice := got.Choices[0]
	if choice.Label != "El Mango Natural (closed)" || choice.Patch["recipe_rkey"] != "fast" || choice.Patch["brewer_rkey"] != "v60" || choice.Patch["bean_rkey"] != "mango" || choice.Patch["temperature"] != 93.0 || choice.Patch["coffee_amount"] != 19.0 {
		t.Fatalf("lost unresolved label/full guard context: %+v", choice)
	}
}
