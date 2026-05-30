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
	"tangled.org/arabica.social/arabica/internal/middleware"
	"tangled.org/arabica.social/arabica/internal/moderation"
	moderationsqlite "tangled.org/arabica.social/arabica/internal/moderation/sqlite"
	"tangled.org/arabica.social/arabica/internal/ogcard"
	"tangled.org/arabica.social/arabica/internal/records"
	"tangled.org/arabica.social/arabica/internal/signup"
	"tangled.org/arabica.social/arabica/internal/social"
	"tangled.org/arabica.social/arabica/internal/web/assets"
	"tangled.org/arabica.social/arabica/internal/web/bff"
	"tangled.org/arabica.social/arabica/internal/web/components"
	"tangled.org/arabica.social/arabica/internal/web/feedviews"
	"tangled.org/arabica.social/arabica/internal/web/pages"
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

type StaticPageRenderer func(context.Context, http.ResponseWriter, *components.LayoutData) error

type StaticPageRenderers struct {
	About StaticPageRenderer
	Terms StaticPageRenderer
}

type HomeReadinessChecker func(context.Context, records.Store) (bool, error)

type HomeBehavior struct {
	OGDescription    string
	SiteCardOpts     ogcard.SiteCardOpts
	ReadinessChecker HomeReadinessChecker
}

