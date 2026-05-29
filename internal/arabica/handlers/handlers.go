// Package coffeehandlers carries the arabica-app-specific HTTP handlers.
//
// Each handler hangs off *Handlers, which embeds *handlers.Handler so the
// shared infrastructure (auth, atproto store, witness cache, layout data,
// etc.) remains accessible via method promotion.
package coffeehandlers

import (
	"context"
	"net/http"

	"tangled.org/arabica.social/arabica/internal/arabica/onboarding"
	arabicastore "tangled.org/arabica.social/arabica/internal/arabica/store"
	coffee "tangled.org/arabica.social/arabica/internal/arabica/web/components"
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/ogcard"
	"tangled.org/arabica.social/arabica/internal/records"
)

// Handlers is the arabica-specific handler set. It embeds the shared
// *handlers.Handler so promoted methods give access to common helpers.
type Handlers struct {
	*handlers.Handler
}

// GetArabicaStore returns the authenticated request's Arabica-typed store.
func (h *Handlers) GetArabicaStore(r *http.Request) (arabicastore.Store, bool) {
	store, ok := h.GetRecordStore(r)
	if !ok {
		return nil, false
	}
	arabicaStore, ok := store.(arabicastore.Store)
	if !ok {
		return nil, false
	}
	return arabicaStore, true
}

// New constructs a Handlers wrapper over an already-configured base.
// The base handler is shared across all per-app handler sets in a binary.
func New(base *handlers.Handler) *Handlers {
	base.SetFeedViews(coffee.FeedViews())
	base.SetHomeBehavior(handlers.HomeBehavior{
		OGDescription: "Coffee journaling for the open social web. Track, share, and own your brews.",
		SiteCardOpts: ogcard.SiteCardOpts{
			AppName:  "arabica",
			Wordmark: "arabica.social",
			Tagline:  "coffee journaling for the open social web",
			Detail:   "track, share, and own your brews",
		},
		ReadinessChecker: func(ctx context.Context, store records.Store) (bool, error) {
			arabicaStore, ok := store.(arabicastore.Store)
			if !ok {
				return true, nil
			}
			status, err := onboarding.CheckBrewReadiness(ctx, arabicaStore)
			if err != nil {
				return true, err
			}
			return status.Ready(), nil
		},
	})
	return &Handlers{Handler: base}
}
