package coffee

import (
	"fmt"

	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/lexicons"
	"tangled.org/arabica.social/arabica/internal/web/feedviews"

	// Ensure arabica descriptors are registered before we attach hooks.
	_ "tangled.org/arabica.social/arabica/internal/arabica/entities"
)

// FeedViews returns the per-record-type feed view metadata for Arabica. Only
// the metadata needed by the JSON feed path (filter labels, action URLs,
// compactness) is populated; the legacy templ Render/RenderPrefs renderers
// were removed with the templ stack.
func FeedViews() feedviews.Registry {
	return feedviews.Registry{
		lexicons.RecordTypeBean: {
			CardClassNoun: "bean",
			FilterLabel:   "Beans",
			ShareURL:      shareURL("beans"),
			DeleteURL:     apiDeleteURL("beans"),
			EditModalURL:  modalEditURL("bean"),
		},
		lexicons.RecordTypeRoaster: {
			Compact:       true,
			CardClassNoun: "roaster",
			ShareURL:      shareURL("roasters"),
			DeleteURL:     apiDeleteURL("roasters"),
			EditModalURL:  modalEditURL("roaster"),
		},
		lexicons.RecordTypeGrinder: {
			Compact:       true,
			CardClassNoun: "grinder",
			FilterLabel:   "Grinders",
			ShareURL:      shareURL("grinders"),
			DeleteURL:     apiDeleteURL("grinders"),
			EditModalURL:  modalEditURL("grinder"),
		},
		lexicons.RecordTypeBrewer: {
			Compact:       true,
			CardClassNoun: "brewer",
			FilterLabel:   "Brewers",
			ShareURL:      shareURL("brewers"),
			DeleteURL:     apiDeleteURL("brewers"),
			EditModalURL:  modalEditURL("brewer"),
		},
		lexicons.RecordTypeRecipe: {
			CardClassNoun: "recipe",
			FilterLabel:   "Recipes",
			ShareURL:      shareURL("recipes"),
			DeleteURL:     apiDeleteURL("recipes"),
			EditModalURL:  modalEditURL("recipe"),
		},
		lexicons.RecordTypeBrew: {
			CardClassNoun: "brew",
			FilterLabel:   "Brews",
			ShareURL:      shareURL("brews"),
			DeleteURL:     pageDeleteURL("brews"),
			EditURL:       editPageURL("brews"),
		},
	}
}

func shareURL(path string) feedviews.ActionURL {
	return func(item *feed.FeedItem) string {
		if rkey := item.RKey(); rkey != "" {
			return fmt.Sprintf("/%s/%s/%s", path, feedviews.Actor(item), rkey)
		}
		return ""
	}
}

func apiDeleteURL(path string) feedviews.ActionURL {
	return func(item *feed.FeedItem) string {
		if rkey := item.RKey(); rkey != "" {
			return fmt.Sprintf("/api/%s/%s", path, rkey)
		}
		return ""
	}
}

func pageDeleteURL(path string) feedviews.ActionURL {
	return func(item *feed.FeedItem) string {
		if rkey := item.RKey(); rkey != "" {
			return fmt.Sprintf("/%s/%s", path, rkey)
		}
		return ""
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
