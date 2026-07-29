package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	arabicastore "tangled.org/arabica.social/arabica/internal/arabica/store"
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

// HandleAPIListAll returns all of the user's records (beans, roasters,
// grinders, brewers, brews) in one response for client-side caching.
func (h *Handlers) HandleAPIListAll(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		handlers.WriteJSONError(w, http.StatusUnauthorized, "authentication_required", "Authentication required")
		return
	}

	userDID, _ := atpmiddleware.GetDID(r.Context())

	ctx := r.Context()

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
		handlers.HandleStoreJSONError(w, err, "Failed to fetch data")
		return
	}

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

	handlers.WriteJSON(w, response, "data")
}

// HandleBeanCreate creates a bean from JSON or form data.
func (h *Handlers) HandleBeanCreate(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req arabica.CreateBeanRequest

	if err := handlers.DecodeRequest(r, &req, func() error {
		req = arabica.CreateBeanRequest{
			Name:        r.FormValue("name"),
			Origin:      r.FormValue("origin"),
			Variety:     r.FormValue("variety"),
			RoastLevel:  r.FormValue("roast_level"),
			RoastDate:   r.FormValue("roast_date"),
			Process:     r.FormValue("process"),
			Description: r.FormValue("description"),
			Notes:       r.FormValue("notes"),
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
	if redirect := r.FormValue("__redirect"); redirect != "" {
		w.Header().Set("HX-Redirect", redirect)
		w.WriteHeader(http.StatusOK)
		return
	}
	handlers.WriteJSON(w, bean, "bean")
}

// HandleRoasterCreate creates a roaster from JSON or form data.
func (h *Handlers) HandleRoasterCreate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.RequireRecordStore(w, r)
	if !ok {
		return
	}
	handlers.RecordCRUDWrite(
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

// HandleIncompleteRecordsPartial returns the user's incomplete records as JSON.
func (h *Handlers) HandleIncompleteRecordsPartial(w http.ResponseWriter, r *http.Request) {
	h.HandleIncompleteRecordsJSON(w, r)
}

// HandleManageRefresh invalidates all caches and re-fetches records from the
// user's PDS, writing them through to the witness cache so subsequent reads
// are up to date. Returns the refreshed manage partial.
func (h *Handlers) HandleManageRefresh(w http.ResponseWriter, r *http.Request) {
	_, authenticated := h.GetArabicaStore(r)
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

	// The SPA refetches /api/manage and /api/brews after this confirmation.
	handlers.WriteJSON(w, map[string]any{"refreshed": true}, "manage refresh")
}

// HandleBeanUpdate updates an existing bean from JSON or form data.
func (h *Handlers) HandleBeanUpdate(w http.ResponseWriter, r *http.Request) {
	rkey := handlers.ValidateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req arabica.UpdateBeanRequest

	if err := handlers.DecodeRequest(r, &req, func() error {
		req = arabica.UpdateBeanRequest{
			Name:        r.FormValue("name"),
			Origin:      r.FormValue("origin"),
			Variety:     r.FormValue("variety"),
			RoastLevel:  r.FormValue("roast_level"),
			RoastDate:   r.FormValue("roast_date"),
			Process:     r.FormValue("process"),
			Description: r.FormValue("description"),
			Notes:       r.FormValue("notes"),
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
	if redirect := r.FormValue("__redirect"); redirect != "" {
		w.Header().Set("HX-Redirect", redirect)
		w.WriteHeader(http.StatusOK)
		return
	}
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

// HandleRoasterUpdate updates an existing roaster.
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
	handlers.RecordCRUDWrite(
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

// Grinder entity handlers.
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
	handlers.RecordCRUDWrite(
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
	handlers.RecordCRUDWrite(
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

// Brewer entity handlers.
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
	handlers.RecordCRUDWrite(
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
	handlers.RecordCRUDWrite(
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
	h.DeleteEntity(w, r, func(ctx context.Context, rkey string) error {
		return store.RemoveRecord(ctx, arabica.NSIDBrewer, rkey)
	}, "brewer", arabica.NSIDBrewer)
}
