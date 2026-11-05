package handlers

import (
	"net/http"

	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/firehose"

	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/rs/zerolog/log"
)

// ExploreResponseJSON is the JSON envelope returned by GET /api/explore for the
// SvelteKit SPA. See docs/api/explore.md for the contract.
type ExploreResponseJSON struct {
	Items       []FeedItemJSON                      `json:"items"`
	Documents   map[string]firehose.ExploreDocument `json:"documents"`
	FacetCounts []firehose.ExploreFacetCount        `json:"facet_counts"`
	NextCursor  string                              `json:"next_cursor"`
	Health      firehose.ExploreHealth              `json:"health"`
}

// HandleExploreJSON returns explore results as JSON for the SvelteKit SPA.
// Reuses the same getModeratedExplore pipeline as the HTML handler.
func (h *Handler) HandleExploreJSON(w http.ResponseWriter, r *http.Request) {
	_, authenticated := h.getArabicaStore(r)
	if !authenticated {
		WriteJSONError(w, http.StatusUnauthorized, "authentication_required", "Authentication required")
		return
	}
	if h.FeedIndex() == nil {
		WriteJSONError(w, http.StatusServiceUnavailable, "service_unavailable", "Explore is unavailable")
		return
	}

	viewerDID, _ := atpmiddleware.GetDID(r.Context())

	query := parseExploreQuery(r)
	cf := h.LoadContentFilter(r.Context())
	result, err := h.getModeratedExplore(r, query, cf)
	if err != nil {
		log.Error().Err(err).Msg("failed to query explore")
		WriteJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to load explore")
		return
	}

	// Populate viewer-context fields (IsLikedByViewer, IsOwner).
	uris := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		uris = append(uris, item.SubjectURI)
		if item.Author != nil {
			item.IsOwner = item.Author.DID == viewerDID
		}
	}
	liked := h.FeedIndex().HasUserLikedBatch(r.Context(), viewerDID, uris)
	for _, item := range result.Items {
		item.IsLikedByViewer = liked[item.SubjectURI]
	}

	items := make([]FeedItemJSON, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, NewFeedItemJSON(item))
	}
	// Apply moderation context so moderators get hide/block controls in
	// explore — mirroring the feed JSON path.
	modCtx := h.BuildModerationContext(r.Context(), viewerDID, result.Items)
	ApplyModerationContext(items, modCtx)

	health := h.FeedIndex().ExploreReadiness(r.Context())

	WriteJSON(w, ExploreResponseJSON{
		Items:       items,
		Documents:   result.Documents,
		FacetCounts: result.FacetCounts,
		NextCursor:  result.NextCursor,
		Health:      health,
	}, "explore")
}

// feed import is referenced only for type aliases used by getModeratedExplore.
var _ = feed.FeedSortRecent
