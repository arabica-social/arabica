package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"arabica/internal/atproto"
	"arabica/internal/suggestions"

	"github.com/rs/zerolog/log"
)

// entityTypeToNSID maps URL path segments to collection NSIDs
var entityTypeToNSID = map[string]string{
	"roasters": atproto.NSIDRoaster,
	"grinders": atproto.NSIDGrinder,
	"brewers":  atproto.NSIDBrewer,
	"beans":    atproto.NSIDBean,
}

// HandleEntitySuggestions returns typeahead suggestions for entity creation
func (h *Handler) HandleEntitySuggestions(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	if _, authenticated := h.getAtprotoStore(r); !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	entityType := r.PathValue("entity")
	nsid, ok := entityTypeToNSID[entityType]
	if !ok {
		http.Error(w, "Unknown entity type", http.StatusBadRequest)
		return
	}

	query := r.URL.Query().Get("q")
	if len(query) < 2 {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
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
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	results, err := suggestions.Search(h.feedIndex, nsid, query, limit)
	if err != nil {
		log.Error().Err(err).Str("entity", entityType).Str("query", query).Msg("Failed to search suggestions")
		http.Error(w, "Failed to search suggestions", http.StatusInternalServerError)
		return
	}

	if results == nil {
		results = []suggestions.EntitySuggestion{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		log.Error().Err(err).Msg("Failed to encode suggestions response")
	}
}