type FeedPresentation struct {
	EmptyState pages.FeedEmptyState
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

	staticPages      StaticPageRenderers
	homeBehavior     HomeBehavior
	feedPresentation FeedPresentation
	assets           assets.Manifest
	feedViews        feedviews.Registry

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

// SetStaticPageRenderers wires app-owned static page templates into the shared
// page handlers. Nil renderers fall back to the default Arabica pages.
func (h *Handler) SetStaticPageRenderers(renderers StaticPageRenderers) {
	h.staticPages = renderers
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

func (h *Handler) SetFeedPresentation(presentation FeedPresentation) {
	h.feedPresentation = presentation
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
// startup from cmd/{arabica,oolong}/main.go after constructing the App.
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

// CookieNames returns the auth cookie names for a given app. Arabica
// keeps the legacy unprefixed names so prod sessions don't break;
// every other app gets a per-app prefix so multiple apps can run on
// localhost without clobbering each other's cookies (loopback OAuth
// pins us to 127.0.0.1, so the browser shares one cookie jar across
// ports).
func CookieNames(app string) (did, sess string) {
	if app == "" || app == "arabica" {
		return "account_did", "session_id"
	}
	return app + "_account_did", app + "_session_id"
}

// cookieNames returns this handler's auth cookie names.
func (h *Handler) cookieNames() (did, sess string) {
	return CookieNames(appName(h.app))
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

// WriteJSON encodes and writes a JSON response
func WriteJSON(w http.ResponseWriter, v any, entityName string) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error().Err(err).Msg("Failed to encode " + entityName + " response")
	}
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
	// social NSIDs are plumbed in so oolong stores write to
	// social.oolong.alpha.{like,comment} rather than arabica's collections.
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

// layoutDataFromRequest extracts auth state from the request and builds layout data.
// Returns the layout data, the user's DID (empty if not authenticated), and whether authenticated.
func (h *Handler) LayoutDataFromRequest(r *http.Request, title string) (layoutData *components.LayoutData, didStr string, isAuthenticated bool) {
	didStr, isAuthenticated = atpmiddleware.GetDID(r.Context())

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.GetUserProfile(r.Context(), didStr)
	}

	layoutData = h.BuildLayoutData(r, title, isAuthenticated, didStr, userProfile)
	return
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

// deleteEntity validates the rkey, calls the delete function, removes the record
// from the firehose feed index, and returns 200.
func (h *Handler) DeleteEntity(w http.ResponseWriter, r *http.Request, deleteFn func(context.Context, string) error, entityName string, collection string) {
	rkey := ValidateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	if err := deleteFn(r.Context(), rkey); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to delete " + entityName)
		HandleStoreError(w, err, "Failed to delete "+entityName)
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

// PopulateOGFields sets the standard OG metadata fields for an entity page.
// The title follows the pattern "{type} from {owner} on arabica.social".
// The subtitle (OG description) shows record-specific detail like the bean name.
func PopulateOGFields(layoutData *components.LayoutData, subtitle, recordType, owner, baseURL, shareURL string) {
	layoutData.OGType = "article"

	if owner != "" {
		layoutData.OGTitle = fmt.Sprintf("%s from %s on arabica.social", recordType, owner)
	} else {
		layoutData.OGTitle = fmt.Sprintf("%s on arabica.social", recordType)
	}

	layoutData.OGDescription = subtitle

	if baseURL != "" && shareURL != "" {
		layoutData.OGUrl = baseURL + shareURL
		if idx := strings.Index(shareURL, "?"); idx >= 0 {
			layoutData.OGImage = baseURL + shareURL[:idx] + "/og-image" + shareURL[idx:]
		} else {
			layoutData.OGImage = baseURL + shareURL + "/og-image"
		}
	}
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

// buildLayoutData creates a LayoutData struct with common fields populated from the request
func (h *Handler) BuildLayoutData(r *http.Request, title string, isAuthenticated bool, didStr string, userProfile *bff.UserProfile) *components.LayoutData {
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
		BrandName:               h.brand.DisplayName,
		BrandTagline:            h.brand.Tagline,
		AppName:                 appName(h.app),
		Assets:                  h.assets,
	}
}

// HandleCommentCreate handles creating a new comment
func (h *Handler) HandleCommentCreate(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getSocialStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	didStr, _ := atpmiddleware.GetDID(r.Context())

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

	if len(text) > social.MaxCommentLength {
		log.Warn().Int("length", len(text)).Int("max", social.MaxCommentLength).Msg("Comment create: text too long")
		http.Error(w, "comment text is too long", http.StatusBadRequest)
		return
	}

	// Validate that parent fields are either both present or both absent
	if (parentURI != "" && parentCID == "") || (parentURI == "" && parentCID != "") {
		log.Warn().Str("parent_uri", parentURI).Str("parent_cid", parentCID).Msg("Comment create: incomplete parent reference")
		http.Error(w, "both parent_uri and parent_cid must be provided together", http.StatusBadRequest)
		return
	}

	req := &social.CreateCommentRequest{
		SubjectURI: subjectURI,
		SubjectCID: subjectCID,
		Text:       text,
		ParentURI:  parentURI,
		ParentCID:  parentCID,
	}

	comment, err := store.CreateComment(r.Context(), req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create comment")
		HandleStoreError(w, err, "Failed to create comment")
		return
	}

	metrics.CommentsTotal.WithLabelValues("create").Inc()

	// Update firehose index (pass parent URI and comment's CID for threading)
	if h.feedIndex != nil {
		if err := h.feedIndex.UpsertComment(r.Context(), didStr, comment.RKey, subjectURI, parentURI, comment.CID, text, comment.CreatedAt); err != nil {
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

// HandleCommentList returns the comment section for a subject
func (h *Handler) HandleCommentList(w http.ResponseWriter, r *http.Request) {
	subjectURI := r.URL.Query().Get("subject_uri")
	if subjectURI == "" {
		http.Error(w, "subject_uri is required", http.StatusBadRequest)
		return
	}

	// Get authenticated user if any
	didStr, isAuthenticated := atpmiddleware.GetDID(r.Context())

	// Get the subject CID from query params (for the form)
	subjectCID := r.URL.Query().Get("subject_cid")

	// Get threaded comments from firehose index
	var comments []firehose.IndexedComment
	if h.feedIndex != nil {
		comments = h.feedIndex.GetThreadedCommentsForSubject(r.Context(), subjectURI, 100, didStr)
		comments = h.FilterHiddenComments(r.Context(), comments)
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

// HandleCreateAccount renders the account creation page (GET /join/create).
// PDS server options come from the internal/signup catalog.
func (h *Handler) HandleCreateAccount(w http.ResponseWriter, r *http.Request) {
	layoutData, _, _ := h.LayoutDataFromRequest(r, "Create Account")

	props := pages.CreateAccountProps{
		Error:      r.URL.Query().Get("error"),
		Categories: signup.Categories(h.devMode),
	}

	if err := pages.CreateAccount(layoutData, props).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render create account page")
	}
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
