package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/suggestions"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/rs/zerolog/log"
)

// entityTypeToNSID maps URL path segments to collection NSIDs.
// Built from the descriptor registry so new entities appear automatically.
var entityTypeToNSID = func() map[string]string {
	m := make(map[string]string)
	for _, d := range entities.All() {
		if d.NSID != "" {
			m[d.URLPath] = d.NSID
		}
	}
	return m
}()

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

	// Exclude the current user's records from suggestions so they only see
	// community records, not their own data echoed back.
	excludeDID, _ := atpmiddleware.GetDID(r.Context())

	results, err := suggestions.Search(r.Context(), h.feedIndex, nsid, query, limit, excludeDID)
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
