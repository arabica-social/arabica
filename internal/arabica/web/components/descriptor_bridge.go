package coffee

import (
	"tangled.org/arabica.social/arabica/internal/lexicons"
	"tangled.org/arabica.social/arabica/internal/web/feedviews"

	// Ensure arabica descriptors are registered before we attach hooks.
	_ "tangled.org/arabica.social/arabica/internal/arabica/entities"
)

func FeedViews() feedviews.Registry {
	return feedviews.Registry{
		lexicons.RecordTypeBean:    {Render: BeanFeedContent},
		lexicons.RecordTypeRoaster: {Render: RoasterFeedContent, Compact: true},
		lexicons.RecordTypeGrinder: {Render: GrinderFeedContent, Compact: true},
		lexicons.RecordTypeBrewer:  {Render: BrewerFeedContent, Compact: true},
		lexicons.RecordTypeRecipe:  {Render: FeedRecipeContent},
		lexicons.RecordTypeBrew:    {Render: FeedBrewContentClickable},
	}
}
