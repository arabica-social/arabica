package handlers

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/entities/arabica"
	"tangled.org/arabica.social/arabica/internal/metrics"
	"tangled.org/arabica.social/arabica/internal/moderation"
	"tangled.org/arabica.social/arabica/internal/web/bff"
	"tangled.org/arabica.social/arabica/internal/web/components"
	"tangled.org/arabica.social/arabica/internal/web/pages"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"
	"tangled.org/pdewey.com/atp"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// ProfileDataBundle holds all user data fetched from their PDS for profile display
type ProfileDataBundle struct {
	Beans    []*arabica.Bean
	Roasters []*arabica.Roaster
	Grinders []*arabica.Grinder
	Brewers  []*arabica.Brewer
	Brews    []*arabica.Brew
}

// fetchUserProfileData fetches all user data for profile display.
// Tries the witness cache first (firehose index), falling back to the PDS via publicClient.
// Brews are sorted in reverse chronological order (newest first).
func (h *Handler) fetchUserProfileData(ctx context.Context, did string, publicClient *atp.PublicClient) (*ProfileDataBundle, error) {
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
		arabica.NSIDBean, arabica.NSIDRoaster, arabica.NSIDGrinder,
		arabica.NSIDBrewer, arabica.NSIDBrew,
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
	beanMap := make(map[string]*arabica.Bean)
	beanRoasterRefMap := make(map[string]string)
	beans := make([]*arabica.Bean, 0, len(results[arabica.NSIDBean]))
	for _, wr := range results[arabica.NSIDBean] {
		m, err := atproto.WitnessRecordToMap(wr)
		if err != nil {
			continue
		}
		bean, err := arabica.RecordToBean(m, wr.URI)
		if err != nil {
			continue
		}
		bean.RKey = wr.RKey
		beans = append(beans, bean)
		beanMap[wr.URI] = bean
		if roasterRef, ok := m["roasterRef"].(string); ok && roasterRef != "" {
			beanRoasterRefMap[wr.URI] = roasterRef
			if rkey := atp.RKeyFromURI(roasterRef); rkey != "" {
				bean.RoasterRKey = rkey
			}
		}
	}

	roasterMap := make(map[string]*arabica.Roaster)
	roasters := make([]*arabica.Roaster, 0, len(results[arabica.NSIDRoaster]))
	for _, wr := range results[arabica.NSIDRoaster] {
		m, err := atproto.WitnessRecordToMap(wr)
		if err != nil {
			continue
		}
		roaster, err := arabica.RecordToRoaster(m, wr.URI)
		if err != nil {
			continue
		}
		roaster.RKey = wr.RKey
		roasters = append(roasters, roaster)
		roasterMap[wr.URI] = roaster
	}

	grinderMap := make(map[string]*arabica.Grinder)
	grinders := make([]*arabica.Grinder, 0, len(results[arabica.NSIDGrinder]))
	for _, wr := range results[arabica.NSIDGrinder] {
		m, err := atproto.WitnessRecordToMap(wr)
		if err != nil {
			continue
		}
		grinder, err := arabica.RecordToGrinder(m, wr.URI)
		if err != nil {
			continue
		}
		grinder.RKey = wr.RKey
		grinders = append(grinders, grinder)
		grinderMap[wr.URI] = grinder
	}

	brewerMap := make(map[string]*arabica.Brewer)
	brewers := make([]*arabica.Brewer, 0, len(results[arabica.NSIDBrewer]))
	for _, wr := range results[arabica.NSIDBrewer] {
		m, err := atproto.WitnessRecordToMap(wr)
		if err != nil {
			continue
		}
		brewer, err := arabica.RecordToBrewer(m, wr.URI)
		if err != nil {
			continue
		}
		brewer.RKey = wr.RKey
		brewers = append(brewers, brewer)
		brewerMap[wr.URI] = brewer
	}

	brews := make([]*arabica.Brew, 0, len(results[arabica.NSIDBrew]))
	for _, wr := range results[arabica.NSIDBrew] {
		m, err := atproto.WitnessRecordToMap(wr)
		if err != nil {
			continue
		}
		brew, err := arabica.RecordToBrew(m, wr.URI)
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
		if roasterRef, found := beanRoasterRefMap[atp.BuildATURI(did, arabica.NSIDBean, bean.RKey)]; found {
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
func (h *Handler) fetchProfileFromPDS(ctx context.Context, did string, publicClient *atp.PublicClient) (*ProfileDataBundle, error) {
	metrics.WitnessCacheMissesTotal.WithLabelValues("profile").Inc()

	// Fetch all user data in parallel
	g, gCtx := errgroup.WithContext(ctx)

	var brews []*arabica.Brew
	var beans []*arabica.Bean
	var roasters []*arabica.Roaster
	var grinders []*arabica.Grinder
	var brewers []*arabica.Brewer

	// Maps for resolving references
	var beanMap map[string]*arabica.Bean
	var beanRoasterRefMap map[string]string
	var roasterMap map[string]*arabica.Roaster
	var brewerMap map[string]*arabica.Brewer
	var grinderMap map[string]*arabica.Grinder

	// Fetch beans
	g.Go(func() error {
		records, _, err := publicClient.ListPublicRecords(gCtx, did, arabica.NSIDBean, atp.ListPublicRecordsOpts{Limit: 100, Reverse: true})
		if err != nil {
			return err
		}
		beanMap = make(map[string]*arabica.Bean)
		beanRoasterRefMap = make(map[string]string)
		beans = make([]*arabica.Bean, 0, len(records))
		for _, record := range records {
			bean, err := arabica.RecordToBean(record.Value, record.URI)
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
		records, _, err := publicClient.ListPublicRecords(gCtx, did, arabica.NSIDRoaster, atp.ListPublicRecordsOpts{Limit: 100, Reverse: true})
		if err != nil {
			return err
		}
		roasterMap = make(map[string]*arabica.Roaster)
		roasters = make([]*arabica.Roaster, 0, len(records))
		for _, record := range records {
			roaster, err := arabica.RecordToRoaster(record.Value, record.URI)
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
		records, _, err := publicClient.ListPublicRecords(gCtx, did, arabica.NSIDGrinder, atp.ListPublicRecordsOpts{Limit: 100, Reverse: true})
		if err != nil {
			return err
		}
		grinderMap = make(map[string]*arabica.Grinder)
		grinders = make([]*arabica.Grinder, 0, len(records))
		for _, record := range records {
			grinder, err := arabica.RecordToGrinder(record.Value, record.URI)
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
		records, _, err := publicClient.ListPublicRecords(gCtx, did, arabica.NSIDBrewer, atp.ListPublicRecordsOpts{Limit: 100, Reverse: true})
		if err != nil {
			return err
		}
		brewerMap = make(map[string]*arabica.Brewer)
		brewers = make([]*arabica.Brewer, 0, len(records))
		for _, record := range records {
			brewer, err := arabica.RecordToBrewer(record.Value, record.URI)
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
		records, _, err := publicClient.ListPublicRecords(gCtx, did, arabica.NSIDBrew, atp.ListPublicRecordsOpts{Limit: 100, Reverse: true})
		if err != nil {
			return err
		}
		brews = make([]*arabica.Brew, 0, len(records))
		for _, record := range records {
			brew, err := arabica.RecordToBrew(record.Value, record.URI)
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
		if roasterRef, found := beanRoasterRefMap[atp.BuildATURI(did, arabica.NSIDBean, bean.RKey)]; found {
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
		did = actor
	} else {
		// Try feed index cache first, fall back to API
		if h.feedIndex != nil {
			did, _ = h.feedIndex.GetDIDByHandle(ctx, actor)
		}
		if did == "" {
			did, err = publicClient.ResolveHandle(ctx, actor)
			if err != nil {
				log.Warn().Err(err).Str("handle", actor).Msg("Failed to resolve handle")
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}
		}
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

	// Fetch profile — try feed index cache first, fall back to API
	var profile *atproto.Profile
	if h.feedIndex != nil {
		profile, _ = h.feedIndex.GetProfile(ctx, did)
	}
	if profile == nil {
		profile, err = publicClient.GetProfile(ctx, did)
		if err != nil {
			log.Warn().Err(err).Str("did", did).Msg("Failed to fetch profile")
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
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
	pageTitle := "@" + viewedProfile.Handle
	if viewedProfile.DisplayName != "" {
		pageTitle = viewedProfile.DisplayName + " (@" + viewedProfile.Handle + ")"
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

	// Parse pagination params for brews tab
	brewsOffset, _ := strconv.Atoi(r.URL.Query().Get("brews_offset"))
	brewsLimit, _ := strconv.Atoi(r.URL.Query().Get("brews_limit"))
	if brewsLimit <= 0 || brewsLimit > 100 {
		brewsLimit = 25
	}

	// Determine if actor is a DID or handle
	var did string
	var err error

	if strings.HasPrefix(actor, "did:") {
		did = actor
	} else {
		// Try feed index cache first, fall back to API
		if h.feedIndex != nil {
			did, _ = h.feedIndex.GetDIDByHandle(ctx, actor)
		}
		if did == "" {
			did, err = publicClient.ResolveHandle(ctx, actor)
			if err != nil {
				log.Warn().Err(err).Str("handle", actor).Msg("Failed to resolve handle")
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}
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
		profileData.Brews = moderation.FilterSlice(cf, profileData.Brews, func(b *arabica.Brew) (string, string) {
			return atp.BuildATURI(did, arabica.NSIDBrew, b.RKey), did
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
	didStr, isAuthenticated := atpmiddleware.GetDID(ctx)
	isOwnProfile := isAuthenticated && didStr == did

	// Get profile for card rendering — try feed index cache first
	var profile *atproto.Profile
	if h.feedIndex != nil {
		profile, _ = h.feedIndex.GetProfile(ctx, did)
	}
	if profile == nil {
		profile, err = publicClient.GetProfile(ctx, did)
		if err != nil {
			log.Warn().Err(err).Str("did", did).Msg("Failed to fetch profile for profile partial")
			// Continue without profile - cards will show limited info
		}
	}

	// Use handle from profile or fallback
	profileHandle := actor
	if profile != nil {
		profileHandle = profile.Handle
	} else if strings.HasPrefix(actor, "did:") {
		profileHandle = did // Fallback to DID if we can't get handle
	}

	// Get like counts, comment counts, and CIDs for brews from firehose index
	brewLikeCounts := make(map[string]int)
	brewCommentCounts := make(map[string]int)
	brewLikedByUser := make(map[string]bool)
	brewCIDs := make(map[string]string)
	var beanBrewCounts, grinderBrewCounts, brewerBrewCounts, roasterBeanCounts map[string]int
	var beanAvgBrewRatings, roasterAvgBrewRatings map[string]float64
	if h.feedIndex != nil && profile != nil {
		// Collect all brew URIs for batch lookup
		brewURIs := make([]string, 0, len(profileData.Brews))
		uriToRKey := make(map[string]string, len(profileData.Brews))
		for _, brew := range profileData.Brews {
			uri := atp.BuildATURI(profile.DID, arabica.NSIDBrew, brew.RKey)
			brewURIs = append(brewURIs, uri)
			uriToRKey[uri] = brew.RKey
		}

		// Batch fetch like counts, liked status, records, and comment counts
		batchLikes := h.feedIndex.GetLikeCountsBatch(ctx, brewURIs)
		batchRecords := h.feedIndex.GetRecordsBatch(ctx, brewURIs)
		var batchLiked map[string]bool
		if isAuthenticated {
			batchLiked = h.feedIndex.HasUserLikedBatch(ctx, didStr, brewURIs)
		}
		batchComments := h.feedIndex.GetCommentCountsBatch(ctx, brewURIs)

		for uri, rkey := range uriToRKey {
			brewLikeCounts[rkey] = batchLikes[uri]
			if batchLiked != nil {
				brewLikedByUser[rkey] = batchLiked[uri]
			}
			if rec, ok := batchRecords[uri]; ok {
				brewCIDs[rkey] = rec.CID
			}
			brewCommentCounts[rkey] = batchComments[uri]
		}

		// Entity usage counts
		beanBrewCounts = h.feedIndex.BrewCountsByBeanURI(ctx, did)
		grinderBrewCounts = h.feedIndex.BrewCountsByGrinderURI(ctx, did)
		brewerBrewCounts = h.feedIndex.BrewCountsByBrewerURI(ctx, did)
		roasterBeanCounts = h.feedIndex.BeanCountsByRoasterURI(ctx, did)

		// Average brew ratings — respect profile visibility settings
		statsVis := h.feedIndex.GetProfileStatsVisibility(ctx, did)
		if isOwnProfile || statsVis.BeanAvgRating == arabica.VisibilityPublic {
			beanAvgBrewRatings = make(map[string]float64)
			for uri, stats := range h.feedIndex.AvgBrewRatingByBeanURI(ctx, did) {
				beanAvgBrewRatings[uri] = stats.Average
			}
		}
		if isOwnProfile || statsVis.RoasterAvgRating == arabica.VisibilityPublic {
			roasterAvgBrewRatings = make(map[string]float64)
			for uri, stats := range h.feedIndex.AvgBrewRatingByRoasterURI(ctx, did) {
				roasterAvgBrewRatings[uri] = stats.Average
			}
		}
	}

	// Apply pagination to brews (brews are already sorted newest-first)
	totalBrews := len(profileData.Brews)
	brewEnd := brewsOffset + brewsLimit
	if brewEnd > totalBrews {
		brewEnd = totalBrews
	}
	brewsHasMore := brewEnd < totalBrews
	if brewsOffset < totalBrews {
		profileData.Brews = profileData.Brews[brewsOffset:brewEnd]
	} else {
		profileData.Brews = nil
	}

	// On load-more requests (offset > 0), render just the brew cards fragment
	if brewsOffset > 0 {
		if err := components.ProfileBrewCards(components.ProfileBrewCardsProps{
			Brews:           profileData.Brews,
			IsOwnProfile:    isOwnProfile,
			ProfileHandle:   profileHandle,
			Profile:         profile,
			LikeCounts:      brewLikeCounts,
			CommentCounts:   brewCommentCounts,
			LikedByUser:     brewLikedByUser,
			BrewCIDs:        brewCIDs,
			IsAuthenticated: isAuthenticated,
			HasMore:         brewsHasMore,
			NextOffset:      brewEnd,
		}).Render(r.Context(), w); err != nil {
			http.Error(w, "Failed to render content", http.StatusInternalServerError)
			log.Error().Err(err).Msg("Failed to render profile brew cards")
		}
		return
	}

	if err := components.ProfileContentPartial(components.ProfileContentPartialProps{
		Brews:                 profileData.Brews,
		Beans:                 profileData.Beans,
		Roasters:              profileData.Roasters,
		Grinders:              profileData.Grinders,
		Brewers:               profileData.Brewers,
		IsOwnProfile:          isOwnProfile,
		ProfileHandle:         profileHandle,
		Profile:               profile,
		BrewLikeCounts:        brewLikeCounts,
		BrewCommentCounts:     brewCommentCounts,
		BrewLikedByUser:       brewLikedByUser,
		BrewCIDs:              brewCIDs,
		IsAuthenticated:       isAuthenticated,
		BeanBrewCounts:        beanBrewCounts,
		GrinderBrewCounts:     grinderBrewCounts,
		BrewerBrewCounts:      brewerBrewCounts,
		RoasterBeanCounts:     roasterBeanCounts,
		BeanAvgBrewRatings:    beanAvgBrewRatings,
		RoasterAvgBrewRatings: roasterAvgBrewRatings,
		ProfileDID:            did,
		BrewsHasMore:          brewsHasMore,
		BrewsNextOffset:       brewEnd,
	}).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render content", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render profile partial")
	}
}
