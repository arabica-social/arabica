package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"arabica/internal/atproto"
	"arabica/internal/database"
	"arabica/internal/database/boltstore"
	"arabica/internal/email"
	"arabica/internal/feed"
	"arabica/internal/firehose"
	"arabica/internal/metrics"
	"arabica/internal/middleware"
	"arabica/internal/models"
	"arabica/internal/moderation"
	"arabica/internal/web/bff"
	"arabica/internal/web/components"

	"github.com/rs/zerolog/log"
)

// Config holds handler configuration options
type Config struct {
	// SecureCookies sets the Secure flag on authentication cookies
	// Should be true in production (HTTPS), false for local development (HTTP)
	SecureCookies bool

	// PublicURL is the public-facing URL for the server (e.g., https://arabica.social)
	// Used for constructing absolute URLs in OpenGraph metadata
	PublicURL string
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
	feedIndex     *firehose.FeedIndex

	// Moderation dependencies (optional)
	moderationService *moderation.Service
	moderationStore   moderation.Store

	// Join request dependencies (optional)
	emailSender   *email.Sender
	joinStore     *boltstore.JoinStore
	pdsAdminURL   string
	pdsAdminToken string

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

// SetFeedIndex configures the handler to use the firehose feed index for like lookups
func (h *Handler) SetFeedIndex(idx *firehose.FeedIndex) {
	h.feedIndex = idx
}

// SetModeration configures the handler with moderation service and store
func (h *Handler) SetModeration(svc *moderation.Service, store moderation.Store) {
	h.moderationService = svc
	h.moderationStore = store
}

// SetJoin configures the handler with email sender and join request store
func (h *Handler) SetJoin(sender *email.Sender, store *boltstore.JoinStore, pdsURL, pdsAdminToken string) {
	h.emailSender = sender
	h.joinStore = store
	h.pdsAdminURL = pdsURL
	h.pdsAdminToken = pdsAdminToken
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

// isJSONRequest checks if the request Content-Type is JSON
func isJSONRequest(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	return strings.Contains(contentType, "application/json")
}

// decodeRequest decodes either JSON or form data into the target interface based on Content-Type.
// The parseForm function is called when the request is form-encoded (not JSON).
// Returns an error if parsing fails.
func decodeRequest(r *http.Request, target interface{}, parseForm func() error) error {
	if isJSONRequest(r) {
		// Parse as JSON
		if err := json.NewDecoder(r.Body).Decode(target); err != nil {
			return err
		}
	} else {
		// Parse as form data using the provided function
		if err := r.ParseForm(); err != nil {
			return err
		}
		if err := parseForm(); err != nil {
			return err
		}
	}
	return nil
}

// writeJSON encodes and writes a JSON response
func writeJSON(w http.ResponseWriter, v interface{}, entityName string) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error().Err(err).Msg("Failed to encode " + entityName + " response")
	}
}

// getUserProfile fetches the profile for an authenticated user.
// Returns nil if unable to fetch profile (non-fatal error).
func (h *Handler) getUserProfile(ctx context.Context, did string) *bff.UserProfile {
	if did == "" {
		return nil
	}

	publicClient := atproto.NewPublicClient()
	profile, err := publicClient.GetProfile(ctx, did)
	if err != nil {
		log.Warn().Err(err).Str("did", did).Msg("Failed to fetch user profile for header")
		return nil
	}

	userProfile := &bff.UserProfile{
		Handle: profile.Handle,
	}
	if profile.DisplayName != nil {
		userProfile.DisplayName = *profile.DisplayName
	}
	if profile.Avatar != nil {
		userProfile.Avatar = *profile.Avatar
	}

	return userProfile
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

// layoutDataFromRequest extracts auth state from the request and builds layout data.
// Returns the layout data, the user's DID (empty if not authenticated), and whether authenticated.
func (h *Handler) layoutDataFromRequest(r *http.Request, title string) (layoutData *components.LayoutData, didStr string, isAuthenticated bool) {
	didStr, err := atproto.GetAuthenticatedDID(r.Context())
	isAuthenticated = err == nil && didStr != ""

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.getUserProfile(r.Context(), didStr)
	}

	layoutData = h.buildLayoutData(r, title, isAuthenticated, didStr, userProfile)
	return
}

