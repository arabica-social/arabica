package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"arabica/internal/atproto"
	"arabica/internal/firehose"
	"arabica/internal/metrics"
	"arabica/internal/models"
	"arabica/internal/moderation"
	"arabica/internal/ogcard"
	"arabica/internal/web/bff"
	"arabica/internal/web/components"
	"arabica/internal/web/pages"

	"github.com/rs/zerolog/log"
)

// populateBrewOGMetadata sets OpenGraph metadata on layoutData for a brew page.
// This enriches social media previews when brew links are shared.
func (h *Handler) populateBrewOGMetadata(layoutData *components.LayoutData, brew *models.Brew, owner, baseURL, shareURL string) {
	if brew == nil {
		return
	}

	var ogTitle string
	if brew.Bean != nil {
		ogTitle = brew.Bean.Name
	} else {
		ogTitle = "Coffee Brew"
	}

	populateOGFields(layoutData, ogTitle, "brew", owner, baseURL, shareURL)
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

	// Fetch brew (witness cache first, then PDS fallback)
	var brew *models.Brew
	brewURI := atproto.BuildATURI(ownerDID, atproto.NSIDBrew, rkey)
	if h.witnessCache != nil {
		if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), brewURI); wr != nil {
			if m, err := atproto.WitnessRecordToMap(wr); err == nil {
				if b, err := atproto.RecordToBrew(m, wr.URI); err == nil {
					metrics.WitnessCacheHitsTotal.WithLabelValues("brew_og").Inc()
					brew = b
					brew.RKey = rkey
					atproto.ExtractBrewRefRKeys(brew, m)
					h.resolveBrewRefsFromWitness(r.Context(), brew, ownerDID, m)
				}
			}
		}
	}
	if brew == nil {
		metrics.WitnessCacheMissesTotal.WithLabelValues("brew_og").Inc()
		record, err := publicClient.GetRecord(r.Context(), ownerDID, atproto.NSIDBrew, rkey)
		if err != nil {
			log.Error().Err(err).Str("did", ownerDID).Str("rkey", rkey).Msg("Failed to get brew for OG image")
			http.Error(w, "Brew not found", http.StatusNotFound)
			return
		}
		brew, err = atproto.RecordToBrew(record.Value, record.URI)
		if err != nil {
			log.Error().Err(err).Msg("Failed to convert brew record for OG image")
			http.Error(w, "Failed to load brew", http.StatusInternalServerError)
			return
		}
		if err := h.resolveBrewReferences(r.Context(), brew, ownerDID, record.Value); err != nil {
			log.Warn().Err(err).Msg("Failed to resolve some brew references for OG image")
		}
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

	brews, err := store.ListBrews(r.Context(), 1) // User ID is not used with atproto
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch brews")
		handleStoreError(w, err, "Failed to fetch brews")
		return
	}

	if err := components.BrewListTablePartial(components.BrewListTableProps{
		Brews:        brews,
		IsOwnProfile: true,
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

	brewFormProps := pages.BrewFormProps{
		Brew:           nil,
		RecipeRKey:     r.URL.Query().Get("recipe"),
		RecipeOwnerDID: r.URL.Query().Get("recipe_owner"),
	}

	if err := pages.BrewFormPage(layoutData, brewFormProps).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render brew form")
	}
}

