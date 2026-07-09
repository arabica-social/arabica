package coffeehandlers

import (
	"net/http"

	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/firehose"
	"tangled.org/arabica.social/arabica/internal/handlers"

	"github.com/rs/zerolog/log"
)

// ExploreResponseJSON is the JSON envelope returned by GET /api/explore for the
// SvelteKit SPA. See docs/api/explore.md for the contract.
type ExploreResponseJSON struct {
	Items       []handlers.FeedItemJSON             `json:"items"`
	Documents   map[string]firehose.ExploreDocument `json:"documents"`
	FacetCounts []firehose.ExploreFacetCount        `json:"facet_counts"`
	NextCursor  string                              `json:"next_cursor"`
	Health      firehose.ExploreHealth              `json:"health"`
}

// HandleExploreJSON returns explore results as JSON for the SvelteKit SPA.
// Reuses the same getModeratedExplore pipeline as the HTML handler.
func (h *Handlers) HandleExploreJSON(w http.ResponseWriter, r *http.Request) {
	_, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		handlers.WriteJSONError(w, http.StatusUnauthorized, "authentication_required", "Authentication required")
		return
	}
	if h.FeedIndex() == nil {
		handlers.WriteJSONError(w, http.StatusServiceUnavailable, "service_unavailable", "Explore is unavailable")
		return
	}

	_, viewerDID, _ := h.LayoutDataFromRequest(r, "Explore")

	query := parseExploreQuery(r)
	cf := h.LoadContentFilter(r.Context())
	result, err := h.getModeratedExplore(r, query, cf)
	if err != nil {
		log.Error().Err(err).Msg("failed to query explore")
		handlers.WriteJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to load explore")
		return
	}

	// Populate IsLikedByViewer and IsOwner
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

	// Convert feed items to JSON form
	items := make([]handlers.FeedItemJSON, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, handlers.NewFeedItemJSON(item))
	}

	health := h.FeedIndex().ExploreReadiness(r.Context())

	handlers.WriteJSON(w, ExploreResponseJSON{
		Items:       items,
		Documents:   result.Documents,
		FacetCounts: result.FacetCounts,
		NextCursor:  result.NextCursor,
		Health:      health,
	}, "explore")
}

// Ensure feed import is used (for type references in getModeratedExplore).
var _ = feed.FeedSortRecent
