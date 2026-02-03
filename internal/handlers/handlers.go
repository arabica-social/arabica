package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"arabica/internal/atproto"
	"arabica/internal/database"
	"arabica/internal/feed"
	"arabica/internal/firehose"
	"arabica/internal/middleware"
	"arabica/internal/models"
	"arabica/internal/web/bff"
	"arabica/internal/web/components"
	"arabica/internal/web/pages"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// Config holds handler configuration options
type Config struct {
	// SecureCookies sets the Secure flag on authentication cookies
	// Should be true in production (HTTPS), false for local development (HTTP)
	SecureCookies bool
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
	return &components.LayoutData{
		Title:           title,
		IsAuthenticated: isAuthenticated,
		UserDID:         didStr,
		UserProfile:     userProfile,
		CSPNonce:        middleware.CSPNonceFromContext(r.Context()),
	}
}

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

	if err := pages.FeedPartial(feedItems, isAuthenticated).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render feed", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render feed partial")
	}
}

// Brew list partial (loaded async via HTMX)
func (h *Handler) HandleBrewListPartial(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	brews, err := store.ListBrews(r.Context(), 1) // User ID is not used with atproto
	if err != nil {
		http.Error(w, "Failed to fetch brews", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to fetch brews")
		return
	}

	if err := components.BrewListTablePartial(components.BrewListTableProps{
		Brews:        brews,
		IsOwnProfile: true,
	}).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render content", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render brew list partial")
	}
}

// Manage page partial (loaded async via HTMX)
func (h *Handler) HandleManagePartial(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	ctx := r.Context()

	// Fetch all collections in parallel using errgroup for proper error handling
	// and automatic context cancellation on first error
	g, ctx := errgroup.WithContext(ctx)

	var beans []*models.Bean
	var roasters []*models.Roaster
	var grinders []*models.Grinder
	var brewers []*models.Brewer

	g.Go(func() error {
		var err error
		beans, err = store.ListBeans(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		roasters, err = store.ListRoasters(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		grinders, err = store.ListGrinders(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		brewers, err = store.ListBrewers(ctx)
		return err
	})

	if err := g.Wait(); err != nil {
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to fetch manage page data")
		return
	}

	// Link beans to their roasters
	atproto.LinkBeansToRoasters(beans, roasters)

	// Render manage partial
	if err := components.ManagePartial(components.ManagePartialProps{
		Beans:    beans,
		Roasters: roasters,
		Grinders: grinders,
		Brewers:  brewers,
	}).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render content", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render manage partial")
	}
}

// List all brews
func (h *Handler) HandleBrewList(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	_, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	didStr, _ := atproto.GetAuthenticatedDID(r.Context())
	userProfile := h.getUserProfile(r.Context(), didStr)

	// Create layout data
	layoutData := h.buildLayoutData(r, "Your Brews", authenticated, didStr, userProfile)

	// Create brew list props
	brewListProps := pages.BrewListProps{}

	// Render using templ component
	if err := pages.BrewList(layoutData, brewListProps).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render brew list page")
	}
}

// Show new brew form
func (h *Handler) HandleBrewNew(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	_, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	didStr, _ := atproto.GetAuthenticatedDID(r.Context())
	userProfile := h.getUserProfile(r.Context(), didStr)

	// Don't fetch data from PDS - client will populate dropdowns from cache
	// This makes the page load much faster
	layoutData := h.buildLayoutData(r, "New Brew", authenticated, didStr, userProfile)

	brewFormProps := pages.BrewFormProps{
		Brew: nil,
	}

	if err := pages.BrewFormPage(layoutData, brewFormProps).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render brew form")
	}
}

