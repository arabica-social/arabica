package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"arabica/internal/atproto"
	"arabica/internal/matching"
	"arabica/internal/models"
	"arabica/internal/web/components"
	"arabica/internal/web/pages"

	"github.com/rs/zerolog/log"
)

// HandleRecipeCreate creates a new recipe
func (h *Handler) HandleRecipeCreate(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req models.CreateRecipeRequest

	if err := decodeRequest(r, &req, func() error {
		req = models.CreateRecipeRequest{
			Name:       r.FormValue("name"),
			BrewerRKey: r.FormValue("brewer_rkey"),
			BrewerType: r.FormValue("brewer_type"),
			Notes:      r.FormValue("notes"),
			SourceRef:  r.FormValue("source_ref"),
		}
		if v := r.FormValue("coffee_amount"); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				req.CoffeeAmount = f
			}
		}
		if v := r.FormValue("water_amount"); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				req.WaterAmount = f
			}
		}
		req.Pours = parsePours(r)
		return nil
	}); err != nil {
		log.Warn().Err(err).Msg("Failed to decode recipe create request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		log.Warn().Err(err).Str("name", req.Name).Msg("Recipe create validation failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if errMsg := validateOptionalRKey(req.BrewerRKey, "Brewer selection"); errMsg != "" {
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	recipe, err := store.CreateRecipe(r.Context(), &req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create recipe")
		handleStoreError(w, err, "Failed to create recipe")
		return
	}

	h.invalidateFeedCache()
	writeJSON(w, recipe, "recipe")
}

// HandleRecipeUpdate updates an existing recipe
func (h *Handler) HandleRecipeUpdate(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req models.UpdateRecipeRequest

	if err := decodeRequest(r, &req, func() error {
		req = models.UpdateRecipeRequest{
			Name:       r.FormValue("name"),
			BrewerRKey: r.FormValue("brewer_rkey"),
			BrewerType: r.FormValue("brewer_type"),
			Notes:      r.FormValue("notes"),
		}
		if v := r.FormValue("coffee_amount"); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				req.CoffeeAmount = f
			}
		}
		if v := r.FormValue("water_amount"); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				req.WaterAmount = f
			}
		}
		req.Pours = parsePours(r)
		return nil
	}); err != nil {
		log.Warn().Err(err).Msg("Failed to decode recipe update request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		log.Warn().Err(err).Str("name", req.Name).Msg("Recipe update validation failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if errMsg := validateOptionalRKey(req.BrewerRKey, "Brewer selection"); errMsg != "" {
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	if err := store.UpdateRecipeByRKey(r.Context(), rkey, &req); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update recipe")
		handleStoreError(w, err, "Failed to update recipe")
		return
	}

	h.invalidateFeedCache()
	w.WriteHeader(http.StatusOK)
}

// HandleRecipeDelete deletes a recipe
func (h *Handler) HandleRecipeDelete(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if err := store.DeleteRecipeByRKey(r.Context(), rkey); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to delete recipe")
		handleStoreError(w, err, "Failed to delete recipe")
		return
	}

	// Remove from firehose feed index
	if h.feedIndex != nil {
		didStr, _ := atproto.GetAuthenticatedDID(r.Context())
		if didStr != "" {
			if err := h.feedIndex.DeleteRecord(didStr, atproto.NSIDRecipe, rkey); err != nil {
				log.Warn().Err(err).Str("rkey", rkey).Msg("Failed to delete recipe from feed index")
			}
		}
	}

	h.invalidateFeedCache()
	w.Header().Set("HX-Trigger", "entityDeleted")
	w.WriteHeader(http.StatusOK)
}

