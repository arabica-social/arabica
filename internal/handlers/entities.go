package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/entities/arabica"
	"tangled.org/arabica.social/arabica/internal/tracing"
	"tangled.org/arabica.social/arabica/internal/web/components"
	"tangled.org/arabica.social/arabica/internal/arabica/web/components"
	"tangled.org/arabica.social/arabica/internal/arabica/web/pages"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/errgroup"
	"tangled.org/pdewey.com/atp"
)

// Manage page partial (loaded async via HTMX)
func (h *Handler) HandleManagePartial(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Invalidate the session cache so we read from the witness cache (or PDS)
	// rather than serving potentially stale in-memory data. The witness cache
	// is a local SQLite read so this is cheap.
	if sessionID, ok := atpmiddleware.GetSessionID(r.Context()); ok {
		h.sessionCache.Invalidate(sessionID)
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
		grinders, err = store.ListGrinders(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		brewers, err = store.ListBrewers(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		recipes, err = store.ListRecipes(ctx)
		return err
	})

	if err := g.Wait(); err != nil {
		log.Error().Err(err).Msg("Failed to fetch manage page data")
		handleStoreError(w, err, "Failed to fetch data")
		return
	}

	// Link beans to their roasters
	atproto.LinkBeansToRoasters(beans, roasters)

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
	if h.feedIndex != nil {
		did, _ := atpmiddleware.GetDID(r.Context())
		props.OwnerDID = did
		props.BeanBrewCounts = h.feedIndex.BrewCountsByBeanURI(r.Context(), did)
		props.GrinderBrewCounts = h.feedIndex.BrewCountsByGrinderURI(r.Context(), did)
		props.BrewerBrewCounts = h.feedIndex.BrewCountsByBrewerURI(r.Context(), did)
		props.RoasterBeanCounts = h.feedIndex.BeanCountsByRoasterURI(r.Context(), did)
		props.BeanAvgBrewRatings = make(map[string]float64)
		for uri, stats := range h.feedIndex.AvgBrewRatingByBeanURI(r.Context(), did) {
			props.BeanAvgBrewRatings[uri] = stats.Average
		}
		props.RoasterAvgBrewRatings = make(map[string]float64)
		for uri, stats := range h.feedIndex.AvgBrewRatingByRoasterURI(r.Context(), did) {
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
// Used by client-side cache for faster page loads
func (h *Handler) HandleAPIListAll(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getAtprotoStore(r)
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
		grinders, err = store.ListGrinders(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		brewers, err = store.ListBrewers(ctx)
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
		handleStoreError(w, err, "Failed to fetch data")
		return
	}

	// Link beans to roasters
	atproto.LinkBeansToRoasters(beans, roasters)

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
func (h *Handler) HandleBeanCreate(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req arabica.CreateBeanRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = arabica.CreateBeanRequest{
			Name:        r.FormValue("name"),
			Origin:      r.FormValue("origin"),
			Variety:     r.FormValue("variety"),
			RoastLevel:  r.FormValue("roast_level"),
			Process:     r.FormValue("process"),
			Description: r.FormValue("description"),
			RoasterRKey: r.FormValue("roaster_rkey"),
			Rating:      parseOptionalInt(r.FormValue("rating")),
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
			handleStoreError(w, roasterErr, "Failed to create roaster")
			return
		}
		req.RoasterRKey = roaster.RKey
		log.Info().Str("roaster_rkey", roaster.RKey).Str("name", newRoasterName).Msg("Auto-created roaster for bean")
	}

	// Validate optional roaster rkey
	if errMsg := validateOptionalRKey(req.RoasterRKey, "Roaster selection"); errMsg != "" {
		log.Warn().Str("roaster_rkey", req.RoasterRKey).Msg("Bean create: invalid roaster rkey")
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	bean, err := store.CreateBean(r.Context(), &req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create bean")
		handleStoreError(w, err, "Failed to create bean")
		return
	}

	h.invalidateFeedCache()
	writeJSON(w, bean, "bean")
}

// API endpoint to create roaster
func (h *Handler) HandleRoasterCreate(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req arabica.CreateRoasterRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = arabica.CreateRoasterRequest{
			Name:      r.FormValue("name"),
			Location:  r.FormValue("location"),
			Website:   r.FormValue("website"),
			SourceRef: r.FormValue("source_ref"),
		}
		return nil
	}); err != nil {
		log.Warn().Err(err).Msg("Failed to decode roaster create request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		log.Warn().Err(err).Str("name", req.Name).Msg("Roaster create validation failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	roaster, err := store.CreateRoaster(r.Context(), &req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create roaster")
		handleStoreError(w, err, "Failed to create roaster")
		return
	}

	h.invalidateFeedCache()
	writeJSON(w, roaster, "roaster")
}

// Manage page
func (h *Handler) HandleManage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/my-coffee", http.StatusMovedPermanently)
}

// HandleMyCoffee renders the unified My Coffee page (replaces both /brews and /manage)
func (h *Handler) HandleMyCoffee(w http.ResponseWriter, r *http.Request) {
	_, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	layoutData, _, _ := h.layoutDataFromRequest(r, "My Coffee")

	if err := coffeepages.MyCoffee(layoutData, coffeepages.MyCoffeeProps{}).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render my coffee page")
	}
}

// HandleIncompleteRecordsPartial returns an HTML fragment for incomplete records on the home dashboard.
func (h *Handler) HandleIncompleteRecordsPartial(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getAtprotoStore(r)
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
		grinders, err = store.ListGrinders(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		brewers, err = store.ListBrewers(ctx)
		return err
	})

	if err := g.Wait(); err != nil {
		log.Error().Err(err).Msg("Failed to fetch data for incomplete records")
		return
	}

	records := components.CollectIncompleteRecords(beans, grinders, brewers, 5)

	if err := components.IncompleteRecords(components.IncompleteRecordsProps{
		Records: records,
	}).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render incomplete records")
	}
}

// HandleManageRefresh invalidates all caches and re-fetches records from the
// user's PDS, writing them through to the witness cache so subsequent reads
// are up to date. Returns the refreshed manage partial.
func (h *Handler) HandleManageRefresh(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getAtprotoStore(r)
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
	h.sessionCache.Invalidate(sessionID)

	// Re-fetch all entity collections from PDS and write-through to witness
	entityCollections := []string{
		arabica.NSIDBean, arabica.NSIDRoaster, arabica.NSIDGrinder,
		arabica.NSIDBrewer, arabica.NSIDRecipe, arabica.NSIDBrew,
	}

	if h.witnessCache != nil {
		refreshCtx, refreshSpan := tracing.HandlerSpan(r.Context(), "manage.refresh.witness_sync",
			attribute.String("user.did", didStr),
		)
		atpClient, err := h.atprotoClient.AtpClient(refreshCtx, did, sessionID)
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
		if err := h.witnessCache.UpsertWitnessRecordBatch(refreshCtx, batch); err != nil {
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
		grinders, err = store.ListGrinders(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		brewers, err = store.ListBrewers(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		recipes, err = store.ListRecipes(ctx)
		return err
	})

	if err := g.Wait(); err != nil {
		log.Error().Err(err).Msg("Failed to fetch manage page data after refresh")
		handleStoreError(w, err, "Failed to fetch data")
		return
	}

	atproto.LinkBeansToRoasters(beans, roasters)

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
	if h.feedIndex != nil {
		refreshProps.OwnerDID = didStr
		refreshProps.BeanBrewCounts = h.feedIndex.BrewCountsByBeanURI(r.Context(), didStr)
		refreshProps.GrinderBrewCounts = h.feedIndex.BrewCountsByGrinderURI(r.Context(), didStr)
		refreshProps.BrewerBrewCounts = h.feedIndex.BrewCountsByBrewerURI(r.Context(), didStr)
		refreshProps.RoasterBeanCounts = h.feedIndex.BeanCountsByRoasterURI(r.Context(), didStr)
		refreshProps.BeanAvgBrewRatings = make(map[string]float64)
		for uri, stats := range h.feedIndex.AvgBrewRatingByBeanURI(r.Context(), didStr) {
			refreshProps.BeanAvgBrewRatings[uri] = stats.Average
		}
		refreshProps.RoasterAvgBrewRatings = make(map[string]float64)
		for uri, stats := range h.feedIndex.AvgBrewRatingByRoasterURI(r.Context(), didStr) {
			refreshProps.RoasterAvgBrewRatings[uri] = stats.Average
		}
	}

	if err := coffee.ManagePartial(refreshProps).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render content", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render manage partial after refresh")
	}
}

// Bean update/delete handlers
func (h *Handler) HandleBeanUpdate(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req arabica.UpdateBeanRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = arabica.UpdateBeanRequest{
			Name:        r.FormValue("name"),
			Origin:      r.FormValue("origin"),
			Variety:     r.FormValue("variety"),
			RoastLevel:  r.FormValue("roast_level"),
			Process:     r.FormValue("process"),
			Description: r.FormValue("description"),
			RoasterRKey: r.FormValue("roaster_rkey"),
			Rating:      parseOptionalInt(r.FormValue("rating")),
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
			handleStoreError(w, roasterErr, "Failed to create roaster")
			return
		}
		req.RoasterRKey = roaster.RKey
		log.Info().Str("roaster_rkey", roaster.RKey).Str("name", newRoasterName).Msg("Auto-created roaster for bean update")
	}

	// Validate optional roaster rkey
	if errMsg := validateOptionalRKey(req.RoasterRKey, "Roaster selection"); errMsg != "" {
		log.Warn().Str("rkey", rkey).Str("roaster_rkey", req.RoasterRKey).Msg("Bean update: invalid roaster rkey")
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	if err := store.UpdateBeanByRKey(r.Context(), rkey, &req); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update bean")
		handleStoreError(w, err, "Failed to update bean")
		return
	}

	bean, err := store.GetBeanByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Failed to fetch updated bean", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get bean after update")
		return
	}

	h.invalidateFeedCache()
	writeJSON(w, bean, "bean")
}