// Show brew view page
func (h *Handler) HandleBrewView(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Check if owner (DID or handle) is specified in query params
	owner := r.URL.Query().Get("owner")

	// Check authentication
	didStr, err := atproto.GetAuthenticatedDID(r.Context())
	isAuthenticated := err == nil && didStr != ""

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.getUserProfile(r.Context(), didStr)
	}

	var brew *models.Brew
	var brewOwnerDID string
	var isOwner bool
	var subjectURI, subjectCID string

	if owner != "" {
		// Viewing someone else's brew - use public client
		publicClient := atproto.NewPublicClient()

		// Resolve owner to DID if it's a handle
		if strings.HasPrefix(owner, "did:") {
			brewOwnerDID = owner
		} else {
			resolved, err := publicClient.ResolveHandle(r.Context(), owner)
			if err != nil {
				log.Warn().Err(err).Str("handle", owner).Msg("Failed to resolve handle for brew view")
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}
			brewOwnerDID = resolved
		}

		// Fetch the brew record from the owner's PDS
		record, err := publicClient.GetRecord(r.Context(), brewOwnerDID, atproto.NSIDBrew, rkey)
		if err != nil {
			log.Error().Err(err).Str("did", brewOwnerDID).Str("rkey", rkey).Msg("Failed to get brew record")
			http.Error(w, "Brew not found", http.StatusNotFound)
			return
		}

		// Store URI and CID for like button
		subjectURI = record.URI
		subjectCID = record.CID

		// Convert record to brew
		brew, err = atproto.RecordToBrew(record.Value, record.URI)
		if err != nil {
			log.Error().Err(err).Msg("Failed to convert brew record")
			http.Error(w, "Failed to load brew", http.StatusInternalServerError)
			return
		}

		// Resolve references (bean, grinder, brewer)
		if err := h.resolveBrewReferences(r.Context(), brew, brewOwnerDID, record.Value); err != nil {
			log.Warn().Err(err).Msg("Failed to resolve some brew references")
			// Don't fail the request, just log the warning
		}

		// Check if viewing user is the owner
		isOwner = isAuthenticated && didStr == brewOwnerDID
	} else {
		// Viewing own brew - require authentication
		store, authenticated := h.getAtprotoStore(r)
		if !authenticated {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// Use type assertion to access GetBrewRecordByRKey
		atprotoStore, ok := store.(*atproto.AtprotoStore)
		if !ok {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			log.Error().Msg("Failed to cast store to AtprotoStore")
			return
		}

		brewRecord, err := atprotoStore.GetBrewRecordByRKey(r.Context(), rkey)
		if err != nil {
			http.Error(w, "Brew not found", http.StatusNotFound)
			log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get brew for view")
			return
		}

		brew = brewRecord.Brew
		subjectURI = brewRecord.URI
		subjectCID = brewRecord.CID
		isOwner = true
	}

	// Create layout data
	layoutData := h.buildLayoutData(r, "Brew Details", isAuthenticated, didStr, userProfile)

	// Get like data
	var isLiked bool
	var likeCount int
	if h.feedIndex != nil && subjectURI != "" {
		likeCount = h.feedIndex.GetLikeCount(subjectURI)
		if isAuthenticated {
			isLiked = h.feedIndex.HasUserLiked(didStr, subjectURI)
		}
	}

	// Construct share URL
	var shareURL string
	if owner != "" {
		shareURL = fmt.Sprintf("/brews/%s?owner=%s", rkey, owner)
	} else if userProfile != nil && userProfile.Handle != "" {
		shareURL = fmt.Sprintf("/brews/%s?owner=%s", rkey, userProfile.Handle)
	}

	// Create brew view props
	brewViewProps := pages.BrewViewProps{
		Brew:            brew,
		IsOwnProfile:    isOwner,
		IsAuthenticated: isAuthenticated,
		SubjectURI:      subjectURI,
		SubjectCID:      subjectCID,
		IsLiked:         isLiked,
		LikeCount:       likeCount,
		ShareURL:        shareURL,
	}

	// Render using templ component
	if err := pages.BrewView(layoutData, brewViewProps).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render brew view")
	}
}