// HandleRecipeGet returns a single recipe as JSON (for autofill)
// Accepts optional ?owner= query param to fetch from another user's PDS.
func (h *Handler) HandleRecipeGet(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	ownerDID := r.URL.Query().Get("owner")

	var recipe *models.Recipe
	if ownerDID != "" {
		// Fetch from the recipe owner's PDS via public client
		publicClient := atproto.NewPublicClient()
		record, err := publicClient.GetRecord(r.Context(), ownerDID, atproto.NSIDRecipe, rkey)
		if err != nil {
			http.Error(w, "Recipe not found", http.StatusNotFound)
			log.Warn().Err(err).Str("rkey", rkey).Str("owner", ownerDID).Msg("Failed to get recipe from owner PDS")
			return
		}

		recipe, err = atproto.RecordToRecipe(record.Value, record.URI)
		if err != nil {
			http.Error(w, "Failed to parse recipe", http.StatusInternalServerError)
			log.Error().Err(err).Str("rkey", rkey).Msg("Failed to parse recipe record")
			return
		}
		recipe.RKey = rkey
		recipe.AuthorDID = ownerDID

		// Resolve brewer reference: fetch source brewer info, then match
		// against the current user's brewers so the returned brewer_rkey
		// is usable in the current user's brew form.
		if brewerRef, ok := record.Value["brewerRef"].(string); ok && brewerRef != "" {
			brewerRKey := atproto.ExtractRKeyFromURI(brewerRef)
			if brewerRKey != "" {
				brewerRecord, err := publicClient.GetRecord(r.Context(), ownerDID, atproto.NSIDBrewer, brewerRKey)
				if err == nil {
					if brewer, err := atproto.RecordToBrewer(brewerRecord.Value, brewerRecord.URI); err == nil {
						brewer.RKey = brewerRKey
						recipe.BrewerObj = brewer

						// Try to match source brewer to the current user's brewers
						if userBrewers, err := store.ListBrewers(r.Context()); err == nil {
							candidates := make([]matching.Candidate, len(userBrewers))
							for i, b := range userBrewers {
								candidates[i] = matching.Candidate{RKey: b.RKey, Name: b.Name, Type: b.BrewerType}
							}
							if m := matching.Match(brewer.Name, brewer.BrewerType, candidates); m != nil {
								recipe.BrewerRKey = m.RKey
							}
						}
					}
				}
			}
		}
	} else {
		// Fetch from the logged-in user's own PDS
		var err error
		recipe, err = store.GetRecipeByRKey(r.Context(), rkey)
		if err != nil {
			http.Error(w, "Recipe not found", http.StatusNotFound)
			log.Warn().Err(err).Str("rkey", rkey).Msg("Failed to get recipe")
			return
		}
	}

	recipe.Interpolate()
	writeJSON(w, recipe, "recipe")
}

// HandleRecipeCreateFromBrew creates a recipe pre-populated from an existing brew's parameters
func (h *Handler) HandleRecipeCreateFromBrew(w http.ResponseWriter, r *http.Request) {
	brewRKey := validateRKey(w, r.PathValue("id"))
	if brewRKey == "" {
		return
	}

	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Get the brew to extract parameters
	brew, err := store.GetBrewByRKey(r.Context(), brewRKey)
	if err != nil {
		http.Error(w, "Brew not found", http.StatusNotFound)
		log.Warn().Err(err).Str("rkey", brewRKey).Msg("Failed to get brew for recipe creation")
		return
	}

	// Name is required from the form
	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "Recipe name is required", http.StatusBadRequest)
		return
	}

	// Build recipe from brew parameters
	req := &models.CreateRecipeRequest{
		Name:         name,
		BrewerRKey:   brew.BrewerRKey,
		CoffeeAmount: float64(brew.CoffeeAmount),
		WaterAmount:  float64(brew.WaterAmount),
	}

	// Copy pours
	if len(brew.Pours) > 0 {
		req.Pours = make([]models.CreatePourData, len(brew.Pours))
		for i, pour := range brew.Pours {
			req.Pours[i] = models.CreatePourData{
				WaterAmount: pour.WaterAmount,
				TimeSeconds: pour.TimeSeconds,
			}
		}
	}

	// If the brew has a brewer but no brewer type, get the brewer type
	if brew.BrewerObj != nil {
		req.BrewerType = brew.BrewerObj.BrewerType
	}

	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	recipe, err := store.CreateRecipe(r.Context(), req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create recipe from brew")
		handleStoreError(w, err, "Failed to create recipe")
		return
	}

	h.invalidateFeedCache()
	writeJSON(w, recipe, "recipe")
}

