package coffeehandlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	arabicastore "tangled.org/arabica.social/arabica/internal/arabica/store"
	coffee "tangled.org/arabica.social/arabica/internal/arabica/web/components"
	coffeepages "tangled.org/arabica.social/arabica/internal/arabica/web/pages"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/records"
	"tangled.org/arabica.social/arabica/internal/tracing"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/errgroup"
	"tangled.org/pdewey.com/atp"
)

// Manage page partial (loaded async via HTMX)
func (h *Handlers) HandleManagePartial(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Invalidate the session cache so we read from the witness cache (or PDS)
	// rather than serving potentially stale in-memory data. The witness cache
	// is a local SQLite read so this is cheap.
	if sessionID, ok := atpmiddleware.GetSessionID(r.Context()); ok {
		h.SessionCache().Invalidate(sessionID)
	}

	ctx := r.Context()

	// Fetch all collections in parallel using errgroup for proper error handling
	// and automatic context cancellation on first error
	g, ctx := errgroup.WithContext(ctx)

	var beans []*arabica.Bean
	var roasters []*arabica.Roaster
	var grinders []*arabica.Grinder
	var brewers []*arabica.Brewer
	var recipes []*arabica.Recipe

	g.Go(func() error {
		var err error
		beans, err = store.ListBeans(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		roasters, err = store.ListRoasters(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		grinders, err = listGrinders(ctx, store)
		return err
	})
	g.Go(func() error {
		var err error
		brewers, err = listBrewers(ctx, store)
		return err
	})
	g.Go(func() error {
		var err error
		recipes, err = store.ListRecipes(ctx)
		return err
	})

	if err := g.Wait(); err != nil {
		log.Error().Err(err).Msg("Failed to fetch manage page data")
		handlers.HandleStoreError(w, err, "Failed to fetch data")
		return
	}

	// Link beans to their roasters
	arabicastore.LinkBeansToRoasters(beans, roasters)

	// Link recipes to their brewers
	brewerMap := make(map[string]*arabica.Brewer, len(brewers))
	for _, b := range brewers {
		brewerMap[b.RKey] = b
	}
	for _, recipe := range recipes {
		if recipe.BrewerRKey != "" {
			recipe.BrewerObj = brewerMap[recipe.BrewerRKey]
		}
	}

	// Fetch entity usage counts and avg ratings from witness cache
	props := coffee.ManagePartialProps{
		Beans:    beans,
		Roasters: roasters,
		Grinders: grinders,
		Brewers:  brewers,
		Recipes:  recipes,
	}
	if h.FeedIndex() != nil {
		did, _ := atpmiddleware.GetDID(r.Context())
		props.OwnerDID = did
		props.BeanBrewCounts = h.FeedIndex().BrewCountsByBeanURI(r.Context(), did)
		props.GrinderBrewCounts = h.FeedIndex().BrewCountsByGrinderURI(r.Context(), did)
		props.BrewerBrewCounts = h.FeedIndex().BrewCountsByBrewerURI(r.Context(), did)
		props.RoasterBeanCounts = h.FeedIndex().BeanCountsByRoasterURI(r.Context(), did)
		props.BeanAvgBrewRatings = make(map[string]float64)
		for uri, stats := range h.FeedIndex().AvgBrewRatingByBeanURI(r.Context(), did) {
			props.BeanAvgBrewRatings[uri] = stats.Average
		}
		props.RoasterAvgBrewRatings = make(map[string]float64)
		for uri, stats := range h.FeedIndex().AvgBrewRatingByRoasterURI(r.Context(), did) {
			props.RoasterAvgBrewRatings[uri] = stats.Average
		}
	}

	// Render manage partial
	if err := coffee.ManagePartial(props).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render content", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render manage partial")
	}
}

