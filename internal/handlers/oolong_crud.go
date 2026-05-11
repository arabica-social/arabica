package handlers

import (
	"context"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	atp "tangled.org/pdewey.com/atp"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/entities/oolong"
)

// Oolong CRUD handlers. Each Create handler decodes the request, builds
// a typed model, calls oolong.XToRecord to produce the PDS record map,
// and writes it via the generic store.PutRecord. Update follows the
// same path with a non-empty rkey. Delete extracts the rkey from the
// URL and calls store.RemoveRecord.
//
// These handlers intentionally do not re-fetch the created/updated record
// to populate joined fields (Vendor on Tea, BrewerObj on Recipe, etc.) —
// the client refreshes via /api/data or by navigating to the entity's
// view page. Cleaner ref-resolution lands alongside the typed Get*
// wrappers in a future phase.

// putOolongRecord is the shared write path for all 7 oolong CRUD Create
// and Update endpoints. The encode callback receives the validated
// request (already decoded into a target struct by the caller) and
// returns the record map to write. rkey is "" for Create and the
// existing record key for Update.
func putOolongRecord(
	ctx context.Context,
	store *atproto.AtprotoStore,
	nsid, rkey string,
	encode func(s *atproto.AtprotoStore) (map[string]any, error),
) (resultRKey string, err error) {
	rec, err := encode(store)
	if err != nil {
		return "", err
	}
	newRKey, _, err := store.PutRecord(ctx, nsid, rkey, rec)
	if err != nil {
		return "", err
	}
	if newRKey == "" {
		newRKey = rkey
	}
	return newRKey, nil
}

func (h *Handler) requireOolongStore(w http.ResponseWriter, r *http.Request) (*atproto.AtprotoStore, bool) {
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return nil, false
	}
	atpStore, ok := store.(*atproto.AtprotoStore)
	if !ok {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return nil, false
	}
	return atpStore, true
}

// --- Tea -------------------------------------------------------------

func (h *Handler) HandleTeaCreate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	var req oolong.CreateTeaRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tea := teaFromCreateRequest(&req)
	rkey, err := putOolongRecord(r.Context(), store, oolong.NSIDTea, "", func(s *atproto.AtprotoStore) (map[string]any, error) {
		var vendorURI string
		if req.VendorRKey != "" {
			vendorURI = atp.BuildATURI(s.DID(), oolong.NSIDVendor, req.VendorRKey)
		}
		return oolong.TeaToRecord(tea, vendorURI)
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create tea")
		handleStoreError(w, err, "Failed to create tea")
		return
	}
	tea.RKey = rkey
	h.invalidateFeedCache()
	writeJSON(w, tea, "tea")
}

func (h *Handler) HandleTeaUpdate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	var req oolong.UpdateTeaRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	createReq := oolong.CreateTeaRequest(req)
	if err := createReq.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tea := teaFromCreateRequest(&createReq)
	tea.RKey = rkey
	if _, err := putOolongRecord(r.Context(), store, oolong.NSIDTea, rkey, func(s *atproto.AtprotoStore) (map[string]any, error) {
		var vendorURI string
		if createReq.VendorRKey != "" {
			vendorURI = atp.BuildATURI(s.DID(), oolong.NSIDVendor, createReq.VendorRKey)
		}
		return oolong.TeaToRecord(tea, vendorURI)
	}); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update tea")
		handleStoreError(w, err, "Failed to update tea")
		return
	}
	h.invalidateFeedCache()
	writeJSON(w, tea, "tea")
}

func (h *Handler) HandleTeaDelete(w http.ResponseWriter, r *http.Request) {
	h.handleOolongDelete(w, r, oolong.NSIDTea, "tea")
}

func teaFromCreateRequest(req *oolong.CreateTeaRequest) *oolong.Tea {
	return &oolong.Tea{
		Name:        req.Name,
		Category:    req.Category,
		SubStyle:    req.SubStyle,
		Origin:      req.Origin,
		Cultivar:    req.Cultivar,
		Farm:        req.Farm,
		HarvestYear: req.HarvestYear,
		Processing:  req.Processing,
		Description: req.Description,
		VendorRKey:  req.VendorRKey,
		Rating:      req.Rating,
		Closed:      req.Closed,
		SourceRef:   req.SourceRef,
		CreatedAt:   time.Now(),
	}
}

