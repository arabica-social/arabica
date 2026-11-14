package handlers

import (
	"tangled.org/arabica.social/arabica/internal/lexicons"
	"tangled.org/arabica.social/arabica/internal/web/feedviews"
)

// FeedViews returns Arabica's per-record-type JSON feed filter metadata.
func FeedViews() feedviews.Registry {
	return feedviews.Registry{
		lexicons.RecordTypeBean: {
			FilterLabel: "Beans",
			FilterValue: "bean",
		},
		lexicons.RecordTypeRoaster: {
			FilterLabel: "Roasters",
			FilterValue: "roaster",
		},
		lexicons.RecordTypeBrew: {
			FilterLabel: "Brews",
			FilterValue: "brew",
		},
	}
}