// API endpoint to list all user data (beans, roasters, grinders, brewers, brews)
// Used by client-side cache for faster page loads. Arabica-specific —
// oolong has its own /api/data handler registered via teahandlers.
func (h *Handlers) HandleAPIListAll(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Get user DID for cache validation
	userDID, _ := atpmiddleware.GetDID(r.Context())

	ctx := r.Context()

	// Fetch all collections in parallel using errgroup
	g, ctx := errgroup.WithContext(ctx)

	var beans []*arabica.Bean
	var roasters []*arabica.Roaster
	var grinders []*arabica.Grinder
	var brewers []*arabica.Brewer
	var recipes []*arabica.Recipe
	var brews []*arabica.Brew

	g.Go(func() error {
		var err error
		beans, err = store.ListBeans(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		roasters, err = store.ListRoasters(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		grinders, err = listGrinders(ctx, store)
		return err
	})
	g.Go(func() error {
		var err error
		brewers, err = listBrewers(ctx, store)
		return err
	})
	g.Go(func() error {
		var err error
		recipes, err = store.ListRecipes(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		brews, err = store.ListBrews(ctx, 1, 0, 0) // limit=0 returns all
		return err
	})

	if err := g.Wait(); err != nil {
		log.Error().Err(err).Msg("Failed to fetch all data for API")
		handlers.HandleStoreError(w, err, "Failed to fetch data")
		return
	}

	// Link beans to roasters
	arabicastore.LinkBeansToRoasters(beans, roasters)

	response := map[string]any{
		"did":      userDID,
		"beans":    beans,
		"roasters": roasters,
		"grinders": grinders,
		"brewers":  brewers,
		"recipes":  recipes,
		"brews":    brews,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().Err(err).Msg("Failed to encode API response")
	}
}

// API endpoint to create bean
func (h *Handlers) HandleBeanCreate(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req arabica.CreateBeanRequest

	// Decode request (JSON or form)
	if err := handlers.DecodeRequest(r, &req, func() error {
		req = arabica.CreateBeanRequest{
			Name:        r.FormValue("name"),
			Origin:      r.FormValue("origin"),
			Variety:     r.FormValue("variety"),
			RoastLevel:  r.FormValue("roast_level"),
			Process:     r.FormValue("process"),
			Description: r.FormValue("description"),
			Link:        r.FormValue("link"),
			RoasterRKey: r.FormValue("roaster_rkey"),
			Rating:      handlers.ParseOptionalInt(r.FormValue("rating")),
			Closed:      r.FormValue("closed") == "true",
			SourceRef:   r.FormValue("source_ref"),
		}
		log.Debug().
			Str("name", req.Name).
			Str("closed_value", r.FormValue("closed")).
			Bool("closed_parsed", req.Closed).
			Msg("Parsed bean create form")
		return nil
	}); err != nil {
		log.Warn().Err(err).Msg("Failed to decode bean create request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		log.Warn().Err(err).Str("name", req.Name).Msg("Bean create validation failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// If a new roaster name was provided and no existing roaster selected, create it
	if newRoasterName := r.FormValue("new_roaster_name"); newRoasterName != "" && req.RoasterRKey == "" {
		roaster, roasterErr := store.CreateRoaster(r.Context(), &arabica.CreateRoasterRequest{
			Name:     newRoasterName,
			Location: r.FormValue("new_roaster_location"),
			Website:  r.FormValue("new_roaster_website"),
		})
		if roasterErr != nil {
			log.Error().Err(roasterErr).Str("name", newRoasterName).Msg("Failed to create roaster for bean")
			handlers.HandleStoreError(w, roasterErr, "Failed to create roaster")
			return
		}
		req.RoasterRKey = roaster.RKey
		log.Info().Str("roaster_rkey", roaster.RKey).Str("name", newRoasterName).Msg("Auto-created roaster for bean")
	}

	// Validate optional roaster rkey
	if errMsg := handlers.ValidateOptionalRKey(req.RoasterRKey, "Roaster selection"); errMsg != "" {
		log.Warn().Str("roaster_rkey", req.RoasterRKey).Msg("Bean create: invalid roaster rkey")
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	bean, err := store.CreateBean(r.Context(), &req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create bean")
		handlers.HandleStoreError(w, err, "Failed to create bean")
		return
	}

	h.InvalidateFeedCache()
	handlers.WriteJSON(w, bean, "bean")
}

// API endpoint to create roaster
func (h *Handlers) HandleRoasterCreate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.RequireRecordStore(w, r)
	if !ok {
		return
	}
	handlers.RecordCRUDWrite[arabica.CreateRoasterRequest, *arabica.CreateRoasterRequest, arabica.Roaster](
		w, r, store, arabica.NSIDRoaster, "roaster", "", decodeRoasterCreateForm,
		func(req *arabica.CreateRoasterRequest) *arabica.Roaster {
			return roasterFromCreate(req, time.Now())
		},
		func(m *arabica.Roaster, rkey string) { m.RKey = rkey },
		func(_ records.Store, _ *arabica.CreateRoasterRequest, m *arabica.Roaster) (map[string]any, error) {
			return arabica.RoasterToRecord(m)
		},
		h.InvalidateFeedCache, false,
	)
}

// Manage page
func (h *Handlers) HandleManage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/my-coffee", http.StatusMovedPermanently)
}

// HandleMyCoffee renders the unified My Coffee page (replaces both /brews and /manage)
func (h *Handlers) HandleMyCoffee(w http.ResponseWriter, r *http.Request) {
	_, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	layoutData, _, _ := h.LayoutDataFromRequest(r, "My Coffee")

	if err := coffeepages.MyCoffee(layoutData, coffeepages.MyCoffeeProps{}).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render my coffee page")
	}
}

// HandleIncompleteRecordsPartial returns an HTML fragment for incomplete records on the home dashboard.
func (h *Handlers) HandleIncompleteRecordsPartial(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		return
	}

	ctx := r.Context()
	g, ctx := errgroup.WithContext(ctx)

	var beans []*arabica.Bean
	var grinders []*arabica.Grinder
	var brewers []*arabica.Brewer

	g.Go(func() error {
		var err error
		beans, err = store.ListBeans(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		grinders, err = listGrinders(ctx, store)
		return err
	})
	g.Go(func() error {
		var err error
		brewers, err = listBrewers(ctx, store)
		return err
	})

	if err := g.Wait(); err != nil {
		log.Error().Err(err).Msg("Failed to fetch data for incomplete records")
		return
	}

	records := coffee.CollectIncompleteRecords(beans, grinders, brewers, 5)

	if err := coffee.IncompleteRecords(coffee.IncompleteRecordsProps{
		Records: records,
	}).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render incomplete records")
	}
}

// HandleManageRefresh invalidates all caches and re-fetches records from the
// user's PDS, writing them through to the witness cache so subsequent reads
// are up to date. Returns the refreshed manage partial.
func (h *Handlers) HandleManageRefresh(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	sessionID, ok := atpmiddleware.GetSessionID(r.Context())
	if !ok {
		http.Error(w, "Session required", http.StatusUnauthorized)
		return
	}

	didStr, ok := atpmiddleware.GetDID(r.Context())
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	did, err := syntax.ParseDID(didStr)
	if err != nil {
		http.Error(w, "Invalid DID", http.StatusInternalServerError)
		return
	}

	// Nuke the entire session cache so List* calls fall through to PDS
	h.SessionCache().Invalidate(sessionID)

	// Re-fetch all entity collections from PDS and write-through to witness
	entityCollections := []string{
		arabica.NSIDBean, arabica.NSIDRoaster, arabica.NSIDGrinder,
		arabica.NSIDBrewer, arabica.NSIDRecipe, arabica.NSIDBrew,
	}

	if h.WitnessCache() != nil {
		refreshCtx, refreshSpan := tracing.HandlerSpan(r.Context(), "manage.refresh.witness_sync",
			attribute.String("user.did", didStr),
		)
		atpClient, err := h.AtprotoClient().AtpClient(refreshCtx, did, sessionID)
		if err != nil {
			log.Warn().Err(err).Msg("refresh: failed to get atp client")
			refreshSpan.End()
			return
		}
		var batch []atproto.WitnessWriteRecord
		for _, collection := range entityCollections {
			records, err := atpClient.ListAllRecords(refreshCtx, collection)
			if err != nil {
				log.Warn().Err(err).Str("collection", collection).Msg("refresh: failed to list records from PDS")
				continue
			}
			for _, rec := range records {
				rkey := atp.RKeyFromURI(rec.URI)
				if rkey == "" {
					continue
				}
				recordJSON, jsonErr := json.Marshal(rec.Value)
				if jsonErr != nil {
					continue
				}
				batch = append(batch, atproto.WitnessWriteRecord{
					DID:        didStr,
					Collection: collection,
					RKey:       rkey,
					CID:        rec.CID,
					Record:     recordJSON,
				})
			}
			short := collection[strings.LastIndex(collection, ".")+1:]
			log.Info().Str("collection", short).Int("count", len(records)).Msg("refresh: fetched collection from PDS")
		}
		if err := h.WitnessCache().UpsertWitnessRecordBatch(refreshCtx, batch); err != nil {
			log.Error().Err(err).Msg("refresh: failed to batch upsert records")
		}
		refreshSpan.SetAttributes(attribute.Int("refresh.total_records", len(batch)))
		refreshSpan.End()
	}

	// Now fetch and render the manage partial with fresh PDS data
	ctx := r.Context()
	g, ctx := errgroup.WithContext(ctx)

	var beans []*arabica.Bean
	var roasters []*arabica.Roaster
	var grinders []*arabica.Grinder
	var brewers []*arabica.Brewer
	var recipes []*arabica.Recipe

	g.Go(func() error {
		var err error
		beans, err = store.ListBeans(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		roasters, err = store.ListRoasters(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		grinders, err = listGrinders(ctx, store)
		return err
	})
	g.Go(func() error {
		var err error
		brewers, err = listBrewers(ctx, store)
		return err
	})
	g.Go(func() error {
		var err error
		recipes, err = store.ListRecipes(ctx)
		return err
	})

	if err := g.Wait(); err != nil {
		log.Error().Err(err).Msg("Failed to fetch manage page data after refresh")
		handlers.HandleStoreError(w, err, "Failed to fetch data")
		return
	}

	arabicastore.LinkBeansToRoasters(beans, roasters)

	brewerMap := make(map[string]*arabica.Brewer, len(brewers))
	for _, b := range brewers {
		brewerMap[b.RKey] = b
	}
	for _, recipe := range recipes {
		if recipe.BrewerRKey != "" {
			recipe.BrewerObj = brewerMap[recipe.BrewerRKey]
		}
	}

	refreshProps := coffee.ManagePartialProps{
		Beans:    beans,
		Roasters: roasters,
		Grinders: grinders,
		Brewers:  brewers,
		Recipes:  recipes,
	}
	if h.FeedIndex() != nil {
		refreshProps.OwnerDID = didStr
		refreshProps.BeanBrewCounts = h.FeedIndex().BrewCountsByBeanURI(r.Context(), didStr)
		refreshProps.GrinderBrewCounts = h.FeedIndex().BrewCountsByGrinderURI(r.Context(), didStr)
		refreshProps.BrewerBrewCounts = h.FeedIndex().BrewCountsByBrewerURI(r.Context(), didStr)
		refreshProps.RoasterBeanCounts = h.FeedIndex().BeanCountsByRoasterURI(r.Context(), didStr)
		refreshProps.BeanAvgBrewRatings = make(map[string]float64)
		for uri, stats := range h.FeedIndex().AvgBrewRatingByBeanURI(r.Context(), didStr) {
			refreshProps.BeanAvgBrewRatings[uri] = stats.Average
		}
		refreshProps.RoasterAvgBrewRatings = make(map[string]float64)
		for uri, stats := range h.FeedIndex().AvgBrewRatingByRoasterURI(r.Context(), didStr) {
			refreshProps.RoasterAvgBrewRatings[uri] = stats.Average
		}
	}

	if err := coffee.ManagePartial(refreshProps).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render content", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render manage partial after refresh")
	}
}

// Bean update/delete handlers
func (h *Handlers) HandleBeanUpdate(w http.ResponseWriter, r *http.Request) {
	rkey := handlers.ValidateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req arabica.UpdateBeanRequest

	// Decode request (JSON or form)
	if err := handlers.DecodeRequest(r, &req, func() error {
		req = arabica.UpdateBeanRequest{
			Name:        r.FormValue("name"),
			Origin:      r.FormValue("origin"),
			Variety:     r.FormValue("variety"),
			RoastLevel:  r.FormValue("roast_level"),
			Process:     r.FormValue("process"),
			Description: r.FormValue("description"),
			Link:        r.FormValue("link"),
			RoasterRKey: r.FormValue("roaster_rkey"),
			Rating:      handlers.ParseOptionalInt(r.FormValue("rating")),
			Closed:      r.FormValue("closed") == "true",
			SourceRef:   r.FormValue("source_ref"),
		}
		log.Debug().
			Str("rkey", rkey).
			Str("name", req.Name).
			Str("closed_value", r.FormValue("closed")).
			Bool("closed_parsed", req.Closed).
			Msg("Parsed bean update form")
		return nil
	}); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msg("Failed to decode bean update request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msg("Bean update validation failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// If a new roaster name was provided and no existing roaster selected, create it
	if newRoasterName := r.FormValue("new_roaster_name"); newRoasterName != "" && req.RoasterRKey == "" {
		roaster, roasterErr := store.CreateRoaster(r.Context(), &arabica.CreateRoasterRequest{
			Name:     newRoasterName,
			Location: r.FormValue("new_roaster_location"),
			Website:  r.FormValue("new_roaster_website"),
		})
		if roasterErr != nil {
			log.Error().Err(roasterErr).Str("name", newRoasterName).Msg("Failed to create roaster for bean update")
			handlers.HandleStoreError(w, roasterErr, "Failed to create roaster")
			return
		}
		req.RoasterRKey = roaster.RKey
		log.Info().Str("roaster_rkey", roaster.RKey).Str("name", newRoasterName).Msg("Auto-created roaster for bean update")
	}

	// Validate optional roaster rkey
	if errMsg := handlers.ValidateOptionalRKey(req.RoasterRKey, "Roaster selection"); errMsg != "" {
		log.Warn().Str("rkey", rkey).Str("roaster_rkey", req.RoasterRKey).Msg("Bean update: invalid roaster rkey")
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	if err := store.UpdateBeanByRKey(r.Context(), rkey, &req); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update bean")
		handlers.HandleStoreError(w, err, "Failed to update bean")
		return
	}

	bean, err := store.GetBeanByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Failed to fetch updated bean", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get bean after update")
		return
	}

	h.InvalidateFeedCache()
	handlers.WriteJSON(w, bean, "bean")
}

func (h *Handlers) HandleBeanDelete(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	h.DeleteEntity(w, r, store.DeleteBeanByRKey, "bean", arabica.NSIDBean)
}

// Roaster update/delete handlers
func (h *Handlers) HandleRoasterUpdate(w http.ResponseWriter, r *http.Request) {
	rkey := handlers.ValidateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	store, ok := h.RequireRecordStore(w, r)
	if !ok {
		return
	}
	createdAt := handlers.ExistingCreatedAt(r.Context(), store, arabica.NSIDRoaster, rkey)
	handlers.RecordCRUDWrite[arabica.UpdateRoasterRequest, *arabica.UpdateRoasterRequest, arabica.Roaster](
		w, r, store, arabica.NSIDRoaster, "roaster", rkey, decodeRoasterUpdateForm,
		func(req *arabica.UpdateRoasterRequest) *arabica.Roaster {
			m := roasterFromUpdate(req, createdAt)
			m.RKey = rkey
			return m
		},
		func(m *arabica.Roaster, rkey string) { m.RKey = rkey },
		func(_ records.Store, _ *arabica.UpdateRoasterRequest, m *arabica.Roaster) (map[string]any, error) {
			return arabica.RoasterToRecord(m)
		},
		h.InvalidateFeedCache, false,
	)
}

func (h *Handlers) HandleRoasterDelete(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	h.DeleteEntity(w, r, store.DeleteRoasterByRKey, "roaster", arabica.NSIDRoaster)
}

// Grinder CRUD handlers
func grinderFormDecoder(r *http.Request) arabica.CreateGrinderRequest {
	return arabica.CreateGrinderRequest{
		Name: r.FormValue("name"), GrinderType: r.FormValue("grinder_type"),
		BurrType: r.FormValue("burr_type"), Notes: r.FormValue("notes"),
		Link: r.FormValue("link"), SourceRef: r.FormValue("source_ref"),
	}
}

func (h *Handlers) HandleGrinderCreate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.RequireRecordStore(w, r)
	if !ok {
		return
	}
	handlers.RecordCRUDWrite[arabica.CreateGrinderRequest, *arabica.CreateGrinderRequest, arabica.Grinder](
		w, r, store, arabica.NSIDGrinder, "grinder", "",
		func(r *http.Request, req *arabica.CreateGrinderRequest) error {
			*req = grinderFormDecoder(r)
			return nil
		},
		func(req *arabica.CreateGrinderRequest) *arabica.Grinder { return grinderFromCreate(req, time.Now()) },
		func(m *arabica.Grinder, rkey string) { m.RKey = rkey },
		func(_ records.Store, _ *arabica.CreateGrinderRequest, m *arabica.Grinder) (map[string]any, error) {
			return arabica.GrinderToRecord(m)
		},
		h.InvalidateFeedCache, false,
	)
}

