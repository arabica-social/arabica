package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"tangled.org/arabica.social/arabica/internal/arabica/web/components"
	"tangled.org/arabica.social/arabica/internal/arabica/web/pages"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/entities/arabica"
	"tangled.org/arabica.social/arabica/internal/metrics"
	"tangled.org/arabica.social/arabica/internal/ogcard"
	"tangled.org/arabica.social/arabica/internal/web/bff"
	"tangled.org/arabica.social/arabica/internal/web/components"
	"tangled.org/pdewey.com/atp"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/rs/zerolog/log"
)

// populateBrewOGMetadata sets OpenGraph metadata on layoutData for a brew page.
// This enriches social media previews when brew links are shared.
func (h *Handler) populateBrewOGMetadata(layoutData *components.LayoutData, brew *arabica.Brew, owner, baseURL, shareURL string) {
	if brew == nil {
		return
	}

	var subtitle string
	if brew.Bean != nil {
		subtitle = brew.Bean.Name
		if brew.Bean.Roaster != nil && brew.Bean.Roaster.Name != "" {
			subtitle += " from " + brew.Bean.Roaster.Name
		}
	}

	populateOGFields(layoutData, subtitle, "brew", owner, baseURL, shareURL)
}

// HandleBrewOGImage generates a 1200x630 PNG preview card for a brew.
// Used as the og:image for social media embeds.
func (h *Handler) HandleBrewOGImage(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	owner := r.URL.Query().Get("owner")
	if owner == "" {
		http.Error(w, "owner parameter required", http.StatusBadRequest)
		return
	}

	// Resolve owner to DID
	publicClient := atproto.NewPublicClient()
	var ownerDID string
	if strings.HasPrefix(owner, "did:") {
		ownerDID = owner
	} else {
		resolved, err := publicClient.ResolveHandle(r.Context(), owner)
		if err != nil {
			log.Warn().Err(err).Str("handle", owner).Msg("Failed to resolve handle for OG image")
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		ownerDID = resolved
	}

	// Fetch brew (witness cache first, then PDS fallback). Refs are
	// resolved via the source-bound lookup so both paths share one walk.
	var brew *arabica.Brew
	brewURI := atp.BuildATURI(ownerDID, arabica.NSIDBrew, rkey)
	if h.witnessCache != nil {
		if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), brewURI); wr != nil {
			if m, err := atproto.WitnessRecordToMap(wr); err == nil {
				if b, err := arabica.RecordToBrew(m, wr.URI); err == nil {
					metrics.WitnessCacheHitsTotal.WithLabelValues("brew_og").Inc()
					brew = b
					brew.RKey = rkey
					atproto.ExtractBrewRefRKeys(brew, m)
					resolveBrewRefsViaLookup(brew, m, h.witnessLookup(r.Context()))
				}
			}
		}
	}
	if brew == nil {
		metrics.WitnessCacheMissesTotal.WithLabelValues("brew_og").Inc()
		record, err := publicClient.GetPublicRecord(r.Context(), ownerDID, arabica.NSIDBrew, rkey)
		if err != nil {
			log.Error().Err(err).Str("did", ownerDID).Str("rkey", rkey).Msg("Failed to get brew for OG image")
			http.Error(w, "Brew not found", http.StatusNotFound)
			return
		}
		brew, err = arabica.RecordToBrew(record.Value, record.URI)
		if err != nil {
			log.Error().Err(err).Msg("Failed to convert brew record for OG image")
			http.Error(w, "Failed to load brew", http.StatusInternalServerError)
			return
		}
		brew.RKey = rkey
		atproto.ExtractBrewRefRKeys(brew, record.Value)
		resolveBrewRefsViaLookup(brew, record.Value, publicLookup(r.Context()))
	}

	// Generate card
	card, err := ogcard.DrawBrewCard(brew)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate OG image")
		http.Error(w, "Failed to generate image", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400") // 24 hours
	if err := card.EncodePNG(w); err != nil {
		log.Error().Err(err).Msg("Failed to encode OG image")
	}
}

