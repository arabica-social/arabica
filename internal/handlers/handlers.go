package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/backup"
	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/firehose"
	"tangled.org/arabica.social/arabica/internal/metrics"
	"tangled.org/arabica.social/arabica/internal/moderation"
	moderationsqlite "tangled.org/arabica.social/arabica/internal/moderation/sqlite"
	"tangled.org/arabica.social/arabica/internal/ogcard"
	"tangled.org/arabica.social/arabica/internal/records"
	"tangled.org/arabica.social/arabica/internal/signup"
	"tangled.org/arabica.social/arabica/internal/social"
	"tangled.org/arabica.social/arabica/internal/web/assets"
	"tangled.org/arabica.social/arabica/internal/web/bff"
	"tangled.org/arabica.social/arabica/internal/web/feedviews"
	"tangled.org/arabica.social/arabica/internal/web/spa"
	"tangled.org/pdewey.com/atp"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/bluesky-social/indigo/atproto/syntax"
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

type HomeReadinessChecker func(context.Context, records.Store) (bool, error)

type HomeBehavior struct {
	OGDescription    string
	SiteCardOpts     ogcard.SiteCardOpts
	ReadinessChecker HomeReadinessChecker
}

// Handler contains all HTTP handler methods and their dependencies.
// Dependencies are injected via the constructor for better testability.
type Handler struct {
	oauth         *atp.OAuthApp
	atprotoClient *atproto.Client
	sessionCache  *atproto.SessionCache
	config        Config
	feedService   *feed.Service
	feedRegistry  *feed.Registry
	feedIndex     *firehose.FeedIndex
	witnessCache  atproto.WitnessCache

	// Moderation dependencies (optional)
	moderationService *moderation.Service
	moderationStore   *moderationsqlite.ModerationStore

	// Backup service (optional) — exposes per-source status to admin views.
	backupService *backup.Service

	// Brand carries the per-app display name and tagline. Set via
	// SetBrand at startup; consumed by buildLayoutData so templ
	// components can read brand strings without hardcoding "Arabica".
	brand domain.BrandConfig

	// app carries the per-app config so handlers that need the entity
	// list (admin export, NSID-keyed loops) can read app.NSIDs(). Set
	// via SetApp at startup.
	app *domain.App

	// devMode reflects <APP>_DEV at startup. Gates dev-only PDS providers
	// in the signup catalog and any other developer-facing affordances.
	devMode bool

	homeBehavior HomeBehavior
	assets       assets.Manifest
	feedViews    feedviews.Registry

	// storeOverride supports focused handler tests without constructing an
	// OAuth-backed ATProto client. Production code leaves it nil.
	storeOverride records.Store
}

// SetStoreOverrideForTest injects a request-scoped store for handler tests.
// Authentication context is still required; only the concrete store creation is
// bypassed. Passing nil clears the override.
func (h *Handler) SetStoreOverrideForTest(store records.Store) {
	h.storeOverride = store
}

// SetHomeReadinessChecker wires app-owned first-run readiness logic into the
// shared home handler.
func (h *Handler) SetHomeReadinessChecker(checker HomeReadinessChecker) {
	h.homeBehavior.ReadinessChecker = checker
}

// SetHomeBehavior wires app-owned home-page behavior into the shared home
// handler.
func (h *Handler) SetHomeBehavior(behavior HomeBehavior) {
	h.homeBehavior = behavior
}

// SetAssetManifest wires the server's configured asset hrefs into layout data.
func (h *Handler) SetAssetManifest(manifest assets.Manifest) {
	h.assets = manifest
}

func (h *Handler) SetFeedViews(views feedviews.Registry) {
	h.feedViews = views
}

// SetRecordStoreOverrideForTest injects an app-generic record store for
// handler tests that do not need Arabica's typed store interface.
func (h *Handler) SetRecordStoreOverrideForTest(store records.Store) {
	h.storeOverride = store
}

// SetDevMode toggles dev-mode features. Called once at startup from the
// server bootstrap based on the <APP>_DEV env var.
func (h *Handler) SetDevMode(v bool) {
	h.devMode = v
}