// resolveBrewReferences resolves bean, grinder, and brewer references for a brew
func (h *Handler) resolveBrewReferences(ctx context.Context, brew *models.Brew, ownerDID string, record map[string]interface{}) error {
	publicClient := atproto.NewPublicClient()

	// Resolve bean reference
	if beanRef, ok := record["beanRef"].(string); ok && beanRef != "" {
		beanRecord, err := publicClient.GetRecord(ctx, ownerDID, atproto.NSIDBean, atproto.ExtractRKeyFromURI(beanRef))
		if err == nil {
			if bean, err := atproto.RecordToBean(beanRecord.Value, beanRecord.URI); err == nil {
				brew.Bean = bean

				// Resolve roaster reference for the bean
				if roasterRef, ok := beanRecord.Value["roasterRef"].(string); ok && roasterRef != "" {
					roasterRecord, err := publicClient.GetRecord(ctx, ownerDID, atproto.NSIDRoaster, atproto.ExtractRKeyFromURI(roasterRef))
					if err == nil {
						if roaster, err := atproto.RecordToRoaster(roasterRecord.Value, roasterRecord.URI); err == nil {
							brew.Bean.Roaster = roaster
						}
					}
				}
			}
		}
	}

	// Resolve grinder reference
	if grinderRef, ok := record["grinderRef"].(string); ok && grinderRef != "" {
		grinderRecord, err := publicClient.GetRecord(ctx, ownerDID, atproto.NSIDGrinder, atproto.ExtractRKeyFromURI(grinderRef))
		if err == nil {
			if grinder, err := atproto.RecordToGrinder(grinderRecord.Value, grinderRecord.URI); err == nil {
				brew.GrinderObj = grinder
			}
		}
	}

	// Resolve brewer reference
	if brewerRef, ok := record["brewerRef"].(string); ok && brewerRef != "" {
		brewerRecord, err := publicClient.GetRecord(ctx, ownerDID, atproto.NSIDBrewer, atproto.ExtractRKeyFromURI(brewerRef))
		if err == nil {
			if brewer, err := atproto.RecordToBrewer(brewerRecord.Value, brewerRecord.URI); err == nil {
				brew.BrewerObj = brewer
			}
		}
	}

	return nil
}

// Show edit brew form
func (h *Handler) HandleBrewEdit(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	didStr, _ := atproto.GetAuthenticatedDID(r.Context())
	userProfile := h.getUserProfile(r.Context(), didStr)

	brew, err := store.GetBrewByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Brew not found", http.StatusNotFound)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get brew for edit")
		return
	}

	// Don't fetch dropdown data from PDS - client will populate from cache
	// This makes the page load much faster
	layoutData := h.buildLayoutData(r, "Edit Brew", authenticated, didStr, userProfile)

	brewFormProps := pages.BrewFormProps{
		Brew:      brew,
		PoursJSON: bff.PoursToJSON(brew.Pours),
	}

	if err := pages.BrewFormPage(layoutData, brewFormProps).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render brew edit form")
	}
}

// maxPours is the maximum number of pours allowed in a single brew
const maxPours = 100

// parsePours extracts pour data from form values with bounds checking
func parsePours(r *http.Request) []models.CreatePourData {
	var pours []models.CreatePourData

	for i := 0; i < maxPours; i++ {
		waterKey := "pour_water_" + strconv.Itoa(i)
		timeKey := "pour_time_" + strconv.Itoa(i)

		waterStr := r.FormValue(waterKey)
		timeStr := r.FormValue(timeKey)

		if waterStr == "" && timeStr == "" {
			break
		}

		water, _ := strconv.Atoi(waterStr)
		pourTime, _ := strconv.Atoi(timeStr)

		if water > 0 && pourTime >= 0 {
			pours = append(pours, models.CreatePourData{
				WaterAmount: water,
				TimeSeconds: pourTime,
			})
		}
	}

	return pours
}

// ValidationError represents a validation error with field name and message
type ValidationError struct {
	Field   string
	Message string
}

