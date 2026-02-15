package handlers

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"arabica/internal/atproto"
	"arabica/internal/models"
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

// fetchUserProfileData fetches all user data from their PDS in parallel.
// This includes beans, roasters, grinders, brewers, and brews with all references resolved.
// Brews are sorted in reverse chronological order (newest first).
func (h *Handler) fetchUserProfileData(ctx context.Context, did string, publicClient *atproto.PublicClient) (*ProfileDataBundle, error) {
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
	didStr, err := atproto.GetAuthenticatedDID(ctx)
	isAuthenticated := err == nil && didStr != ""

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.getUserProfile(ctx, didStr)
	}

	// Check if this is an Arabica user (has records or is registered in feed)
	isArabicaUser := h.feedRegistry.IsRegistered(did) ||
		len(profileData.Brews) > 0 || len(profileData.Beans) > 0 ||
		len(profileData.Roasters) > 0 || len(profileData.Grinders) > 0 ||
		len(profileData.Brewers) > 0

	if !isArabicaUser {
		layoutData := h.buildLayoutData(r, "Profile Not Found", isAuthenticated, didStr, userProfile)
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
	layoutData := h.buildLayoutData(r, pageTitle, isAuthenticated, didStr, userProfile)

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

	// Fetch all user data from their PDS
	profileData, err := h.fetchUserProfileData(ctx, did, publicClient)
	if err != nil {
		log.Error().Err(err).Str("did", did).Msg("Failed to fetch user data for profile partial")
		http.Error(w, "Failed to load profile data", http.StatusInternalServerError)
		return
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
	if h.feedIndex != nil && profile != nil {
		for _, brew := range profileData.Brews {
			subjectURI := atproto.BuildATURI(profile.DID, atproto.NSIDBrew, brew.RKey)
			brewLikeCounts[brew.RKey] = h.feedIndex.GetLikeCount(subjectURI)
			if isAuthenticated {
				brewLikedByUser[brew.RKey] = h.feedIndex.HasUserLiked(didStr, subjectURI)
			}
			// Get CID from the firehose index record
			if record, err := h.feedIndex.GetRecord(subjectURI); err == nil && record != nil {
				brewCIDs[brew.RKey] = record.CID
			}
		}
	}

	if err := components.ProfileContentPartial(components.ProfileContentPartialProps{
		Brews:           profileData.Brews,
		Beans:           profileData.Beans,
		Roasters:        profileData.Roasters,
		Grinders:        profileData.Grinders,
		Brewers:         profileData.Brewers,
		IsOwnProfile:    isOwnProfile,
		ProfileHandle:   profileHandle,
		Profile:         profile,
		BrewLikeCounts:  brewLikeCounts,
		BrewLikedByUser: brewLikedByUser,
		BrewCIDs:        brewCIDs,
		IsAuthenticated: isAuthenticated,
	}).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render content", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render profile partial")
	}
}
