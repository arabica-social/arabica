package handlers

import (
	"context"
	"net/http"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/lexicons"
	"tangled.org/arabica.social/arabica/internal/metrics"
	"tangled.org/arabica.social/arabica/internal/models"
	"tangled.org/arabica.social/arabica/internal/moderation"
	"tangled.org/arabica.social/arabica/internal/ogcard"
	"tangled.org/arabica.social/arabica/internal/web/components"
	"tangled.org/arabica.social/arabica/internal/web/pages"

	"github.com/rs/zerolog/log"
)

// buildModerationContext creates moderation context for feed rendering
// Returns empty context if moderation is not configured or user is not a moderator
func (h *Handler) buildModerationContext(ctx context.Context, viewerDID string, items []*feed.FeedItem) pages.FeedModerationContext {
	modCtx := pages.FeedModerationContext{
		HiddenURIs: make(map[string]bool),
	}

	// Check if moderation is configured and user is a moderator
	if h.moderationService == nil || viewerDID == "" {
		return modCtx
	}

	if !h.moderationService.IsModerator(viewerDID) {
		return modCtx
	}

	modCtx.IsModerator = true
	modCtx.CanHideRecord = h.moderationService.HasPermission(viewerDID, moderation.PermissionHideRecord)
	modCtx.CanBlockUser = h.moderationService.HasPermission(viewerDID, moderation.PermissionBlacklistUser)

	// Load all hidden URIs in one query and intersect with feed items
	if h.moderationStore != nil {
		if hiddenURIs, err := h.moderationStore.ListHiddenURIs(ctx); err == nil {
			hiddenSet := make(map[string]bool, len(hiddenURIs))
			for _, uri := range hiddenURIs {
				hiddenSet[uri] = true
			}
			for _, item := range items {
				if item.SubjectURI != "" && hiddenSet[item.SubjectURI] {
					modCtx.HiddenURIs[item.SubjectURI] = true
				}
			}
		}
	}

	return modCtx
}

// Home page
func (h *Handler) HandleHome(w http.ResponseWriter, r *http.Request) {
	layoutData, didStr, isAuthenticated := h.layoutDataFromRequest(r, "Home")

	// Set OG metadata for the home page
	layoutData.OGTitle = "Arabica"
	layoutData.OGDescription = "Coffee journaling for the open social web. Track, share, and own your brews."
	baseURL := h.publicBaseURL(r)
	if baseURL != "" {
		layoutData.OGImage = baseURL + "/og-image"
		layoutData.OGUrl = baseURL + "/"
	}

	// Create home props
	homeProps := pages.HomeProps{
		IsAuthenticated: isAuthenticated,
		UserDID:         didStr,
	}

	// Render using templ component
	if err := pages.Home(layoutData, homeProps).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render home page")
	}
}

// HandleSiteOGImage generates a 1200x630 PNG preview card for the site.
func (h *Handler) HandleSiteOGImage(w http.ResponseWriter, r *http.Request) {
	card, err := ogcard.DrawSiteCard()
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate site OG image")
		http.Error(w, "Failed to generate image", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	if err := card.EncodePNG(w); err != nil {
		log.Error().Err(err).Msg("Failed to encode site OG image")
	}
}

// Community feed partial (loaded async via HTMX)
func (h *Handler) HandleFeedPartial(w http.ResponseWriter, r *http.Request) {
	var feedItems []*feed.FeedItem
	var nextCursor string

	// Check if user is authenticated
	viewerDID, err := atproto.GetAuthenticatedDID(r.Context())
	isAuthenticated := err == nil

	// Parse query parameters
	typeParam := r.URL.Query().Get("type")
	typeFilter := lexicons.ParseRecordType(typeParam)
	var typeFilters []lexicons.RecordType
	if typeParam == "equipment" {
		typeFilters = []lexicons.RecordType{lexicons.RecordTypeGrinder, lexicons.RecordTypeBrewer}
		typeFilter = "" // Clear single filter when using multi
	}
	sortBy := feed.FeedSort(r.URL.Query().Get("sort"))
	cursor := r.URL.Query().Get("cursor")

	if sortBy != feed.FeedSortPopular {
		sortBy = feed.FeedSortRecent
	}

	if h.feedService != nil {
		if isAuthenticated {
			result, err := h.feedService.GetFeedWithQuery(r.Context(), feed.FeedQuery{
				Limit:       feed.FeedLimit,
				Cursor:      cursor,
				TypeFilter:  typeFilter,
				TypeFilters: typeFilters,
				Sort:        sortBy,
			})
			if err != nil {
				log.Error().Err(err).Str("sort", string(sortBy)).Str("type", string(typeFilter)).Msg("Failed to query feed")
			}
			if result != nil {
				feedItems = result.Items
				nextCursor = result.NextCursor
			}
		} else {
			// Unauthenticated users get a limited feed from the cache (no filtering)
			var err error
			feedItems, err = h.feedService.GetCachedPublicFeed(r.Context())
			if err != nil {
				log.Error().Err(err).Msg("Failed to get cached public feed")
			}
		}
	}

	// Populate IsLikedByViewer and IsOwner for each feed item if user is authenticated
	if isAuthenticated {
		// Batch fetch liked status for all feed items
		var likedByViewer map[string]bool
		if h.feedIndex != nil {
			uris := make([]string, 0, len(feedItems))
			for _, item := range feedItems {
				if item.SubjectURI != "" {
					uris = append(uris, item.SubjectURI)
				}
			}
			likedByViewer = h.feedIndex.HasUserLikedBatch(r.Context(), viewerDID, uris)
		}
		for _, item := range feedItems {
			if item.Author != nil {
				item.IsOwner = item.Author.DID == viewerDID
			}
			if likedByViewer != nil {
				item.IsLikedByViewer = likedByViewer[item.SubjectURI]
			}
		}
	}

	// Build moderation context for moderators
	modCtx := h.buildModerationContext(r.Context(), viewerDID, feedItems)

	// Build query state for template
	typeFilterStr := string(typeFilter)
	if len(typeFilters) > 0 {
		typeFilterStr = "equipment"
	}
	// Determine column count from query param or cookie
	cols := 2 // default to 2-column masonry
	if c := r.URL.Query().Get("cols"); c == "1" {
		cols = 1
	} else if c == "" {
		// No query param — check cookie (set by client JS on initial page load)
		if ck, err := r.Cookie("feed_cols"); err == nil && ck.Value == "1" {
			cols = 1
		}
	}

	queryState := pages.FeedQueryState{
		TypeFilter:      typeFilterStr,
		Sort:            string(sortBy),
		NextCursor:      nextCursor,
		IsAuthenticated: isAuthenticated,
		Cols:            cols,
	}

	// If this is a "load more" request (has cursor), render just the additional items
	if cursor != "" {
		if err := pages.FeedMoreItems(feedItems, isAuthenticated, modCtx, queryState).Render(r.Context(), w); err != nil {
			http.Error(w, "Failed to render feed", http.StatusInternalServerError)
			log.Error().Err(err).Msg("Failed to render feed partial")
		}
		return
	}

	if err := pages.FeedPartialWithModeration(feedItems, isAuthenticated, modCtx, queryState).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render feed", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render feed partial")
	}
}

