package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"arabica/internal/atproto"
	"arabica/internal/database"
	"arabica/internal/feed"
	"arabica/internal/models"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// Config holds handler configuration options
type Config struct {
	// SecureCookies sets the Secure flag on authentication cookies
	// Should be true in production (HTTPS), false for local development (HTTP)
	SecureCookies bool
}

// Handler contains all HTTP handler methods and their dependencies.
// Dependencies are injected via the constructor for better testability.
type Handler struct {
	oauth         *atproto.OAuthManager
	atprotoClient *atproto.Client
	sessionCache  *atproto.SessionCache
	config        Config
	feedService   *feed.Service
	feedRegistry  *feed.Registry
}

// NewHandler creates a new Handler with all required dependencies.
// This constructor pattern ensures the Handler is always fully initialized.
func NewHandler(
	oauth *atproto.OAuthManager,
	atprotoClient *atproto.Client,
	sessionCache *atproto.SessionCache,
	feedService *feed.Service,
	feedRegistry *feed.Registry,
	config Config,
) *Handler {
	return &Handler{
		oauth:         oauth,
		atprotoClient: atprotoClient,
		sessionCache:  sessionCache,
		config:        config,
		feedService:   feedService,
		feedRegistry:  feedRegistry,
	}
}

// validateRKey validates and returns an rkey from a path parameter.
// Returns the rkey if valid, or writes an error response and returns empty string if invalid.
func validateRKey(w http.ResponseWriter, rkey string) string {
	if rkey == "" {
		http.Error(w, "Record key is required", http.StatusBadRequest)
		return ""
	}
	if !atproto.ValidateRKey(rkey) {
		http.Error(w, "Invalid record key format", http.StatusBadRequest)
		return ""
	}
	return rkey
}

// validateOptionalRKey validates an optional rkey from form data.
// Returns an error message if invalid, empty string if valid or empty.
func validateOptionalRKey(rkey, fieldName string) string {
	if rkey == "" {
		return ""
	}
	if !atproto.ValidateRKey(rkey) {
		return fieldName + " has invalid format"
	}
	return ""
}

// getAtprotoStore creates a user-scoped atproto store from the request context.
// Returns the store and true if authenticated, or nil and false if not authenticated.
func (h *Handler) getAtprotoStore(r *http.Request) (database.Store, bool) {
	// Get authenticated DID from context
	didStr, err := atproto.GetAuthenticatedDID(r.Context())
	if err != nil {
		return nil, false
	}

	// Parse DID string to syntax.DID
	did, err := atproto.ParseDID(didStr)
	if err != nil {
		return nil, false
	}

	// Get session ID from context
	sessionID, err := atproto.GetSessionIDFromContext(r.Context())
	if err != nil {
		return nil, false
	}

	// Create user-scoped atproto store with injected cache
	store := atproto.NewAtprotoStore(h.atprotoClient, did, sessionID, h.sessionCache)
	return store, true
}

// SPA fallback handler - serves index.html for client-side routes
func (h *Handler) HandleSPAFallback(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/app/index.html")
}

// Home page

// API endpoint for feed (JSON)
func (h *Handler) HandleFeedAPI(w http.ResponseWriter, r *http.Request) {
	var feedItems []*feed.FeedItem

	// Check if user is authenticated
	_, err := atproto.GetAuthenticatedDID(r.Context())
	isAuthenticated := err == nil

	if h.feedService != nil {
		if isAuthenticated {
			feedItems, _ = h.feedService.GetRecentRecords(r.Context(), feed.FeedLimit)
		} else {
			// Unauthenticated users get a limited feed from the cache
			feedItems, _ = h.feedService.GetCachedPublicFeed(r.Context())
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"items":           feedItems,
		"isAuthenticated": isAuthenticated,
	}); err != nil {
		log.Error().Err(err).Msg("Failed to encode feed API response")
	}
}

