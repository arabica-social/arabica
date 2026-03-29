package handlers

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"arabica/internal/atproto"
	"arabica/internal/metrics"
	"arabica/internal/models"
	"arabica/internal/moderation"
	"arabica/internal/web/bff"
	"arabica/internal/web/components"
	"arabica/internal/web/pages"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// ProfileDataBundle holds all user data fetched from their PDS for profile display
type ProfileDataBundle struct {
	Beans    []*models.Bean
	Roasters []*models.Roaster
	Grinders []*models.Grinder
	Brewers  []*models.Brewer
	Brews    []*models.Brew
}

// fetchUserProfileData fetches all user data for profile display.
// Tries the witness cache first (firehose index), falling back to the PDS via publicClient.
// Brews are sorted in reverse chronological order (newest first).
func (h *Handler) fetchUserProfileData(ctx context.Context, did string, publicClient *atproto.PublicClient) (*ProfileDataBundle, error) {
	// Try witness cache first — all records for this user may already be indexed
	if bundle := h.fetchProfileFromWitness(ctx, did); bundle != nil {
		return bundle, nil
	}

	return h.fetchProfileFromPDS(ctx, did, publicClient)
}

// fetchProfileFromWitness loads all profile data from the witness cache.
// Returns nil if the witness cache is not configured or the user has no indexed records.
func (h *Handler) fetchProfileFromWitness(ctx context.Context, did string) *ProfileDataBundle {
	if h.witnessCache == nil {
		return nil
	}

	// Load all collections from witness cache
	type collectionResult struct {
		collection string
		records    []*atproto.WitnessRecord
	}

	collections := []string{
		atproto.NSIDBean, atproto.NSIDRoaster, atproto.NSIDGrinder,
		atproto.NSIDBrewer, atproto.NSIDBrew,
	}

	results := make(map[string][]*atproto.WitnessRecord)
	totalRecords := 0
	for _, coll := range collections {
		records, err := h.witnessCache.ListWitnessRecords(ctx, did, coll)
		if err != nil {
			log.Debug().Err(err).Str("did", did).Str("collection", coll).Msg("witness: profile collection error")
			return nil
		}
		results[coll] = records
		totalRecords += len(records)
	}

	// If the witness cache has zero records for this user, fall back to PDS
	// (user may not have been backfilled/indexed yet)
	if totalRecords == 0 {
		return nil
	}

	metrics.WitnessCacheHitsTotal.WithLabelValues("profile").Inc()

	// Convert witness records to models
	beanMap := make(map[string]*models.Bean)
	beanRoasterRefMap := make(map[string]string)
	beans := make([]*models.Bean, 0, len(results[atproto.NSIDBean]))
	for _, wr := range results[atproto.NSIDBean] {
		m, err := atproto.WitnessRecordToMap(wr)
		if err != nil {
			continue
		}
		bean, err := atproto.RecordToBean(m, wr.URI)
		if err != nil {
			continue
		}
		bean.RKey = wr.RKey
		beans = append(beans, bean)
		beanMap[wr.URI] = bean
		if roasterRef, ok := m["roasterRef"].(string); ok && roasterRef != "" {
			beanRoasterRefMap[wr.URI] = roasterRef
			if c, err := atproto.ResolveATURI(roasterRef); err == nil {
				bean.RoasterRKey = c.RKey
			}
		}
	}

	roasterMap := make(map[string]*models.Roaster)
	roasters := make([]*models.Roaster, 0, len(results[atproto.NSIDRoaster]))
	for _, wr := range results[atproto.NSIDRoaster] {
		m, err := atproto.WitnessRecordToMap(wr)
		if err != nil {
			continue
		}
		roaster, err := atproto.RecordToRoaster(m, wr.URI)
		if err != nil {
			continue
		}
		roaster.RKey = wr.RKey
		roasters = append(roasters, roaster)
		roasterMap[wr.URI] = roaster
	}

	grinderMap := make(map[string]*models.Grinder)
	grinders := make([]*models.Grinder, 0, len(results[atproto.NSIDGrinder]))
	for _, wr := range results[atproto.NSIDGrinder] {
		m, err := atproto.WitnessRecordToMap(wr)
		if err != nil {
			continue
		}
		grinder, err := atproto.RecordToGrinder(m, wr.URI)
		if err != nil {
			continue
		}
		grinder.RKey = wr.RKey
		grinders = append(grinders, grinder)
		grinderMap[wr.URI] = grinder
	}

	brewerMap := make(map[string]*models.Brewer)
	brewers := make([]*models.Brewer, 0, len(results[atproto.NSIDBrewer]))
	for _, wr := range results[atproto.NSIDBrewer] {
		m, err := atproto.WitnessRecordToMap(wr)
		if err != nil {
			continue
		}
		brewer, err := atproto.RecordToBrewer(m, wr.URI)
		if err != nil {
			continue
		}
		brewer.RKey = wr.RKey
		brewers = append(brewers, brewer)
		brewerMap[wr.URI] = brewer
	}

	brews := make([]*models.Brew, 0, len(results[atproto.NSIDBrew]))
	for _, wr := range results[atproto.NSIDBrew] {
		m, err := atproto.WitnessRecordToMap(wr)
		if err != nil {
			continue
		}
		brew, err := atproto.RecordToBrew(m, wr.URI)
		if err != nil {
			continue
		}
		brew.RKey = wr.RKey
		// Store full AT-URI refs for resolution below
		if beanRef, ok := m["beanRef"].(string); ok {
			brew.BeanRKey = beanRef
		}
		if grinderRef, ok := m["grinderRef"].(string); ok {
			brew.GrinderRKey = grinderRef
		}
		if brewerRef, ok := m["brewerRef"].(string); ok {
			brew.BrewerRKey = brewerRef
		}
		brews = append(brews, brew)
	}

	// Resolve references (same logic as PDS path)
	for _, bean := range beans {
		if roasterRef, found := beanRoasterRefMap[atproto.BuildATURI(did, atproto.NSIDBean, bean.RKey)]; found {
			if roaster, found := roasterMap[roasterRef]; found {
				bean.Roaster = roaster
			}
		}
	}

	for _, brew := range brews {
		if brew.BeanRKey != "" {
			if bean, found := beanMap[brew.BeanRKey]; found {
				brew.Bean = bean
			}
		}
		if brew.GrinderRKey != "" {
			if grinder, found := grinderMap[brew.GrinderRKey]; found {
				brew.GrinderObj = grinder
			}
		}
		if brew.BrewerRKey != "" {
			if brewer, found := brewerMap[brew.BrewerRKey]; found {
				brew.BrewerObj = brewer
			}
		}
	}

	sort.Slice(brews, func(i, j int) bool {
		return brews[i].CreatedAt.After(brews[j].CreatedAt)
	})

	return &ProfileDataBundle{
		Beans:    beans,
		Roasters: roasters,
		Grinders: grinders,
		Brewers:  brewers,
		Brews:    brews,
	}
}

