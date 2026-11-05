package suggestions

import (
	"context"
	"strings"
	"tangled.org/arabica.social/arabica/internal/matching"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
)

var entityConfigs = map[string]FieldConfig{
	arabica.NSIDRoaster: {
		AllFields:    []string{"name", "location", "website"},
		SearchFields: []string{"name", "location", "website"},
		NameField:    "name",
		DedupKey:     roasterDedupKey,
	},
	arabica.NSIDGrinder: {
		AllFields:    []string{"name", "grinderType", "burrType", "link"},
		SearchFields: []string{"name", "grinderType", "burrType"},
		NameField:    "name",
		DedupKey:     grinderDedupKey,
	},
	arabica.NSIDBrewer: {
		AllFields:    []string{"name", "brewerType", "description", "link"},
		SearchFields: []string{"name", "brewerType"},
		NameField:    "name",
		DedupKey:     brewerDedupKey,
	},
	arabica.NSIDBean: {
		AllFields:    []string{"name", "origin", "roastLevel", "process", "link"},
		SearchFields: []string{"name", "origin", "roastLevel"},
		NameField:    "name",
		DedupKey:     beanDedupKey,
		Prepare: func(ctx context.Context, source RecordSource) any {
			return BuildNameMap(ctx, source, arabica.NSIDRoaster)
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
	},
	arabica.NSIDRecipe: {
		AllFields:    []string{"name", "brewerType"},
		SearchFields: []string{"name", "brewerType"},
		NameField:    "name",
		DedupKey:     recipeDedupKey,
	},
}

// roasterDedupKey: fuzzy name + normalized location.
// Website is intentionally excluded — too sparse, would split duplicates.
func roasterDedupKey(fields map[string]string) string {
	parts := []string{matching.FuzzyName(fields["name"])}
	if loc := matching.Normalize(fields["location"]); loc != "" {
		parts = append(parts, loc)
	}
	return strings.Join(parts, "|")
}

func grinderDedupKey(fields map[string]string) string {
	parts := []string{matching.Normalize(fields["name"])}
	if gt := matching.Normalize(fields["grinderType"]); gt != "" {
		parts = append(parts, gt)
	}
	if bt := matching.Normalize(fields["burrType"]); bt != "" {
		parts = append(parts, bt)
	}
	return strings.Join(parts, "|")
}

func brewerDedupKey(fields map[string]string) string {
	parts := []string{matching.Normalize(fields["name"])}
	if bt := matching.Normalize(fields["brewerType"]); bt != "" {
		parts = append(parts, bt)
	}
	return strings.Join(parts, "|")
}

func beanDedupKey(fields map[string]string) string {
	parts := []string{matching.Normalize(fields["name"])}
	if o := matching.Normalize(fields["origin"]); o != "" {
		parts = append(parts, o)
	}
	if p := matching.Normalize(fields["process"]); p != "" {
		parts = append(parts, p)
	}
	return strings.Join(parts, "|")
}

func recipeDedupKey(fields map[string]string) string {
	parts := []string{matching.Normalize(fields["name"])}
	if bt := matching.Normalize(fields["brewerType"]); bt != "" {
		parts = append(parts, bt)
	}
	return strings.Join(parts, "|")
}