// SetBrand wires the per-app branding into the handler. Called once at
// startup from cmd/arabica or cmd/server after constructing the App.
func (h *Handler) SetBrand(b domain.BrandConfig) {
	h.brand = b
}

// SetApp wires the per-app config into the handler so app-aware code
// paths (admin export, etc.) can read entity lists without depending on
// arabica-specific globals.
func (h *Handler) SetApp(a *domain.App) {
	h.app = a
}

// appName returns the running app's lowercase identifier, falling back
// to "arabica" when SetApp wasn't called (legacy tests, ad-hoc handler
// construction). The empty default matches the layout's stylesheet
// switch which serves arabica's CSS for unknown app names.
func appName(a *domain.App) string {
	if a == nil {
		return ""
	}
	return a.Name
}

// appName returns the running app's lowercase identifier via the package
// helper, for handlers that want a method form.
func (h *Handler) appName() string {
	return appName(h.app)
}

// brandName returns the app's brand display name for OG titles and other
// user-facing strings, falling back to the app name then "Arabica" for
// unconfigured handlers (legacy tests, ad-hoc construction).
func (h *Handler) brandName() string {
	if h.app != nil {
		if h.app.Brand.DisplayName != "" {
			return h.app.Brand.DisplayName
		}
		if h.app.Name != "" {
			return h.app.Name
		}
	}
	return "Arabica"
}

// appClient returns a lowercased app identifier for the X-Client header on
// outbound PDS search requests, derived from the app name. Falls back to
// "arabica" for unconfigured handlers.
func (h *Handler) appClient() string {
	if h.app != nil && h.app.Name != "" {
		return h.app.Name
	}
	return "arabica"
}

// CookieNames returns the auth cookie names for a given app. Apps with
// App.LegacyUnprefixedCookies true (Arabica) keep the legacy unprefixed
// names so prod sessions don't break; every other app gets a per-app
// prefix so multiple apps can run on localhost without clobbering each
// other's cookies (loopback OAuth pins us to 127.0.0.1, so the browser
// shares one cookie jar across ports).
func CookieNames(app *domain.App) (did, sess string) {
	name := ""
	if app != nil {
		name = app.Name
	}
	if app != nil && app.LegacyUnprefixedCookies {
		return "account_did", "session_id"
	}
	if name == "" {
		name = "arabica"
	}
	return name + "_account_did", name + "_session_id"
}

// cookieNames returns this handler's auth cookie names.
func (h *Handler) cookieNames() (did, sess string) {
	return CookieNames(h.app)
}

// appNSIDs returns the running app's NSID list. Returns nil if SetApp
// was never called — admin handlers handle nil gracefully (empty
// export rather than crash) so tests that skip wiring still work.
func (h *Handler) appNSIDs() []string {
	if h.app != nil {
		return h.app.NSIDs()
	}
	return nil
}

