// Package tea: descriptor_bridge.go wires oolong entities' templ render
// hooks into the entities.Descriptor registry. Importing this package
// for side effects (blank import in cmd/server) runs init() to populate
// hooks for all oolong record types. Mirrors the arabica equivalent in
// internal/arabica/web/components/descriptor_bridge.go.
package tea

import (
	"fmt"

	"github.com/a-h/templ"

	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/lexicons"

	// Ensure oolong descriptors are registered before we attach hooks.
	_ "tangled.org/arabica.social/arabica/internal/entities/oolong"
)

func init() {
	if d := entities.Get(lexicons.RecordTypeOolongTea); d != nil {
		d.RenderFeedContent = func(item any) templ.Component {
			return TeaFeedContent(item.(*feed.FeedItem))
		}
	}
	if d := entities.Get(lexicons.RecordTypeOolongVendor); d != nil {
		d.RenderFeedContent = func(item any) templ.Component {
			return VendorFeedContent(item.(*feed.FeedItem))
		}
		d.FeedCardCompact = true
	}
	if d := entities.Get(lexicons.RecordTypeOolongVessel); d != nil {
		d.RenderFeedContent = func(item any) templ.Component {
			return VesselFeedContent(item.(*feed.FeedItem))
		}
		d.FeedCardCompact = true
	}
	if d := entities.Get(lexicons.RecordTypeOolongInfuser); d != nil {
		d.RenderFeedContent = func(item any) templ.Component {
			return InfuserFeedContent(item.(*feed.FeedItem))
		}
		d.FeedCardCompact = true
	}
	if d := entities.Get(lexicons.RecordTypeOolongBrew); d != nil {
		d.RenderFeedContent = func(item any) templ.Component {
			return BrewFeedContent(item.(*feed.FeedItem))
		}
		d.EditURL = func(item any) string {
			it := item.(*feed.FeedItem)
			rkey := it.RKey()
			if rkey == "" {
				return ""
			}
			return fmt.Sprintf("/brews/%s/edit", rkey)
		}
	}
	// Cafe and Drink hooks are wired but the descriptors aren't registered
	// for v1, so the if-blocks no-op. Re-enable when those entities ship.
}
