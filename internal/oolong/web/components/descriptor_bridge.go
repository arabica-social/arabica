package tea

import (
	"fmt"

	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/lexicons"
	"tangled.org/arabica.social/arabica/internal/web/feedviews"

	// Ensure oolong descriptors are registered before we attach hooks.
	_ "tangled.org/arabica.social/arabica/internal/oolong/entities"
)

func FeedViews() feedviews.Registry {
	return feedviews.Registry{
		lexicons.RecordTypeOolongTea: {
			Render:  TeaFeedContent,
			EditURL: editPageURL("teas"),
		},
		lexicons.RecordTypeOolongVendor: {
			Render:       VendorFeedContent,
			Compact:      true,
			EditModalURL: modalEditURL("vendor"),
		},
		lexicons.RecordTypeOolongVessel: {
			Render:       VesselFeedContent,
			Compact:      true,
			EditModalURL: modalEditURL("vessel"),
		},
		lexicons.RecordTypeOolongInfuser: {
			Render:       InfuserFeedContent,
			Compact:      true,
			EditModalURL: modalEditURL("infuser"),
		},
		lexicons.RecordTypeOolongBrew: {
			Render:  BrewFeedContent,
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