func (h *Handlers) HandleGrinderUpdate(w http.ResponseWriter, r *http.Request) {
	rkey := handlers.ValidateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	store, ok := h.RequireRecordStore(w, r)
	if !ok {
		return
	}
	createdAt := handlers.ExistingCreatedAt(r.Context(), store, arabica.NSIDGrinder, rkey)
	handlers.RecordCRUDWrite[arabica.UpdateGrinderRequest, *arabica.UpdateGrinderRequest, arabica.Grinder](
		w, r, store, arabica.NSIDGrinder, "grinder", rkey,
		func(r *http.Request, req *arabica.UpdateGrinderRequest) error {
			*req = arabica.UpdateGrinderRequest(grinderFormDecoder(r))
			return nil
		},
		func(req *arabica.UpdateGrinderRequest) *arabica.Grinder {
			m := grinderFromUpdate(req, createdAt)
			m.RKey = rkey
			return m
		},
		func(m *arabica.Grinder, rkey string) { m.RKey = rkey },
		func(_ records.Store, _ *arabica.UpdateGrinderRequest, m *arabica.Grinder) (map[string]any, error) {
			return arabica.GrinderToRecord(m)
		},
		h.InvalidateFeedCache, false,
	)
}

func (h *Handlers) HandleGrinderDelete(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	h.DeleteEntity(w, r, func(ctx context.Context, rkey string) error {
		return store.RemoveRecord(ctx, arabica.NSIDGrinder, rkey)
	}, "grinder", arabica.NSIDGrinder)
}

