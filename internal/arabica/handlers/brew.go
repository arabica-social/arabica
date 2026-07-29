package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	coffeeogcard "tangled.org/arabica.social/arabica/internal/arabica/ogcard"
	arabicastore "tangled.org/arabica.social/arabica/internal/arabica/store"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/metrics"
	"tangled.org/pdewey.com/atp"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/rs/zerolog/log"
)

// HandleBrewOGImage generates a 1200x630 PNG preview card for a brew.
// Used as the og:image for social media embeds.
func (h *Handlers) HandleBrewOGImage(w http.ResponseWriter, r *http.Request) {
	rkey := handlers.ValidateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	owner := r.URL.Query().Get("owner")
	if owner == "" {
		http.Error(w, "owner parameter required", http.StatusBadRequest)
		return
	}

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
	if h.WitnessCache() != nil {
		if wr, _ := h.WitnessCache().GetWitnessRecord(r.Context(), brewURI); wr != nil {
			if m, err := atproto.WitnessRecordToMap(wr); err == nil {
				if b, err := arabica.RecordToBrew(m, wr.URI); err == nil {
					metrics.WitnessCacheHitsTotal.WithLabelValues("brew_og").Inc()
					brew = b
					brew.RKey = rkey
					arabicastore.ExtractBrewRefRKeys(brew, m)
					arabica.HydrateBrewRefs(brew, m, h.WitnessLookup(r.Context()))
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
		arabicastore.ExtractBrewRefRKeys(brew, record.Value)
		arabica.HydrateBrewRefs(brew, record.Value, handlers.PublicLookup(r.Context()))
	}

	card, err := coffeeogcard.DrawBrewCard(brew)
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

// HandleBrewList serves the user's paginated brew list as JSON.
func (h *Handlers) HandleBrewList(w http.ResponseWriter, r *http.Request) {
	h.handleBrewListJSON(w, r)
}

// brewListResult holds fetched brews and pagination state.
type brewListResult struct {
	brews         []*arabica.Brew
	hasMore       bool
	nextOffset    int
	profileHandle string
}

// fetchBrewList loads a page of the user's brews with limit+1 detection.
func (h *Handlers) fetchBrewList(r *http.Request) (*brewListResult, error) {
	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		return nil, errBrewListUnauth
	}

	didStr, _ := atpmiddleware.GetDID(r.Context())
	var profileHandle string
	if p := h.GetUserProfile(r.Context(), didStr); p != nil {
		profileHandle = p.Handle
	}
	if profileHandle == "" {
		profileHandle = didStr
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	brews, err := store.ListBrews(r.Context(), 1, offset, limit+1)
	if err != nil {
		return nil, err
	}

	hasMore := len(brews) > limit
	if hasMore {
		brews = brews[:limit]
	}

	return &brewListResult{
		brews:         brews,
		hasMore:       hasMore,
		nextOffset:    offset + limit,
		profileHandle: profileHandle,
	}, nil
}

// BrewListJSONResponse is the JSON envelope returned by GET /api/brews for the
// SvelteKit SPA. See docs/api/brews.md for the contract.
type BrewListJSONResponse struct {
	Brews      []*arabica.Brew `json:"brews"`
	HasMore    bool            `json:"has_more"`
	NextOffset int             `json:"next_offset"`
}

// handleBrewListJSON returns the brew list as JSON for the SvelteKit SPA.
func (h *Handlers) handleBrewListJSON(w http.ResponseWriter, r *http.Request) {
	res, err := h.fetchBrewList(r)
	if err != nil {
		if err == errBrewListUnauth {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		log.Error().Err(err).Msg("Failed to fetch brews")
		handlers.HandleStoreError(w, err, "Failed to fetch brews")
		return
	}
	handlers.WriteJSON(w, BrewListJSONResponse{
		Brews:      res.brews,
		HasMore:    res.hasMore,
		NextOffset: res.nextOffset,
	}, "brews")
}

// errBrewListUnauth is a sentinel for unauthenticated requests.
var errBrewListUnauth = &brewListError{msg: "Authentication required"}

type brewListError struct{ msg string }

func (e *brewListError) Error() string { return e.msg }

// HandleBrewViewJSON returns brew detail data as JSON for the SvelteKit SPA.
func (h *Handlers) HandleBrewViewJSON(w http.ResponseWriter, r *http.Request) {
	h.RenderEntityViewJSON(w, r, h.brewViewConfig())
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
	if tempStr := r.FormValue("temperature"); tempStr != "" {
		var err error
		temperature, err = strconv.ParseFloat(tempStr, 64)
		if err != nil {
			errs = append(errs, ValidationError{Field: "temperature", Message: "invalid temperature format"})
		} else if temperature < 0 || temperature > 212 {
			errs = append(errs, ValidationError{Field: "temperature", Message: "temperature must be between 0 and 212"})
		}
	}

	if waterStr := r.FormValue("water_amount"); waterStr != "" {
		var err error
		waterAmount, err = strconv.Atoi(waterStr)
		if err != nil {
			errs = append(errs, ValidationError{Field: "water_amount", Message: "invalid water amount"})
		} else if waterAmount < 0 || waterAmount > 10000 {
			errs = append(errs, ValidationError{Field: "water_amount", Message: "water amount must be between 0 and 10000ml"})
		}
	}

	if coffeeStr := r.FormValue("coffee_amount"); coffeeStr != "" {
		var err error
		coffeeAmount, err = strconv.Atoi(coffeeStr)
		if err != nil {
			errs = append(errs, ValidationError{Field: "coffee_amount", Message: "invalid coffee amount"})
		} else if coffeeAmount < 0 || coffeeAmount > 1000 {
			errs = append(errs, ValidationError{Field: "coffee_amount", Message: "coffee amount must be between 0 and 1000g"})
		}
	}

	if timeStr := r.FormValue("time_seconds"); timeStr != "" {
		var err error
		timeSeconds, err = strconv.Atoi(timeStr)
		if err != nil {
			errs = append(errs, ValidationError{Field: "time_seconds", Message: "invalid time"})
		} else if timeSeconds < 0 || timeSeconds > 3600 {
			errs = append(errs, ValidationError{Field: "time_seconds", Message: "brew time must be between 0 and 3600 seconds"})
		}
	}

	if ratingStr := r.FormValue("rating"); ratingStr != "" {
		var err error
		rating, err = strconv.Atoi(ratingStr)
		if err != nil {
			errs = append(errs, ValidationError{Field: "rating", Message: "invalid rating"})
		} else if rating < 0 || rating > 10 {
			errs = append(errs, ValidationError{Field: "rating", Message: "rating must be between 0 and 10"})
		}
	}

	pours = parsePours(r)

	return
}

func (h *Handlers) HandleBrewCreate(w http.ResponseWriter, r *http.Request) {
	// The SPA posts with Accept: application/json;
	// honor that so an expired/missing session surfaces as a JSON 401 the
	// client can react to (opening the session-expired modal) instead of a
	// same-tab redirect that would discard an in-progress brew form.
	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		handlers.WriteRequestError(w, r, http.StatusUnauthorized, "authentication_required", "Authentication required")
		return
	}

	if err := handlers.ParseFormOrMultipart(r, 0); err != nil {
		log.Warn().Err(err).Msg("Failed to parse brew create form")
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	temperature, waterAmount, coffeeAmount, timeSeconds, rating, pours, validationErrs := validateBrewRequest(r)
	if len(validationErrs) > 0 {
		log.Warn().Str("field", validationErrs[0].Field).Str("error", validationErrs[0].Message).Msg("Brew create validation failed")
		http.Error(w, validationErrs[0].Message, http.StatusBadRequest)
		return
	}

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

	grinderRKey := r.FormValue("grinder_rkey")
	if errMsg := handlers.ValidateOptionalRKey(grinderRKey, "Grinder selection"); errMsg != "" {
		log.Warn().Str("grinder_rkey", grinderRKey).Msg("Brew create: invalid grinder rkey")
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	brewerRKey := r.FormValue("brewer_rkey")
	if errMsg := handlers.ValidateOptionalRKey(brewerRKey, "Brewer selection"); errMsg != "" {
		log.Warn().Str("brewer_rkey", brewerRKey).Msg("Brew create: invalid brewer rkey")
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	recipeRKey := r.FormValue("recipe_rkey")
	if errMsg := handlers.ValidateOptionalRKey(recipeRKey, "Recipe selection"); errMsg != "" {
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

	brew, err := store.CreateBrew(r.Context(), req, 1) // User ID not used with atproto
	if err != nil {
		log.Error().Err(err).Msg("Failed to create brew")
		handlers.HandleStoreErrorForRequest(w, r, err, "Failed to create brew")
		return
	}

	h.InvalidateFeedCache()

	var nudge *BeanNudge
	ctx := r.Context()
	if beanRKey != "" {
		if bean, beanErr := store.GetBeanByRKey(ctx, beanRKey); beanErr == nil && bean != nil && bean.IsIncomplete() {
			nudge = &BeanNudge{
				EntityType:    "bean",
				RKey:          bean.RKey,
				Name:          bean.Name,
				MissingFields: strings.Join(bean.MissingFields(), ", "),
			}
		}
	}

	// JSON callers receive the model and incomplete-bean nudge.
	if handlers.WantsJSON(r) {
		authorDID, _ := atpmiddleware.GetDID(r.Context())
		handlers.WriteJSON(w, BrewMutationJSONResponse{Brew: brew, AuthorDID: authorDID, IncompleteNudge: nudge}, "brew")
		return
	}

	// Form callers receive the nudge header and redirect target.
	if nudge != nil {
		w.Header().Set("X-Incomplete-Nudge", fmt.Sprintf(`{"entity_type":"bean","rkey":"%s","name":"%s","missing":"%s"}`,
			nudge.RKey, nudge.Name, nudge.MissingFields))
	}
	w.Header().Set("HX-Redirect", "/my-coffee")
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) HandleBrewUpdate(w http.ResponseWriter, r *http.Request) {
	rkey := handlers.ValidateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if err := handlers.ParseFormOrMultipart(r, 0); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msg("Failed to parse brew update form")
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	temperature, waterAmount, coffeeAmount, timeSeconds, rating, pours, validationErrs := validateBrewRequest(r)
	if len(validationErrs) > 0 {
		log.Warn().Str("rkey", rkey).Str("field", validationErrs[0].Field).Str("error", validationErrs[0].Message).Msg("Brew update validation failed")
		http.Error(w, validationErrs[0].Message, http.StatusBadRequest)
		return
	}

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

	grinderRKey := r.FormValue("grinder_rkey")
	if errMsg := handlers.ValidateOptionalRKey(grinderRKey, "Grinder selection"); errMsg != "" {
		log.Warn().Str("rkey", rkey).Str("grinder_rkey", grinderRKey).Msg("Brew update: invalid grinder rkey")
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	brewerRKey := r.FormValue("brewer_rkey")
	if errMsg := handlers.ValidateOptionalRKey(brewerRKey, "Brewer selection"); errMsg != "" {
		log.Warn().Str("rkey", rkey).Str("brewer_rkey", brewerRKey).Msg("Brew update: invalid brewer rkey")
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	recipeRKey := r.FormValue("recipe_rkey")
	if errMsg := handlers.ValidateOptionalRKey(recipeRKey, "Recipe selection"); errMsg != "" {
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
		handlers.HandleStoreErrorForRequest(w, r, err, "Failed to update brew")
		return
	}

	h.InvalidateFeedCache()

	// JSON callers receive the updated model.
	if handlers.WantsJSON(r) {
		updated, err := store.GetBrewByRKey(r.Context(), rkey)
		if err != nil {
			log.Error().Err(err).Str("rkey", rkey).Msg("Failed to fetch updated brew for JSON response")
			handlers.HandleStoreErrorForRequest(w, r, err, "Failed to fetch updated brew")
			return
		}
		authorDID, _ := atpmiddleware.GetDID(r.Context())
		handlers.WriteJSON(w, BrewMutationJSONResponse{Brew: updated, AuthorDID: authorDID}, "brew")
		return
	}

	w.Header().Set("HX-Redirect", "/my-coffee")
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) HandleBrewDelete(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	h.DeleteEntity(w, r, store.DeleteBrewByRKey, "brew", arabica.NSIDBrew)
}

func (h *Handlers) HandleBrewExport(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	brews, err := store.ListBrews(r.Context(), 1, 0, 0) // limit=0 returns all
	if err != nil {
		log.Error().Err(err).Msg("Failed to list brews for export")
		handlers.HandleStoreError(w, err, "Failed to fetch brews")
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