// HandleLikeToggle handles creating or deleting a like on a record
func (h *Handler) HandleLikeToggle(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	didStr, _ := atproto.GetAuthenticatedDID(r.Context())

	if err := r.ParseForm(); err != nil {
		log.Warn().Err(err).Msg("Failed to parse like toggle form")
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	subjectURI := r.FormValue("subject_uri")
	subjectCID := r.FormValue("subject_cid")

	if subjectURI == "" || subjectCID == "" {
		log.Warn().Str("subject_uri", subjectURI).Str("subject_cid", subjectCID).Msg("Like toggle: missing required fields")
		http.Error(w, "subject_uri and subject_cid are required", http.StatusBadRequest)
		return
	}

	// Check if user already liked this record
	existingLike, err := store.GetUserLikeForSubject(r.Context(), subjectURI)
	if err != nil {
		log.Error().Err(err).Msg("Failed to check existing like")
		handleStoreError(w, err, "Failed to check like status")
		return
	}

	var isLiked bool
	var likeCount int

	if existingLike != nil {
		// Unlike: delete the existing like
		if err := store.DeleteLikeByRKey(r.Context(), existingLike.RKey); err != nil {
			log.Error().Err(err).Msg("Failed to delete like")
			handleStoreError(w, err, "Failed to unlike")
			return
		}
		isLiked = false
		metrics.LikesTotal.WithLabelValues("delete").Inc()

		// Update firehose index
		if h.feedIndex != nil {
			if err := h.feedIndex.DeleteLike(r.Context(), didStr, subjectURI); err != nil {
				log.Warn().Err(err).Str("did", didStr).Str("subject_uri", subjectURI).Msg("Failed to delete like from feed index")
			}
			h.feedIndex.DeleteLikeNotification(didStr, subjectURI)
			likeCount = h.feedIndex.GetLikeCount(r.Context(), subjectURI)
		}
	} else {
		// Like: create a new like
		req := &models.CreateLikeRequest{
			SubjectURI: subjectURI,
			SubjectCID: subjectCID,
		}
		like, err := store.CreateLike(r.Context(), req)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create like")
			handleStoreError(w, err, "Failed to like")
			return
		}
		isLiked = true
		metrics.LikesTotal.WithLabelValues("create").Inc()

		// Update firehose index
		if h.feedIndex != nil {
			if err := h.feedIndex.UpsertLike(r.Context(), didStr, like.RKey, subjectURI); err != nil {
				log.Warn().Err(err).Str("did", didStr).Str("subject_uri", subjectURI).Msg("Failed to upsert like in feed index")
			}
			likeCount = h.feedIndex.GetLikeCount(r.Context(), subjectURI)
		}
	}

	// Return the updated like button component
	if err := components.LikeButton(components.LikeButtonProps{
		SubjectURI:      subjectURI,
		SubjectCID:      subjectCID,
		IsLiked:         isLiked,
		LikeCount:       likeCount,
		IsAuthenticated: true,
	}).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render like button")
	}
}
