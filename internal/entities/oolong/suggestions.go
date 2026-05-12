package oolong

import (
	"strings"

	"tangled.org/arabica.social/arabica/internal/suggestions"
)

// init wires the oolong entity collections into the suggestions
// registry. Lives in oolong's entity package so cmd/arabica's binary
// can stay free of oolong's NSID constants and dedup logic.
func init() {
	suggestions.Register(NSIDTea, suggestions.FieldConfig{
		AllFields:    []string{"name", "category", "subStyle", "origin", "cultivar"},
		SearchFields: []string{"name", "category", "origin", "cultivar"},
		NameField:    "name",
		DedupKey:     teaDedupKey,
	})
	suggestions.Register(NSIDBrewer, suggestions.FieldConfig{
		AllFields:    []string{"name", "style", "material"},
		SearchFields: []string{"name", "style"},
		NameField:    "name",
		DedupKey:     teaBrewerDedupKey,
	})
	suggestions.Register(NSIDRecipe, suggestions.FieldConfig{
		AllFields:    []string{"name", "style"},
		SearchFields: []string{"name", "style"},
		NameField:    "name",
		DedupKey:     teaRecipeDedupKey,
	})
	suggestions.Register(NSIDVendor, suggestions.FieldConfig{
		AllFields:    []string{"name", "location", "website"},
		SearchFields: []string{"name", "location"},
		NameField:    "name",
		DedupKey:     vendorDedupKey,
	})
	suggestions.Register(NSIDCafe, suggestions.FieldConfig{
		AllFields:    []string{"name", "location", "website"},
		SearchFields: []string{"name", "location"},
		NameField:    "name",
		DedupKey:     teaCafeDedupKey,
	})
}

// teaDedupKey: name + category + origin. Different origins keep
// same-named teas distinct (a Da Hong Pao from Wuyi vs Anxi reads
// as two records, not one).
func teaDedupKey(fields map[string]string) string {
	parts := []string{suggestions.Normalize(fields["name"])}
	if cat := suggestions.Normalize(fields["category"]); cat != "" {
		parts = append(parts, cat)
	}
	if o := suggestions.Normalize(fields["origin"]); o != "" {
		parts = append(parts, o)
	}
	return strings.Join(parts, "|")
}

func teaBrewerDedupKey(fields map[string]string) string {
	parts := []string{suggestions.Normalize(fields["name"])}
	if s := suggestions.Normalize(fields["style"]); s != "" {
		parts = append(parts, s)
	}
	return strings.Join(parts, "|")
}

func teaRecipeDedupKey(fields map[string]string) string {
	parts := []string{suggestions.Normalize(fields["name"])}
	if s := suggestions.Normalize(fields["style"]); s != "" {
		parts = append(parts, s)
	}
	return strings.Join(parts, "|")
}

func vendorDedupKey(fields map[string]string) string {
	parts := []string{suggestions.FuzzyName(fields["name"])}
	if loc := suggestions.Normalize(fields["location"]); loc != "" {
		parts = append(parts, loc)
	}
	return strings.Join(parts, "|")
}

// teaCafeDedupKey: exact name + location (cafes are physical, two
// branches of the same chain in different cities aren't dupes).
func teaCafeDedupKey(fields map[string]string) string {
	parts := []string{suggestions.Normalize(fields["name"])}
	if loc := suggestions.Normalize(fields["location"]); loc != "" {
		parts = append(parts, loc)
	}
	return strings.Join(parts, "|")
}