// Brew list partial (loaded async via HTMX)
func (h *Handler) HandleBrewListPartial(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	didStr, _ := atpmiddleware.GetDID(r.Context())
	var profileHandle string
	if p := h.getUserProfile(r.Context(), didStr); p != nil {
		profileHandle = p.Handle
	}
	if profileHandle == "" {
		profileHandle = didStr
	}

	// Parse pagination params
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	// Request limit+1 to detect if there are more results beyond this page.
	brews, err := store.ListBrews(r.Context(), 1, offset, limit+1)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch brews")
		handleStoreError(w, err, "Failed to fetch brews")
		return
	}

	// If we got limit+1 records, the extra one signals there are more coffeepages.
	hasMore := len(brews) > limit
	if hasMore {
		brews = brews[:limit]
	}

	if err := coffee.BrewListTablePartial(coffee.BrewListTableProps{
		Brews:         brews,
		IsOwnProfile:  true,
		ProfileHandle: profileHandle,
		HasMore:       hasMore,
		NextOffset:    offset + limit,
	}).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render content", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render brew list partial")
	}
}

// List all brews
func (h *Handler) HandleBrewList(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/my-coffee", http.StatusMovedPermanently)
}

// Show new brew form
func (h *Handler) HandleBrewNew(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	_, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Don't fetch data from PDS - client will populate dropdowns from cache
	// This makes the page load much faster
	layoutData, _, _ := h.layoutDataFromRequest(r, "New Brew")

	brewFormProps := coffeepages.BrewFormProps{
		Brew:           nil,
		RecipeRKey:     r.URL.Query().Get("recipe"),
		RecipeOwnerDID: r.URL.Query().Get("recipe_owner"),
	}

	if err := coffeepages.BrewFormPage(layoutData, brewFormProps).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render brew form")
	}
}

// Show brew view page
func (h *Handler) HandleBrewView(w http.ResponseWriter, r *http.Request) {
	h.handleEntityView(w, r, h.brewViewConfig())
}

// Show edit brew form
func (h *Handler) HandleBrewEdit(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	brew, err := store.GetBrewByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Brew not found", http.StatusNotFound)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get brew for edit")
		return
	}

	// Don't fetch dropdown data from PDS - client will populate from cache
	// This makes the page load much faster
	layoutData, _, _ := h.layoutDataFromRequest(r, "Edit Brew")

	brewFormProps := coffeepages.BrewFormProps{
		Brew:      brew,
		PoursJSON: bff.PoursToJSON(brew.Pours),
	}

	if err := coffeepages.BrewFormPage(layoutData, brewFormProps).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render brew edit form")
	}
}

// parseEspressoParams extracts espresso-specific params from form values.
// Returns nil if no espresso params were provided.
func parseEspressoParams(r *http.Request) *arabica.EspressoParams {
	yieldStr := r.FormValue("espresso_yield_weight")
	pressureStr := r.FormValue("espresso_pressure")
	preInfStr := r.FormValue("espresso_pre_infusion_seconds")

	if yieldStr == "" && pressureStr == "" && preInfStr == "" {
		return nil
	}

	ep := &arabica.EspressoParams{}
	if v, err := strconv.ParseFloat(yieldStr, 64); err == nil && v > 0 {
		ep.YieldWeight = v
	}
	if v, err := strconv.ParseFloat(pressureStr, 64); err == nil && v > 0 {
		ep.Pressure = v
	}
	if v, err := strconv.Atoi(preInfStr); err == nil && v > 0 {
		ep.PreInfusionSeconds = v
	}
	return ep
}

