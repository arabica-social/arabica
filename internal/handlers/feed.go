package handlers

import (
	"context"
	"net/http"

	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/lexicons"
	"tangled.org/arabica.social/arabica/internal/moderation"
	"tangled.org/arabica.social/arabica/internal/ogcard"
	"tangled.org/arabica.social/arabica/internal/web/pages"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/rs/zerolog/log"
)

// BuildModerationContext creates moderation context for feed rendering
// Returns empty context if moderation is not configured or user is not a moderator
func (h *Handler) BuildModerationContext(ctx context.Context, viewerDID string, items []*feed.FeedItem) pages.FeedModerationContext {
	modCtx := pages.FeedModerationContext{
		HiddenURIs: make(map[string]bool),
	}

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

func (h *Handler) homeOGDescription() string {
	if h.homeBehavior.OGDescription != "" {
		return h.homeBehavior.OGDescription
	}
	return "Coffee journaling for the open social web. Track, share, and own your brews."
}

// siteCardOpts builds per-app SiteCardOpts so the site OG image picks up
// the right brand name, tagline, and logo for the running binary.
func (h *Handler) siteCardOpts() ogcard.SiteCardOpts {
	if h.homeBehavior.SiteCardOpts != (ogcard.SiteCardOpts{}) {
		return h.homeBehavior.SiteCardOpts
	}
	// Fallback (e.g. tests without app behavior wired): derive from brand so
	// the default isn't hardcoded to Arabica.
	name := h.appName()
	return ogcard.SiteCardOpts{
		AppName:  name,
		Wordmark: h.brandName(),
		Tagline:  "journaling for the open social web",
		Detail:   "track, share, and own your brews",
	}
}

// HandleSiteOGImage generates a 1200x630 PNG preview card for the site.
func (h *Handler) HandleSiteOGImage(w http.ResponseWriter, r *http.Request) {
	card, err := ogcard.DrawSiteCard(h.siteCardOpts())
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

// HandleFeed serves the community feed as JSON for the SvelteKit SPA.
func (h *Handler) HandleFeed(w http.ResponseWriter, r *http.Request) {
	h.handleFeedJSON(w, r)
}

// feedQueryResult holds the fetched feed items plus the resolved query state
// shared by the JSON render path.
type feedQueryResult struct {
	items           []*feed.FeedItem
	nextCursor      string
	viewerDID       string
	isAuthenticated bool
	typeFilter      lexicons.RecordType
	sortBy          feed.FeedSort
}

// fetchFeed loads feed items for the request, applying type/sort/cursor query
// params and populating per-viewer IsLikedByViewer / IsOwner fields.
func (h *Handler) fetchFeed(r *http.Request) feedQueryResult {
	viewerDID, isAuthenticated := atpmiddleware.GetDID(r.Context())

	typeParam := r.URL.Query().Get("type")
	// Filter pills send the app entity route noun (e.g. "brew", "tea").
	// Resolve through the running app so shared nouns like "brew" map to the
	// current product's record type instead of the global lexicon default.
	var typeFilter lexicons.RecordType
	if typeParam != "" {
		if h.app != nil {
			if route, ok := h.app.EntityRouteByNoun(typeParam); ok {
				typeFilter = route.Type
			} else {
				typeFilter = lexicons.ParseRecordType(typeParam)
			}
		} else {
			typeFilter = lexicons.ParseRecordType(typeParam)
		}
	}
	sortBy := feed.FeedSort(r.URL.Query().Get("sort"))
	cursor := r.URL.Query().Get("cursor")

	if sortBy != feed.FeedSortPopular {
		sortBy = feed.FeedSortRecent
	}

	res := feedQueryResult{
		viewerDID:       viewerDID,
		isAuthenticated: isAuthenticated,
		typeFilter:      typeFilter,
		sortBy:          sortBy,
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
				res.items = result.Items
				res.nextCursor = result.NextCursor
			}
		} else {
			// Unauthenticated users get a limited feed from the cache (no filtering)
			items, err := h.feedService.GetCachedPublicFeed(r.Context())
			if err != nil {
				log.Error().Err(err).Msg("Failed to get cached public feed")
			}
			res.items = items
		}
	}

	if isAuthenticated {
		var likedByViewer map[string]bool
		if h.feedIndex != nil {
			uris := make([]string, 0, len(res.items))
			for _, item := range res.items {
				if item.SubjectURI != "" {
					uris = append(uris, item.SubjectURI)
				}
			}
			likedByViewer = h.feedIndex.HasUserLikedBatch(r.Context(), viewerDID, uris)
		}
		for _, item := range res.items {
			if item.Author != nil {
				item.IsOwner = item.Author.DID == viewerDID
			}
			if likedByViewer != nil {
				item.IsLikedByViewer = likedByViewer[item.SubjectURI]
			}
		}
	}

	return res
}

// handleFeedJSON returns the feed as JSON for the SvelteKit SPA.
func (h *Handler) handleFeedJSON(w http.ResponseWriter, r *http.Request) {
	res := h.fetchFeed(r)
	items := make([]FeedItemJSON, 0, len(res.items))
	for _, item := range res.items {
		items = append(items, NewFeedItemJSON(item))
	}
	// Build moderation context so moderators see hide/block controls and
	// hidden-record badges in the feed.
	modCtx := h.BuildModerationContext(r.Context(), res.viewerDID, res.items)
	ApplyModerationContext(items, modCtx)
	// Build app-scoped filter tabs so each app gets its own labels.
	var tabs []FeedFilterTabJSON
	if h.app != nil {
		tabs = BuildFeedTabs(h.app.Descriptors, h.feedViews)
	}
	WriteJSON(w, FeedResponseJSON{
		Items:           items,
		NextCursor:      res.nextCursor,
		IsAuthenticated: res.isAuthenticated,
		Query: FeedQueryJSON{
			Type: string(res.typeFilter),
			Sort: string(res.sortBy),
		},
		Tabs: tabs,
	}, "feed")
}

// HandleLikeToggle handles creating or deleting a like on a record. The SPA
// always sends Accept: application/json, so this delegates to the JSON path.
func (h *Handler) HandleLikeToggle(w http.ResponseWriter, r *http.Request) {
	h.HandleLikeToggleJSON(w, r)
}