// Show brew view page
func (h *Handler) HandleBrewView(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Check if owner (DID or handle) is specified in query params
	owner := r.URL.Query().Get("owner")

	// Check authentication
	didStr, err := atproto.GetAuthenticatedDID(r.Context())
	isAuthenticated := err == nil && didStr != ""

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.getUserProfile(r.Context(), didStr) //nolint: only partially uses layoutDataFromRequest due to complex flow
	}

	var brew *models.Brew
	var brewOwnerDID string
	var isOwner bool
	var subjectURI, subjectCID string

	if owner != "" {
		// Viewing someone else's brew - use public client
		publicClient := atproto.NewPublicClient()

		// Resolve owner to DID if it's a handle
		if strings.HasPrefix(owner, "did:") {
			brewOwnerDID = owner
		} else {
			resolved, err := publicClient.ResolveHandle(r.Context(), owner)
			if err != nil {
				log.Warn().Err(err).Str("handle", owner).Msg("Failed to resolve handle for brew view")
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}
			brewOwnerDID = resolved
		}

		// Try witness cache first for the brew and all its references
		brewURI := atproto.BuildATURI(brewOwnerDID, atproto.NSIDBrew, rkey)
		if h.witnessCache != nil {
			if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), brewURI); wr != nil {
				if m, err := atproto.WitnessRecordToMap(wr); err == nil {
					if b, err := atproto.RecordToBrew(m, wr.URI); err == nil {
						metrics.WitnessCacheHitsTotal.WithLabelValues("brew").Inc()
						brew = b
						brew.RKey = rkey
						atproto.ExtractBrewRefRKeys(brew, m)
						subjectURI = wr.URI
						subjectCID = wr.CID
						h.resolveBrewRefsFromWitness(r.Context(), brew, brewOwnerDID, m)
					}
				}
			}
		}

		if brew == nil {
			// PDS fallback
			metrics.WitnessCacheMissesTotal.WithLabelValues("brew").Inc()
			record, err := publicClient.GetRecord(r.Context(), brewOwnerDID, atproto.NSIDBrew, rkey)
			if err != nil {
				log.Error().Err(err).Str("did", brewOwnerDID).Str("rkey", rkey).Msg("Failed to get brew record")
				http.Error(w, "Brew not found", http.StatusNotFound)
				return
			}

			// Store URI and CID for like button
			subjectURI = record.URI
			subjectCID = record.CID

			// Convert record to brew
			brew, err = atproto.RecordToBrew(record.Value, record.URI)
			if err != nil {
				log.Error().Err(err).Msg("Failed to convert brew record")
				http.Error(w, "Failed to load brew", http.StatusInternalServerError)
				return
			}

			// Resolve references (bean, grinder, brewer)
			if err := h.resolveBrewReferences(r.Context(), brew, brewOwnerDID, record.Value); err != nil {
				log.Warn().Err(err).Msg("Failed to resolve some brew references")
			}
		}

		// Check if viewing user is the owner
		isOwner = isAuthenticated && didStr == brewOwnerDID
	} else {
		// Viewing own brew - require authentication
		store, authenticated := h.getAtprotoStore(r)
		if !authenticated {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// Use type assertion to access GetBrewRecordByRKey
		atprotoStore, ok := store.(*atproto.AtprotoStore)
		if !ok {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			log.Error().Msg("Failed to cast store to AtprotoStore")
			return
		}

		brewRecord, err := atprotoStore.GetBrewRecordByRKey(r.Context(), rkey)
		if err != nil {
			http.Error(w, "Brew not found", http.StatusNotFound)
			log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get brew for view")
			return
		}

		brew = brewRecord.Brew
		subjectURI = brewRecord.URI
		subjectCID = brewRecord.CID
		isOwner = true
	}

	// Construct share URL (needed for both OG metadata and props)
	var shareURL string
	if owner != "" {
		shareURL = fmt.Sprintf("/brews/%s?owner=%s", rkey, owner)
	} else if userProfile != nil && userProfile.Handle != "" {
		shareURL = fmt.Sprintf("/brews/%s?owner=%s", rkey, userProfile.Handle)
	}

	// Create layout data with OpenGraph metadata
	layoutData := h.buildLayoutData(r, "Brew Details", isAuthenticated, didStr, userProfile)
	h.populateBrewOGMetadata(layoutData, brew, owner, h.publicBaseURL(r), shareURL)

	// Get like data
	var isLiked bool
	var likeCount int
	if h.feedIndex != nil && subjectURI != "" {
		likeCount = h.feedIndex.GetLikeCount(r.Context(), subjectURI)
		if isAuthenticated {
			isLiked = h.feedIndex.HasUserLiked(r.Context(), didStr, subjectURI)
		}
	}

	// Get comment data
	var commentCount int
	var comments []firehose.IndexedComment
	if h.feedIndex != nil && subjectURI != "" {
		commentCount = h.feedIndex.GetCommentCount(r.Context(), subjectURI)
		comments = h.feedIndex.GetThreadedCommentsForSubject(r.Context(), subjectURI, 100, didStr)
		comments = h.filterHiddenComments(r.Context(), comments)
	}

	// Get moderation data
	var isModerator, canHideRecord, canBlockUser, isRecordHidden bool
	if h.moderationService != nil && isAuthenticated {
		isModerator = h.moderationService.IsModerator(didStr)
		canHideRecord = h.moderationService.HasPermission(didStr, moderation.PermissionHideRecord)
		canBlockUser = h.moderationService.HasPermission(didStr, moderation.PermissionBlacklistUser)
	}
	if h.moderationStore != nil && isModerator && subjectURI != "" {
		isRecordHidden = h.moderationStore.IsRecordHidden(r.Context(), subjectURI)
	}

	// Create brew view props
	brewViewProps := pages.BrewViewProps{
		Brew:            brew,
		IsOwnProfile:    isOwner,
		IsAuthenticated: isAuthenticated,
		SubjectURI:      subjectURI,
		SubjectCID:      subjectCID,
		IsLiked:         isLiked,
		LikeCount:       likeCount,
		CommentCount:    commentCount,
		Comments:        comments,
		CurrentUserDID:  didStr,
		ShareURL:        shareURL,
		IsModerator:     isModerator,
		CanHideRecord:   canHideRecord,
		CanBlockUser:    canBlockUser,
		IsRecordHidden:  isRecordHidden,
		AuthorDID:       brewOwnerDID,
	}

	// Render using templ component
	if err := pages.BrewView(layoutData, brewViewProps).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render brew view")
	}
}