// parsePouroverParams extracts pour-over-specific params from form values.
// Returns nil if no pour-over params were provided.
func parsePouroverParams(r *http.Request) *arabica.PouroverParams {
	bloomWaterStr := r.FormValue("pourover_bloom_water")
	bloomSecsStr := r.FormValue("pourover_bloom_seconds")
	drawdownStr := r.FormValue("pourover_drawdown_seconds")
	bypassStr := r.FormValue("pourover_bypass_water")
	filterStr := strings.TrimSpace(r.FormValue("pourover_filter"))

	if bloomWaterStr == "" && bloomSecsStr == "" && drawdownStr == "" && bypassStr == "" && filterStr == "" {
		return nil
	}

	pp := &arabica.PouroverParams{}
	if v, err := strconv.Atoi(bloomWaterStr); err == nil && v > 0 {
		pp.BloomWater = v
	}
	if v, err := strconv.Atoi(bloomSecsStr); err == nil && v > 0 {
		pp.BloomSeconds = v
	}
	if v, err := strconv.Atoi(drawdownStr); err == nil && v > 0 {
		pp.DrawdownSeconds = v
	}
	if v, err := strconv.Atoi(bypassStr); err == nil && v > 0 {
		pp.BypassWater = v
	}
	pp.Filter = filterStr
	return pp
}

// maxPours is the maximum number of pours allowed in a single brew
const maxPours = 100

// parsePours extracts pour data from form values with bounds checking
func parsePours(r *http.Request) []arabica.CreatePourData {
	var pours []arabica.CreatePourData

	for i := range maxPours {
		waterKey := "pour_water_" + strconv.Itoa(i)
		timeKey := "pour_time_" + strconv.Itoa(i)

		waterStr := r.FormValue(waterKey)
		timeStr := r.FormValue(timeKey)

		if waterStr == "" && timeStr == "" {
			break
		}

		water, _ := strconv.Atoi(waterStr)
		pourTime, _ := strconv.Atoi(timeStr)

		if water > 0 && pourTime >= 0 {
			pours = append(pours, arabica.CreatePourData{
				WaterAmount: water,
				TimeSeconds: pourTime,
			})
		}
	}

	return pours
}

// ValidationError represents a validation error with field name and message
type ValidationError struct {
	Field   string
	Message string
}

// validateBrewRequest validates brew form input and returns any validation errors
func validateBrewRequest(r *http.Request) (temperature float64, waterAmount, coffeeAmount, timeSeconds, rating int, pours []arabica.CreatePourData, errs []ValidationError) {
	// Parse and validate temperature
	if tempStr := r.FormValue("temperature"); tempStr != "" {
		var err error
		temperature, err = strconv.ParseFloat(tempStr, 64)
		if err != nil {
			errs = append(errs, ValidationError{Field: "temperature", Message: "invalid temperature format"})
		} else if temperature < 0 || temperature > 212 {
			errs = append(errs, ValidationError{Field: "temperature", Message: "temperature must be between 0 and 212"})
		}
	}

	// Parse and validate water amount
	if waterStr := r.FormValue("water_amount"); waterStr != "" {
		var err error
		waterAmount, err = strconv.Atoi(waterStr)
		if err != nil {
			errs = append(errs, ValidationError{Field: "water_amount", Message: "invalid water amount"})
		} else if waterAmount < 0 || waterAmount > 10000 {
			errs = append(errs, ValidationError{Field: "water_amount", Message: "water amount must be between 0 and 10000ml"})
		}
	}

	// Parse and validate coffee amount
	if coffeeStr := r.FormValue("coffee_amount"); coffeeStr != "" {
		var err error
		coffeeAmount, err = strconv.Atoi(coffeeStr)
		if err != nil {
			errs = append(errs, ValidationError{Field: "coffee_amount", Message: "invalid coffee amount"})
		} else if coffeeAmount < 0 || coffeeAmount > 1000 {
			errs = append(errs, ValidationError{Field: "coffee_amount", Message: "coffee amount must be between 0 and 1000g"})
		}
	}

	// Parse and validate time
	if timeStr := r.FormValue("time_seconds"); timeStr != "" {
		var err error
		timeSeconds, err = strconv.Atoi(timeStr)
		if err != nil {
			errs = append(errs, ValidationError{Field: "time_seconds", Message: "invalid time"})
		} else if timeSeconds < 0 || timeSeconds > 3600 {
			errs = append(errs, ValidationError{Field: "time_seconds", Message: "brew time must be between 0 and 3600 seconds"})
		}
	}

	// Parse and validate rating
	if ratingStr := r.FormValue("rating"); ratingStr != "" {
		var err error
		rating, err = strconv.Atoi(ratingStr)
		if err != nil {
			errs = append(errs, ValidationError{Field: "rating", Message: "invalid rating"})
		} else if rating < 0 || rating > 10 {
			errs = append(errs, ValidationError{Field: "rating", Message: "rating must be between 0 and 10"})
		}
	}

	// Parse pours
	pours = parsePours(r)

	return
}