// --- Vendor ----------------------------------------------------------

func (h *Handler) HandleOolongVendorCreate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	var req oolong.CreateVendorRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	v := &oolong.Vendor{
		Name:        req.Name,
		Location:    req.Location,
		Website:     req.Website,
		Description: req.Description,
		SourceRef:   req.SourceRef,
		CreatedAt:   time.Now(),
	}
	rkey, err := putOolongRecord(r.Context(), store, oolong.NSIDVendor, "", func(*atproto.AtprotoStore) (map[string]any, error) {
		return oolong.VendorToRecord(v)
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create vendor")
		handleStoreError(w, err, "Failed to create vendor")
		return
	}
	v.RKey = rkey
	h.invalidateFeedCache()
	writeJSON(w, v, "vendor")
}

func (h *Handler) HandleOolongVendorUpdate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	var req oolong.UpdateVendorRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	createReq := oolong.CreateVendorRequest{
		Name: req.Name, Location: req.Location, Website: req.Website,
		Description: req.Description, SourceRef: req.SourceRef,
	}
	if err := createReq.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	v := &oolong.Vendor{
		RKey: rkey,
		Name: createReq.Name, Location: createReq.Location, Website: createReq.Website,
		Description: createReq.Description, SourceRef: createReq.SourceRef,
		CreatedAt: time.Now(),
	}
	if _, err := putOolongRecord(r.Context(), store, oolong.NSIDVendor, rkey, func(*atproto.AtprotoStore) (map[string]any, error) {
		return oolong.VendorToRecord(v)
	}); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update vendor")
		handleStoreError(w, err, "Failed to update vendor")
		return
	}
	h.invalidateFeedCache()
	writeJSON(w, v, "vendor")
}

func (h *Handler) HandleOolongVendorDelete(w http.ResponseWriter, r *http.Request) {
	h.handleOolongDelete(w, r, oolong.NSIDVendor, "vendor")
}

// --- Brewer ----------------------------------------------------------

func (h *Handler) HandleOolongBrewerCreate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	var req oolong.CreateBrewerRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	b := &oolong.Brewer{
		Name: req.Name, Style: req.Style, CapacityMl: req.CapacityMl,
		Material: req.Material, Description: req.Description,
		SourceRef: req.SourceRef, CreatedAt: time.Now(),
	}
	rkey, err := putOolongRecord(r.Context(), store, oolong.NSIDBrewer, "", func(*atproto.AtprotoStore) (map[string]any, error) {
		return oolong.BrewerToRecord(b)
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create tea brewer")
		handleStoreError(w, err, "Failed to create brewer")
		return
	}
	b.RKey = rkey
	h.invalidateFeedCache()
	writeJSON(w, b, "brewer")
}

func (h *Handler) HandleOolongBrewerUpdate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	var req oolong.UpdateBrewerRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	createReq := oolong.CreateBrewerRequest(req)
	if err := createReq.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	b := &oolong.Brewer{
		RKey: rkey,
		Name: createReq.Name, Style: createReq.Style, CapacityMl: createReq.CapacityMl,
		Material: createReq.Material, Description: createReq.Description,
		SourceRef: createReq.SourceRef, CreatedAt: time.Now(),
	}
	if _, err := putOolongRecord(r.Context(), store, oolong.NSIDBrewer, rkey, func(*atproto.AtprotoStore) (map[string]any, error) {
		return oolong.BrewerToRecord(b)
	}); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update tea brewer")
		handleStoreError(w, err, "Failed to update brewer")
		return
	}
	h.invalidateFeedCache()
	writeJSON(w, b, "brewer")
}

func (h *Handler) HandleOolongBrewerDelete(w http.ResponseWriter, r *http.Request) {
	h.handleOolongDelete(w, r, oolong.NSIDBrewer, "brewer")
}