// HandleProfileAPI returns profile data for a given actor (handle or DID)
func (h *Handler) HandleProfileAPI(w http.ResponseWriter, r *http.Request) {
	actor := r.PathValue("actor")
	if actor == "" {
		http.Error(w, "Actor parameter required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Check if current user is authenticated
	currentUserDID, err := atproto.GetAuthenticatedDID(ctx)
	isAuthenticated := err == nil && currentUserDID != ""

	// Resolve actor to DID
	var targetDID string
	if strings.HasPrefix(actor, "did:") {
		targetDID = actor
	} else {
		// Resolve handle to DID
		publicClient := atproto.NewPublicClient()
		resolvedDID, err := publicClient.ResolveHandle(ctx, actor)
		if err != nil {
			http.Error(w, "Failed to resolve handle", http.StatusNotFound)
			log.Error().Err(err).Str("actor", actor).Msg("Failed to resolve handle")
			return
		}
		targetDID = resolvedDID
	}

	// Check if viewing own profile
	isOwnProfile := isAuthenticated && currentUserDID == targetDID

	// Get profile info
	publicClient := atproto.NewPublicClient()
	profile, err := publicClient.GetProfile(ctx, targetDID)
	if err != nil {
		log.Warn().Err(err).Str("did", targetDID).Msg("Failed to fetch profile")
		http.Error(w, "Profile not found", http.StatusNotFound)
		return
	}

	// Fetch user's data using public client (works for any user)

	// Fetch all collections in parallel
	g, ctx := errgroup.WithContext(ctx)

	var brewRecords, beanRecords, roasterRecords, grinderRecords, brewerRecords *atproto.PublicListRecordsOutput

	g.Go(func() error {
		var err error
		brewRecords, err = publicClient.ListRecords(ctx, targetDID, atproto.NSIDBrew, 100)
		return err
	})
	g.Go(func() error {
		var err error
		beanRecords, err = publicClient.ListRecords(ctx, targetDID, atproto.NSIDBean, 100)
		return err
	})
	g.Go(func() error {
		var err error
		roasterRecords, err = publicClient.ListRecords(ctx, targetDID, atproto.NSIDRoaster, 100)
		return err
	})
	g.Go(func() error {
		var err error
		grinderRecords, err = publicClient.ListRecords(ctx, targetDID, atproto.NSIDGrinder, 100)
		return err
	})
	g.Go(func() error {
		var err error
		brewerRecords, err = publicClient.ListRecords(ctx, targetDID, atproto.NSIDBrewer, 100)
		return err
	})

	if err := g.Wait(); err != nil {
		http.Error(w, "Failed to fetch profile data", http.StatusInternalServerError)
		log.Error().Err(err).Str("did", targetDID).Msg("Failed to fetch profile data")
		return
	}

	// Convert records to models
	brews := make([]*models.Brew, 0, len(brewRecords.Records))
	for _, rec := range brewRecords.Records {
		brew, err := atproto.RecordToBrew(rec.Value, rec.URI)
		if err == nil {
			brews = append(brews, brew)
		}
	}

	beans := make([]*models.Bean, 0, len(beanRecords.Records))
	for _, rec := range beanRecords.Records {
		bean, err := atproto.RecordToBean(rec.Value, rec.URI)
		if err == nil {
			beans = append(beans, bean)
		}
	}

	roasters := make([]*models.Roaster, 0, len(roasterRecords.Records))
	for _, rec := range roasterRecords.Records {
		roaster, err := atproto.RecordToRoaster(rec.Value, rec.URI)
		if err == nil {
			roasters = append(roasters, roaster)
		}
	}

	grinders := make([]*models.Grinder, 0, len(grinderRecords.Records))
	for _, rec := range grinderRecords.Records {
		grinder, err := atproto.RecordToGrinder(rec.Value, rec.URI)
		if err == nil {
			grinders = append(grinders, grinder)
		}
	}

	brewers := make([]*models.Brewer, 0, len(brewerRecords.Records))
	for _, rec := range brewerRecords.Records {
		brewer, err := atproto.RecordToBrewer(rec.Value, rec.URI)
		if err == nil {
			brewers = append(brewers, brewer)
		}
	}

	// Link beans to roasters
	atproto.LinkBeansToRoasters(beans, roasters)

	// Resolve references in brews
	for _, brew := range brews {
		if brew.BeanRKey != "" {
			for _, bean := range beans {
				if bean.RKey == brew.BeanRKey {
					brew.Bean = bean
					break
				}
			}
		}
		if brew.GrinderRKey != "" {
			for _, grinder := range grinders {
				if grinder.RKey == brew.GrinderRKey {
					brew.GrinderObj = grinder
					break
				}
			}
		}
		if brew.BrewerRKey != "" {
			for _, brewer := range brewers {
				if brewer.RKey == brew.BrewerRKey {
					brew.BrewerObj = brewer
					break
				}
			}
		}
	}

	response := map[string]interface{}{
		"profile":      profile,
		"brews":        brews,
		"beans":        beans,
		"roasters":     roasters,
		"grinders":     grinders,
		"brewers":      brewers,
		"isOwnProfile": isOwnProfile,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().Err(err).Msg("Failed to encode profile API response")
	}
}

// Brew list partial (loaded async via HTMX)
// resolveBrewReferences resolves bean, grinder, and brewer references for a brew
func (h *Handler) resolveBrewReferences(ctx context.Context, brew *models.Brew, ownerDID string, record map[string]interface{}) error {
	publicClient := atproto.NewPublicClient()

	// Resolve bean reference
	if beanRef, ok := record["beanRef"].(string); ok && beanRef != "" {
		beanRecord, err := publicClient.GetRecord(ctx, ownerDID, atproto.NSIDBean, atproto.ExtractRKeyFromURI(beanRef))
		if err == nil {
			if bean, err := atproto.RecordToBean(beanRecord.Value, beanRecord.URI); err == nil {
				brew.Bean = bean

				// Resolve roaster reference for the bean
				if roasterRef, ok := beanRecord.Value["roasterRef"].(string); ok && roasterRef != "" {
					roasterRecord, err := publicClient.GetRecord(ctx, ownerDID, atproto.NSIDRoaster, atproto.ExtractRKeyFromURI(roasterRef))
					if err == nil {
						if roaster, err := atproto.RecordToRoaster(roasterRecord.Value, roasterRecord.URI); err == nil {
							brew.Bean.Roaster = roaster
						}
					}
				}
			}
		}
	}

	// Resolve grinder reference
	if grinderRef, ok := record["grinderRef"].(string); ok && grinderRef != "" {
		grinderRecord, err := publicClient.GetRecord(ctx, ownerDID, atproto.NSIDGrinder, atproto.ExtractRKeyFromURI(grinderRef))
		if err == nil {
			if grinder, err := atproto.RecordToGrinder(grinderRecord.Value, grinderRecord.URI); err == nil {
				brew.GrinderObj = grinder
			}
		}
	}

	// Resolve brewer reference
	if brewerRef, ok := record["brewerRef"].(string); ok && brewerRef != "" {
		brewerRecord, err := publicClient.GetRecord(ctx, ownerDID, atproto.NSIDBrewer, atproto.ExtractRKeyFromURI(brewerRef))
		if err == nil {
			if brewer, err := atproto.RecordToBrewer(brewerRecord.Value, brewerRecord.URI); err == nil {
				brew.BrewerObj = brewer
			}
		}
	}

	return nil
}

// Show edit brew form

// maxPours is the maximum number of pours allowed in a single brew
const maxPours = 100

// parsePours extracts pour data from form values with bounds checking
func parsePours(r *http.Request) []models.CreatePourData {
	var pours []models.CreatePourData

	for i := 0; i < maxPours; i++ {
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
			pours = append(pours, models.CreatePourData{
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
func validateBrewRequest(r *http.Request) (temperature float64, waterAmount, coffeeAmount, timeSeconds, rating int, pours []models.CreatePourData, errs []ValidationError) {
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

	// Check content type - support both JSON and form data
	contentType := r.Header.Get("Content-Type")
	isJSON := strings.Contains(contentType, "application/json")

	var req models.CreateBrewRequest

	if isJSON {
		// Parse JSON body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
	} else {
		// Parse form data (legacy support)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		// Validate input
		temperature, waterAmount, coffeeAmount, timeSeconds, rating, pours, validationErrs := validateBrewRequest(r)
		if len(validationErrs) > 0 {
			// Return first validation error
			http.Error(w, validationErrs[0].Message, http.StatusBadRequest)
			return
		}

		// Validate required fields
		beanRKey := r.FormValue("bean_rkey")
		if beanRKey == "" {
			http.Error(w, "Bean selection is required", http.StatusBadRequest)
			return
		}
		if !atproto.ValidateRKey(beanRKey) {
			http.Error(w, "Invalid bean selection", http.StatusBadRequest)
			return
		}

		// Validate optional rkeys
		grinderRKey := r.FormValue("grinder_rkey")
		if errMsg := validateOptionalRKey(grinderRKey, "Grinder selection"); errMsg != "" {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		brewerRKey := r.FormValue("brewer_rkey")
		if errMsg := validateOptionalRKey(brewerRKey, "Brewer selection"); errMsg != "" {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}

		req = models.CreateBrewRequest{
			BeanRKey:     beanRKey,
			Method:       r.FormValue("method"),
			Temperature:  temperature,
			WaterAmount:  waterAmount,
			CoffeeAmount: coffeeAmount,
			TimeSeconds:  timeSeconds,
			GrindSize:    r.FormValue("grind_size"),
			GrinderRKey:  grinderRKey,
			BrewerRKey:   brewerRKey,
			TastingNotes: r.FormValue("tasting_notes"),
			Rating:       rating,
			Pours:        pours,
		}
	}

	// Validate JSON request
	if isJSON {
		if req.BeanRKey == "" {
			http.Error(w, "Bean selection is required", http.StatusBadRequest)
			return
		}
		if !atproto.ValidateRKey(req.BeanRKey) {
			http.Error(w, "Invalid bean selection", http.StatusBadRequest)
			return
		}
		if errMsg := validateOptionalRKey(req.GrinderRKey, "Grinder selection"); errMsg != "" {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		if errMsg := validateOptionalRKey(req.BrewerRKey, "Brewer selection"); errMsg != "" {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
	}

	brew, err := store.CreateBrew(r.Context(), &req, 1) // User ID not used with atproto
	if err != nil {
		http.Error(w, "Failed to create brew", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to create brew")
		return
	}

	// Return JSON for API calls, redirect for HTMX
	if isJSON {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(brew); err != nil {
			log.Error().Err(err).Msg("Failed to encode brew response")
		}
	} else {
		// Redirect to brew list
		w.Header().Set("HX-Redirect", "/brews")
		w.WriteHeader(http.StatusOK)
	}
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

	// Check content type - support both JSON and form data
	contentType := r.Header.Get("Content-Type")
	isJSON := strings.Contains(contentType, "application/json")

	var req models.CreateBrewRequest

	if isJSON {
		// Parse JSON body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
	} else {
		// Parse form data (legacy support)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		// Validate input
		temperature, waterAmount, coffeeAmount, timeSeconds, rating, pours, validationErrs := validateBrewRequest(r)
		if len(validationErrs) > 0 {
			http.Error(w, validationErrs[0].Message, http.StatusBadRequest)
			return
		}

		// Validate required fields
		beanRKey := r.FormValue("bean_rkey")
		if beanRKey == "" {
			http.Error(w, "Bean selection is required", http.StatusBadRequest)
			return
		}
		if !atproto.ValidateRKey(beanRKey) {
			http.Error(w, "Invalid bean selection", http.StatusBadRequest)
			return
		}

		// Validate optional rkeys
		grinderRKey := r.FormValue("grinder_rkey")
		if errMsg := validateOptionalRKey(grinderRKey, "Grinder selection"); errMsg != "" {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		brewerRKey := r.FormValue("brewer_rkey")
		if errMsg := validateOptionalRKey(brewerRKey, "Brewer selection"); errMsg != "" {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}

		req = models.CreateBrewRequest{
			BeanRKey:     beanRKey,
			Method:       r.FormValue("method"),
			Temperature:  temperature,
			WaterAmount:  waterAmount,
			CoffeeAmount: coffeeAmount,
			TimeSeconds:  timeSeconds,
			GrindSize:    r.FormValue("grind_size"),
			GrinderRKey:  grinderRKey,
			BrewerRKey:   brewerRKey,
			TastingNotes: r.FormValue("tasting_notes"),
			Rating:       rating,
			Pours:        pours,
		}
	}

	// Validate JSON request
	if isJSON {
		if req.BeanRKey == "" {
			http.Error(w, "Bean selection is required", http.StatusBadRequest)
			return
		}
		if !atproto.ValidateRKey(req.BeanRKey) {
			http.Error(w, "Invalid bean selection", http.StatusBadRequest)
			return
		}
		if errMsg := validateOptionalRKey(req.GrinderRKey, "Grinder selection"); errMsg != "" {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		if errMsg := validateOptionalRKey(req.BrewerRKey, "Brewer selection"); errMsg != "" {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
	}

	err := store.UpdateBrewByRKey(r.Context(), rkey, &req)
	if err != nil {
		http.Error(w, "Failed to update brew", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update brew")
		return
	}

	// Return JSON for API calls, redirect for HTMX
	if isJSON {
		w.WriteHeader(http.StatusOK)
	} else {
		// Redirect to brew list
		w.Header().Set("HX-Redirect", "/brews")
		w.WriteHeader(http.StatusOK)
	}
}

// Delete brew
func (h *Handler) HandleBrewDelete(w http.ResponseWriter, r *http.Request) {
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

	if err := store.DeleteBrewByRKey(r.Context(), rkey); err != nil {
		http.Error(w, "Failed to delete brew", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to delete brew")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Export brews as JSON

// Get a public brew by AT-URI
func (h *Handler) HandleBrewGetPublic(w http.ResponseWriter, r *http.Request) {
	atURI := r.URL.Query().Get("uri")
	if atURI == "" {
		http.Error(w, "Missing 'uri' query parameter", http.StatusBadRequest)
		return
	}

	// Parse AT-URI to extract DID, collection, and rkey
	components, err := atproto.ResolveATURI(atURI)
	if err != nil {
		http.Error(w, "Invalid AT-URI", http.StatusBadRequest)
		log.Error().Err(err).Str("uri", atURI).Msg("Failed to parse AT-URI")
		return
	}

	// Fetch the record from the user's PDS using PublicClient
	publicClient := atproto.NewPublicClient()
	recordEntry, err := publicClient.GetRecord(r.Context(), components.DID, components.Collection, components.RKey)
	if err != nil {
		http.Error(w, "Failed to fetch brew", http.StatusInternalServerError)
		log.Error().Err(err).Str("uri", atURI).Msg("Failed to fetch public brew")
		return
	}

	// Convert the record to a Brew model
	brew, err := atproto.RecordToBrew(recordEntry.Value, atURI)
	if err != nil {
		http.Error(w, "Failed to parse brew record", http.StatusInternalServerError)
		log.Error().Err(err).Str("uri", atURI).Msg("Failed to convert record to brew")
		return
	}

	// Fetch referenced entities (bean, grinder, brewer) if they exist
	if brew.BeanRKey != "" {
		beanURI := atproto.BuildATURI(components.DID, atproto.NSIDBean, brew.BeanRKey)
		beanRecord, err := publicClient.GetRecord(r.Context(), components.DID, atproto.NSIDBean, brew.BeanRKey)
		if err == nil {
			bean, err := atproto.RecordToBean(beanRecord.Value, beanURI)
			if err == nil {
				brew.Bean = bean

				// Fetch roaster if referenced
				if bean.RoasterRKey != "" {
					roasterURI := atproto.BuildATURI(components.DID, atproto.NSIDRoaster, bean.RoasterRKey)
					roasterRecord, err := publicClient.GetRecord(r.Context(), components.DID, atproto.NSIDRoaster, bean.RoasterRKey)
					if err == nil {
						roaster, err := atproto.RecordToRoaster(roasterRecord.Value, roasterURI)
						if err == nil {
							brew.Bean.Roaster = roaster
						}
					}
				}
			}
		}
	}

	if brew.GrinderRKey != "" {
		grinderURI := atproto.BuildATURI(components.DID, atproto.NSIDGrinder, brew.GrinderRKey)
		grinderRecord, err := publicClient.GetRecord(r.Context(), components.DID, atproto.NSIDGrinder, brew.GrinderRKey)
		if err == nil {
			grinder, err := atproto.RecordToGrinder(grinderRecord.Value, grinderURI)
			if err == nil {
				brew.GrinderObj = grinder
			}
		}
	}

	if brew.BrewerRKey != "" {
		brewerURI := atproto.BuildATURI(components.DID, atproto.NSIDBrewer, brew.BrewerRKey)
		brewerRecord, err := publicClient.GetRecord(r.Context(), components.DID, atproto.NSIDBrewer, brew.BrewerRKey)
		if err == nil {
			brewer, err := atproto.RecordToBrewer(brewerRecord.Value, brewerURI)
			if err == nil {
				brew.BrewerObj = brewer
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(brew)
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
	userDID, _ := atproto.GetAuthenticatedDID(r.Context())

	ctx := r.Context()

	// Fetch all collections in parallel using errgroup
	g, ctx := errgroup.WithContext(ctx)

	var beans []*models.Bean
	var roasters []*models.Roaster
	var grinders []*models.Grinder
	var brewers []*models.Brewer
	var brews []*models.Brew

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
		brews, err = store.ListBrews(ctx, 1) // User ID not used with atproto
		return err
	})

	if err := g.Wait(); err != nil {
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to fetch all data for API")
		return
	}

	// Link beans to roasters
	atproto.LinkBeansToRoasters(beans, roasters)

	response := map[string]interface{}{
		"did":      userDID,
		"beans":    beans,
		"roasters": roasters,
		"grinders": grinders,
		"brewers":  brewers,
		"brews":    brews,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().Err(err).Msg("Failed to encode API response")
	}
}

// API endpoint to get current user info
func (h *Handler) HandleAPIMe(w http.ResponseWriter, r *http.Request) {
	// Get authenticated DID from context
	didStr, err := atproto.GetAuthenticatedDID(r.Context())
	if err != nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Fetch user profile
	publicClient := atproto.NewPublicClient()
	profile, err := publicClient.GetProfile(r.Context(), didStr)
	if err != nil {
		log.Warn().Err(err).Str("did", didStr).Msg("Failed to fetch user profile")
		http.Error(w, "Failed to fetch user profile", http.StatusInternalServerError)
		return
	}

	displayName := ""
	if profile.DisplayName != nil {
		displayName = *profile.DisplayName
	}
	avatar := ""
	if profile.Avatar != nil {
		avatar = *profile.Avatar
	}

	response := map[string]interface{}{
		"did":         didStr,
		"handle":      profile.Handle,
		"displayName": displayName,
		"avatar":      avatar,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().Err(err).Msg("Failed to encode API response")
	}
}

// API endpoint to create bean
func (h *Handler) HandleBeanCreate(w http.ResponseWriter, r *http.Request) {
	var req models.CreateBeanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate optional roaster rkey
	if errMsg := validateOptionalRKey(req.RoasterRKey, "Roaster selection"); errMsg != "" {
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	bean, err := store.CreateBean(r.Context(), &req)
	if err != nil {
		http.Error(w, "Failed to create bean", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to create bean")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(bean); err != nil {
		log.Error().Err(err).Msg("Failed to encode bean response")
	}
}

// API endpoint to create roaster
func (h *Handler) HandleRoasterCreate(w http.ResponseWriter, r *http.Request) {
	var req models.CreateRoasterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	roaster, err := store.CreateRoaster(r.Context(), &req)
	if err != nil {
		http.Error(w, "Failed to create roaster", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to create roaster")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(roaster); err != nil {
		log.Error().Err(err).Msg("Failed to encode roaster response")
	}
}

// Manage page

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

	var req models.UpdateBeanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate optional roaster rkey
	if errMsg := validateOptionalRKey(req.RoasterRKey, "Roaster selection"); errMsg != "" {
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	if err := store.UpdateBeanByRKey(r.Context(), rkey, &req); err != nil {
		http.Error(w, "Failed to update bean", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update bean")
		return
	}

	bean, err := store.GetBeanByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Failed to fetch updated bean", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get bean after update")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(bean); err != nil {
		log.Error().Err(err).Msg("Failed to encode bean response")
	}
}

func (h *Handler) HandleBeanDelete(w http.ResponseWriter, r *http.Request) {
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

	if err := store.DeleteBeanByRKey(r.Context(), rkey); err != nil {
		http.Error(w, "Failed to delete bean", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to delete bean")
		return
	}

	w.WriteHeader(http.StatusOK)
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

	var req models.UpdateRoasterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := store.UpdateRoasterByRKey(r.Context(), rkey, &req); err != nil {
		http.Error(w, "Failed to update roaster", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update roaster")
		return
	}

	roaster, err := store.GetRoasterByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Failed to fetch updated roaster", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get roaster after update")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(roaster); err != nil {
		log.Error().Err(err).Msg("Failed to encode roaster response")
	}
}

func (h *Handler) HandleRoasterDelete(w http.ResponseWriter, r *http.Request) {
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

	if err := store.DeleteRoasterByRKey(r.Context(), rkey); err != nil {
		http.Error(w, "Failed to delete roaster", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to delete roaster")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Grinder CRUD handlers
func (h *Handler) HandleGrinderCreate(w http.ResponseWriter, r *http.Request) {
	var req models.CreateGrinderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	grinder, err := store.CreateGrinder(r.Context(), &req)
	if err != nil {
		http.Error(w, "Failed to create grinder", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to create grinder")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(grinder); err != nil {
		log.Error().Err(err).Msg("Failed to encode grinder response")
	}
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

	var req models.UpdateGrinderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := store.UpdateGrinderByRKey(r.Context(), rkey, &req); err != nil {
		http.Error(w, "Failed to update grinder", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update grinder")
		return
	}

	grinder, err := store.GetGrinderByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Failed to fetch updated grinder", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get grinder after update")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(grinder); err != nil {
		log.Error().Err(err).Msg("Failed to encode grinder response")
	}
}

func (h *Handler) HandleGrinderDelete(w http.ResponseWriter, r *http.Request) {
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

	if err := store.DeleteGrinderByRKey(r.Context(), rkey); err != nil {
		http.Error(w, "Failed to delete grinder", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to delete grinder")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Brewer CRUD handlers
func (h *Handler) HandleBrewerCreate(w http.ResponseWriter, r *http.Request) {
	var req models.CreateBrewerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	brewer, err := store.CreateBrewer(r.Context(), &req)
	if err != nil {
		http.Error(w, "Failed to create brewer", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to create brewer")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(brewer); err != nil {
		log.Error().Err(err).Msg("Failed to encode brewer response")
	}
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

	var req models.UpdateBrewerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := store.UpdateBrewerByRKey(r.Context(), rkey, &req); err != nil {
		http.Error(w, "Failed to update brewer", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update brewer")
		return
	}

	brewer, err := store.GetBrewerByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Failed to fetch updated brewer", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get brewer after update")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(brewer); err != nil {
		log.Error().Err(err).Msg("Failed to encode brewer response")
	}
}

func (h *Handler) HandleBrewerDelete(w http.ResponseWriter, r *http.Request) {
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

	if err := store.DeleteBrewerByRKey(r.Context(), rkey); err != nil {
		http.Error(w, "Failed to delete brewer", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to delete brewer")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// About page

// Terms of Service page

// fetchAllData is a helper that fetches all data types in parallel using errgroup.
// This is used by handlers that need beans, roasters, grinders, and brewers.
func fetchAllData(ctx context.Context, store database.Store) (
	beans []*models.Bean,
	roasters []*models.Roaster,
	grinders []*models.Grinder,
	brewers []*models.Brewer,
	err error,
) {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var fetchErr error
		beans, fetchErr = store.ListBeans(ctx)
		return fetchErr
	})
	g.Go(func() error {
		var fetchErr error
		roasters, fetchErr = store.ListRoasters(ctx)
		return fetchErr
	})
	g.Go(func() error {
		var fetchErr error
		grinders, fetchErr = store.ListGrinders(ctx)
		return fetchErr
	})
	g.Go(func() error {
		var fetchErr error
		brewers, fetchErr = store.ListBrewers(ctx)
		return fetchErr
	})

	err = g.Wait()
	return
}

// HandleProfile displays a user's public profile with their brews and gear
