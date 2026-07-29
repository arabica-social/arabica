package handlers

import (
	"context"
	"net/http"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	arabicastore "tangled.org/arabica.social/arabica/internal/arabica/store"
	"tangled.org/arabica.social/arabica/internal/handlers"

	"golang.org/x/sync/errgroup"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"
)

// ManageStatsJSON holds the entity usage counts and average ratings computed
// from the witness cache. All maps are keyed by AT-URI.
type ManageStatsJSON struct {
	BeanBrewCounts        map[string]int     `json:"bean_brew_counts"`
	GrinderBrewCounts     map[string]int     `json:"grinder_brew_counts"`
	BrewerBrewCounts      map[string]int     `json:"brewer_brew_counts"`
	RoasterBeanCounts     map[string]int     `json:"roaster_bean_counts"`
	BeanAvgBrewRatings    map[string]float64 `json:"bean_avg_brew_ratings"`
	RoasterAvgBrewRatings map[string]float64 `json:"roaster_avg_brew_ratings"`
}

// ManageResponseJSON is the JSON envelope returned by GET /api/manage. It
// combines all the user's records with witness-cache-derived usage stats in a
// single response, so the SvelteKit manage/my-coffee page can render without a
// second round-trip. See docs/api/manage.md for the contract.
type ManageResponseJSON struct {
	DID      string             `json:"did"`
	Beans    []*arabica.Bean    `json:"beans"`
	Roasters []*arabica.Roaster `json:"roasters"`
	Grinders []*arabica.Grinder `json:"grinders"`
	Brewers  []*arabica.Brewer  `json:"brewers"`
	Recipes  []*arabica.Recipe  `json:"recipes"`
	Stats    ManageStatsJSON    `json:"stats"`
}

// manageData holds the records fetched in parallel, shared by the HTML and
// JSON manage handlers.
type manageData struct {
	beans    []*arabica.Bean
	roasters []*arabica.Roaster
	grinders []*arabica.Grinder
	brewers  []*arabica.Brewer
	recipes  []*arabica.Recipe
}

// fetchManageData loads all entity collections in parallel, links beans to
// roasters and recipes to brewers, and returns the assembled bundle. Shared by
// HandleManagePartial (HTML) and HandleManageJSON (JSON).
func (h *Handlers) fetchManageData(r *http.Request) (*manageData, error) {
	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		return nil, errManageUnauth
	}

	// Invalidate the session cache so we read from the witness cache (or PDS)
	// rather than serving potentially stale in-memory data.
	if sessionID, ok := atpmiddleware.GetSessionID(r.Context()); ok {
		h.SessionCache().Invalidate(sessionID)
	}

	ctx := r.Context()
	g, ctx := errgroup.WithContext(ctx)

	data := &manageData{}
	g.Go(func() error {
		var err error
		data.beans, err = store.ListBeans(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		data.roasters, err = store.ListRoasters(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		data.grinders, err = listGrinders(ctx, store)
		return err
	})
	g.Go(func() error {
		var err error
		data.brewers, err = listBrewers(ctx, store)
		return err
	})
	g.Go(func() error {
		var err error
		data.recipes, err = store.ListRecipes(ctx)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Link beans to roasters and recipes to brewers.
	arabicastore.LinkBeansToRoasters(data.beans, data.roasters)

	brewerMap := make(map[string]*arabica.Brewer, len(data.brewers))
	for _, b := range data.brewers {
		brewerMap[b.RKey] = b
	}
	for _, recipe := range data.recipes {
		if recipe.BrewerRKey != "" {
			recipe.BrewerObj = brewerMap[recipe.BrewerRKey]
		}
	}

	return data, nil
}

// computeManageStats fetches entity usage counts and average ratings from the
// witness cache for the given owner DID.
func (h *Handlers) computeManageStats(ctx context.Context, did string) ManageStatsJSON {
	stats := ManageStatsJSON{
		BeanBrewCounts:    map[string]int{},
		GrinderBrewCounts: map[string]int{},
		BrewerBrewCounts:  map[string]int{},
		RoasterBeanCounts: map[string]int{},
	}
	if h.FeedIndex() == nil || did == "" {
		return stats
	}
	stats.BeanBrewCounts = h.FeedIndex().BrewCountsByBeanURI(ctx, did)
	stats.GrinderBrewCounts = h.FeedIndex().BrewCountsByGrinderURI(ctx, did)
	stats.BrewerBrewCounts = h.FeedIndex().BrewCountsByBrewerURI(ctx, did)
	stats.RoasterBeanCounts = h.FeedIndex().BeanCountsByRoasterURI(ctx, did)
	stats.BeanAvgBrewRatings = make(map[string]float64, len(stats.BeanBrewCounts))
	for uri, s := range h.FeedIndex().AvgBrewRatingByBeanURI(ctx, did) {
		stats.BeanAvgBrewRatings[uri] = s.Average
	}
	stats.RoasterAvgBrewRatings = make(map[string]float64, len(stats.RoasterBeanCounts))
	for uri, s := range h.FeedIndex().AvgBrewRatingByRoasterURI(ctx, did) {
		stats.RoasterAvgBrewRatings[uri] = s.Average
	}
	return stats
}

// HandleManageAPI serves the manage page data as JSON for the SvelteKit SPA.
// The SPA always sends Accept: application/json.
func (h *Handlers) HandleManageAPI(w http.ResponseWriter, r *http.Request) {
	h.HandleManageJSON(w, r)
}

// HandleManageJSON returns the user's records + usage stats as JSON for the
// SvelteKit SPA. This is the heavier counterpart to /api/data (which returns
// raw records without stats for the lightweight appCache use case).
func (h *Handlers) HandleManageJSON(w http.ResponseWriter, r *http.Request) {
	data, err := h.fetchManageData(r)
	if err != nil {
		if err == errManageUnauth {
			handlers.WriteJSONError(w, http.StatusUnauthorized, "authentication_required", "Authentication required")
			return
		}
		handlers.HandleStoreJSONError(w, err, "Failed to fetch data")
		return
	}

	did, _ := atpmiddleware.GetDID(r.Context())
	stats := h.computeManageStats(r.Context(), did)

	handlers.WriteJSON(w, ManageResponseJSON{
		DID:      did,
		Beans:    data.beans,
		Roasters: data.roasters,
		Grinders: data.grinders,
		Brewers:  data.brewers,
		Recipes:  data.recipes,
		Stats:    stats,
	}, "manage")
}

// errManageUnauth is a sentinel returned by fetchManageData when the request
// is unauthenticated, so the JSON handler can map it to a 401 without a type
// assertion.
var errManageUnauth = &manageError{msg: "Authentication required"}

type manageError struct{ msg string }

func (e *manageError) Error() string { return e.msg }