// --- Recipe ----------------------------------------------------------

func (h *Handler) HandleOolongRecipeCreate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	var req oolong.CreateRecipeRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rec := recipeFromCreateRequest(&req)
	rkey, err := putOolongRecord(r.Context(), store, oolong.NSIDRecipe, "", func(s *atproto.AtprotoStore) (map[string]any, error) {
		brewerURI := buildOolongRef(s, req.BrewerRKey, oolong.NSIDBrewer)
		teaURI := buildOolongRef(s, req.TeaRKey, oolong.NSIDTea)
		return oolong.RecipeToRecord(rec, brewerURI, teaURI)
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create tea recipe")
		handleStoreError(w, err, "Failed to create recipe")
		return
	}
	rec.RKey = rkey
	h.invalidateFeedCache()
	writeJSON(w, rec, "recipe")
}

func (h *Handler) HandleOolongRecipeUpdate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	var req oolong.UpdateRecipeRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	createReq := oolong.CreateRecipeRequest(req)
	if err := createReq.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rec := recipeFromCreateRequest(&createReq)
	rec.RKey = rkey
	if _, err := putOolongRecord(r.Context(), store, oolong.NSIDRecipe, rkey, func(s *atproto.AtprotoStore) (map[string]any, error) {
		brewerURI := buildOolongRef(s, createReq.BrewerRKey, oolong.NSIDBrewer)
		teaURI := buildOolongRef(s, createReq.TeaRKey, oolong.NSIDTea)
		return oolong.RecipeToRecord(rec, brewerURI, teaURI)
	}); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update tea recipe")
		handleStoreError(w, err, "Failed to update recipe")
		return
	}
	h.invalidateFeedCache()
	writeJSON(w, rec, "recipe")
}

func (h *Handler) HandleOolongRecipeDelete(w http.ResponseWriter, r *http.Request) {
	h.handleOolongDelete(w, r, oolong.NSIDRecipe, "recipe")
}

func recipeFromCreateRequest(req *oolong.CreateRecipeRequest) *oolong.Recipe {
	return &oolong.Recipe{
		Name:         req.Name,
		BrewerRKey:   req.BrewerRKey,
		Style:        req.Style,
		TeaRKey:      req.TeaRKey,
		Temperature:  req.Temperature,
		TimeSeconds:  req.TimeSeconds,
		LeafGrams:    req.LeafGrams,
		VesselMl:     req.VesselMl,
		MethodParams: req.MethodParams,
		Notes:        req.Notes,
		SourceRef:    req.SourceRef,
		CreatedAt:    time.Now(),
	}
}

// --- Brew (steep) ----------------------------------------------------

func (h *Handler) HandleOolongBrewCreate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	var req oolong.CreateBrewRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	b := brewFromCreateRequest(&req)
	rkey, err := putOolongRecord(r.Context(), store, oolong.NSIDBrew, "", func(s *atproto.AtprotoStore) (map[string]any, error) {
		teaURI := buildOolongRef(s, req.TeaRKey, oolong.NSIDTea)
		brewerURI := buildOolongRef(s, req.BrewerRKey, oolong.NSIDBrewer)
		recipeURI := buildOolongRef(s, req.RecipeRKey, oolong.NSIDRecipe)
		return oolong.BrewToRecord(b, teaURI, brewerURI, recipeURI)
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create tea brew")
		handleStoreError(w, err, "Failed to create brew")
		return
	}
	b.RKey = rkey
	h.invalidateFeedCache()
	writeJSON(w, b, "brew")
}

func (h *Handler) HandleOolongBrewUpdate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	var req oolong.CreateBrewRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	b := brewFromCreateRequest(&req)
	b.RKey = rkey
	if _, err := putOolongRecord(r.Context(), store, oolong.NSIDBrew, rkey, func(s *atproto.AtprotoStore) (map[string]any, error) {
		teaURI := buildOolongRef(s, req.TeaRKey, oolong.NSIDTea)
		brewerURI := buildOolongRef(s, req.BrewerRKey, oolong.NSIDBrewer)
		recipeURI := buildOolongRef(s, req.RecipeRKey, oolong.NSIDRecipe)
		return oolong.BrewToRecord(b, teaURI, brewerURI, recipeURI)
	}); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update tea brew")
		handleStoreError(w, err, "Failed to update brew")
		return
	}
	h.invalidateFeedCache()
	writeJSON(w, b, "brew")
}

