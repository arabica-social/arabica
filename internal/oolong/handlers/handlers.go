// Package teahandlers carries the oolong-app-specific HTTP handlers.
//
// Each handler hangs off *Handlers, which embeds *handlers.Handler so the
// shared infrastructure (auth, atproto store, witness cache, layout data,
// etc.) remains accessible via method promotion.
package teahandlers

import (
	"context"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/ogcard"
	tea "tangled.org/arabica.social/arabica/internal/oolong/web/components"
	"tangled.org/arabica.social/arabica/internal/records"
	"tangled.org/arabica.social/arabica/internal/web/pages"
)

// Handlers is the oolong-specific handler set. It embeds the shared
// *handlers.Handler so promoted methods give access to common helpers.
type Handlers struct {
	*handlers.Handler
}

// New constructs a Handlers wrapper over an already-configured base.
// The base handler is shared across all per-app handler sets in a binary.
func New(base *handlers.Handler) *Handlers {
	h := &Handlers{Handler: base}
	base.SetFeedViews(tea.FeedViews())
	base.SetFeedPresentation(handlers.FeedPresentation{
		EmptyState: pages.FeedEmptyState{
			Icon:  "🍵",
			Title: "The tea table's quiet today",
			Body:  "Follow some tea people, or log your first steep to get started.",
		},
	})
	base.SetHomeBehavior(handlers.HomeBehavior{
		OGDescription: "Tea journaling for the open social web. Log every steep, track your teaware, share your tea story.",
		SiteCardOpts: ogcard.SiteCardOpts{
			AppName:  "oolong",
			Wordmark: "Oolong",
			Tagline:  "tea journaling for the open social web",
			Detail:   "log every steep, share your tea story",
		},
		ReadinessChecker: func(ctx context.Context, store records.Store) (bool, error) {
			atpStore, ok := store.(*atproto.AtprotoStore)
			if !ok {
				return true, nil
			}
			return h.oolongReadyToBrew(ctx, atpStore)
		},
	})
	return h
}
