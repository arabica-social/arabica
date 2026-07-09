package handlers

import (
	"context"
	"net/http"
	"strconv"

	"tangled.org/arabica.social/arabica/internal/firehose"
	"tangled.org/arabica.social/arabica/internal/suggestions"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/rs/zerolog/log"
)

// suggestionSource adapts *firehose.FeedIndex to suggestions.RecordSource.
// The suggestions package defines its own IndexedRecord type so it can
// stay free of the firehose dependency (which in turn imports the
// arabica entity package and would create a cycle).
type suggestionSource struct {
	idx *firehose.FeedIndex
}

func (s suggestionSource) ListRecordsByCollectionOldest(ctx context.Context, collection string) ([]suggestions.IndexedRecord, error) {
	raw, err := s.idx.ListRecordsByCollectionOldest(ctx, collection)
	if err != nil {
		return nil, err
	}
	out := make([]suggestions.IndexedRecord, len(raw))
	for i, r := range raw {
		out[i] = suggestions.IndexedRecord{URI: r.URI, DID: r.DID, Record: r.Record}
	}
	return out, nil
}

func (s suggestionSource) CountReferencesToURI(ctx context.Context, uri string) (int, error) {
	return s.idx.CountReferencesToURI(ctx, uri)
}

// nsidForEntity returns the NSID to query for the requested entity URL
// segment using only the running app's descriptor set. Unknown paths and
// handlers without an app return an empty string so suggestions cannot bleed
// across app registries when URL paths collide.
func (h *Handler) nsidForEntity(entityType string) string {
	route, ok := h.app.EntityRouteByPath(entityType)
	if !ok {
		return ""
	}
	if d := h.app.DescriptorByType(route.Type); d != nil {
		return d.NSID
	}
	return ""
}

// HandleEntitySuggestions returns typeahead suggestions for entity creation
func (h *Handler) HandleEntitySuggestions(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	if _, authenticated := h.GetRecordStore(r); !authenticated {
		WriteJSONError(w, http.StatusUnauthorized, "authentication_required", "Authentication required")
		return
	}

	entityType := r.PathValue("entity")
	nsid := h.nsidForEntity(entityType)
	if nsid == "" {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Unknown entity type")
		return
	}

	query := r.URL.Query().Get("q")
	if len(query) < 2 {
		WriteJSON(w, []suggestions.EntitySuggestion{}, "suggestions")
		return
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 20 {
		limit = 20
	}

	if h.feedIndex == nil {
		WriteJSON(w, []suggestions.EntitySuggestion{}, "suggestions")
		return
	}

	// Exclude the current user's records from suggestions so they only see
	// community records, not their own data echoed back.
	excludeDID, _ := atpmiddleware.GetDID(r.Context())

	results, err := suggestions.Search(r.Context(), suggestionSource{idx: h.feedIndex}, nsid, query, limit, excludeDID)
	if err != nil {
		log.Error().Err(err).Str("entity", entityType).Str("query", query).Msg("Failed to search suggestions")
		WriteJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to search suggestions")
		return
	}

	if results == nil {
		results = []suggestions.EntitySuggestion{}
	}

	WriteJSON(w, results, "suggestions")
}