func (h *Handler) HandleBeanDelete(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	h.deleteEntity(w, r, store.DeleteBeanByRKey, "bean", arabica.NSIDBean)
}

// Roaster update/delete handlers
func (h *Handler) HandleRoasterUpdate(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req arabica.UpdateRoasterRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = arabica.UpdateRoasterRequest{
			Name:      r.FormValue("name"),
			Location:  r.FormValue("location"),
			Website:   r.FormValue("website"),
			SourceRef: r.FormValue("source_ref"),
		}
		return nil
	}); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msg("Failed to decode roaster update request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msg("Roaster update validation failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := store.UpdateRoasterByRKey(r.Context(), rkey, &req); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update roaster")
		handleStoreError(w, err, "Failed to update roaster")
		return
	}

	roaster, err := store.GetRoasterByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Failed to fetch updated roaster", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get roaster after update")
		return
	}

	h.invalidateFeedCache()
	writeJSON(w, roaster, "roaster")
}

func (h *Handler) HandleRoasterDelete(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	h.deleteEntity(w, r, store.DeleteRoasterByRKey, "roaster", arabica.NSIDRoaster)
}

// Grinder CRUD handlers
func (h *Handler) HandleGrinderCreate(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req arabica.CreateGrinderRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = arabica.CreateGrinderRequest{
			Name:        r.FormValue("name"),
			GrinderType: r.FormValue("grinder_type"),
			BurrType:    r.FormValue("burr_type"),
			Notes:       r.FormValue("notes"),
			SourceRef:   r.FormValue("source_ref"),
		}
		return nil
	}); err != nil {
		log.Warn().Err(err).Msg("Failed to decode grinder create request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		log.Warn().Err(err).Str("name", req.Name).Msg("Grinder create validation failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	grinder, err := store.CreateGrinder(r.Context(), &req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create grinder")
		handleStoreError(w, err, "Failed to create grinder")
		return
	}

	h.invalidateFeedCache()
	writeJSON(w, grinder, "grinder")
}

func (h *Handler) HandleGrinderUpdate(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req arabica.UpdateGrinderRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = arabica.UpdateGrinderRequest{
			Name:        r.FormValue("name"),
			GrinderType: r.FormValue("grinder_type"),
			BurrType:    r.FormValue("burr_type"),
			Notes:       r.FormValue("notes"),
			SourceRef:   r.FormValue("source_ref"),
		}
		return nil
	}); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msg("Failed to decode grinder update request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msg("Grinder update validation failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := store.UpdateGrinderByRKey(r.Context(), rkey, &req); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update grinder")
		handleStoreError(w, err, "Failed to update grinder")
		return
	}

	grinder, err := store.GetGrinderByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Failed to fetch updated grinder", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get grinder after update")
		return
	}

	h.invalidateFeedCache()
	writeJSON(w, grinder, "grinder")
}