func (h *Handler) HandleOolongBrewDelete(w http.ResponseWriter, r *http.Request) {
	h.handleOolongDelete(w, r, oolong.NSIDBrew, "brew")
}

func brewFromCreateRequest(req *oolong.CreateBrewRequest) *oolong.Brew {
	return &oolong.Brew{
		TeaRKey:      req.TeaRKey,
		Style:        req.Style,
		BrewerRKey:   req.BrewerRKey,
		RecipeRKey:   req.RecipeRKey,
		Temperature:  req.Temperature,
		LeafGrams:    req.LeafGrams,
		VesselMl:     req.VesselMl,
		TimeSeconds:  req.TimeSeconds,
		TastingNotes: req.TastingNotes,
		Rating:       req.Rating,
		MethodParams: req.MethodParams,
		CreatedAt:    time.Now(),
	}
}

// --- Cafe ------------------------------------------------------------

func (h *Handler) HandleOolongCafeCreate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	var req oolong.CreateCafeRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	c := &oolong.Cafe{
		Name: req.Name, Location: req.Location, Address: req.Address,
		Website: req.Website, Description: req.Description,
		VendorRKey: req.VendorRKey, SourceRef: req.SourceRef,
		CreatedAt: time.Now(),
	}
	rkey, err := putOolongRecord(r.Context(), store, oolong.NSIDCafe, "", func(s *atproto.AtprotoStore) (map[string]any, error) {
		vendorURI := buildOolongRef(s, req.VendorRKey, oolong.NSIDVendor)
		return oolong.CafeToRecord(c, vendorURI)
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create tea cafe")
		handleStoreError(w, err, "Failed to create cafe")
		return
	}
	c.RKey = rkey
	h.invalidateFeedCache()
	writeJSON(w, c, "cafe")
}

func (h *Handler) HandleOolongCafeUpdate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	var req oolong.UpdateCafeRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	createReq := oolong.CreateCafeRequest(req)
	if err := createReq.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	c := &oolong.Cafe{
		RKey: rkey,
		Name: createReq.Name, Location: createReq.Location, Address: createReq.Address,
		Website: createReq.Website, Description: createReq.Description,
		VendorRKey: createReq.VendorRKey, SourceRef: createReq.SourceRef,
		CreatedAt: time.Now(),
	}
	if _, err := putOolongRecord(r.Context(), store, oolong.NSIDCafe, rkey, func(s *atproto.AtprotoStore) (map[string]any, error) {
		vendorURI := buildOolongRef(s, createReq.VendorRKey, oolong.NSIDVendor)
		return oolong.CafeToRecord(c, vendorURI)
	}); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update tea cafe")
		handleStoreError(w, err, "Failed to update cafe")
		return
	}
	h.invalidateFeedCache()
	writeJSON(w, c, "cafe")
}

func (h *Handler) HandleOolongCafeDelete(w http.ResponseWriter, r *http.Request) {
	h.handleOolongDelete(w, r, oolong.NSIDCafe, "cafe")
}

// --- Drink -----------------------------------------------------------

func (h *Handler) HandleOolongDrinkCreate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	var req oolong.CreateDrinkRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	d := drinkFromCreateRequest(&req)
	rkey, err := putOolongRecord(r.Context(), store, oolong.NSIDDrink, "", func(s *atproto.AtprotoStore) (map[string]any, error) {
		cafeURI := buildOolongRef(s, req.CafeRKey, oolong.NSIDCafe)
		teaURI := buildOolongRef(s, req.TeaRKey, oolong.NSIDTea)
		return oolong.DrinkToRecord(d, cafeURI, teaURI)
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create tea drink")
		handleStoreError(w, err, "Failed to create drink")
		return
	}
	d.RKey = rkey
	h.invalidateFeedCache()
	writeJSON(w, d, "drink")
}