// validateBrewRequest validates brew form input and returns any validation errors
func validateBrewRequest(r *http.Request) (temperature float64, waterAmount, coffeeAmount, timeSeconds, rating int, pours []models.CreatePourData, errs []ValidationError) {
	// Parse and validate temperature
	if tempStr := r.FormValue("temperature"); tempStr != "" {
		var err error
		temperature, err = strconv.ParseFloat(tempStr, 64)
		if err != nil {
			errs = append(errs, ValidationError{Field: "temperature", Message: "invalid temperature format"})
		} else if temperature < 0 || temperature > 212 {
			errs = append(errs, ValidationError{Field: "temperature", Message: "temperature must be between 0 and 212"})
		}
	}

	// Parse and validate water amount
	if waterStr := r.FormValue("water_amount"); waterStr != "" {
		var err error
		waterAmount, err = strconv.Atoi(waterStr)
		if err != nil {
			errs = append(errs, ValidationError{Field: "water_amount", Message: "invalid water amount"})
		} else if waterAmount < 0 || waterAmount > 10000 {
			errs = append(errs, ValidationError{Field: "water_amount", Message: "water amount must be between 0 and 10000ml"})
		}
	}

	// Parse and validate coffee amount
	if coffeeStr := r.FormValue("coffee_amount"); coffeeStr != "" {
		var err error
		coffeeAmount, err = strconv.Atoi(coffeeStr)
		if err != nil {
			errs = append(errs, ValidationError{Field: "coffee_amount", Message: "invalid coffee amount"})
		} else if coffeeAmount < 0 || coffeeAmount > 1000 {
			errs = append(errs, ValidationError{Field: "coffee_amount", Message: "coffee amount must be between 0 and 1000g"})
		}
	}

	// Parse and validate time
	if timeStr := r.FormValue("time_seconds"); timeStr != "" {
		var err error
		timeSeconds, err = strconv.Atoi(timeStr)
		if err != nil {
			errs = append(errs, ValidationError{Field: "time_seconds", Message: "invalid time"})
		} else if timeSeconds < 0 || timeSeconds > 3600 {
			errs = append(errs, ValidationError{Field: "time_seconds", Message: "brew time must be between 0 and 3600 seconds"})
		}
	}

	// Parse and validate rating
	if ratingStr := r.FormValue("rating"); ratingStr != "" {
		var err error
		rating, err = strconv.Atoi(ratingStr)
		if err != nil {
			errs = append(errs, ValidationError{Field: "rating", Message: "invalid rating"})
		} else if rating < 0 || rating > 10 {
			errs = append(errs, ValidationError{Field: "rating", Message: "rating must be between 0 and 10"})
		}
	}

	// Parse pours
	pours = parsePours(r)

	return
}