// HandleRecipeFork creates a copy of another user's recipe in the current user's PDS.
// The source recipe is identified by rkey + owner query param (handle or DID).
func (h *Handler) HandleRecipeFork(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	owner := r.URL.Query().Get("owner")
	if owner == "" {
		http.Error(w, "owner query parameter is required", http.StatusBadRequest)
		return
	}

	ownerDID, err := resolveOwnerDID(r.Context(), owner)
	if err != nil {
		http.Error(w, "Could not resolve owner", http.StatusNotFound)
		return
	}

	// Fetch the source recipe via public client
	publicClient := atproto.NewPublicClient()
	record, err := publicClient.GetRecord(r.Context(), ownerDID, atproto.NSIDRecipe, rkey)
	if err != nil {
		http.Error(w, "Recipe not found", http.StatusNotFound)
		log.Warn().Err(err).Str("rkey", rkey).Str("owner", owner).Msg("Failed to fetch recipe for fork")
		return
	}

	sourceRecipe, err := atproto.RecordToRecipe(record.Value, record.URI)
	if err != nil {
		http.Error(w, "Failed to parse recipe", http.StatusInternalServerError)
		return
	}

	// Build the source AT-URI for provenance
	sourceURI := atproto.BuildATURI(ownerDID, atproto.NSIDRecipe, rkey)

	// Resolve brewer: try to match source brewer to current user's brewers
	var brewerRKey, brewerType string
	if brewerRef, ok := record.Value["brewerRef"].(string); ok && brewerRef != "" {
		// Fetch the source brewer to get name and type for matching
		brewerRKeySource := atproto.ExtractRKeyFromURI(brewerRef)
		if brewerRKeySource != "" {
			if sourceBrewer, err := publicClient.GetRecord(r.Context(), ownerDID, atproto.NSIDBrewer, brewerRKeySource); err == nil {
				var sourceName, sourceType string
				if n, ok := sourceBrewer.Value["name"].(string); ok {
					sourceName = n
				}
				if t, ok := sourceBrewer.Value["brewerType"].(string); ok {
					sourceType = t
					brewerType = t
				}

				// Match against the current user's brewers
				if userBrewers, err := store.ListBrewers(r.Context()); err == nil {
					candidates := make([]matching.Candidate, len(userBrewers))
					for i, b := range userBrewers {
						candidates[i] = matching.Candidate{RKey: b.RKey, Name: b.Name, Type: b.BrewerType}
					}
					if m := matching.Match(sourceName, sourceType, candidates); m != nil {
						brewerRKey = m.RKey
						log.Debug().Str("matched", m.Name).Float64("score", m.Score).Msg("Matched brewer for recipe fork")
					}
				}
			}
		}
	}
	if brewerType == "" {
		brewerType = sourceRecipe.BrewerType
	}

	// Create a copy in the current user's PDS
	req := &models.CreateRecipeRequest{
		Name:         sourceRecipe.Name,
		BrewerRKey:   brewerRKey,
		BrewerType:   brewerType,
		CoffeeAmount: sourceRecipe.CoffeeAmount,
		WaterAmount:  sourceRecipe.WaterAmount,
		Notes:        sourceRecipe.Notes,
		SourceRef:    sourceURI,
	}

	// Copy pours
	if len(sourceRecipe.Pours) > 0 {
		req.Pours = make([]models.CreatePourData, len(sourceRecipe.Pours))
		for i, pour := range sourceRecipe.Pours {
			req.Pours[i] = models.CreatePourData{
				WaterAmount: pour.WaterAmount,
				TimeSeconds: pour.TimeSeconds,
			}
		}
	}

	recipe, err := store.CreateRecipe(r.Context(), req)
	if err != nil {
		log.Error().Err(err).Str("source", sourceURI).Msg("Failed to fork recipe")
		handleStoreError(w, err, "Failed to fork recipe")
		return
	}

	h.invalidateFeedCache()
	writeJSON(w, recipe, "recipe")
}

// HandleRecipeSuggestions returns filtered recipes from all users via the feed index.
// Query params: q (text search), brewer_type, min_coffee, max_coffee, min_water, max_water, category
func (h *Handler) HandleRecipeSuggestions(w http.ResponseWriter, r *http.Request) {
	_, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	filter := models.RecipeFilter{
		Query:      r.URL.Query().Get("q"),
		BrewerType: r.URL.Query().Get("brewer_type"),
		Category:   r.URL.Query().Get("category"),
	}
	if v := r.URL.Query().Get("min_coffee"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			filter.MinCoffee = f
		}
	}
	if v := r.URL.Query().Get("max_coffee"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			filter.MaxCoffee = f
		}
	}
	if v := r.URL.Query().Get("min_water"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			filter.MinWater = f
		}
	}
	if v := r.URL.Query().Get("max_water"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			filter.MaxWater = f
		}
	}

	recipes, err := h.listAllRecipesFromIndex(r.Context())
	if err != nil {
		http.Error(w, "Failed to list recipes", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to list recipes from feed index")
		return
	}

	filtered := models.FilterRecipes(recipes, filter)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(filtered); err != nil {
		log.Error().Err(err).Msg("Failed to encode recipe suggestions response")
	}
}