// Brewer CRUD handlers
func brewerFormDecoder(r *http.Request) arabica.CreateBrewerRequest {
	return arabica.CreateBrewerRequest{
		Name: r.FormValue("name"), BrewerType: r.FormValue("brewer_type"),
		Description: r.FormValue("description"), Link: r.FormValue("link"),
		SourceRef: r.FormValue("source_ref"),
	}
}

func (h *Handlers) HandleBrewerCreate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.RequireRecordStore(w, r)
	if !ok {
		return
	}
	handlers.RecordCRUDWrite[arabica.CreateBrewerRequest, *arabica.CreateBrewerRequest, arabica.Brewer](
		w, r, store, arabica.NSIDBrewer, "brewer", "",
		func(r *http.Request, req *arabica.CreateBrewerRequest) error { *req = brewerFormDecoder(r); return nil },
		func(req *arabica.CreateBrewerRequest) *arabica.Brewer { return brewerFromCreate(req, time.Now()) },
		func(m *arabica.Brewer, rkey string) { m.RKey = rkey },
		func(_ records.Store, _ *arabica.CreateBrewerRequest, m *arabica.Brewer) (map[string]any, error) {
			return arabica.BrewerToRecord(m)
		},
		h.InvalidateFeedCache, false,
	)
}

func (h *Handlers) HandleBrewerUpdate(w http.ResponseWriter, r *http.Request) {
	rkey := handlers.ValidateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	store, ok := h.RequireRecordStore(w, r)
	if !ok {
		return
	}
	createdAt := handlers.ExistingCreatedAt(r.Context(), store, arabica.NSIDBrewer, rkey)
	handlers.RecordCRUDWrite[arabica.UpdateBrewerRequest, *arabica.UpdateBrewerRequest, arabica.Brewer](
		w, r, store, arabica.NSIDBrewer, "brewer", rkey,
		func(r *http.Request, req *arabica.UpdateBrewerRequest) error {
			*req = arabica.UpdateBrewerRequest(brewerFormDecoder(r))
			return nil
		},
		func(req *arabica.UpdateBrewerRequest) *arabica.Brewer {
			m := brewerFromUpdate(req, createdAt)
			m.RKey = rkey
			return m
		},
		func(m *arabica.Brewer, rkey string) { m.RKey = rkey },
		func(_ records.Store, _ *arabica.UpdateBrewerRequest, m *arabica.Brewer) (map[string]any, error) {
			return arabica.BrewerToRecord(m)
		},
		h.InvalidateFeedCache, false,
	)
}

func (h *Handlers) HandleBrewerDelete(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	h.DeleteEntity(w, r, func(ctx context.Context, rkey string) error { return store.RemoveRecord(ctx, arabica.NSIDBrewer, rkey) }, "brewer", arabica.NSIDBrewer)
}
