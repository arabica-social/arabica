package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/records"
)

func contextLibrary(rows map[string][]records.RawRecord) fakeLibrary {
	return fakeLibrary{did: ownDID, fetch: func(_ context.Context, nsid string) ([]records.RawRecord, error) {
		return rows[strings.TrimPrefix(nsid, arabica.NSIDBase+".")], nil
	}}
}
func contextRef(kind, key string) string {
	return "at://" + ownDID + "/" + arabica.NSIDBase + "." + kind + "/" + key
}
func contextRow(key, name string, extra map[string]any) records.RawRecord {
	r := records.RawRecord{RKey: key, Record: map[string]any{"name": name}}
	for k, v := range extra {
		r.Record[k] = v
	}
	return r
}
func findContext(t *testing.T, c catalog, kind, key string) libraryItem {
	t.Helper()
	for _, item := range c {
		if item.Kind == kind && item.rkey == key {
			return item
		}
	}
	t.Fatalf("missing %s/%s", kind, key)
	return libraryItem{}
}
func TestCatalogRecipeContextAndBaseline(t *testing.T) {
	rows := map[string][]records.RawRecord{
		"brewer": {contextRow("own-brewer", "Orea", map[string]any{"brewerType": "pourover"})},
		"recipe": {contextRow("own-recipe", "3 pour", map[string]any{"brewerRef": contextRef("brewer", "own-brewer"), "coffeeAmount": int64(180), "waterAmount": float64(3000), "notes": "93°C, medium grind, 3 minutes", "sourceRef": "at://did:plc:private/source/key", "pours": []any{map[string]any{"waterAmount": float64(100), "timeSeconds": float64(30)}}})},
	}
	c, err := loadCatalog(context.Background(), contextLibrary(rows), Draft{}, "Orea 3 pour")
	if err != nil {
		t.Fatal(err)
	}
	recipe := findContext(t, c, "recipe", "own-recipe")
	if recipe.Defaults["coffee_amount"] != float64(18) || recipe.Defaults["water_amount"] != float64(300) || recipe.Instructions != "93°C, medium grind, 3 minutes" || recipe.BrewerLabel != "Orea" {
		t.Fatalf("context: %+v", recipe)
	}
	patch, category, err := c.recipePatch("own-recipe")
	if err != nil || category != "pourover" || patch["brewer_rkey"] != "own-brewer" || patch["recipe_rkey"] != "own-recipe" {
		t.Fatalf("patch %v %s %v", patch, category, err)
	}
	if patch["pours"].([]any)[0].(map[string]any)["water_amount"] != float64(100) {
		t.Fatal("pours scaled")
	}
	if _, ok := patch["tasting_notes"]; ok {
		t.Fatal("instructions copied to tasting notes")
	}
	data, _ := json.Marshal(c)
	for _, private := range []string{ownDID, "own-recipe", "own-brewer", "at://", "sourceRef"} {
		if strings.Contains(string(data), private) {
			t.Fatalf("private identifier in catalog: %s", private)
		}
	}
	converted, _, err := c.convert(map[string]any{"recipe_rkey": "own-recipe"}, true)
	if err != nil || converted["recipe_rkey"] != recipe.ID {
		t.Fatal("recipe reference not opaque")
	}
	if got := c.deriveLabel(map[string]any{"recipe_rkey": "own-recipe"}); got != "3 pour · Orea" {
		t.Fatal(got)
	}
}
func TestCatalogRecipeFractionalMalformedAndForeign(t *testing.T) {
	rows := map[string][]records.RawRecord{"brewer": {contextRow("same", "Own", nil)}, "recipe": {contextRow("fraction", "Fraction", map[string]any{"coffeeAmount": float64(185), "waterAmount": float64(3005), "brewerRef": "at://did:plc:foreign/" + arabica.NSIDBase + ".brewer/same", "pours": []any{map[string]any{"waterAmount": -1, "timeSeconds": 2}}})}}
	c, err := loadCatalog(context.Background(), contextLibrary(rows), Draft{})
	if err != nil {
		t.Fatal(err)
	}
	item := findContext(t, c, "recipe", "fraction")
	if item.Defaults["coffee_amount"] != 18.5 || item.Defaults["water_amount"] != 300.5 || item.BrewerLabel != "" {
		t.Fatalf("lost precision/foreign join: %+v", item)
	}
	patch, category, err := c.recipePatch("fraction")
	if err != nil || len(patch) != 1 || category != "" {
		t.Fatalf("invalid baseline %v %s %v", patch, category, err)
	}
	if _, _, err := c.recipePatch("missing"); err == nil {
		t.Fatal("unknown recipe accepted")
	}
}
func TestCatalogRecipeRetention(t *testing.T) {
	rows := map[string][]records.RawRecord{}
	for i := 0; i < 35; i++ {
		rows["recipe"] = append(rows["recipe"], contextRow(fmt.Sprintf("r%d", i), fmt.Sprintf("Recipe %d", i), nil))
	}
	c, err := loadCatalog(context.Background(), contextLibrary(rows), Draft{Fields: map[string]any{"recipe_rkey": "r34"}}, "Recipe 33")
	if err != nil {
		t.Fatal(err)
	}
	if len(c) != 20 {
		t.Fatalf("cap %d", len(c))
	}
	findContext(t, c, "recipe", "r34")
	findContext(t, c, "recipe", "r33")
}

func TestCatalogRecipeOptionalDetailsBounded(t *testing.T) {
	for _, details := range []map[string]any{
		{"coffeeAmount": "180", "waterAmount": -1, "pours": []any{nil}},
		{"coffeeAmount": 180.5, "waterAmount": 1e100, "pours": make([]any, 101)},
	} {
		details["notes"] = strings.Repeat("x", 2100)
		rows := map[string][]records.RawRecord{"recipe": {contextRow("malformed", "Recipe", details)}}
		c, err := loadCatalog(context.Background(), contextLibrary(rows), Draft{})
		if err != nil {
			t.Fatal(err)
		}
		item := findContext(t, c, "recipe", "malformed")
		if len(item.Instructions) != 2000 || len(item.Defaults) != 0 {
			t.Fatalf("unbounded/invalid context: %+v", item)
		}
		patch, _, err := c.recipePatch("malformed")
		if err != nil || len(patch) != 1 {
			t.Fatalf("optional malformed details lost recipe: %v %v", patch, err)
		}
	}
}