// NewHandler creates a new Handler with all required dependencies.
// This constructor pattern ensures the Handler is always fully initialized.
func NewHandler(
	oauth *atp.OAuthApp,
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

// FeedIndex returns the feed index for health checks.
func (h *Handler) FeedIndex() *firehose.FeedIndex {
	return h.feedIndex
}

// SetWitnessCache configures the handler to use the witness cache for cache-first reads.
func (h *Handler) SetWitnessCache(wc atproto.WitnessCache) {
	h.witnessCache = wc
}

// WitnessCache exposes the witness cache for per-app handler packages.
func (h *Handler) WitnessCache() atproto.WitnessCache { return h.witnessCache }

// AtprotoClient exposes the AT Protocol client for per-app handler packages.
func (h *Handler) AtprotoClient() *atproto.Client { return h.atprotoClient }

// SessionCache exposes the session cache for per-app handler packages.
func (h *Handler) SessionCache() *atproto.SessionCache { return h.sessionCache }

// FeedRegistry exposes the feed registry for per-app handler packages.
func (h *Handler) FeedRegistry() *feed.Registry { return h.feedRegistry }

// App exposes the per-app config for handler packages that need to inspect
// app identity (e.g. for branch logic in cross-app endpoints).
func (h *Handler) App() *domain.App { return h.app }

// SetModeration configures the handler with moderation service and SQLite store.
func (h *Handler) SetModeration(svc *moderation.Service, store *moderationsqlite.ModerationStore) {
	h.moderationService = svc
	h.moderationStore = store
}

// SetBackupService wires the backup service so admin handlers can surface
// per-source backup status. Optional — handlers tolerate a nil service.
func (h *Handler) SetBackupService(svc *backup.Service) {
	h.backupService = svc
}

// invalidateFeedCache clears the public feed cache after a mutation.
func (h *Handler) InvalidateFeedCache() {
	if h.feedService != nil {
		h.feedService.InvalidatePublicFeedCache()
	}
}

// loadContentFilter creates a ContentFilter from the moderation store.
// Returns nil if moderation is not configured.
func (h *Handler) LoadContentFilter(ctx context.Context) *moderation.ContentFilter {
	if h.moderationStore == nil {
		return nil
	}
	f, err := moderation.LoadFilter(ctx, h.moderationStore)
	if err != nil {
		log.Warn().Err(err).Msg("failed to load content filter")
		return nil
	}
	return f
}

// ValidateRKey validates and returns an rkey from a path parameter.
// Returns the rkey if valid, or writes an error response and returns empty string if invalid.
func ValidateRKey(w http.ResponseWriter, rkey string) string {
	if rkey == "" {
		http.Error(w, "Record key is required", http.StatusBadRequest)
		return ""
	}
	if !atp.ValidateRKey(rkey) {
		http.Error(w, "Invalid record key format", http.StatusBadRequest)
		return ""
	}
	return rkey
}

// ValidateRKeyJSON validates a record key and writes the stable JSON error
// envelope when validation fails.
func ValidateRKeyJSON(w http.ResponseWriter, rkey string) string {
	if rkey == "" {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Record key is required")
		return ""
	}
	if !atp.ValidateRKey(rkey) {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid record key format")
		return ""
	}
	return rkey
}

// ValidateOptionalRKey validates an optional rkey from form data.
// Returns an error message if invalid, empty string if valid or empty.
func ValidateOptionalRKey(rkey, fieldName string) string {
	if rkey == "" {
		return ""
	}
	if !atp.ValidateRKey(rkey) {
		return fieldName + " has invalid format"
	}
	return ""
}

// IsJSONRequest checks if the request Content-Type is JSON
func IsJSONRequest(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	return strings.Contains(contentType, "application/json")
}

// DecodeRequest decodes either JSON or form data into the target interface based on Content-Type.
// The parseForm function is called when the request is form-encoded (not JSON).
// Returns an error if parsing fails.
func DecodeRequest(r *http.Request, target any, parseForm func() error) error {
	if IsJSONRequest(r) {
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

// ParseOptionalInt parses a form value as *int. Returns nil for empty strings.
func ParseOptionalInt(s string) *int {
	if s == "" {
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &v
}

// JSONErrorResponse is the stable error envelope returned by SPA-facing JSON
// endpoints. Code is intentionally low-cardinality so clients can branch on
// it without parsing the human-readable message.
type JSONErrorResponse struct {
	Error  string            `json:"error"`
	Code   string            `json:"code"`
	Fields map[string]string `json:"fields,omitempty"`
}

// WriteJSONStatus encodes one JSON response with the requested status.
func WriteJSONStatus(w http.ResponseWriter, status int, v any, entityName string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error().Err(err).Msg("Failed to encode " + entityName + " response")
	}
}

// WriteJSONError writes the stable JSON error envelope.
func WriteJSONError(w http.ResponseWriter, status int, code, message string) {
	WriteJSONStatus(w, status, JSONErrorResponse{Error: message, Code: code}, "error")
}

// WriteJSONValidationError writes the standard field-validation response.
func WriteJSONValidationError(w http.ResponseWriter, fields map[string]string) {
	WriteJSONStatus(w, http.StatusBadRequest, JSONErrorResponse{
		Error:  "Please correct the highlighted fields.",
		Code:   "validation_failed",
		Fields: fields,
	}, "validation error")
}

// WriteRequestError returns JSON when the request selected the JSON
// representation, while preserving plain-text errors for legacy form/HTMX
// callers during the migration.
func WriteRequestError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	if IsJSONRequest(r) || WantsJSON(r) {
		WriteJSONError(w, status, code, message)
		return
	}
	http.Error(w, message, status)
}

// WriteJSON encodes and writes a successful JSON response.
func WriteJSON(w http.ResponseWriter, v any, entityName string) {
	WriteJSONStatus(w, http.StatusOK, v, entityName)
}

// getUserProfile fetches the profile for an authenticated user.
// Routes through feedIndex (invalidated by ProfileWatcher on profile updates)
// so the header stays fresh without a separate cache layer.
// Returns nil if unable to fetch profile (non-fatal error).
func (h *Handler) GetUserProfile(ctx context.Context, did string) *bff.UserProfile {
	if did == "" {
		return nil
	}

	var profile *atproto.Profile
	var err error
	if h.feedIndex != nil {
		profile, err = h.feedIndex.GetProfile(ctx, did)
	} else {
		profile, err = atproto.NewPublicClient().GetProfile(ctx, did)
	}
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

// GetRecordStore creates a user-scoped app-generic record store from the request context.
// Returns the store and true if authenticated, or nil and false if not authenticated.
func (h *Handler) GetRecordStore(r *http.Request) (records.Store, bool) {
	// Get authenticated DID from context
	didStr, ok := atpmiddleware.GetDID(r.Context())
	if !ok {
		return nil, false
	}

	// Parse DID string to syntax.DID
	did, err := syntax.ParseDID(didStr)
	if err != nil {
		return nil, false
	}

	// Get session ID from context
	sessionID, ok := atpmiddleware.GetSessionID(r.Context())
	if !ok {
		return nil, false
	}
	if h.storeOverride != nil {
		return h.storeOverride, true
	}

	// Create user-scoped atproto store with injected cache. App-specific
	// social NSIDs are plumbed in so a sister app writes to its own
	// social.<base>.alpha.{like,comment} collections rather than arabica's.
	var likeNSID, commentNSID string
	if h.app != nil {
		likeNSID = h.app.LikeNSID()
		commentNSID = h.app.CommentNSID()
	}
	store := atproto.NewAtprotoStoreForApp(h.atprotoClient, did, sessionID, h.sessionCache, h.witnessCache, likeNSID, commentNSID)
	if h.app != nil && h.app.RecordStore != nil {
		return h.app.RecordStore(store), true
	}
	return store, true
}

type socialStore interface {
	records.Store
	CreateLike(ctx context.Context, req *social.CreateLikeRequest) (*social.Like, error)
	DeleteLikeByRKey(ctx context.Context, rkey string) error
	GetUserLikeForSubject(ctx context.Context, subjectURI string) (*social.Like, error)
	CreateComment(ctx context.Context, req *social.CreateCommentRequest) (*social.Comment, error)
	DeleteCommentByRKey(ctx context.Context, rkey string) error
}

func (h *Handler) getSocialStore(r *http.Request) (socialStore, bool) {
	store, ok := h.GetRecordStore(r)
	if !ok {
		return nil, false
	}
	social, ok := store.(socialStore)
	if !ok {
		return nil, false
	}
	return social, true
}

// HandleStoreError writes the appropriate HTTP error for a store operation failure.
// If the error indicates an expired OAuth session, it returns 401 Unauthorized with
// a user-friendly message. Otherwise it returns 500 with the fallbackMessage.
func HandleStoreError(w http.ResponseWriter, err error, fallbackMessage string) {
	if errors.Is(err, atproto.ErrSessionExpired) {
		http.Error(w, "Your session has expired. Please log in again.", http.StatusUnauthorized)
		return
	}
	http.Error(w, fallbackMessage, http.StatusInternalServerError)
}

// HandleStoreJSONError maps store failures to the stable JSON error envelope
// without exposing internal dependency details.
func HandleStoreJSONError(w http.ResponseWriter, err error, fallbackMessage string) {
	if errors.Is(err, atproto.ErrSessionExpired) {
		WriteJSONError(w, http.StatusUnauthorized, "session_expired", "Your session has expired. Please log in again.")
		return
	}
	WriteJSONError(w, http.StatusInternalServerError, "internal_error", fallbackMessage)
}

// HandleStoreErrorForRequest preserves the existing store-error semantics but
// uses the JSON envelope when the request selected JSON.
func HandleStoreErrorForRequest(w http.ResponseWriter, r *http.Request, err error, fallbackMessage string) {
	if errors.Is(err, atproto.ErrSessionExpired) {
		WriteRequestError(w, r, http.StatusUnauthorized, "session_expired", "Your session has expired. Please log in again.")
		return
	}
	WriteRequestError(w, r, http.StatusInternalServerError, "internal_error", fallbackMessage)
}

// deleteEntity validates the rkey, calls the delete function, removes the record
// from the firehose feed index, and returns 200.
func (h *Handler) DeleteEntity(w http.ResponseWriter, r *http.Request, deleteFn func(context.Context, string) error, entityName string, collection string) {
	rkey := ValidateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	if err := deleteFn(r.Context(), rkey); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to delete " + entityName)
		HandleStoreErrorForRequest(w, r, err, "Failed to delete "+entityName)
		return
	}
	// Remove from firehose feed index
	if h.feedIndex != nil && collection != "" {
		didStr, _ := atpmiddleware.GetDID(r.Context())
		if didStr != "" {
			if err := h.feedIndex.DeleteRecord(r.Context(), didStr, collection, rkey); err != nil {
				log.Warn().Err(err).Str("rkey", rkey).Str("collection", collection).Msg("Failed to delete record from feed index")
			}
		}
	}
	h.InvalidateFeedCache()
	if WantsJSON(r) {
		WriteJSON(w, map[string]bool{"deleted": true}, entityName+" delete")
		return
	}
	w.Header().Set("HX-Trigger", "entityDeleted")
	w.WriteHeader(http.StatusOK)
}

// resolveOwnerHandle returns a human-readable handle for the owner string.
// If the owner is already a handle, it is returned as-is. If it is a DID,
// the feed index profile cache is consulted to resolve it to a handle.
func (h *Handler) ResolveOwnerHandle(ctx context.Context, owner string) string {
	if !strings.HasPrefix(owner, "did:") {
		return owner
	}
	if h.feedIndex != nil {
		if profile, err := h.feedIndex.GetProfile(ctx, owner); err == nil && profile.Handle != "" {
			return profile.Handle
		}
	}
	return owner
}

// publicBaseURL returns the public-facing base URL for constructing absolute URLs.
// It prefers the configured PublicURL, falling back to deriving it from the request.
func (h *Handler) PublicBaseURL(r *http.Request) string {
	if h.config.PublicURL != "" {
		return h.config.PublicURL
	}
	scheme := "https"
	if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") == "" {
		scheme = "http"
	}
	return scheme + "://" + r.Host
}

// ResolveSessionData returns the authenticated user's display state for the
// SPA shell. It is the SPA equivalent of the profile/moderator/unread fields
// that BuildLayoutData populates for templ pages. Returns zero values when
// the DID is empty or the backing stores are unavailable.
//
// This is called once per SPA page load by the shell handler's session
// resolver; it relies on the feed index profile cache so it stays cheap.
func (h *Handler) ResolveSessionData(ctx context.Context, did string) spa.SessionData {
	if did == "" {
		return spa.SessionData{}
	}

	var out spa.SessionData
	if profile := h.GetUserProfile(ctx, did); profile != nil {
		out.Handle = profile.Handle
		out.DisplayName = profile.DisplayName
		out.Avatar = profile.Avatar
	}
	if h.moderationService != nil && h.moderationService.IsModerator(did) {
		out.IsModerator = true
	}
	if h.feedIndex != nil {
		out.UnreadNotificationCount = h.feedIndex.GetUnreadCount(did)
		out.TemperatureUnit = string(h.feedIndex.GetUserPreferences(ctx, did).TemperatureUnit)
	}
	return out
}

// HandleCommentCreate handles creating a new comment. The SPA always sends
// Accept: application/json, so this delegates to the JSON handler.
func (h *Handler) HandleCommentCreate(w http.ResponseWriter, r *http.Request) {
	h.HandleCommentCreateJSON(w, r)
}

// HandleCommentDelete handles deleting a comment
func (h *Handler) HandleCommentDelete(w http.ResponseWriter, r *http.Request) {
	if WantsJSON(r) {
		h.HandleCommentDeleteJSON(w, r)
		return
	}
	// Require authentication
	store, authenticated := h.getSocialStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	didStr, _ := atpmiddleware.GetDID(r.Context())

	rkey := r.PathValue("id")
	if rkey == "" {
		http.Error(w, "Comment ID is required", http.StatusBadRequest)
		return
	}

	// Delete the comment from the user's PDS
	if err := store.DeleteCommentByRKey(r.Context(), rkey); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Str("did", didStr).Msg("Failed to delete comment from PDS")
		HandleStoreError(w, err, "Failed to delete comment")
		return
	}

	metrics.CommentsTotal.WithLabelValues("delete").Inc()

	// Update firehose index and remove notifications
	if h.feedIndex != nil {
		// Look up subject URI before deletion for notification cleanup
		subjectURI := h.feedIndex.GetCommentSubjectURI(didStr, rkey)

		if err := h.feedIndex.DeleteComment(r.Context(), didStr, rkey, ""); err != nil {
			log.Warn().Err(err).Str("did", didStr).Str("rkey", rkey).Msg("Failed to delete comment from feed index")
		}

		if subjectURI != "" {
			h.feedIndex.DeleteCommentNotification(didStr, subjectURI, "")
		}
	}

	// Return empty response (the comment element will be removed via hx-swap="outerHTML")
	w.Header().Set("HX-Trigger", "entityDeleted")
	w.WriteHeader(http.StatusOK)
}