// listAllRecipesFromIndex loads all recipe records from the feed index,
// converts them to Recipe models, and populates author info.
func (h *Handler) listAllRecipesFromIndex(ctx context.Context) ([]*models.Recipe, error) {
	if h.feedIndex == nil {
		return nil, fmt.Errorf("feed index not available")
	}

	records, err := h.feedIndex.ListRecordsByCollection(atproto.NSIDRecipe)
	if err != nil {
		return nil, err
	}

	// Batch-collect unique DIDs for profile lookups (authors + source refs)
	didSet := make(map[string]struct{}, len(records))
	for _, rec := range records {
		didSet[rec.DID] = struct{}{}
	}

	// Pre-scan records for sourceRef DIDs so we can batch profile lookups
	type parsedRecord struct {
		uri        string
		did        string
		data       map[string]interface{}
		recipe     *models.Recipe
		sourceRef  string
		sourceDID  string
		sourceRKey string
	}
	parsed := make([]parsedRecord, 0, len(records))
	for i := range records {
		var recordData map[string]interface{}
		if err := json.Unmarshal(records[i].Record, &recordData); err != nil {
			continue
		}
		recipe, err := atproto.RecordToRecipe(recordData, records[i].URI)
		if err != nil {
			continue
		}

		pr := parsedRecord{uri: records[i].URI, did: records[i].DID, data: recordData, recipe: recipe}
		if recipe.SourceRef != "" {
			if c, err := atproto.ResolveATURI(recipe.SourceRef); err == nil {
				pr.sourceDID = c.DID
				pr.sourceRKey = c.RKey
				pr.sourceRef = recipe.SourceRef
				didSet[c.DID] = struct{}{}
			}
		}
		parsed = append(parsed, pr)
	}

	// Resolve profiles for all DIDs (authors + source authors)
	profiles := make(map[string]*atproto.Profile, len(didSet))
	for did := range didSet {
		profile, err := h.feedIndex.GetProfile(ctx, did)
		if err == nil && profile != nil {
			profiles[did] = profile
		}
	}

	// Build fork map: source URI -> list of forker DIDs
	type forkInfo struct {
		count   int
		avatars []string
	}
	forkMap := make(map[string]*forkInfo)
	for _, pr := range parsed {
		if pr.sourceRef != "" {
			fi, ok := forkMap[pr.sourceRef]
			if !ok {
				fi = &forkInfo{}
				forkMap[pr.sourceRef] = fi
			}
			fi.count++
			if len(fi.avatars) < 5 {
				if p, ok := profiles[pr.did]; ok && p.Avatar != nil && *p.Avatar != "" {
					fi.avatars = append(fi.avatars, *p.Avatar)
				}
			}
		}
	}

	// Batch query brew counts per recipe
	brewCounts := h.feedIndex.BrewCountsByRecipeURI()

	// Build final recipe list
	recipes := make([]*models.Recipe, 0, len(parsed))
	for _, pr := range parsed {
		recipe := pr.recipe

		// Resolve brewer reference from the record data
		if brewerRef, ok := pr.data["brewerRef"].(string); ok && brewerRef != "" {
			if c, parseErr := atproto.ResolveATURI(brewerRef); parseErr == nil {
				recipe.BrewerRKey = c.RKey
			}
			if brewerRec, getErr := h.feedIndex.GetRecord(brewerRef); getErr == nil && brewerRec != nil {
				var brewerData map[string]interface{}
				if err := json.Unmarshal(brewerRec.Record, &brewerData); err == nil {
					if brewer, err := atproto.RecordToBrewer(brewerData, brewerRef); err == nil {
						recipe.BrewerObj = brewer
					}
				}
			}
		}
		if recipe.BrewerType == "" && recipe.BrewerObj != nil {
			recipe.BrewerType = recipe.BrewerObj.BrewerType
		}

		// Populate author info
		recipe.AuthorDID = pr.did
		if profile, ok := profiles[pr.did]; ok {
			recipe.AuthorHandle = profile.Handle
			if profile.Avatar != nil {
				recipe.AuthorAvatar = *profile.Avatar
			}
			if profile.DisplayName != nil {
				recipe.AuthorDisplay = *profile.DisplayName
			}
		}

		// Populate source author info
		if pr.sourceDID != "" {
			if profile, ok := profiles[pr.sourceDID]; ok {
				recipe.SourceAuthorHandle = profile.Handle
				if profile.Avatar != nil {
					recipe.SourceAuthorAvatar = *profile.Avatar
				}
				if profile.DisplayName != nil && *profile.DisplayName != "" {
					recipe.SourceAuthorDisplay = *profile.DisplayName
				} else {
					recipe.SourceAuthorDisplay = profile.Handle
				}
			}
		}

		// Populate social stats
		recipeURI := pr.uri
		if fi, ok := forkMap[recipeURI]; ok {
			recipe.ForkCount = fi.count
			recipe.ForkerAvatars = fi.avatars
		}
		if bc, ok := brewCounts[recipeURI]; ok {
			recipe.BrewCount = bc
		}

		recipe.Interpolate()
		recipes = append(recipes, recipe)
	}

	return recipes, nil
}