// resolveBrewReferences resolves bean, grinder, and brewer references for a brew
func (h *Handler) resolveBrewReferences(ctx context.Context, brew *models.Brew, ownerDID string, record map[string]any) error {
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

// resolveBrewRefsFromWitness resolves a brew's references (bean, grinder, brewer, recipe)
// from the witness cache, avoiding PDS calls for public brew views.
func (h *Handler) resolveBrewRefsFromWitness(ctx context.Context, brew *models.Brew, ownerDID string, record map[string]any) {
	if h.witnessCache == nil {
		return
	}

	// Resolve bean (and its roaster)
	if beanRef, _ := record["beanRef"].(string); beanRef != "" {
		if wr, _ := h.witnessCache.GetWitnessRecord(ctx, beanRef); wr != nil {
			if m, err := atproto.WitnessRecordToMap(wr); err == nil {
				if bean, err := atproto.RecordToBean(m, wr.URI); err == nil {
					bean.RKey = wr.RKey
					// Resolve roaster
					if roasterRef, ok := m["roasterRef"].(string); ok && roasterRef != "" {
						if c, err := atproto.ResolveATURI(roasterRef); err == nil {
							bean.RoasterRKey = c.RKey
						}
						if rwr, _ := h.witnessCache.GetWitnessRecord(ctx, roasterRef); rwr != nil {
							if rm, err := atproto.WitnessRecordToMap(rwr); err == nil {
								if roaster, err := atproto.RecordToRoaster(rm, rwr.URI); err == nil {
									roaster.RKey = rwr.RKey
									bean.Roaster = roaster
								}
							}
						}
					}
					brew.Bean = bean
				}
			}
		}
	}

	// Resolve grinder
	if grinderRef, _ := record["grinderRef"].(string); grinderRef != "" {
		if wr, _ := h.witnessCache.GetWitnessRecord(ctx, grinderRef); wr != nil {
			if m, err := atproto.WitnessRecordToMap(wr); err == nil {
				if grinder, err := atproto.RecordToGrinder(m, wr.URI); err == nil {
					grinder.RKey = wr.RKey
					brew.GrinderObj = grinder
				}
			}
		}
	}

	// Resolve brewer
	if brewerRef, _ := record["brewerRef"].(string); brewerRef != "" {
		if wr, _ := h.witnessCache.GetWitnessRecord(ctx, brewerRef); wr != nil {
			if m, err := atproto.WitnessRecordToMap(wr); err == nil {
				if brewer, err := atproto.RecordToBrewer(m, wr.URI); err == nil {
					brewer.RKey = wr.RKey
					brew.BrewerObj = brewer
				}
			}
		}
	}

	// Resolve recipe
	if recipeRef, _ := record["recipeRef"].(string); recipeRef != "" {
		if wr, _ := h.witnessCache.GetWitnessRecord(ctx, recipeRef); wr != nil {
			if m, err := atproto.WitnessRecordToMap(wr); err == nil {
				if recipe, err := atproto.RecordToRecipe(m, wr.URI); err == nil {
					recipe.RKey = wr.RKey
					// Resolve recipe's brewer
					if brewerRef, ok := m["brewerRef"].(string); ok && brewerRef != "" {
						if c, err := atproto.ResolveATURI(brewerRef); err == nil {
							recipe.BrewerRKey = c.RKey
						}
						if bwr, _ := h.witnessCache.GetWitnessRecord(ctx, brewerRef); bwr != nil {
							if bm, err := atproto.WitnessRecordToMap(bwr); err == nil {
								if brewer, err := atproto.RecordToBrewer(bm, bwr.URI); err == nil {
									brewer.RKey = bwr.RKey
									recipe.BrewerObj = brewer
								}
							}
						}
					}
					brew.RecipeObj = recipe
				}
			}
		}
	}
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

	brewFormProps := pages.BrewFormProps{
		Brew:      brew,
		PoursJSON: bff.PoursToJSON(brew.Pours),
	}

	if err := pages.BrewFormPage(layoutData, brewFormProps).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render brew edit form")
	}
}