// Create new brew
func (h *Handler) HandleBrewCreate(w http.ResponseWriter, r *http.Request) {
	// Require authentication first
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Validate input
	temperature, waterAmount, coffeeAmount, timeSeconds, rating, pours, validationErrs := validateBrewRequest(r)
	if len(validationErrs) > 0 {
		// Return first validation error
		http.Error(w, validationErrs[0].Message, http.StatusBadRequest)
		return
	}

	// Validate required fields
	beanRKey := r.FormValue("bean_rkey")
	if beanRKey == "" {
		http.Error(w, "Bean selection is required", http.StatusBadRequest)
		return
	}
	if !atproto.ValidateRKey(beanRKey) {
		http.Error(w, "Invalid bean selection", http.StatusBadRequest)
		return
	}

	// Validate optional rkeys
	grinderRKey := r.FormValue("grinder_rkey")
	if errMsg := validateOptionalRKey(grinderRKey, "Grinder selection"); errMsg != "" {
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	brewerRKey := r.FormValue("brewer_rkey")
	if errMsg := validateOptionalRKey(brewerRKey, "Brewer selection"); errMsg != "" {
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	req := &models.CreateBrewRequest{
		BeanRKey:     beanRKey,
		Method:       r.FormValue("method"),
		Temperature:  temperature,
		WaterAmount:  waterAmount,
		CoffeeAmount: coffeeAmount,
		TimeSeconds:  timeSeconds,
		GrindSize:    r.FormValue("grind_size"),
		GrinderRKey:  grinderRKey,
		BrewerRKey:   brewerRKey,
		TastingNotes: r.FormValue("tasting_notes"),
		Rating:       rating,
		Pours:        pours,
	}

	_, err := store.CreateBrew(r.Context(), req, 1) // User ID not used with atproto
	if err != nil {
		http.Error(w, "Failed to create brew", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to create brew")
		return
	}

	// Redirect to brew list
	w.Header().Set("HX-Redirect", "/brews")
	w.WriteHeader(http.StatusOK)
}

// Update existing brew
func (h *Handler) HandleBrewUpdate(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Validate input
	temperature, waterAmount, coffeeAmount, timeSeconds, rating, pours, validationErrs := validateBrewRequest(r)
	if len(validationErrs) > 0 {
		http.Error(w, validationErrs[0].Message, http.StatusBadRequest)
		return
	}

	// Validate required fields
	beanRKey := r.FormValue("bean_rkey")
	if beanRKey == "" {
		http.Error(w, "Bean selection is required", http.StatusBadRequest)
		return
	}
	if !atproto.ValidateRKey(beanRKey) {
		http.Error(w, "Invalid bean selection", http.StatusBadRequest)
		return
	}

	// Validate optional rkeys
	grinderRKey := r.FormValue("grinder_rkey")
	if errMsg := validateOptionalRKey(grinderRKey, "Grinder selection"); errMsg != "" {
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	brewerRKey := r.FormValue("brewer_rkey")
	if errMsg := validateOptionalRKey(brewerRKey, "Brewer selection"); errMsg != "" {
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	req := &models.CreateBrewRequest{
		BeanRKey:     beanRKey,
		Method:       r.FormValue("method"),
		Temperature:  temperature,
		WaterAmount:  waterAmount,
		CoffeeAmount: coffeeAmount,
		TimeSeconds:  timeSeconds,
		GrindSize:    r.FormValue("grind_size"),
		GrinderRKey:  grinderRKey,
		BrewerRKey:   brewerRKey,
		TastingNotes: r.FormValue("tasting_notes"),
		Rating:       rating,
		Pours:        pours,
	}

	err := store.UpdateBrewByRKey(r.Context(), rkey, req)
	if err != nil {
		http.Error(w, "Failed to update brew", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update brew")
		return
	}

	// Redirect to brew list
	w.Header().Set("HX-Redirect", "/brews")
	w.WriteHeader(http.StatusOK)
}

// Delete brew
func (h *Handler) HandleBrewDelete(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if err := store.DeleteBrewByRKey(r.Context(), rkey); err != nil {
		http.Error(w, "Failed to delete brew", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to delete brew")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Export brews as JSON
func (h *Handler) HandleBrewExport(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	brews, err := store.ListBrews(r.Context(), 1) // User ID is not used with atproto
	if err != nil {
		http.Error(w, "Failed to fetch brews", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to list brews for export")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=arabica-brews.json")

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(brews); err != nil {
		log.Error().Err(err).Msg("Failed to encode brews for export")
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
		SubjectURI: subjectURI,
		SubjectCID: subjectCID,
		IsLiked:    isLiked,
		LikeCount:  likeCount,
	}).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render like button")
	}
}

// API endpoint to list all user data (beans, roasters, grinders, brewers, brews)
// Used by client-side cache for faster page loads
func (h *Handler) HandleAPIListAll(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Get user DID for cache validation
	userDID, _ := atproto.GetAuthenticatedDID(r.Context())

	ctx := r.Context()

	// Fetch all collections in parallel using errgroup
	g, ctx := errgroup.WithContext(ctx)

	var beans []*models.Bean
	var roasters []*models.Roaster
	var grinders []*models.Grinder
	var brewers []*models.Brewer
	var brews []*models.Brew

	g.Go(func() error {
		var err error
		beans, err = store.ListBeans(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		roasters, err = store.ListRoasters(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		grinders, err = store.ListGrinders(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		brewers, err = store.ListBrewers(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		brews, err = store.ListBrews(ctx, 1) // User ID not used with atproto
		return err
	})

	if err := g.Wait(); err != nil {
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to fetch all data for API")
		return
	}

	// Link beans to roasters
	atproto.LinkBeansToRoasters(beans, roasters)

	response := map[string]interface{}{
		"did":      userDID,
		"beans":    beans,
		"roasters": roasters,
		"grinders": grinders,
		"brewers":  brewers,
		"brews":    brews,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().Err(err).Msg("Failed to encode API response")
	}
}

// API endpoint to create bean
func (h *Handler) HandleBeanCreate(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req models.CreateBeanRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = models.CreateBeanRequest{
			Name:        r.FormValue("name"),
			Origin:      r.FormValue("origin"),
			RoastLevel:  r.FormValue("roast_level"),
			Process:     r.FormValue("process"),
			Description: r.FormValue("description"),
			RoasterRKey: r.FormValue("roaster_rkey"),
			Closed:      r.FormValue("closed") == "true",
		}
		log.Debug().
			Str("name", req.Name).
			Str("closed_value", r.FormValue("closed")).
			Bool("closed_parsed", req.Closed).
			Msg("Parsed bean create form")
		return nil
	}); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate optional roaster rkey
	if errMsg := validateOptionalRKey(req.RoasterRKey, "Roaster selection"); errMsg != "" {
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	bean, err := store.CreateBean(r.Context(), &req)
	if err != nil {
		http.Error(w, "Failed to create bean", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to create bean")
		return
	}

	writeJSON(w, bean, "bean")
}

// API endpoint to create roaster
func (h *Handler) HandleRoasterCreate(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req models.CreateRoasterRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = models.CreateRoasterRequest{
			Name:     r.FormValue("name"),
			Location: r.FormValue("location"),
			Website:  r.FormValue("website"),
		}
		return nil
	}); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	roaster, err := store.CreateRoaster(r.Context(), &req)
	if err != nil {
		http.Error(w, "Failed to create roaster", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to create roaster")
		return
	}

	writeJSON(w, roaster, "roaster")
}

// Manage page
func (h *Handler) HandleManage(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	_, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	didStr, _ := atproto.GetAuthenticatedDID(r.Context())
	userProfile := h.getUserProfile(r.Context(), didStr)

	// Create layout data
	layoutData := h.buildLayoutData(r, "Manage", authenticated, didStr, userProfile)

	// Create manage props
	manageProps := pages.ManageProps{}

	// Render using templ component
	if err := pages.Manage(layoutData, manageProps).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render manage page")
	}
}

// Bean update/delete handlers
func (h *Handler) HandleBeanUpdate(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req models.UpdateBeanRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = models.UpdateBeanRequest{
			Name:        r.FormValue("name"),
			Origin:      r.FormValue("origin"),
			RoastLevel:  r.FormValue("roast_level"),
			Process:     r.FormValue("process"),
			Description: r.FormValue("description"),
			RoasterRKey: r.FormValue("roaster_rkey"),
			Closed:      r.FormValue("closed") == "true",
		}
		log.Debug().
			Str("rkey", rkey).
			Str("name", req.Name).
			Str("closed_value", r.FormValue("closed")).
			Bool("closed_parsed", req.Closed).
			Msg("Parsed bean update form")
		return nil
	}); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate optional roaster rkey
	if errMsg := validateOptionalRKey(req.RoasterRKey, "Roaster selection"); errMsg != "" {
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	if err := store.UpdateBeanByRKey(r.Context(), rkey, &req); err != nil {
		http.Error(w, "Failed to update bean", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update bean")
		return
	}

	bean, err := store.GetBeanByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Failed to fetch updated bean", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get bean after update")
		return
	}

	writeJSON(w, bean, "bean")
}

func (h *Handler) HandleBeanDelete(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if err := store.DeleteBeanByRKey(r.Context(), rkey); err != nil {
		http.Error(w, "Failed to delete bean", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to delete bean")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Roaster update/delete handlers
func (h *Handler) HandleRoasterUpdate(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req models.UpdateRoasterRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = models.UpdateRoasterRequest{
			Name:     r.FormValue("name"),
			Location: r.FormValue("location"),
			Website:  r.FormValue("website"),
		}
		return nil
	}); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := store.UpdateRoasterByRKey(r.Context(), rkey, &req); err != nil {
		http.Error(w, "Failed to update roaster", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update roaster")
		return
	}

	roaster, err := store.GetRoasterByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Failed to fetch updated roaster", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get roaster after update")
		return
	}

	writeJSON(w, roaster, "roaster")
}

func (h *Handler) HandleRoasterDelete(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if err := store.DeleteRoasterByRKey(r.Context(), rkey); err != nil {
		http.Error(w, "Failed to delete roaster", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to delete roaster")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Grinder CRUD handlers
func (h *Handler) HandleGrinderCreate(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req models.CreateGrinderRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = models.CreateGrinderRequest{
			Name:        r.FormValue("name"),
			GrinderType: r.FormValue("grinder_type"),
			BurrType:    r.FormValue("burr_type"),
			Notes:       r.FormValue("notes"),
		}
		return nil
	}); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	grinder, err := store.CreateGrinder(r.Context(), &req)
	if err != nil {
		http.Error(w, "Failed to create grinder", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to create grinder")
		return
	}

	writeJSON(w, grinder, "grinder")
}

func (h *Handler) HandleGrinderUpdate(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req models.UpdateGrinderRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = models.UpdateGrinderRequest{
			Name:        r.FormValue("name"),
			GrinderType: r.FormValue("grinder_type"),
			BurrType:    r.FormValue("burr_type"),
			Notes:       r.FormValue("notes"),
		}
		return nil
	}); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := store.UpdateGrinderByRKey(r.Context(), rkey, &req); err != nil {
		http.Error(w, "Failed to update grinder", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update grinder")
		return
	}

	grinder, err := store.GetGrinderByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Failed to fetch updated grinder", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get grinder after update")
		return
	}

	writeJSON(w, grinder, "grinder")
}

func (h *Handler) HandleGrinderDelete(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if err := store.DeleteGrinderByRKey(r.Context(), rkey); err != nil {
		http.Error(w, "Failed to delete grinder", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to delete grinder")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Brewer CRUD handlers
func (h *Handler) HandleBrewerCreate(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req models.CreateBrewerRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = models.CreateBrewerRequest{
			Name:        r.FormValue("name"),
			BrewerType:  r.FormValue("brewer_type"),
			Description: r.FormValue("description"),
		}
		return nil
	}); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	brewer, err := store.CreateBrewer(r.Context(), &req)
	if err != nil {
		http.Error(w, "Failed to create brewer", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to create brewer")
		return
	}

	writeJSON(w, brewer, "brewer")
}

func (h *Handler) HandleBrewerUpdate(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req models.UpdateBrewerRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = models.UpdateBrewerRequest{
			Name:        r.FormValue("name"),
			BrewerType:  r.FormValue("brewer_type"),
			Description: r.FormValue("description"),
		}
		return nil
	}); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := store.UpdateBrewerByRKey(r.Context(), rkey, &req); err != nil {
		http.Error(w, "Failed to update brewer", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update brewer")
		return
	}

	brewer, err := store.GetBrewerByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Failed to fetch updated brewer", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get brewer after update")
		return
	}

	writeJSON(w, brewer, "brewer")
}

func (h *Handler) HandleBrewerDelete(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if err := store.DeleteBrewerByRKey(r.Context(), rkey); err != nil {
		http.Error(w, "Failed to delete brewer", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to delete brewer")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// About page
func (h *Handler) HandleAbout(w http.ResponseWriter, r *http.Request) {
	// Check if user is authenticated
	didStr, err := atproto.GetAuthenticatedDID(r.Context())
	isAuthenticated := err == nil && didStr != ""

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.getUserProfile(r.Context(), didStr)
	}

	data := h.buildLayoutData(r, "About", isAuthenticated, didStr, userProfile)

	// Use templ component
	if err := pages.About(data).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render about page")
	}
}

// Terms of Service page
func (h *Handler) HandleTerms(w http.ResponseWriter, r *http.Request) {
	didStr, err := atproto.GetAuthenticatedDID(r.Context())
	isAuthenticated := err == nil

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.getUserProfile(r.Context(), didStr)
	}

	layoutData := h.buildLayoutData(r, "Terms of Service", isAuthenticated, didStr, userProfile)

	if err := pages.Terms(layoutData).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render terms page")
	}
}

func (h *Handler) HandleATProto(w http.ResponseWriter, r *http.Request) {
	didStr, err := atproto.GetAuthenticatedDID(r.Context())
	isAuthenticated := err == nil

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.getUserProfile(r.Context(), didStr)
	}

	layoutData := h.buildLayoutData(r, "AT Protocol", isAuthenticated, didStr, userProfile)

	if err := pages.ATProto(layoutData).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render AT Protocol page")
	}
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

// Modal dialog handlers for entity management

// HandleBeanModalNew renders a new bean modal dialog
func (h *Handler) HandleBeanModalNew(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Fetch roasters for the select dropdown
	roasters, err := store.ListRoasters(r.Context())
	if err != nil {
		log.Warn().Err(err).Msg("Failed to fetch roasters for bean modal")
		roasters = []*models.Roaster{} // Empty list on error
	}

	// Convert to slice for template
	roastersSlice := make([]models.Roaster, len(roasters))
	for i, r := range roasters {
		roastersSlice[i] = *r
	}

	if err := components.BeanDialogModal(nil, roastersSlice).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render bean modal")
	}
}

// HandleBeanModalEdit renders an edit bean modal dialog
func (h *Handler) HandleBeanModalEdit(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Fetch the bean
	bean, err := store.GetBeanByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Bean not found", http.StatusNotFound)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get bean for modal")
		return
	}

	// Fetch roasters for the select dropdown
	roasters, err := store.ListRoasters(r.Context())
	if err != nil {
		log.Warn().Err(err).Msg("Failed to fetch roasters for bean modal")
		roasters = []*models.Roaster{}
	}

	// Convert to slice for template
	roastersSlice := make([]models.Roaster, len(roasters))
	for i, r := range roasters {
		roastersSlice[i] = *r
	}

	if err := components.BeanDialogModal(bean, roastersSlice).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render bean modal")
	}
}