// HandleRecipeList returns all recipes as JSON
func (h *Handler) HandleRecipeList(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	recipes, err := store.ListRecipes(r.Context())
	if err != nil {
		http.Error(w, "Failed to list recipes", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to list recipes")
		return
	}

	// Resolve brewer references using cached data
	brewers, _ := store.ListBrewers(r.Context())
	brewerMap := make(map[string]*models.Brewer, len(brewers))
	for _, b := range brewers {
		brewerMap[b.RKey] = b
	}
	for _, recipe := range recipes {
		if recipe.BrewerRKey != "" {
			recipe.BrewerObj = brewerMap[recipe.BrewerRKey]
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(recipes); err != nil {
		log.Error().Err(err).Msg("Failed to encode recipes response")
	}
}

// HandleRecipeModalNew returns the recipe creation modal HTML
func (h *Handler) HandleRecipeModalNew(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	brewers, err := store.ListBrewers(r.Context())
	if err != nil {
		log.Warn().Err(err).Msg("Failed to fetch brewers for recipe modal")
		brewers = []*models.Brewer{}
	}

	brewersSlice := make([]models.Brewer, len(brewers))
	for i, b := range brewers {
		brewersSlice[i] = *b
	}

	if err := components.RecipeDialogModal(nil, brewersSlice).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render recipe modal")
	}
}

// HandleRecipeModalEdit returns the recipe edit modal HTML
func (h *Handler) HandleRecipeModalEdit(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	recipe, err := store.GetRecipeByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Recipe not found", http.StatusNotFound)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get recipe for modal")
		return
	}

	brewers, err := store.ListBrewers(r.Context())
	if err != nil {
		log.Warn().Err(err).Msg("Failed to fetch brewers for recipe modal")
		brewers = []*models.Brewer{}
	}

	brewersSlice := make([]models.Brewer, len(brewers))
	for i, b := range brewers {
		brewersSlice[i] = *b
	}

	if err := components.RecipeDialogModal(recipe, brewersSlice).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render recipe modal")
	}
}

// HandleRecipeExplore renders the recipe explore page
func (h *Handler) HandleRecipeExplore(w http.ResponseWriter, r *http.Request) {
	_, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	layoutData, _, _ := h.layoutDataFromRequest(r, "Explore Recipes")

	if err := pages.RecipeExplorePage(layoutData, pages.RecipeExploreProps{
		IsAuthenticated: authenticated,
		UserDID:         layoutData.UserDID,
	}).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render recipe explore page")
	}
}

// HandlePopularRecipesPartial returns an HTML fragment of popular recipes for the home page.
func (h *Handler) HandlePopularRecipesPartial(w http.ResponseWriter, r *http.Request) {
	recipes, err := h.listAllRecipesFromIndex(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch recipes for popular section")
		return
	}

	// Sort by popularity: brew_count + fork_count, descending
	sort.Slice(recipes, func(i, j int) bool {
		si := recipes[i].BrewCount + recipes[i].ForkCount
		sj := recipes[j].BrewCount + recipes[j].ForkCount
		return si > sj
	})

	// Take top 6
	if len(recipes) > 6 {
		recipes = recipes[:6]
	}

	if err := components.PopularRecipes(components.PopularRecipesProps{
		Recipes: recipes,
	}).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render popular recipes")
	}
}
