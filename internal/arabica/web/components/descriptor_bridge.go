package coffee

import (
	"fmt"

	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/lexicons"
	"tangled.org/arabica.social/arabica/internal/web/feedviews"

	// Ensure arabica descriptors are registered before we attach hooks.
	_ "tangled.org/arabica.social/arabica/internal/arabica/entities"
)

func FeedViews() feedviews.Registry {
	return feedviews.Registry{
		lexicons.RecordTypeBean: {
			Render:       BeanFeedContent,
			EditModalURL: modalEditURL("bean"),
		},
		lexicons.RecordTypeRoaster: {
			Render:       RoasterFeedContent,
			Compact:      true,
			EditModalURL: modalEditURL("roaster"),
		},
		lexicons.RecordTypeGrinder: {
			Render:       GrinderFeedContent,
			Compact:      true,
			EditModalURL: modalEditURL("grinder"),
		},
		lexicons.RecordTypeBrewer: {
			Render:       BrewerFeedContent,
			Compact:      true,
			EditModalURL: modalEditURL("brewer"),
		},
		lexicons.RecordTypeRecipe: {
			Render:       FeedRecipeContent,
			EditModalURL: modalEditURL("recipe"),
		},
		lexicons.RecordTypeBrew: {
			Render:  FeedBrewContentClickable,
			EditURL: editPageURL("brews"),
		},
	}
}

func modalEditURL(noun string) feedviews.ActionURL {
	return func(item *feed.FeedItem) string {
		if rkey := item.RKey(); rkey != "" {
			return fmt.Sprintf("/api/modals/%s/%s", noun, rkey)
		}
		return ""
	}
}

func editPageURL(path string) feedviews.ActionURL {
	return func(item *feed.FeedItem) string {
		if rkey := item.RKey(); rkey != "" {
			return fmt.Sprintf("/%s/%s/edit", path, rkey)
		}
		return ""
	}
}