func (h *Handler) HandleGrinderDelete(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	h.deleteEntity(w, r, store.DeleteGrinderByRKey, "grinder", arabica.NSIDGrinder)
}

// Brewer CRUD handlers
func (h *Handler) HandleBrewerCreate(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req arabica.CreateBrewerRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = arabica.CreateBrewerRequest{
			Name:        r.FormValue("name"),
			BrewerType:  r.FormValue("brewer_type"),
			Description: r.FormValue("description"),
			SourceRef:   r.FormValue("source_ref"),
		}
		return nil
	}); err != nil {
		log.Warn().Err(err).Msg("Failed to decode brewer create request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		log.Warn().Err(err).Str("name", req.Name).Msg("Brewer create validation failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	brewer, err := store.CreateBrewer(r.Context(), &req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create brewer")
		handleStoreError(w, err, "Failed to create brewer")
		return
	}

	h.invalidateFeedCache()
	writeJSON(w, brewer, "brewer")
}

func (h *Handler) HandleBrewerUpdate(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req arabica.UpdateBrewerRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = arabica.UpdateBrewerRequest{
			Name:        r.FormValue("name"),
			BrewerType:  r.FormValue("brewer_type"),
			Description: r.FormValue("description"),
			SourceRef:   r.FormValue("source_ref"),
		}
		return nil
	}); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msg("Failed to decode brewer update request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msg("Brewer update validation failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := store.UpdateBrewerByRKey(r.Context(), rkey, &req); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update brewer")
		handleStoreError(w, err, "Failed to update brewer")
		return
	}

	brewer, err := store.GetBrewerByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Failed to fetch updated brewer", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get brewer after update")
		return
	}

	h.invalidateFeedCache()
	writeJSON(w, brewer, "brewer")
}

func (h *Handler) HandleBrewerDelete(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	h.deleteEntity(w, r, store.DeleteBrewerByRKey, "brewer", arabica.NSIDBrewer)
}