// HandleGrinderModalNew renders a new grinder modal dialog
func (h *Handler) HandleGrinderModalNew(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	_, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if err := components.GrinderDialogModal(nil).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render grinder modal")
	}
}

// HandleGrinderModalEdit renders an edit grinder modal dialog
func (h *Handler) HandleGrinderModalEdit(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Fetch the grinder
	grinder, err := store.GetGrinderByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Grinder not found", http.StatusNotFound)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get grinder for modal")
		return
	}

	if err := components.GrinderDialogModal(grinder).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render grinder modal")
	}
}

// HandleBrewerModalNew renders a new brewer modal dialog
func (h *Handler) HandleBrewerModalNew(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	_, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if err := components.BrewerDialogModal(nil).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render brewer modal")
	}
}

// HandleBrewerModalEdit renders an edit brewer modal dialog
func (h *Handler) HandleBrewerModalEdit(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Fetch the brewer
	brewer, err := store.GetBrewerByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Brewer not found", http.StatusNotFound)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get brewer for modal")
		return
	}

	if err := components.BrewerDialogModal(brewer).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render brewer modal")
	}
}

// HandleRoasterModalNew renders a new roaster modal dialog
func (h *Handler) HandleRoasterModalNew(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	_, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if err := components.RoasterDialogModal(nil).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render roaster modal")
	}
}