// filterHiddenComments removes comments that have been hidden by moderation.
// Children of hidden comments are kept but shifted up in depth.
func (h *Handler) FilterHiddenComments(ctx context.Context, comments []firehose.IndexedComment) []firehose.IndexedComment {
	if h.moderationStore == nil || len(comments) == 0 {
		return comments
	}

	// Build set of hidden comment rkeys for depth adjustment
	commentNSID := "social.arabica.alpha.comment"
	if h.app != nil {
		commentNSID = h.app.CommentNSID()
	}
	hiddenRKeys := make(map[string]bool)
	for _, c := range comments {
		uri := fmt.Sprintf("at://%s/%s/%s", c.ActorDID, commentNSID, c.RKey)
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

// HandleCommentList returns the comment section for a subject. The SPA
// always sends Accept: application/json, so this delegates to the JSON handler.
func (h *Handler) HandleCommentList(w http.ResponseWriter, r *http.Request) {
	h.HandleCommentListJSON(w, r)
}

// HandleSignupCategories returns the PDS provider catalog as JSON
// (GET /api/signup/categories). Consumed by the SvelteKit /join/create
// route. Dev-only categories are included when dev mode is enabled.
func (h *Handler) HandleSignupCategories(w http.ResponseWriter, r *http.Request) {
	categories := signup.Categories(h.devMode)
	WriteJSON(w, map[string]any{"categories": categories}, "signup categories")
}

// HandleCreateAccountSubmit initiates the OAuth prompt=create flow (POST /join/create).
func (h *Handler) HandleCreateAccountSubmit(w http.ResponseWriter, r *http.Request) {
	if h.oauth == nil {
		http.Error(w, "OAuth not configured", http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	pdsURL := r.FormValue("pds_url")
	if pdsURL == "" {
		http.Redirect(w, r, "/join/create?error=Please+select+a+server", http.StatusSeeOther)
		return
	}

	if !signup.IsAllowedPDSURL(pdsURL, h.devMode) {
		log.Warn().Str("pds_url", pdsURL).Msg("Signup attempt with unlisted PDS URL")
		http.Redirect(w, r, "/join/create?error=Invalid+server+selection", http.StatusSeeOther)
		return
	}

	// Initiate OAuth flow with prompt=create
	authURL, err := h.oauth.StartSignup(r.Context(), pdsURL)
	if err != nil {
		log.Error().Err(err).Str("pds_url", pdsURL).Msg("Failed to initiate signup flow")
		http.Redirect(w, r, "/join/create?error=Failed+to+connect+to+server.+Please+try+again.", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, authURL, http.StatusFound)
}
