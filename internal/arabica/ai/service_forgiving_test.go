package ai

import (
	"context"
	"fmt"
	"strings"
	"tangled.org/arabica.social/arabica/internal/records"
	"testing"
)

func forgivingLibrary(beans ...map[string]any) fakeLibrary {
	base := namedBundleLibrary()
	original := base.fetch
	base.fetch = func(ctx context.Context, nsid string) ([]records.RawRecord, error) {
		if strings.HasSuffix(nsid, ".bean") {
			rows := make([]records.RawRecord, 0, len(beans))
			for i, bean := range beans {
				rows = append(rows, records.RawRecord{RKey: fmt.Sprintf("bag-%d", i), Record: bean})
			}
			return rows, nil
		}
		return original(ctx, nsid)
	}
	return base
}

func TestChatForgivingOwnReferences(t *testing.T) {
	open := func(name string) map[string]any { return map[string]any{"name": name, "closed": false} }
	closed := map[string]any{"name": "El Mango Natural", "closed": true}
	unknown := map[string]any{"name": "El Mango Natural", "closed": "false"}
	cases := []struct {
		name, message string
		beans         []map[string]any
		current       map[string]any
		fill          bool
		choices       int
	}{
		{name: "short coffee", message: "Use El Mango", beans: []map[string]any{open("El Mango Natural")}, fill: true},
		{name: "one token", message: "mango", beans: []map[string]any{open("El Mango Natural")}, fill: true},
		{name: "minor typo", message: "El Magno", beans: []map[string]any{open("El Mango Natural")}, fill: true},
		{name: "history duplicate", message: "El Mango Natural", beans: []map[string]any{open("El Mango Natural"), closed}, fill: true},
		{name: "actual competing names", message: "mango", beans: []map[string]any{open("El Mango Natural"), open("El Mango Honey")}, choices: 2},
		{name: "processing qualifier distinguishes bags", message: "Brew Mango Natural", beans: []map[string]any{open("El Mango Natural"), open("El Mango Washed")}, fill: true},
		{name: "processing variants without qualifier stay ambiguous", message: "Brew Mango", beans: []map[string]any{open("El Mango Natural"), open("El Mango Washed")}, choices: 2},
		{name: "distinct partial", message: "mango red", beans: []map[string]any{open("El Mango Red Natural"), open("El Mango Yellow Natural")}, fill: true},
		{name: "sole open unspecified", message: "15g coffee, 250g water", beans: []map[string]any{open("El Mango Natural")}, fill: true},
		{name: "closed only", message: "15g coffee, 250g water", beans: []map[string]any{closed}, choices: 1},
		{name: "unknown status", message: "mango", beans: []map[string]any{unknown}, choices: 1},
		{name: "closed fuzzy", message: "mango", beans: []map[string]any{closed}, choices: 1},
		{name: "closed explicit", message: "El Mango Natural", beans: []map[string]any{closed}, fill: true},
		{name: "unrelated followup preserves current", message: "water 250", beans: []map[string]any{open("El Mango Natural"), closed}, current: map[string]any{"bean_rkey": "bag-1"}, choices: 1},
		{name: "clearly named other coffee", message: "El Mango Natural", beans: []map[string]any{open("Another Coffee"), closed}, choices: 1},
	}
	for _, tt := range cases {
		for _, choiceOnly := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/choice=%v", tt.name, choiceOnly), func(t *testing.T) {
				patch, choices := `{"bean_rkey":"bean-1"}`, `[]`
				if choiceOnly {
					patch, choices = `{}`, `[{"label":"model guess","patch":{"bean_rkey":"bean-1"}}]`
				}
				got, err := namedBundleChat(t, tt.message, patch, choices, tt.current, forgivingLibrary(tt.beans...))
				if err != nil {
					t.Fatal(err)
				}
				if (got.Patch["bean_rkey"] == "bag-0") != tt.fill || len(got.Choices) != tt.choices {
					t.Fatalf("got %+v want fill=%v choices=%d", got, tt.fill, tt.choices)
				}
			})
		}
	}
}

func TestChatForgivingEquipmentDoesNotAuthorizeRecipe(t *testing.T) {
	got, err := namedBundleChat(t, "Use mango with V60 and ZP6", `{"bean_rkey":"bean-1","brewer_rkey":"brewer-1","grinder_rkey":"grinder-1","recipe_rkey":"recipe-1"}`, `[]`, nil, namedBundleLibrary())
	if err != nil {
		t.Fatal(err)
	}
	if got.Patch["bean_rkey"] != "mango" || got.Patch["grinder_rkey"] != "zp6" || got.Patch["brewer_rkey"] != nil || got.Patch["recipe_rkey"] != nil || got.Patch["coffee_amount"] != nil || len(got.Choices) != 1 {
		t.Fatalf("equipment guessed a recipe or shorthand rejected: %+v", got)
	}
}

func TestChatForgivingEquipmentOnly(t *testing.T) {
	got, err := namedBundleChat(t, "Use mango with V60 and ZP6", `{}`, `[{"label":"bundle","patch":{"bean_rkey":"bean-1","brewer_rkey":"brewer-1","grinder_rkey":"grinder-1"}}]`, nil, namedBundleLibrary())
	if err != nil {
		t.Fatal(err)
	}
	if got.Patch["bean_rkey"] != "mango" || got.Patch["grinder_rkey"] != "zp6" || got.Patch["brewer_rkey"] != "v60" || len(got.Choices) != 0 {
		t.Fatalf("shorthand remained manual: %+v", got)
	}
}

func TestChatForgivingCountsBeforeProviderCap(t *testing.T) {
	beans := make([]map[string]any, 203)
	for i := range beans {
		beans[i] = map[string]any{"name": fmt.Sprintf("Mango lot %d", i), "closed": false}
	}
	got, err := namedBundleChat(t, "mango", `{}`, `[{"label":"only provider choice","patch":{"bean_rkey":"bean-1"}}]`, nil, forgivingLibrary(beans...))
	if err != nil {
		t.Fatal(err)
	}
	if got.Patch["bean_rkey"] != nil || len(got.Choices) != 8 || !strings.Contains(got.Reply, "203 matching open beans") {
		t.Fatalf("bounded subset decided authority: %+v", got)
	}
	req := validRequest()
	req.Messages[0].Content = "Draft a brew"
	catalog, err := loadCatalog(context.Background(), forgivingLibrary(beans...), req.Draft, req.Messages[0].Content)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range catalog {
		if item.Kind == "bean" && item.SoleOpenDefault {
			t.Fatal("subset used to count sole open default")
		}
	}
	beans = append(beans, map[string]any{"name": "Unrelated selected history", "closed": true})
	req.Draft.Fields["bean_rkey"] = "bag-203"
	catalog, err = loadCatalog(context.Background(), forgivingLibrary(beans...), req.Draft, "mango")
	if err != nil {
		t.Fatal(err)
	}
	pinned := false
	for _, item := range catalog {
		if item.Kind == "bean" && item.rkey == "bag-203" {
			pinned = true
		}
	}
	if !pinned {
		t.Fatal("identity relevance displaced selected reference beyond cap")
	}
	// Exercise the real draft-to-provider conversion, not only retention.
	req.Messages[0].Content = "mango"
	_, err = NewService(outputClient(contextOutput(`{}`, `[]`))).Chat(context.Background(), ownDID, forgivingLibrary(beans...), req)
	if err != nil {
		t.Fatalf("selected draft became unusable after retention: %v", err)
	}
}