// HandleRoasterModalEdit renders an edit roaster modal dialog
func (h *Handler) HandleRoasterModalEdit(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Fetch the roaster
	roaster, err := store.GetRoasterByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Roaster not found", http.StatusNotFound)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get roaster for modal")
		return
	}

	if err := components.RoasterDialogModal(roaster).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render roaster modal")
	}
}

// HandleNotFound renders the 404 page
func (h *Handler) HandleNotFound(w http.ResponseWriter, r *http.Request) {
	// Check if current user is authenticated (for nav bar state)
	didStr, err := atproto.GetAuthenticatedDID(r.Context())
	isAuthenticated := err == nil && didStr != ""

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.getUserProfile(r.Context(), didStr)
	}

	layoutData := h.buildLayoutData(r, "Page Not Found", isAuthenticated, didStr, userProfile)

	w.WriteHeader(http.StatusNotFound)
	if err := pages.NotFound(layoutData).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render 404 page")
	}
}

// HandleReport handles content report submissions
//
// TODO: Implement actual moderation system:
// - Store reports in database (BoltDB bucket or SQLite table)
// - Add admin interface to review reports
// - Implement report status workflow (pending -> reviewed -> dismissed/actioned)
//
// TODO: Reports should be rate limited more strictly by IP than other requests.
// Consider implementing a separate, stricter rate limit for this endpoint
// (e.g., 5 reports per hour per IP) to prevent abuse and report flooding.
func (h *Handler) HandleReport(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	subjectURI := r.FormValue("subject_uri")
	subjectCID := r.FormValue("subject_cid")
	reason := r.FormValue("reason")

	if subjectURI == "" {
		http.Error(w, "subject_uri is required", http.StatusBadRequest)
		return
	}

	// Validate reason
	validReasons := map[string]bool{"spam": true, "inappropriate": true, "other": true}
	if reason == "" || !validReasons[reason] {
		reason = "other"
	}

	// Get reporter info if authenticated
	reporterDID := "anonymous"
	if didStr, err := atproto.GetAuthenticatedDID(r.Context()); err == nil && didStr != "" {
		reporterDID = didStr
	}

	// Get reporter IP for rate limiting tracking
	reporterIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// Use first IP in chain (original client)
		reporterIP = strings.Split(forwarded, ",")[0]
	}

	// Create report record (not persisted yet - just for structured logging)
	report := &models.Report{
		SubjectURI:  subjectURI,
		SubjectCID:  subjectCID,
		Reason:      reason,
		ReporterDID: reporterDID,
		ReporterIP:  reporterIP,
		CreatedAt:   time.Now(),
		Status:      "pending",
	}

	// TODO: Persist report to database
	// For now, log the report for manual review
	log.Info().
		Str("subject_uri", report.SubjectURI).
		Str("subject_cid", report.SubjectCID).
		Str("reason", report.Reason).
		Str("reporter_did", report.ReporterDID).
		Str("reporter_ip", report.ReporterIP).
		Time("created_at", report.CreatedAt).
		Msg("Content report received")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "received"}`))
}
