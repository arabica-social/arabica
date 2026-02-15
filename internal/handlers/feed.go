package handlers

import (
	"context"
	"net/http"

	"arabica/internal/atproto"
	"arabica/internal/feed"
	"arabica/internal/models"
	"arabica/internal/moderation"
	"arabica/internal/web/bff"
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
	// Check if user is authenticated
	didStr, err := atproto.GetAuthenticatedDID(r.Context())
	isAuthenticated := err == nil && didStr != ""

	// Fetch user profile for authenticated users
	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.getUserProfile(r.Context(), didStr)
	}

	// Create layout data
	layoutData := h.buildLayoutData(r, "Home", isAuthenticated, didStr, userProfile)

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

	// Check if user is authenticated
	viewerDID, err := atproto.GetAuthenticatedDID(r.Context())
	isAuthenticated := err == nil

	if h.feedService != nil {
		if isAuthenticated {
			feedItems, _ = h.feedService.GetRecentRecords(r.Context(), feed.FeedLimit)
		} else {
			// Unauthenticated users get a limited feed from the cache
			feedItems, _ = h.feedService.GetCachedPublicFeed(r.Context())
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

	if err := pages.FeedPartialWithModeration(feedItems, isAuthenticated, modCtx).Render(r.Context(), w); err != nil {
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
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	subjectURI := r.FormValue("subject_uri")
	subjectCID := r.FormValue("subject_cid")

	if subjectURI == "" || subjectCID == "" {
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

		// Update firehose index
		if h.feedIndex != nil {
			_ = h.feedIndex.DeleteLike(didStr, subjectURI)
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

		// Update firehose index
		if h.feedIndex != nil {
			_ = h.feedIndex.UpsertLike(didStr, like.RKey, subjectURI)
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
