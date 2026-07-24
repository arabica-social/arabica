package handlers

import (
	"fmt"
	"net/http"
	"strings"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/pdewey.com/atp"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/rs/zerolog/log"
)

// BrewMutationJSONResponse is the JSON envelope returned by brew create/update
// handlers when the client requests JSON (Accept: application/json or no
// __redirect form value). The brew model carries json tags so it serializes
// directly; incomplete_nudge is populated when the referenced bean is missing
// fields, mirroring the X-Incomplete-Nudge header the HTMX path sets.
// AuthorDID carries the owning user's DID so the SPA can navigate to the
// brew's view URL (/brews/{actor}/{rkey}) after a create/update. The Brew
// record model itself does not carry authorship — it is derived from the
// authenticated session.
type BrewMutationJSONResponse struct {
	Brew            *arabica.Brew `json:"brew"`
	AuthorDID       string        `json:"author_did"`
	IncompleteNudge *BeanNudge    `json:"incomplete_nudge,omitempty"`
}

// BeanNudge describes an incomplete bean reference so the SPA can prompt the
// user to fill in missing fields after creating/updating a brew.
type BeanNudge struct {
	EntityType    string `json:"entity_type"`
	RKey          string `json:"rkey"`
	Name          string `json:"name"`
	MissingFields string `json:"missing"`
}

// HandleBrewCreateJSON creates a new brew from a typed JSON request body. It
// mirrors HandleRecipeCreate but uses arabica.CreateBrewRequest and replicates
// the numeric range validation the legacy multipart HandleBrewCreate performs
// via validateBrewRequest. The response is BrewMutationJSONResponse so the SPA
// can navigate to the new brew's view URL and surface an incomplete-bean nudge.
func (h *Handlers) HandleBrewCreateJSON(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		handlers.WriteRequestError(w, r, http.StatusUnauthorized, "authentication_required", "Authentication required")
		return
	}

	var req arabica.CreateBrewRequest
	if err := handlers.DecodeRequest(r, &req, func() error {
		// Form fallback mirrors the legacy multipart handler so non-JSON
		// callers (e.g. legacy HTMX) still work if routed here.
		temperature, waterAmount, coffeeAmount, timeSeconds, rating, pours, errs := validateBrewRequest(r)
		if len(errs) > 0 {
			return fmt.Errorf("%s", errs[0].Message)
		}
		req = arabica.CreateBrewRequest{
			BeanRKey:       r.FormValue("bean_rkey"),
			RecipeRKey:     r.FormValue("recipe_rkey"),
			RecipeOwnerDID: r.FormValue("recipe_owner_did"),
			Method:         r.FormValue("method"),
			Temperature:    temperature,
			WaterAmount:    waterAmount,
			CoffeeAmount:   coffeeAmount,
			TimeSeconds:    timeSeconds,
			GrindSize:      r.FormValue("grind_size"),
			GrinderRKey:    r.FormValue("grinder_rkey"),
			BrewerRKey:     r.FormValue("brewer_rkey"),
			TastingNotes:   r.FormValue("tasting_notes"),
			Rating:         rating,
			Pours:          pours,
		}
		req.EspressoParams = parseEspressoParams(r)
		req.PouroverParams = parsePouroverParams(r)
		return nil
	}); err != nil {
		log.Warn().Err(err).Msg("Failed to decode brew create request")
		handlers.WriteRequestError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		log.Warn().Err(err).Msg("Brew create JSON validation failed")
		handlers.WriteRequestError(w, r, http.StatusBadRequest, "validation_failed", err.Error())
		return
	}

	// Numeric range validation. JSON decoding yields typed values directly, so
	// only range-check (no string parsing). Mirrors validateBrewRequest.
	if msg := validateBrewNumericRanges(&req); msg != "" {
		log.Warn().Str("field", msg).Msg("Brew create JSON numeric validation failed")
		handlers.WriteRequestError(w, r, http.StatusBadRequest, "validation_failed", msg)
		return
	}

	// Required bean rkey.
	if req.BeanRKey == "" {
		handlers.WriteRequestError(w, r, http.StatusBadRequest, "validation_failed", "Bean selection is required")
		return
	}
	if !atp.ValidateRKey(req.BeanRKey) {
		handlers.WriteRequestError(w, r, http.StatusBadRequest, "validation_failed", "Invalid bean selection")
		return
	}

	// Optional rkeys.
	if errMsg := handlers.ValidateOptionalRKey(req.GrinderRKey, "Grinder selection"); errMsg != "" {
		handlers.WriteRequestError(w, r, http.StatusBadRequest, "validation_failed", errMsg)
		return
	}
	if errMsg := handlers.ValidateOptionalRKey(req.BrewerRKey, "Brewer selection"); errMsg != "" {
		handlers.WriteRequestError(w, r, http.StatusBadRequest, "validation_failed", errMsg)
		return
	}
	if errMsg := handlers.ValidateOptionalRKey(req.RecipeRKey, "Recipe selection"); errMsg != "" {
		handlers.WriteRequestError(w, r, http.StatusBadRequest, "validation_failed", errMsg)
		return
	}

	brew, err := store.CreateBrew(r.Context(), &req, 1)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create brew")
		handlers.HandleStoreErrorForRequest(w, r, err, "Failed to create brew")
		return
	}

	h.InvalidateFeedCache()

	// Check if the referenced bean is incomplete and include a nudge so the SPA
	// can prompt the user to fill in missing fields.
	var nudge *BeanNudge
	if req.BeanRKey != "" {
		if bean, beanErr := store.GetBeanByRKey(r.Context(), req.BeanRKey); beanErr == nil && bean != nil && bean.IsIncomplete() {
			nudge = &BeanNudge{
				EntityType:    "bean",
				RKey:          bean.RKey,
				Name:          bean.Name,
				MissingFields: strings.Join(bean.MissingFields(), ", "),
			}
		}
	}

	authorDID, _ := atpmiddleware.GetDID(r.Context())
	handlers.WriteJSON(w, BrewMutationJSONResponse{Brew: brew, AuthorDID: authorDID, IncompleteNudge: nudge}, "brew")
}

