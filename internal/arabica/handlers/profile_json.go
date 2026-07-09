package coffeehandlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/moderation"
	"tangled.org/arabica.social/arabica/internal/web/bff"
	"tangled.org/pdewey.com/atp"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/rs/zerolog/log"
)

// ProfileJSONResponse is the JSON envelope returned by GET /api/profile/{actor}.
// It combines the profile metadata, paginated brews, entity lists, and all the
// social/usage stats the SvelteKit profile page needs in one response. See
// docs/api/profile.md for the contract.
type ProfileJSONResponse struct {
	Profile               *bff.UserProfile   `json:"profile"`
	DID                   string             `json:"did"`
	IsOwnProfile          bool               `json:"is_own_profile"`
	IsAuthenticated       bool               `json:"is_authenticated"`
	IsArabicaUser         bool               `json:"is_arabica_user"`
	Brews                 []*arabica.Brew    `json:"brews"`
	TotalBrews            int                `json:"total_brews"`
	BrewsHasMore          bool               `json:"brews_has_more"`
	BrewsNextOffset       int                `json:"brews_next_offset"`
	Beans                 []*arabica.Bean    `json:"beans"`
	Roasters              []*arabica.Roaster `json:"roasters"`
	Grinders              []*arabica.Grinder `json:"grinders"`
	Brewers               []*arabica.Brewer  `json:"brewers"`
	BrewLikeCounts        map[string]int     `json:"brew_like_counts"`
	BrewCommentCounts     map[string]int     `json:"brew_comment_counts"`
	BrewLikedByUser       map[string]bool    `json:"brew_liked_by_user"`
	BrewCIDs              map[string]string  `json:"brew_cids"`
	BeanBrewCounts        map[string]int     `json:"bean_brew_counts"`
	GrinderBrewCounts     map[string]int     `json:"grinder_brew_counts"`
	BrewerBrewCounts      map[string]int     `json:"brewer_brew_counts"`
	RoasterBeanCounts     map[string]int     `json:"roaster_bean_counts"`
	BeanAvgBrewRatings    map[string]float64 `json:"bean_avg_brew_ratings"`
	RoasterAvgBrewRatings map[string]float64 `json:"roaster_avg_brew_ratings"`
}

// HandleProfileAPI serves profile data with content negotiation.
// Accept: application/json returns the full profile data bundle as JSON for
// the SvelteKit SPA; HX-Request returns the existing HTML partial.
func (h *Handlers) HandleProfileAPI(w http.ResponseWriter, r *http.Request) {
	if handlers.WantsJSON(r) {
		h.HandleProfileJSON(w, r)
		return
	}
	h.HandleProfilePartial(w, r)
}