// deleteEntity validates the rkey, calls the delete function, and returns 200.
func (h *Handler) deleteEntity(w http.ResponseWriter, r *http.Request, deleteFn func(context.Context, string) error, entityName string) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	if err := deleteFn(r.Context(), rkey); err != nil {
		http.Error(w, "Failed to delete "+entityName, http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to delete " + entityName)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// buildLayoutData creates a LayoutData struct with common fields populated from the request
func (h *Handler) buildLayoutData(r *http.Request, title string, isAuthenticated bool, didStr string, userProfile *bff.UserProfile) *components.LayoutData {
	// Check if user is a moderator
	isModerator := false
	if h.moderationService != nil && didStr != "" {
		isModerator = h.moderationService.IsModerator(didStr)
	}

	// Get unread notification count for authenticated users
	var unreadNotifCount int
	if h.feedIndex != nil && didStr != "" {
		unreadNotifCount = h.feedIndex.GetUnreadCount(didStr)
	}

	return &components.LayoutData{
		Title:                   title,
		IsAuthenticated:         isAuthenticated,
		UserDID:                 didStr,
		UserProfile:             userProfile,
		CSPNonce:                middleware.CSPNonceFromContext(r.Context()),
		IsModerator:             isModerator,
		UnreadNotificationCount: unreadNotifCount,
	}
}

// HandleCommentCreate handles creating a new comment
func (h *Handler) HandleCommentCreate(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	didStr, _ := atproto.GetAuthenticatedDID(r.Context())

	if err := r.ParseForm(); err != nil {
		log.Warn().Err(err).Msg("Failed to parse comment create form")
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	subjectURI := r.FormValue("subject_uri")
	subjectCID := r.FormValue("subject_cid")
	text := strings.TrimSpace(r.FormValue("text"))
	parentURI := r.FormValue("parent_uri")
	parentCID := r.FormValue("parent_cid")

	if subjectURI == "" || subjectCID == "" {
		log.Warn().Str("subject_uri", subjectURI).Str("subject_cid", subjectCID).Msg("Comment create: missing required fields")
		http.Error(w, "subject_uri and subject_cid are required", http.StatusBadRequest)
		return
	}

	if text == "" {
		log.Warn().Str("subject_uri", subjectURI).Msg("Comment create: empty text")
		http.Error(w, "comment text is required", http.StatusBadRequest)
		return
	}

	if len(text) > models.MaxCommentLength {
		log.Warn().Int("length", len(text)).Int("max", models.MaxCommentLength).Msg("Comment create: text too long")
		http.Error(w, "comment text is too long", http.StatusBadRequest)
		return
	}

	// Validate that parent fields are either both present or both absent
	if (parentURI != "" && parentCID == "") || (parentURI == "" && parentCID != "") {
		log.Warn().Str("parent_uri", parentURI).Str("parent_cid", parentCID).Msg("Comment create: incomplete parent reference")
		http.Error(w, "both parent_uri and parent_cid must be provided together", http.StatusBadRequest)
		return
	}

	req := &models.CreateCommentRequest{
		SubjectURI: subjectURI,
		SubjectCID: subjectCID,
		Text:       text,
		ParentURI:  parentURI,
		ParentCID:  parentCID,
	}

	comment, err := store.CreateComment(r.Context(), req)
	if err != nil {
		http.Error(w, "Failed to create comment", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to create comment")
		return
	}

	metrics.CommentsTotal.WithLabelValues("create").Inc()

	// Update firehose index (pass parent URI and comment's CID for threading)
	if h.feedIndex != nil {
		if err := h.feedIndex.UpsertComment(didStr, comment.RKey, subjectURI, parentURI, comment.CID, text, comment.CreatedAt); err != nil {
			log.Warn().Err(err).Str("did", didStr).Str("rkey", comment.RKey).Str("subject_uri", subjectURI).Msg("Failed to upsert comment in feed index")
		}
		// Create notification for the comment/reply
		h.feedIndex.CreateCommentNotification(didStr, subjectURI, parentURI)
	}

	// Return the updated comment section with threaded comments
	comments := h.feedIndex.GetThreadedCommentsForSubject(r.Context(), subjectURI, 100, didStr)

	// Build moderation context
	var modCtx components.CommentModerationContext
	if h.moderationService != nil {
		if h.moderationService.IsModerator(didStr) {
			modCtx.IsModerator = true
			modCtx.CanHideRecord = h.moderationService.HasPermission(didStr, moderation.PermissionHideRecord)
			modCtx.CanBlockUser = h.moderationService.HasPermission(didStr, moderation.PermissionBlacklistUser)
		}
	}

	if err := components.CommentSection(components.CommentSectionProps{
		SubjectURI:      subjectURI,
		SubjectCID:      subjectCID,
		Comments:        comments,
		IsAuthenticated: true,
		CurrentUserDID:  didStr,
		ModCtx:          modCtx,
	}).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render comment section")
	}
}

// HandleCommentDelete handles deleting a comment
func (h *Handler) HandleCommentDelete(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	didStr, _ := atproto.GetAuthenticatedDID(r.Context())

	rkey := r.PathValue("id")
	if rkey == "" {
		http.Error(w, "Comment ID is required", http.StatusBadRequest)
		return
	}

	// Delete the comment from the user's PDS
	if err := store.DeleteCommentByRKey(r.Context(), rkey); err != nil {
		http.Error(w, "Failed to delete comment", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Str("did", didStr).Msg("Failed to delete comment from PDS")
		return
	}

	metrics.CommentsTotal.WithLabelValues("delete").Inc()

	// Update firehose index and remove notifications
	if h.feedIndex != nil {
		// Look up subject URI before deletion for notification cleanup
		subjectURI := h.feedIndex.GetCommentSubjectURI(didStr, rkey)

		if err := h.feedIndex.DeleteComment(didStr, rkey, ""); err != nil {
			log.Warn().Err(err).Str("did", didStr).Str("rkey", rkey).Msg("Failed to delete comment from feed index")
		}

		if subjectURI != "" {
			h.feedIndex.DeleteCommentNotification(didStr, subjectURI, "")
		}
	}

	// Return empty response (the comment element will be removed via hx-swap="outerHTML")
	w.WriteHeader(http.StatusOK)
}

// filterHiddenComments removes comments that have been hidden by moderation.
// Children of hidden comments are kept but shifted up in depth.
func (h *Handler) filterHiddenComments(ctx context.Context, comments []firehose.IndexedComment) []firehose.IndexedComment {
	if h.moderationStore == nil || len(comments) == 0 {
		return comments
	}

	// Build set of hidden comment rkeys for depth adjustment
	hiddenRKeys := make(map[string]bool)
	for _, c := range comments {
		uri := fmt.Sprintf("at://%s/social.arabica.alpha.comment/%s", c.ActorDID, c.RKey)
		if h.moderationStore.IsRecordHidden(ctx, uri) {
			hiddenRKeys[c.RKey] = true
		}
	}

	if len(hiddenRKeys) == 0 {
		return comments
	}

	filtered := make([]firehose.IndexedComment, 0, len(comments))
	for _, c := range comments {
		if hiddenRKeys[c.RKey] {
			continue
		}
		// If this comment's parent was hidden, reduce depth by 1
		if c.ParentRKey != "" && hiddenRKeys[c.ParentRKey] && c.Depth > 0 {
			c.Depth--
		}
		filtered = append(filtered, c)
	}
	return filtered
}

// HandleCommentList returns the comment section for a subject
func (h *Handler) HandleCommentList(w http.ResponseWriter, r *http.Request) {
	subjectURI := r.URL.Query().Get("subject_uri")
	if subjectURI == "" {
		http.Error(w, "subject_uri is required", http.StatusBadRequest)
		return
	}

	// Get authenticated user if any
	didStr, err := atproto.GetAuthenticatedDID(r.Context())
	isAuthenticated := err == nil && didStr != ""

	// Get the subject CID from query params (for the form)
	subjectCID := r.URL.Query().Get("subject_cid")

	// Get threaded comments from firehose index
	var comments []firehose.IndexedComment
	if h.feedIndex != nil {
		comments = h.feedIndex.GetThreadedCommentsForSubject(r.Context(), subjectURI, 100, didStr)
		comments = h.filterHiddenComments(r.Context(), comments)
	}

	// Build moderation context
	var modCtx components.CommentModerationContext
	if h.moderationService != nil && isAuthenticated {
		if h.moderationService.IsModerator(didStr) {
			modCtx.IsModerator = true
			modCtx.CanHideRecord = h.moderationService.HasPermission(didStr, moderation.PermissionHideRecord)
			modCtx.CanBlockUser = h.moderationService.HasPermission(didStr, moderation.PermissionBlacklistUser)
		}
	}

	if err := components.CommentSection(components.CommentSectionProps{
		SubjectURI:      subjectURI,
		SubjectCID:      subjectCID,
		Comments:        comments,
		IsAuthenticated: isAuthenticated,
		CurrentUserDID:  didStr,
		ModCtx:          modCtx,
	}).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render comment section")
	}
}
