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
			Render:        TeaFeedContent,
			CardClassNoun: "tea",
			FilterLabel:   "Teas",
			ShareURL:      shareURL("teas"),
			DeleteURL:     apiDeleteURL("teas"),
			EditURL:       editPageURL("teas"),
		},
		lexicons.RecordTypeOolongVendor: {
			Render:        VendorFeedContent,
			Compact:       true,
			CardClassNoun: "vendor",
			FilterLabel:   "Vendors",
			ShareURL:      shareURL("vendors"),
			DeleteURL:     apiDeleteURL("vendors"),
			EditModalURL:  modalEditURL("vendor"),
		},
		lexicons.RecordTypeOolongVessel: {
			Render:        VesselFeedContent,
			Compact:       true,
			CardClassNoun: "vessel",
			FilterLabel:   "Vessels",
			ShareURL:      shareURL("vessels"),
			DeleteURL:     apiDeleteURL("vessels"),
			EditModalURL:  modalEditURL("vessel"),
		},
		lexicons.RecordTypeOolongInfuser: {
			Render:        InfuserFeedContent,
			Compact:       true,
			CardClassNoun: "infuser",
			FilterLabel:   "Infusers",
			ShareURL:      shareURL("infusers"),
			DeleteURL:     apiDeleteURL("infusers"),
			EditModalURL:  modalEditURL("infuser"),
		},
		lexicons.RecordTypeOolongBrew: {
			Render:        BrewFeedContent,
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
