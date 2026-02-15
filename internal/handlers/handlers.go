package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"arabica/internal/atproto"
	"arabica/internal/database"
	"arabica/internal/database/boltstore"
	"arabica/internal/email"
	"arabica/internal/feed"
	"arabica/internal/firehose"
	"arabica/internal/middleware"
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
	moderationStore   *boltstore.ModerationStore

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
func (h *Handler) SetModeration(svc *moderation.Service, store *boltstore.ModerationStore) {
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

// buildLayoutData creates a LayoutData struct with common fields populated from the request
func (h *Handler) buildLayoutData(r *http.Request, title string, isAuthenticated bool, didStr string, userProfile *bff.UserProfile) *components.LayoutData {
	// Check if user is a moderator
	isModerator := false
	if h.moderationService != nil && didStr != "" {
		isModerator = h.moderationService.IsModerator(didStr)
	}

	return &components.LayoutData{
		Title:           title,
		IsAuthenticated: isAuthenticated,
		UserDID:         didStr,
		UserProfile:     userProfile,
		CSPNonce:        middleware.CSPNonceFromContext(r.Context()),
		IsModerator:     isModerator,
	}
}
