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
		AllFields:    []string{"name", "category", "origin"},
		SearchFields: []string{"name", "category", "origin"},
		NameField:    "name",
		DedupKey:     teaDedupKey,
	})
	suggestions.Register(NSIDVessel, suggestions.FieldConfig{
		AllFields:    []string{"name", "style", "material"},
		SearchFields: []string{"name", "style"},
		NameField:    "name",
		DedupKey:     vesselDedupKey,
	})
	suggestions.Register(NSIDInfuser, suggestions.FieldConfig{
		AllFields:    []string{"name", "style", "material"},
		SearchFields: []string{"name", "style"},
		NameField:    "name",
		DedupKey:     infuserDedupKey,
	})
	suggestions.Register(NSIDVendor, suggestions.FieldConfig{
		AllFields:    []string{"name", "location", "website"},
		SearchFields: []string{"name", "location"},
		NameField:    "name",
		DedupKey:     vendorDedupKey,
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

func vesselDedupKey(fields map[string]string) string {
	parts := []string{suggestions.Normalize(fields["name"])}
	if s := suggestions.Normalize(fields["style"]); s != "" {
		parts = append(parts, s)
	}
	return strings.Join(parts, "|")
}

func infuserDedupKey(fields map[string]string) string {
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