// Create new brew
func (h *Handler) HandleBrewCreate(w http.ResponseWriter, r *http.Request) {
	// Require authentication first
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Warn().Err(err).Msg("Failed to parse brew create form")
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Validate input
	temperature, waterAmount, coffeeAmount, timeSeconds, rating, pours, validationErrs := validateBrewRequest(r)
	if len(validationErrs) > 0 {
		log.Warn().Str("field", validationErrs[0].Field).Str("error", validationErrs[0].Message).Msg("Brew create validation failed")
		http.Error(w, validationErrs[0].Message, http.StatusBadRequest)
		return
	}

	// Validate required fields
	beanRKey := r.FormValue("bean_rkey")
	if beanRKey == "" {
		log.Warn().Msg("Brew create: missing bean_rkey")
		http.Error(w, "Bean selection is required", http.StatusBadRequest)
		return
	}
	if !atp.ValidateRKey(beanRKey) {
		log.Warn().Str("bean_rkey", beanRKey).Msg("Brew create: invalid bean rkey format")
		http.Error(w, "Invalid bean selection", http.StatusBadRequest)
		return
	}

	// Validate optional rkeys
	grinderRKey := r.FormValue("grinder_rkey")
	if errMsg := validateOptionalRKey(grinderRKey, "Grinder selection"); errMsg != "" {
		log.Warn().Str("grinder_rkey", grinderRKey).Msg("Brew create: invalid grinder rkey")
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	brewerRKey := r.FormValue("brewer_rkey")
	if errMsg := validateOptionalRKey(brewerRKey, "Brewer selection"); errMsg != "" {
		log.Warn().Str("brewer_rkey", brewerRKey).Msg("Brew create: invalid brewer rkey")
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	recipeRKey := r.FormValue("recipe_rkey")
	if errMsg := validateOptionalRKey(recipeRKey, "Recipe selection"); errMsg != "" {
		log.Warn().Str("recipe_rkey", recipeRKey).Msg("Brew create: invalid recipe rkey")
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	req := &arabica.CreateBrewRequest{
		BeanRKey:       beanRKey,
		RecipeRKey:     recipeRKey,
		RecipeOwnerDID: r.FormValue("recipe_owner_did"),
		Method:         r.FormValue("method"),
		Temperature:    temperature,
		WaterAmount:    waterAmount,
		CoffeeAmount:   coffeeAmount,
		TimeSeconds:    timeSeconds,
		GrindSize:      r.FormValue("grind_size"),
		GrinderRKey:    grinderRKey,
		BrewerRKey:     brewerRKey,
		TastingNotes:   r.FormValue("tasting_notes"),
		Rating:         rating,
		Pours:          pours,
	}
	req.EspressoParams = parseEspressoParams(r)
	req.PouroverParams = parsePouroverParams(r)

	if err := req.Validate(); err != nil {
		log.Warn().Err(err).Msg("Brew create request validation failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := store.CreateBrew(r.Context(), req, 1) // User ID not used with atproto
	if err != nil {
		log.Error().Err(err).Msg("Failed to create brew")
		handleStoreError(w, err, "Failed to create brew")
		return
	}

	h.invalidateFeedCache()

	// Check if the bean is incomplete and include nudge info in response header.
	// The brew form JS reads this before HTMX processes the redirect.
	ctx := r.Context()
	if beanRKey != "" {
		if bean, beanErr := store.GetBeanByRKey(ctx, beanRKey); beanErr == nil && bean != nil && bean.IsIncomplete() {
			nudge := fmt.Sprintf(`{"entity_type":"bean","rkey":"%s","name":"%s","missing":"%s"}`,
				bean.RKey, bean.Name, strings.Join(bean.MissingFields(), ", "))
			w.Header().Set("X-Incomplete-Nudge", nudge)
		}
	}

	w.Header().Set("HX-Redirect", "/my-coffee")
	w.WriteHeader(http.StatusOK)
}

// Update existing brew
func (h *Handler) HandleBrewUpdate(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msg("Failed to parse brew update form")
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Validate input
	temperature, waterAmount, coffeeAmount, timeSeconds, rating, pours, validationErrs := validateBrewRequest(r)
	if len(validationErrs) > 0 {
		log.Warn().Str("rkey", rkey).Str("field", validationErrs[0].Field).Str("error", validationErrs[0].Message).Msg("Brew update validation failed")
		http.Error(w, validationErrs[0].Message, http.StatusBadRequest)
		return
	}

	// Validate required fields
	beanRKey := r.FormValue("bean_rkey")
	if beanRKey == "" {
		log.Warn().Str("rkey", rkey).Msg("Brew update: missing bean_rkey")
		http.Error(w, "Bean selection is required", http.StatusBadRequest)
		return
	}
	if !atp.ValidateRKey(beanRKey) {
		log.Warn().Str("rkey", rkey).Str("bean_rkey", beanRKey).Msg("Brew update: invalid bean rkey format")
		http.Error(w, "Invalid bean selection", http.StatusBadRequest)
		return
	}

	// Validate optional rkeys
	grinderRKey := r.FormValue("grinder_rkey")
	if errMsg := validateOptionalRKey(grinderRKey, "Grinder selection"); errMsg != "" {
		log.Warn().Str("rkey", rkey).Str("grinder_rkey", grinderRKey).Msg("Brew update: invalid grinder rkey")
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	brewerRKey := r.FormValue("brewer_rkey")
	if errMsg := validateOptionalRKey(brewerRKey, "Brewer selection"); errMsg != "" {
		log.Warn().Str("rkey", rkey).Str("brewer_rkey", brewerRKey).Msg("Brew update: invalid brewer rkey")
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	recipeRKey := r.FormValue("recipe_rkey")
	if errMsg := validateOptionalRKey(recipeRKey, "Recipe selection"); errMsg != "" {
		log.Warn().Str("rkey", rkey).Str("recipe_rkey", recipeRKey).Msg("Brew update: invalid recipe rkey")
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	req := &arabica.CreateBrewRequest{
		BeanRKey:       beanRKey,
		RecipeRKey:     recipeRKey,
		RecipeOwnerDID: r.FormValue("recipe_owner_did"),
		Method:         r.FormValue("method"),
		Temperature:    temperature,
		WaterAmount:    waterAmount,
		CoffeeAmount:   coffeeAmount,
		TimeSeconds:    timeSeconds,
		GrindSize:      r.FormValue("grind_size"),
		GrinderRKey:    grinderRKey,
		BrewerRKey:     brewerRKey,
		TastingNotes:   r.FormValue("tasting_notes"),
		Rating:         rating,
		Pours:          pours,
	}
	req.EspressoParams = parseEspressoParams(r)
	req.PouroverParams = parsePouroverParams(r)

	if err := req.Validate(); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msg("Brew update request validation failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := store.UpdateBrewByRKey(r.Context(), rkey, req)
	if err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update brew")
		handleStoreError(w, err, "Failed to update brew")
		return
	}

	h.invalidateFeedCache()

	w.Header().Set("HX-Redirect", "/my-coffee")
	w.WriteHeader(http.StatusOK)
}

// Delete brew
func (h *Handler) HandleBrewDelete(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	h.deleteEntity(w, r, store.DeleteBrewByRKey, "brew", arabica.NSIDBrew)
}

// Export brews as JSON
func (h *Handler) HandleBrewExport(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	brews, err := store.ListBrews(r.Context(), 1, 0, 0) // limit=0 returns all
	if err != nil {
		log.Error().Err(err).Msg("Failed to list brews for export")
		handleStoreError(w, err, "Failed to fetch brews")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=arabica-brews.json")

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(brews); err != nil {
		log.Error().Err(err).Msg("Failed to encode brews for export")
	}
}