// HandleProfileJSON returns a user's profile data as JSON for the SvelteKit SPA.
// It combines the shell-level checks (HandleProfile) with the heavy data fetch
// (HandleProfilePartial): actor resolution, blacklist check, profile fetch,
// paginated brews, entity lists, and all social/usage stats.
func (h *Handlers) HandleProfileJSON(w http.ResponseWriter, r *http.Request) {
	actor := r.PathValue("actor")
	if actor == "" {
		handlers.WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Actor parameter is required")
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

	// Resolve actor to DID
	did, err := h.resolveProfileActor(ctx, actor, publicClient)
	if err != nil {
		handlers.WriteJSONError(w, http.StatusNotFound, "not_found", "User not found")
		return
	}

	// Check if user is blacklisted
	if cf := h.LoadContentFilter(ctx); cf != nil && cf.IsBlocked(did) {
		handlers.WriteJSONError(w, http.StatusNotFound, "not_found", "User not found")
		return
	}

	// Fetch all user data (witness cache first, PDS fallback).
	profileData, err := h.fetchUserProfileData(ctx, did, publicClient, brewsOffset, brewsLimit)
	if err != nil {
		// A PDS fetch failure for an unknown DID is a not-found, not a server
		// error. The witness cache returned nothing (user has no indexed records)
		// and the PDS either returned no records or errored.
		log.Warn().Err(err).Str("did", did).Msg("Failed to fetch user data for profile JSON")
		handlers.WriteJSONError(w, http.StatusNotFound, "not_found", "User not found")
		return
	}

	// Filter moderated content from brews
	if cf := h.LoadContentFilter(ctx); cf != nil {
		profileData.Brews = moderation.FilterSlice(cf, profileData.Brews, func(b *arabica.Brew) (string, string) {
			return atp.BuildATURI(did, arabica.NSIDBrew, b.RKey), did
		})
	}

	// Check if this is an Arabica user
	isArabicaUser := h.FeedRegistry().IsRegistered(did) ||
		len(profileData.Brews) > 0 || len(profileData.Beans) > 0 ||
		len(profileData.Roasters) > 0 || len(profileData.Grinders) > 0 ||
		len(profileData.Brewers) > 0
	if !isArabicaUser {
		handlers.WriteJSONError(w, http.StatusNotFound, "not_found", "User not found")
		return
	}

	// Viewer state
	didStr, isAuthenticated := atpmiddleware.GetDID(ctx)
	isOwnProfile := isAuthenticated && didStr == did

	// Fetch profile for display — try feed index cache first, fall back to
	// API. A failure here is non-fatal: the profile data bundle already
	// confirmed the user exists via their records.
	var profile *atproto.Profile
	if h.FeedIndex() != nil {
		profile, _ = h.FeedIndex().GetProfile(ctx, did)
	}
	if profile == nil {
		profile, err = publicClient.GetProfile(ctx, did)
		if err != nil {
			log.Warn().Err(err).Str("did", did).Msg("Failed to fetch profile for profile JSON")
		}
	}

	// Build the profile summary
	var profileSummary *bff.UserProfile
	if profile != nil {
		profileSummary = &bff.UserProfile{Handle: profile.Handle}
		if profile.DisplayName != nil {
			profileSummary.DisplayName = *profile.DisplayName
		}
		if profile.Avatar != nil {
			profileSummary.Avatar = *profile.Avatar
		}
	} else {
		profileSummary = &bff.UserProfile{Handle: actor}
		if strings.HasPrefix(actor, "did:") {
			profileSummary.Handle = did
		}
	}

	// Compute brew social stats and entity usage counts
	resp := ProfileJSONResponse{
		Profile:           profileSummary,
		DID:               did,
		IsOwnProfile:      isOwnProfile,
		IsAuthenticated:   isAuthenticated,
		IsArabicaUser:     isArabicaUser,
		Brews:             profileData.Brews,
		Beans:             profileData.Beans,
		Roasters:          profileData.Roasters,
		Grinders:          profileData.Grinders,
		Brewers:           profileData.Brewers,
		BrewLikeCounts:    map[string]int{},
		BrewCommentCounts: map[string]int{},
		BrewLikedByUser:   map[string]bool{},
		BrewCIDs:          map[string]string{},
		BeanBrewCounts:    map[string]int{},
		GrinderBrewCounts: map[string]int{},
		BrewerBrewCounts:  map[string]int{},
		RoasterBeanCounts: map[string]int{},
	}

	// Determine pagination state
	totalBrews := profileData.TotalBrews
	if totalBrews == 0 {
		totalBrews = len(profileData.Brews)
	}
	resp.TotalBrews = totalBrews
	brewEnd := min(brewsOffset+brewsLimit, totalBrews)
	resp.BrewsHasMore = brewEnd < totalBrews
	resp.BrewsNextOffset = brewEnd
	if len(resp.Brews) > brewsLimit {
		resp.Brews = resp.Brews[:brewsLimit]
	}

	// Fetch social stats from firehose index
	if h.FeedIndex() != nil && profile != nil {
		brewURIs := make([]string, 0, len(resp.Brews))
		uriToRKey := make(map[string]string, len(resp.Brews))
		for _, brew := range resp.Brews {
			uri := atp.BuildATURI(profile.DID, arabica.NSIDBrew, brew.RKey)
			brewURIs = append(brewURIs, uri)
			uriToRKey[uri] = brew.RKey
		}

		batchLikes := h.FeedIndex().GetLikeCountsBatch(ctx, brewURIs)
		batchRecords := h.FeedIndex().GetRecordsBatch(ctx, brewURIs)
		var batchLiked map[string]bool
		if isAuthenticated {
			batchLiked = h.FeedIndex().HasUserLikedBatch(ctx, didStr, brewURIs)
		}
		batchComments := h.FeedIndex().GetCommentCountsBatch(ctx, brewURIs)

		for uri, rkey := range uriToRKey {
			resp.BrewLikeCounts[rkey] = batchLikes[uri]
			if batchLiked != nil {
				resp.BrewLikedByUser[rkey] = batchLiked[uri]
			}
			if rec, ok := batchRecords[uri]; ok {
				resp.BrewCIDs[rkey] = rec.CID
			}
			resp.BrewCommentCounts[rkey] = batchComments[uri]
		}

		// Entity usage counts
		resp.BeanBrewCounts = h.FeedIndex().BrewCountsByBeanURI(ctx, did)
		resp.GrinderBrewCounts = h.FeedIndex().BrewCountsByGrinderURI(ctx, did)
		resp.BrewerBrewCounts = h.FeedIndex().BrewCountsByBrewerURI(ctx, did)
		resp.RoasterBeanCounts = h.FeedIndex().BeanCountsByRoasterURI(ctx, did)

		// Average brew ratings — respect profile visibility settings
		statsVis := h.FeedIndex().GetProfileStatsVisibility(ctx, did)
		if isOwnProfile || statsVis.BeanAvgRating == arabica.VisibilityPublic {
			resp.BeanAvgBrewRatings = make(map[string]float64)
			for uri, stats := range h.FeedIndex().AvgBrewRatingByBeanURI(ctx, did) {
				resp.BeanAvgBrewRatings[uri] = stats.Average
			}
		}
		if isOwnProfile || statsVis.RoasterAvgRating == arabica.VisibilityPublic {
			resp.RoasterAvgBrewRatings = make(map[string]float64)
			for uri, stats := range h.FeedIndex().AvgBrewRatingByRoasterURI(ctx, did) {
				resp.RoasterAvgBrewRatings[uri] = stats.Average
			}
		}
	}

	handlers.WriteJSON(w, resp, "profile")
}

// resolveProfileActor resolves an actor parameter (DID or handle) to a DID
// string. Tries the feed index cache first, falls back to the public client.
func (h *Handlers) resolveProfileActor(ctx context.Context, actor string, publicClient *atp.PublicClient) (string, error) {
	if strings.HasPrefix(actor, "did:") {
		return actor, nil
	}
	if h.FeedIndex() != nil {
		if did, _ := h.FeedIndex().GetDIDByHandle(ctx, actor); did != "" {
			return did, nil
		}
	}
	return publicClient.ResolveHandle(ctx, actor)
}