// parseEspressoParams extracts espresso-specific params from form values.
// Returns nil if no espresso params were provided.
func parseEspressoParams(r *http.Request) *models.EspressoParams {
	yieldStr := r.FormValue("espresso_yield_weight")
	pressureStr := r.FormValue("espresso_pressure")
	preInfStr := r.FormValue("espresso_pre_infusion_seconds")

	if yieldStr == "" && pressureStr == "" && preInfStr == "" {
		return nil
	}

	ep := &models.EspressoParams{}
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
func parsePouroverParams(r *http.Request) *models.PouroverParams {
	bloomWaterStr := r.FormValue("pourover_bloom_water")
	bloomSecsStr := r.FormValue("pourover_bloom_seconds")
	drawdownStr := r.FormValue("pourover_drawdown_seconds")
	bypassStr := r.FormValue("pourover_bypass_water")
	filterStr := strings.TrimSpace(r.FormValue("pourover_filter"))

	if bloomWaterStr == "" && bloomSecsStr == "" && drawdownStr == "" && bypassStr == "" && filterStr == "" {
		return nil
	}

	pp := &models.PouroverParams{}
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
func parsePours(r *http.Request) []models.CreatePourData {
	var pours []models.CreatePourData

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
	if !atproto.ValidateRKey(beanRKey) {
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

	req := &models.CreateBrewRequest{
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
	if !atproto.ValidateRKey(beanRKey) {
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

	req := &models.CreateBrewRequest{
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
	h.deleteEntity(w, r, store.DeleteBrewByRKey, "brew", atproto.NSIDBrew)
}

// Export brews as JSON
func (h *Handler) HandleBrewExport(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	brews, err := store.ListBrews(r.Context(), 1) // User ID is not used with atproto
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
