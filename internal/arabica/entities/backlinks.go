package arabica

import (
	"strings"

	"tangled.org/arabica.social/arabica/internal/backlinks"
	"tangled.org/arabica.social/arabica/internal/suggestions"
)

func init() {
	backlinks.Register(NSIDBean, backlinks.EntityConfig{
		Noun:      "bean",
		AllFields: []string{"name", "origin", "roastLevel", "process"},
		DedupKey:  beanBacklinkKey,
		UsageRefs: []backlinks.UsageRef{{Collection: NSIDBrew, Field: "beanRef", Label: "brews"}},
	})
	backlinks.Register(NSIDRoaster, backlinks.EntityConfig{
		Noun:      "roaster",
		AllFields: []string{"name", "location", "website"},
		DedupKey:  roasterBacklinkKey,
		UsageRefs: []backlinks.UsageRef{{Collection: NSIDBean, Field: "roasterRef", Label: "beans"}},
	})
	backlinks.Register(NSIDGrinder, backlinks.EntityConfig{
		Noun:      "grinder",
		AllFields: []string{"name", "grinderType", "burrType"},
		DedupKey:  grinderBacklinkKey,
		UsageRefs: []backlinks.UsageRef{{Collection: NSIDBrew, Field: "grinderRef", Label: "brews"}},
	})
	backlinks.Register(NSIDBrewer, backlinks.EntityConfig{
		Noun:      "brewer",
		AllFields: []string{"name", "brewerType", "description"},
		DedupKey:  brewerBacklinkKey,
		UsageRefs: []backlinks.UsageRef{
			{Collection: NSIDBrew, Field: "brewerRef", Label: "brews"},
			{Collection: NSIDRecipe, Field: "brewerRef", Label: "recipes"},
		},
	})
	backlinks.Register(NSIDRecipe, backlinks.EntityConfig{
		Noun:      "recipe",
		AllFields: []string{"name", "brewerType"},
		DedupKey:  recipeBacklinkKey,
		UsageRefs: []backlinks.UsageRef{{Collection: NSIDBrew, Field: "recipeRef", Label: "brews"}},
	})
}

func roasterBacklinkKey(fields map[string]string) string {
	parts := []string{suggestions.FuzzyName(fields["name"])}
	if loc := suggestions.Normalize(fields["location"]); loc != "" {
		parts = append(parts, loc)
	}
	return strings.Join(parts, "|")
}

func grinderBacklinkKey(fields map[string]string) string {
	parts := []string{suggestions.Normalize(fields["name"])}
	if gt := suggestions.Normalize(fields["grinderType"]); gt != "" {
		parts = append(parts, gt)
	}
	if bt := suggestions.Normalize(fields["burrType"]); bt != "" {
		parts = append(parts, bt)
	}
	return strings.Join(parts, "|")
}

func brewerBacklinkKey(fields map[string]string) string {
	parts := []string{suggestions.Normalize(fields["name"])}
	if bt := suggestions.Normalize(fields["brewerType"]); bt != "" {
		parts = append(parts, bt)
	}
	return strings.Join(parts, "|")
}

func beanBacklinkKey(fields map[string]string) string {
	parts := []string{suggestions.Normalize(fields["name"])}
	if o := suggestions.Normalize(fields["origin"]); o != "" {
		parts = append(parts, o)
	}
	if p := suggestions.Normalize(fields["process"]); p != "" {
		parts = append(parts, p)
	}
	return strings.Join(parts, "|")
}

func recipeBacklinkKey(fields map[string]string) string {
	parts := []string{suggestions.Normalize(fields["name"])}
	if bt := suggestions.Normalize(fields["brewerType"]); bt != "" {
		parts = append(parts, bt)
	}
	return strings.Join(parts, "|")
}
