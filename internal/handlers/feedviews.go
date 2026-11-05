package handlers

import (
	"tangled.org/arabica.social/arabica/internal/lexicons"
	"tangled.org/arabica.social/arabica/internal/web/feedviews"
)

// FeedViews returns Arabica's per-record-type JSON feed filter metadata.
// Roaster has no filter tab, so it is deliberately absent.
func FeedViews() feedviews.Registry {
	return feedviews.Registry{
		lexicons.RecordTypeBean: {
			FilterLabel: "Beans",
			FilterValue: "bean",
		},
		lexicons.RecordTypeGrinder: {
			FilterLabel: "Grinders",
			FilterValue: "grinder",
		},
		lexicons.RecordTypeBrewer: {
			FilterLabel: "Brewers",
			FilterValue: "brewer",
		},
		lexicons.RecordTypeRecipe: {
			FilterLabel: "Recipes",
			FilterValue: "recipe",
		},
		lexicons.RecordTypeBrew: {
			FilterLabel: "Brews",
			FilterValue: "brew",
		},
	}
}
