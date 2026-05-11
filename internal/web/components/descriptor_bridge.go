// Package components: descriptor_bridge.go wires arabica entities'
// templ render hooks into the entities.Descriptor registry.
//
// This file lives in internal/web/components/ during Phase 1 of the
// oolong-frontend split (docs/superpowers/specs/2026-05-11-oolong-frontend-split-design.md);
// it moves to internal/arabica/web/components/ in Phase 2 alongside
// the arabica-specific record_*.templ files.
package components

import (
	"fmt"

	"github.com/a-h/templ"

	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/lexicons"

	// Ensure arabica descriptors are registered before we attach hooks.
	_ "tangled.org/arabica.social/arabica/internal/entities/arabica"
)

func init() {
	if d := entities.Get(lexicons.RecordTypeBean); d != nil {
		d.RenderFeedContent = func(item any) templ.Component {
			return BeanFeedContent(item.(*feed.FeedItem))
		}
	}
	if d := entities.Get(lexicons.RecordTypeRoaster); d != nil {
		d.RenderFeedContent = func(item any) templ.Component {
			return RoasterFeedContent(item.(*feed.FeedItem))
		}
		d.FeedCardCompact = true
	}
	if d := entities.Get(lexicons.RecordTypeGrinder); d != nil {
		d.RenderFeedContent = func(item any) templ.Component {
			return GrinderFeedContent(item.(*feed.FeedItem))
		}
		d.FeedCardCompact = true
	}
	if d := entities.Get(lexicons.RecordTypeBrewer); d != nil {
		d.RenderFeedContent = func(item any) templ.Component {
			return BrewerFeedContent(item.(*feed.FeedItem))
		}
		d.FeedCardCompact = true
	}
	if d := entities.Get(lexicons.RecordTypeRecipe); d != nil {
		d.RenderFeedContent = func(item any) templ.Component {
			return FeedRecipeContent(item.(*feed.FeedItem))
		}
	}
	if d := entities.Get(lexicons.RecordTypeBrew); d != nil {
		d.RenderFeedContent = func(item any) templ.Component {
			return FeedBrewContentClickable(item.(*feed.FeedItem))
		}
		d.EditURL = func(item any) string {
			it := item.(*feed.FeedItem)
			if it.Brew() == nil {
				return ""
			}
			return fmt.Sprintf("/brews/%s/edit", it.Brew().RKey)
		}
	}
}