// fetchProfileFromPDS fetches all user data from their PDS via publicClient in parallel.
func (h *Handler) fetchProfileFromPDS(ctx context.Context, did string, publicClient *atproto.PublicClient) (*ProfileDataBundle, error) {
	metrics.WitnessCacheMissesTotal.WithLabelValues("profile").Inc()

	// Fetch all user data in parallel
	g, gCtx := errgroup.WithContext(ctx)

	var brews []*models.Brew
	var beans []*models.Bean
	var roasters []*models.Roaster
	var grinders []*models.Grinder
	var brewers []*models.Brewer

	// Maps for resolving references
	var beanMap map[string]*models.Bean
	var beanRoasterRefMap map[string]string
	var roasterMap map[string]*models.Roaster
	var brewerMap map[string]*models.Brewer
	var grinderMap map[string]*models.Grinder

	// Fetch beans
	g.Go(func() error {
		output, err := publicClient.ListRecords(gCtx, did, atproto.NSIDBean, 100)
		if err != nil {
			return err
		}
		beanMap = make(map[string]*models.Bean)
		beanRoasterRefMap = make(map[string]string)
		beans = make([]*models.Bean, 0, len(output.Records))
		for _, record := range output.Records {
			bean, err := atproto.RecordToBean(record.Value, record.URI)
			if err != nil {
				continue
			}
			beans = append(beans, bean)
			beanMap[record.URI] = bean
			if roasterRef, ok := record.Value["roasterRef"].(string); ok && roasterRef != "" {
				beanRoasterRefMap[record.URI] = roasterRef
			}
		}
		return nil
	})

	// Fetch roasters
	g.Go(func() error {
		output, err := publicClient.ListRecords(gCtx, did, atproto.NSIDRoaster, 100)
		if err != nil {
			return err
		}
		roasterMap = make(map[string]*models.Roaster)
		roasters = make([]*models.Roaster, 0, len(output.Records))
		for _, record := range output.Records {
			roaster, err := atproto.RecordToRoaster(record.Value, record.URI)
			if err != nil {
				continue
			}
			roasters = append(roasters, roaster)
			roasterMap[record.URI] = roaster
		}
		return nil
	})

	// Fetch grinders
	g.Go(func() error {
		output, err := publicClient.ListRecords(gCtx, did, atproto.NSIDGrinder, 100)
		if err != nil {
			return err
		}
		grinderMap = make(map[string]*models.Grinder)
		grinders = make([]*models.Grinder, 0, len(output.Records))
		for _, record := range output.Records {
			grinder, err := atproto.RecordToGrinder(record.Value, record.URI)
			if err != nil {
				continue
			}
			grinders = append(grinders, grinder)
			grinderMap[record.URI] = grinder
		}
		return nil
	})

	// Fetch brewers
	g.Go(func() error {
		output, err := publicClient.ListRecords(gCtx, did, atproto.NSIDBrewer, 100)
		if err != nil {
			return err
		}
		brewerMap = make(map[string]*models.Brewer)
		brewers = make([]*models.Brewer, 0, len(output.Records))
		for _, record := range output.Records {
			brewer, err := atproto.RecordToBrewer(record.Value, record.URI)
			if err != nil {
				continue
			}
			brewers = append(brewers, brewer)
			brewerMap[record.URI] = brewer
		}
		return nil
	})

	// Fetch brews
	g.Go(func() error {
		output, err := publicClient.ListRecords(gCtx, did, atproto.NSIDBrew, 100)
		if err != nil {
			return err
		}
		brews = make([]*models.Brew, 0, len(output.Records))
		for _, record := range output.Records {
			brew, err := atproto.RecordToBrew(record.Value, record.URI)
			if err != nil {
				continue
			}
			// Store the raw record for reference resolution later
			brew.BeanRKey = ""
			if beanRef, ok := record.Value["beanRef"].(string); ok {
				brew.BeanRKey = beanRef
			}
			if grinderRef, ok := record.Value["grinderRef"].(string); ok {
				brew.GrinderRKey = grinderRef
			}
			if brewerRef, ok := record.Value["brewerRef"].(string); ok {
				brew.BrewerRKey = brewerRef
			}
			brews = append(brews, brew)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Resolve references for beans (roaster refs)
	for _, bean := range beans {
		if roasterRef, found := beanRoasterRefMap[atproto.BuildATURI(did, atproto.NSIDBean, bean.RKey)]; found {
			if roaster, found := roasterMap[roasterRef]; found {
				bean.Roaster = roaster
			}
		}
	}

	// Resolve references for brews
	for _, brew := range brews {
		// Resolve bean reference
		if brew.BeanRKey != "" {
			if bean, found := beanMap[brew.BeanRKey]; found {
				brew.Bean = bean
			}
		}
		// Resolve grinder reference
		if brew.GrinderRKey != "" {
			if grinder, found := grinderMap[brew.GrinderRKey]; found {
				brew.GrinderObj = grinder
			}
		}
		// Resolve brewer reference
		if brew.BrewerRKey != "" {
			if brewer, found := brewerMap[brew.BrewerRKey]; found {
				brew.BrewerObj = brewer
			}
		}
	}

	// Sort brews in reverse chronological order (newest first)
	sort.Slice(brews, func(i, j int) bool {
		return brews[i].CreatedAt.After(brews[j].CreatedAt)
	})

	return &ProfileDataBundle{
		Beans:    beans,
		Roasters: roasters,
		Grinders: grinders,
		Brewers:  brewers,
		Brews:    brews,
	}, nil
}

// HandleProfile displays a user's public profile with their brews and gear
func (h *Handler) HandleProfile(w http.ResponseWriter, r *http.Request) {
	actor := r.PathValue("actor")
	if actor == "" {
		http.Error(w, "Actor parameter is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	publicClient := atproto.NewPublicClient()

	// Determine if actor is a DID or handle
	var did string
	var err error

	if strings.HasPrefix(actor, "did:") {
		// It's already a DID
		did = actor
	} else {
		// It's a handle, resolve to DID
		did, err = publicClient.ResolveHandle(ctx, actor)
		if err != nil {
			log.Warn().Err(err).Str("handle", actor).Msg("Failed to resolve handle")
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Redirect to canonical URL with handle (we'll get the handle from profile)
		// For now, continue with the DID we have
	}

	// Check if user is blacklisted
	if cf := h.loadContentFilter(ctx); cf != nil && cf.IsBlocked(did) {
		layoutData, _, _ := h.layoutDataFromRequest(r, "Profile Not Found")
		w.WriteHeader(http.StatusNotFound)
		if err := pages.ProfileNotFound(layoutData).Render(r.Context(), w); err != nil {
			log.Error().Err(err).Msg("Failed to render profile not found page")
		}
		return
	}

	// Fetch profile
	profile, err := publicClient.GetProfile(ctx, did)
	if err != nil {
		log.Warn().Err(err).Str("did", did).Msg("Failed to fetch profile")
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// If the URL used a DID but we have the handle, redirect to the canonical handle URL
	if strings.HasPrefix(actor, "did:") && profile.Handle != "" {
		http.Redirect(w, r, "/profile/"+profile.Handle, http.StatusFound)
		return
	}

	// Fetch all user data from their PDS
	profileData, err := h.fetchUserProfileData(ctx, did, publicClient)
	if err != nil {
		log.Error().Err(err).Str("did", did).Msg("Failed to fetch user data")
		http.Error(w, "Failed to load profile data", http.StatusInternalServerError)
		return
	}

	// Check if current user is authenticated (for nav bar state)
	_, didStr, isAuthenticated := h.layoutDataFromRequest(r, "Profile")

	// Check if this is an Arabica user (has records or is registered in feed)
	isArabicaUser := h.feedRegistry.IsRegistered(did) ||
		len(profileData.Brews) > 0 || len(profileData.Beans) > 0 ||
		len(profileData.Roasters) > 0 || len(profileData.Grinders) > 0 ||
		len(profileData.Brewers) > 0

	if !isArabicaUser {
		layoutData, _, _ := h.layoutDataFromRequest(r, "Profile Not Found")
		w.WriteHeader(http.StatusNotFound)
		if err := pages.ProfileNotFound(layoutData).Render(r.Context(), w); err != nil {
			log.Error().Err(err).Msg("Failed to render profile not found page")
		}
		return
	}

	// Check if the viewing user is the profile owner
	isOwnProfile := isAuthenticated && didStr == did

	// Convert atproto.Profile to bff.UserProfile
	viewedProfile := &bff.UserProfile{
		Handle: profile.Handle,
	}
	if profile.DisplayName != nil {
		viewedProfile.DisplayName = *profile.DisplayName
	}
	if profile.Avatar != nil {
		viewedProfile.Avatar = *profile.Avatar
	}

	// Create layout data
	pageTitle := "Profile"
	if viewedProfile.DisplayName != "" {
		pageTitle = viewedProfile.DisplayName + " - Profile"
	}
	layoutData, _, _ := h.layoutDataFromRequest(r, pageTitle)

	// Create roaster options for own profile
	var roasterOptions []pages.RoasterOption
	if isOwnProfile {
		for _, roaster := range profileData.Roasters {
			roasterOptions = append(roasterOptions, pages.RoasterOption{
				RKey: roaster.RKey,
				Name: roaster.Name,
			})
		}
	}

	// Create profile props
	profileProps := pages.ProfileProps{
		Profile:      viewedProfile,
		IsOwnProfile: isOwnProfile,
		Roasters:     roasterOptions,
	}

	// Render using templ component
	if err := pages.Profile(layoutData, profileProps).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render profile page")
	}
}

// HandleProfilePartial returns profile data content (loaded async via HTMX)
func (h *Handler) HandleProfilePartial(w http.ResponseWriter, r *http.Request) {
	actor := r.PathValue("actor")
	if actor == "" {
		http.Error(w, "Actor parameter is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	publicClient := atproto.NewPublicClient()

	// Determine if actor is a DID or handle
	var did string
	var err error

	if strings.HasPrefix(actor, "did:") {
		did = actor
	} else {
		did, err = publicClient.ResolveHandle(ctx, actor)
		if err != nil {
			log.Warn().Err(err).Str("handle", actor).Msg("Failed to resolve handle")
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
	}

	// Check if user is blacklisted
	cf := h.loadContentFilter(ctx)
	if cf != nil && cf.IsBlocked(did) {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Fetch all user data from their PDS
	profileData, err := h.fetchUserProfileData(ctx, did, publicClient)
	if err != nil {
		log.Error().Err(err).Str("did", did).Msg("Failed to fetch user data for profile partial")
		http.Error(w, "Failed to load profile data", http.StatusInternalServerError)
		return
	}

	// Filter moderated content from profile
	if cf != nil {
		profileData.Brews = moderation.FilterSlice(cf, profileData.Brews, func(b *models.Brew) (string, string) {
			return atproto.BuildATURI(did, atproto.NSIDBrew, b.RKey), did
		})
	}

	// Check if this is an Arabica user (has records or is registered in feed)
	isArabicaUser := h.feedRegistry.IsRegistered(did) ||
		len(profileData.Brews) > 0 || len(profileData.Beans) > 0 ||
		len(profileData.Roasters) > 0 || len(profileData.Grinders) > 0 ||
		len(profileData.Brewers) > 0

	if !isArabicaUser {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Check if the viewing user is the profile owner
	didStr, err := atproto.GetAuthenticatedDID(ctx)
	isAuthenticated := err == nil && didStr != ""
	isOwnProfile := isAuthenticated && didStr == did

	// Get profile for card rendering
	profile, err := publicClient.GetProfile(ctx, did)
	if err != nil {
		log.Warn().Err(err).Str("did", did).Msg("Failed to fetch profile for profile partial")
		// Continue without profile - cards will show limited info
	}

	// Use handle from profile or fallback
	profileHandle := actor
	if profile != nil {
		profileHandle = profile.Handle
	} else if strings.HasPrefix(actor, "did:") {
		profileHandle = did // Fallback to DID if we can't get handle
	}

	// Get like counts and CIDs for brews from firehose index
	brewLikeCounts := make(map[string]int)
	brewLikedByUser := make(map[string]bool)
	brewCIDs := make(map[string]string)
	var beanBrewCounts, grinderBrewCounts, brewerBrewCounts, roasterBeanCounts map[string]int
	var beanAvgBrewRatings, roasterAvgBrewRatings map[string]float64
	if h.feedIndex != nil && profile != nil {
		// Collect all brew URIs for batch lookup
		brewURIs := make([]string, 0, len(profileData.Brews))
		uriToRKey := make(map[string]string, len(profileData.Brews))
		for _, brew := range profileData.Brews {
			uri := atproto.BuildATURI(profile.DID, atproto.NSIDBrew, brew.RKey)
			brewURIs = append(brewURIs, uri)
			uriToRKey[uri] = brew.RKey
		}

		// Batch fetch like counts, liked status, and records
		batchLikes := h.feedIndex.GetLikeCountsBatch(ctx, brewURIs)
		batchRecords := h.feedIndex.GetRecordsBatch(ctx, brewURIs)
		var batchLiked map[string]bool
		if isAuthenticated {
			batchLiked = h.feedIndex.HasUserLikedBatch(ctx, didStr, brewURIs)
		}

		for uri, rkey := range uriToRKey {
			brewLikeCounts[rkey] = batchLikes[uri]
			if batchLiked != nil {
				brewLikedByUser[rkey] = batchLiked[uri]
			}
			if rec, ok := batchRecords[uri]; ok {
				brewCIDs[rkey] = rec.CID
			}
		}

		// Entity usage counts
		beanBrewCounts = h.feedIndex.BrewCountsByBeanURI(ctx, did)
		grinderBrewCounts = h.feedIndex.BrewCountsByGrinderURI(ctx, did)
		brewerBrewCounts = h.feedIndex.BrewCountsByBrewerURI(ctx, did)
		roasterBeanCounts = h.feedIndex.BeanCountsByRoasterURI(ctx, did)

		// Average brew ratings
		beanAvgBrewRatings = make(map[string]float64)
		for uri, stats := range h.feedIndex.AvgBrewRatingByBeanURI(ctx, did) {
			beanAvgBrewRatings[uri] = stats.Average
		}
		roasterAvgBrewRatings = make(map[string]float64)
		for uri, stats := range h.feedIndex.AvgBrewRatingByRoasterURI(ctx, did) {
			roasterAvgBrewRatings[uri] = stats.Average
		}
	}

	if err := components.ProfileContentPartial(components.ProfileContentPartialProps{
		Brews:             profileData.Brews,
		Beans:             profileData.Beans,
		Roasters:          profileData.Roasters,
		Grinders:          profileData.Grinders,
		Brewers:           profileData.Brewers,
		IsOwnProfile:      isOwnProfile,
		ProfileHandle:     profileHandle,
		Profile:           profile,
		BrewLikeCounts:    brewLikeCounts,
		BrewLikedByUser:   brewLikedByUser,
		BrewCIDs:          brewCIDs,
		IsAuthenticated:   isAuthenticated,
		BeanBrewCounts:        beanBrewCounts,
		GrinderBrewCounts:     grinderBrewCounts,
		BrewerBrewCounts:      brewerBrewCounts,
		RoasterBeanCounts:     roasterBeanCounts,
		BeanAvgBrewRatings:    beanAvgBrewRatings,
		RoasterAvgBrewRatings: roasterAvgBrewRatings,
		ProfileDID:            did,
	}).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render content", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render profile partial")
	}
}