func (h *Handler) HandleOolongDrinkUpdate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	var req oolong.UpdateDrinkRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	createReq := oolong.CreateDrinkRequest(req)
	if err := createReq.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	d := drinkFromCreateRequest(&createReq)
	d.RKey = rkey
	if _, err := putOolongRecord(r.Context(), store, oolong.NSIDDrink, rkey, func(s *atproto.AtprotoStore) (map[string]any, error) {
		cafeURI := buildOolongRef(s, createReq.CafeRKey, oolong.NSIDCafe)
		teaURI := buildOolongRef(s, createReq.TeaRKey, oolong.NSIDTea)
		return oolong.DrinkToRecord(d, cafeURI, teaURI)
	}); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update tea drink")
		handleStoreError(w, err, "Failed to update drink")
		return
	}
	h.invalidateFeedCache()
	writeJSON(w, d, "drink")
}

func (h *Handler) HandleOolongDrinkDelete(w http.ResponseWriter, r *http.Request) {
	h.handleOolongDelete(w, r, oolong.NSIDDrink, "drink")
}

func drinkFromCreateRequest(req *oolong.CreateDrinkRequest) *oolong.Drink {
	return &oolong.Drink{
		CafeRKey: req.CafeRKey, TeaRKey: req.TeaRKey,
		Name: req.Name, Style: req.Style, Description: req.Description,
		Rating: req.Rating, TastingNotes: req.TastingNotes,
		PriceUsdCents: req.PriceUsdCents, SourceRef: req.SourceRef,
		CreatedAt: time.Now(),
	}
}

// --- Shared helpers ---------------------------------------------------

// handleOolongDelete is the shared body for every oolong Delete handler.
func (h *Handler) handleOolongDelete(w http.ResponseWriter, r *http.Request, nsid, entityName string) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	if err := store.RemoveRecord(r.Context(), nsid, rkey); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Str("nsid", nsid).Msgf("Failed to delete %s", entityName)
		handleStoreError(w, err, "Failed to delete "+entityName)
		return
	}
	h.invalidateFeedCache()
	w.WriteHeader(http.StatusNoContent)
}

// buildOolongRef returns the AT-URI for a referenced entity owned by the
// authenticated user, or "" if the rkey is empty.
func buildOolongRef(s *atproto.AtprotoStore, rkey, nsid string) string {
	if rkey == "" {
		return ""
	}
	return atp.BuildATURI(s.DID(), nsid, rkey)
}

// decodeOolongForm fills a CreateXRequest struct from form-encoded data
// using its `json:"..."` tags as form field names. Handles the scalar
// types every oolong request struct uses (string, int, float64, bool,
// *int) — slice / interface fields are ignored, since modals don't
// currently surface ProcessingStep or MethodParams. Mirrors the typed
// per-entity FormValue lookups arabica handlers do by hand.
func decodeOolongForm(r *http.Request, target any) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	v := reflect.ValueOf(target).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := strings.Split(field.Tag.Get("json"), ",")[0]
		if tag == "" || tag == "-" {
			continue
		}
		raw := r.FormValue(tag)
		if raw == "" {
			continue
		}
		fv := v.Field(i)
		if !fv.CanSet() {
			continue
		}
		switch fv.Kind() {
		case reflect.String:
			fv.SetString(raw)
		case reflect.Int, reflect.Int32, reflect.Int64:
			if n, err := strconv.Atoi(raw); err == nil {
				fv.SetInt(int64(n))
			}
		case reflect.Float32, reflect.Float64:
			if n, err := strconv.ParseFloat(raw, 64); err == nil {
				fv.SetFloat(n)
			}
		case reflect.Bool:
			fv.SetBool(raw == "true" || raw == "on" || raw == "1")
		case reflect.Ptr:
			if fv.Type().Elem().Kind() == reflect.Int {
				if n, err := strconv.Atoi(raw); err == nil {
					fv.Set(reflect.ValueOf(&n))
				}
			}
		}
	}
	return nil
}
