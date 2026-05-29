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
			Render:      TeaFeedContent,
			FilterLabel: "Teas",
			EditURL:     editPageURL("teas"),
		},
		lexicons.RecordTypeOolongVendor: {
			Render:       VendorFeedContent,
			Compact:      true,
			FilterLabel:  "Vendors",
			EditModalURL: modalEditURL("vendor"),
		},
		lexicons.RecordTypeOolongVessel: {
			Render:       VesselFeedContent,
			Compact:      true,
			FilterLabel:  "Vessels",
			EditModalURL: modalEditURL("vessel"),
		},
		lexicons.RecordTypeOolongInfuser: {
			Render:       InfuserFeedContent,
			Compact:      true,
			FilterLabel:  "Infusers",
			EditModalURL: modalEditURL("infuser"),
		},
		lexicons.RecordTypeOolongBrew: {
			Render:      BrewFeedContent,
			FilterLabel: "Brews",
			EditURL:     editPageURL("brews"),
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