// HandleBrewUpdateJSON updates an existing brew from a typed JSON request body.
// Mirrors HandleRecipeUpdate: validate the path rkey, decode, validate, update,
// re-fetch, and respond with BrewMutationJSONResponse.
func (h *Handlers) HandleBrewUpdateJSON(w http.ResponseWriter, r *http.Request) {
	rkey := handlers.ValidateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		handlers.WriteRequestError(w, r, http.StatusUnauthorized, "authentication_required", "Authentication required")
		return
	}

	var req arabica.CreateBrewRequest
	if err := handlers.DecodeRequest(r, &req, func() error {
		temperature, waterAmount, coffeeAmount, timeSeconds, rating, pours, errs := validateBrewRequest(r)
		if len(errs) > 0 {
			return fmt.Errorf("%s", errs[0].Message)
		}
		req = arabica.CreateBrewRequest{
			BeanRKey:       r.FormValue("bean_rkey"),
			RecipeRKey:     r.FormValue("recipe_rkey"),
			RecipeOwnerDID: r.FormValue("recipe_owner_did"),
			Method:         r.FormValue("method"),
			Temperature:    temperature,
			WaterAmount:    waterAmount,
			CoffeeAmount:   coffeeAmount,
			TimeSeconds:    timeSeconds,
			GrindSize:      r.FormValue("grind_size"),
			GrinderRKey:    r.FormValue("grinder_rkey"),
			BrewerRKey:     r.FormValue("brewer_rkey"),
			TastingNotes:   r.FormValue("tasting_notes"),
			Rating:         rating,
			Pours:          pours,
		}
		req.EspressoParams = parseEspressoParams(r)
		req.PouroverParams = parsePouroverParams(r)
		return nil
	}); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msg("Failed to decode brew update request")
		handlers.WriteRequestError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msg("Brew update JSON validation failed")
		handlers.WriteRequestError(w, r, http.StatusBadRequest, "validation_failed", err.Error())
		return
	}

	if msg := validateBrewNumericRanges(&req); msg != "" {
		log.Warn().Str("rkey", rkey).Str("field", msg).Msg("Brew update JSON numeric validation failed")
		handlers.WriteRequestError(w, r, http.StatusBadRequest, "validation_failed", msg)
		return
	}

	if req.BeanRKey == "" {
		handlers.WriteRequestError(w, r, http.StatusBadRequest, "validation_failed", "Bean selection is required")
		return
	}
	if !atp.ValidateRKey(req.BeanRKey) {
		handlers.WriteRequestError(w, r, http.StatusBadRequest, "validation_failed", "Invalid bean selection")
		return
	}

	if errMsg := handlers.ValidateOptionalRKey(req.GrinderRKey, "Grinder selection"); errMsg != "" {
		handlers.WriteRequestError(w, r, http.StatusBadRequest, "validation_failed", errMsg)
		return
	}
	if errMsg := handlers.ValidateOptionalRKey(req.BrewerRKey, "Brewer selection"); errMsg != "" {
		handlers.WriteRequestError(w, r, http.StatusBadRequest, "validation_failed", errMsg)
		return
	}
	if errMsg := handlers.ValidateOptionalRKey(req.RecipeRKey, "Recipe selection"); errMsg != "" {
		handlers.WriteRequestError(w, r, http.StatusBadRequest, "validation_failed", errMsg)
		return
	}

	if err := store.UpdateBrewByRKey(r.Context(), rkey, &req); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update brew")
		handlers.HandleStoreErrorForRequest(w, r, err, "Failed to update brew")
		return
	}

	h.InvalidateFeedCache()

	updated, err := store.GetBrewByRKey(r.Context(), rkey)
	if err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to fetch updated brew for JSON response")
		handlers.HandleStoreErrorForRequest(w, r, err, "Failed to fetch updated brew")
		return
	}

	authorDID, _ := atpmiddleware.GetDID(r.Context())
	handlers.WriteJSON(w, BrewMutationJSONResponse{Brew: updated, AuthorDID: authorDID}, "brew")
}

// validateBrewNumericRanges checks the typed numeric fields of a
// CreateBrewRequest against the same bounds the legacy multipart
// validateBrewRequest enforces. Returns a human-readable message on the first
// failure, or empty string when all values are in range.
func validateBrewNumericRanges(req *arabica.CreateBrewRequest) string {
	if req.Temperature < 0 || req.Temperature > 212 {
		return "temperature must be between 0 and 212"
	}
	if req.WaterAmount < 0 || req.WaterAmount > 10000 {
		return "water amount must be between 0 and 10000ml"
	}
	if req.CoffeeAmount < 0 || req.CoffeeAmount > 1000 {
		return "coffee amount must be between 0 and 1000g"
	}
	if req.TimeSeconds < 0 || req.TimeSeconds > 3600 {
		return "brew time must be between 0 and 3600 seconds"
	}
	if req.Rating < 0 || req.Rating > 10 {
		return "rating must be between 0 and 10"
	}
	return ""
}
