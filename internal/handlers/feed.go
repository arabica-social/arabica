package handlers

import (
	"context"
	"net/http"

	"arabica/internal/atproto"
	"arabica/internal/feed"
	"arabica/internal/lexicons"
	"arabica/internal/metrics"
	"arabica/internal/models"
	"arabica/internal/moderation"
	"arabica/internal/web/components"
	"arabica/internal/web/pages"

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

	// Build map of hidden URIs for efficient lookup
	if h.moderationStore != nil {
		for _, item := range items {
			if item.SubjectURI != "" {
				if h.moderationStore.IsRecordHidden(ctx, item.SubjectURI) {
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

// Community feed partial (loaded async via HTMX)
func (h *Handler) HandleFeedPartial(w http.ResponseWriter, r *http.Request) {
	var feedItems []*feed.FeedItem
	var nextCursor string

	// Check if user is authenticated
	viewerDID, err := atproto.GetAuthenticatedDID(r.Context())
	isAuthenticated := err == nil

	// Parse query parameters
	typeFilter := lexicons.ParseRecordType(r.URL.Query().Get("type"))
	sortBy := feed.FeedSort(r.URL.Query().Get("sort"))
	cursor := r.URL.Query().Get("cursor")

	if sortBy != feed.FeedSortPopular {
		sortBy = feed.FeedSortRecent
	}

	if h.feedService != nil {
		if isAuthenticated {
			result, err := h.feedService.GetFeedWithQuery(r.Context(), feed.FeedQuery{
				Limit:      feed.FeedLimit,
				Cursor:     cursor,
				TypeFilter: typeFilter,
				Sort:       sortBy,
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
		for _, item := range feedItems {
			// Check if viewer owns this record
			if item.Author != nil {
				item.IsOwner = item.Author.DID == viewerDID
			}
			// Check if viewer liked this record
			if h.feedIndex != nil && item.SubjectURI != "" {
				item.IsLikedByViewer = h.feedIndex.HasUserLiked(viewerDID, item.SubjectURI)
			}
		}
	}

	// Build moderation context for moderators
	modCtx := h.buildModerationContext(r.Context(), viewerDID, feedItems)

	// Build query state for template
	queryState := pages.FeedQueryState{
		TypeFilter:      string(typeFilter),
		Sort:            string(sortBy),
		NextCursor:      nextCursor,
		IsAuthenticated: isAuthenticated,
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
		http.Error(w, "Failed to check like status", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to check existing like")
		return
	}

	var isLiked bool
	var likeCount int

	if existingLike != nil {
		// Unlike: delete the existing like
		if err := store.DeleteLikeByRKey(r.Context(), existingLike.RKey); err != nil {
			http.Error(w, "Failed to unlike", http.StatusInternalServerError)
			log.Error().Err(err).Msg("Failed to delete like")
			return
		}
		isLiked = false
		metrics.LikesTotal.WithLabelValues("delete").Inc()

		// Update firehose index
		if h.feedIndex != nil {
			if err := h.feedIndex.DeleteLike(didStr, subjectURI); err != nil {
				log.Warn().Err(err).Str("did", didStr).Str("subject_uri", subjectURI).Msg("Failed to delete like from feed index")
			}
			likeCount = h.feedIndex.GetLikeCount(subjectURI)
		}
	} else {
		// Like: create a new like
		req := &models.CreateLikeRequest{
			SubjectURI: subjectURI,
			SubjectCID: subjectCID,
		}
		like, err := store.CreateLike(r.Context(), req)
		if err != nil {
			http.Error(w, "Failed to like", http.StatusInternalServerError)
			log.Error().Err(err).Msg("Failed to create like")
			return
		}
		isLiked = true
		metrics.LikesTotal.WithLabelValues("create").Inc()

		// Update firehose index
		if h.feedIndex != nil {
			if err := h.feedIndex.UpsertLike(didStr, like.RKey, subjectURI); err != nil {
				log.Warn().Err(err).Str("did", didStr).Str("subject_uri", subjectURI).Msg("Failed to upsert like in feed index")
			}
			likeCount = h.feedIndex.GetLikeCount(subjectURI)
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
