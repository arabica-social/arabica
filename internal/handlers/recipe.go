package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"arabica/internal/atproto"
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
			GrindSize:  r.FormValue("grind_size"),
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
		http.Error(w, "Failed to create recipe", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to create recipe")
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
			GrindSize:  r.FormValue("grind_size"),
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
		http.Error(w, "Failed to update recipe", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update recipe")
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
		http.Error(w, "Failed to delete recipe", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to delete recipe")
		return
	}

	h.invalidateFeedCache()
	w.WriteHeader(http.StatusOK)
}

// HandleRecipeGet returns a single recipe as JSON (for autofill)
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

	recipe, err := store.GetRecipeByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Recipe not found", http.StatusNotFound)
		log.Warn().Err(err).Str("rkey", rkey).Msg("Failed to get recipe")
		return
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
		GrindSize:    brew.GrindSize,
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
		http.Error(w, "Failed to create recipe", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to create recipe from brew")
		return
	}

	h.invalidateFeedCache()
	writeJSON(w, recipe, "recipe")
}

// HandleRecipeSuggestions returns filtered recipes based on query parameters.
// Query params: q (text search), brewer_type, min_coffee, max_coffee, min_water, max_water, category
func (h *Handler) HandleRecipeSuggestions(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getAtprotoStore(r)
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

	recipes, err := store.ListRecipes(r.Context())
	if err != nil {
		http.Error(w, "Failed to list recipes", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to list recipes for suggestions")
		return
	}

	// Resolve brewer references for display
	brewers, _ := store.ListBrewers(r.Context())
	brewerMap := make(map[string]*models.Brewer, len(brewers))
	for _, b := range brewers {
		brewerMap[b.RKey] = b
	}
	userDID, _ := atproto.GetAuthenticatedDID(r.Context())
	userProfile := h.getUserProfile(r.Context(), userDID)
	for _, recipe := range recipes {
		if recipe.BrewerRKey != "" {
			recipe.BrewerObj = brewerMap[recipe.BrewerRKey]
		}
		// Populate BrewerType from BrewerObj if not set
		if recipe.BrewerType == "" && recipe.BrewerObj != nil {
			recipe.BrewerType = recipe.BrewerObj.BrewerType
		}
		recipe.AuthorDID = userDID
		if userProfile != nil {
			recipe.AuthorHandle = userProfile.Handle
			recipe.AuthorAvatar = userProfile.Avatar
			recipe.AuthorDisplay = userProfile.DisplayName
		}
		recipe.Interpolate()
	}

	filtered := models.FilterRecipes(recipes, filter)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(filtered); err != nil {
		log.Error().Err(err).Msg("Failed to encode recipe suggestions response")
	}
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

	if err := pages.RecipeExplorePage(layoutData, pages.RecipeExploreProps{}).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render recipe explore page")
	}
}
