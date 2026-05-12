package arabica

import (
	"context"
	"strings"

	"tangled.org/arabica.social/arabica/internal/suggestions"
)

// init wires the arabica entity collections into the suggestions
// registry. Lives in the arabica entity package — not in suggestions —
// so cmd/oolong's binary doesn't have to drag in arabica's NSID
// constants or dedup logic.
func init() {
	suggestions.Register(NSIDRoaster, suggestions.FieldConfig{
		AllFields:    []string{"name", "location", "website"},
		SearchFields: []string{"name", "location", "website"},
		NameField:    "name",
		DedupKey:     roasterDedupKey,
	})
	suggestions.Register(NSIDGrinder, suggestions.FieldConfig{
		AllFields:    []string{"name", "grinderType", "burrType"},
		SearchFields: []string{"name", "grinderType", "burrType"},
		NameField:    "name",
		DedupKey:     grinderDedupKey,
	})
	suggestions.Register(NSIDBrewer, suggestions.FieldConfig{
		AllFields:    []string{"name", "brewerType", "description"},
		SearchFields: []string{"name", "brewerType"},
		NameField:    "name",
		DedupKey:     brewerDedupKey,
	})
	suggestions.Register(NSIDBean, suggestions.FieldConfig{
		AllFields:    []string{"name", "origin", "roastLevel", "process"},
		SearchFields: []string{"name", "origin", "roastLevel"},
		NameField:    "name",
		DedupKey:     beanDedupKey,
		Prepare: func(ctx context.Context, source suggestions.RecordSource) any {
			return suggestions.BuildNameMap(ctx, source, NSIDRoaster)
		},
		Enrich: func(prepared any, record map[string]any, fields map[string]string) {
			names, _ := prepared.(map[string]string)
			if names == nil {
				return
			}
			ref, ok := record["roasterRef"].(string)
			if !ok || ref == "" {
				return
			}
			if rn, ok := names[ref]; ok {
				fields["roasterName"] = rn
			}
		},
	})
	suggestions.Register(NSIDRecipe, suggestions.FieldConfig{
		AllFields:    []string{"name", "brewerType"},
		SearchFields: []string{"name", "brewerType"},
		NameField:    "name",
		DedupKey:     recipeDedupKey,
	})
}

// roasterDedupKey: fuzzy name + normalized location.
// Website is intentionally excluded — too sparse, would split duplicates.
func roasterDedupKey(fields map[string]string) string {
	parts := []string{suggestions.FuzzyName(fields["name"])}
	if loc := suggestions.Normalize(fields["location"]); loc != "" {
		parts = append(parts, loc)
	}
	return strings.Join(parts, "|")
}

func grinderDedupKey(fields map[string]string) string {
	parts := []string{suggestions.Normalize(fields["name"])}
	if gt := suggestions.Normalize(fields["grinderType"]); gt != "" {
		parts = append(parts, gt)
	}
	if bt := suggestions.Normalize(fields["burrType"]); bt != "" {
		parts = append(parts, bt)
	}
	return strings.Join(parts, "|")
}

func brewerDedupKey(fields map[string]string) string {
	parts := []string{suggestions.Normalize(fields["name"])}
	if bt := suggestions.Normalize(fields["brewerType"]); bt != "" {
		parts = append(parts, bt)
	}
	return strings.Join(parts, "|")
}

func beanDedupKey(fields map[string]string) string {
	parts := []string{suggestions.Normalize(fields["name"])}
	if o := suggestions.Normalize(fields["origin"]); o != "" {
		parts = append(parts, o)
	}
	if p := suggestions.Normalize(fields["process"]); p != "" {
		parts = append(parts, p)
	}
	return strings.Join(parts, "|")
}

func recipeDedupKey(fields map[string]string) string {
	parts := []string{suggestions.Normalize(fields["name"])}
	if bt := suggestions.Normalize(fields["brewerType"]); bt != "" {
		parts = append(parts, bt)
	}
	return strings.Join(parts, "|")
}
