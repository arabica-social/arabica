package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"tangled.org/arabica.social/arabica/internal/backlinks"
	"tangled.org/arabica.social/arabica/internal/web/bff"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"
)

// BacklinksResponseJSON is the JSON envelope returned by
// GET /api/{entity}/{actor}/{id}/backlinks. See docs/api/backlinks.md.
type BacklinksResponseJSON struct {
	EntityNoun string             `json:"entity_noun"`
	EntityName string             `json:"entity_name"`
	BackURL    string             `json:"back_url"`
	DetailURL  string             `json:"detail_url"`
	Result     *backlinks.Result  `json:"result"`
}

// RenderBacklinksViewJSON is the JSON counterpart to RenderBacklinksView. It
// reuses the same EntityViewLoader.Load pipeline and fetchBacklinksWithOptions,
// then serializes the result to JSON.
func (h *Handler) RenderBacklinksViewJSON(w http.ResponseWriter, r *http.Request, cfg EntityViewConfig) {
	rkey := ValidateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	owner := r.URL.Query().Get("owner")
	didStr, _ := atpmiddleware.GetDID(r.Context())
	isAuthenticated := didStr != ""
	if owner == "" && !isAuthenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.GetUserProfile(r.Context(), didStr)
	}

	loaded, err := h.EntityViewLoader().Load(r, rkey, cfg.loadConfig())
	if err != nil {
		if loadErr, ok := err.(*EntityLoadError); ok {
			http.Error(w, loadErr.Msg, loadErr.HTTPStatus())
		} else {
			http.Error(w, "Failed to load record", http.StatusInternalServerError)
		}
		return
	}

	name := cfg.DisplayName(loaded.Record)
	if name == "" && cfg.Descriptor != nil {
		name = cfg.Descriptor.DisplayName
	}
	ownerID := ownerSegment(owner, userProfile, didStr)
	backURL := fmt.Sprintf("/%s/%s/%s", loaded.Route.Path, ownerID, rkey)
	usageKey := r.URL.Query().Get("usage")
	usagePage, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if usagePage <= 0 {
		usagePage = 1
	}
	result, detailURL := h.fetchBacklinksWithOptions(r.Context(), loaded.SubjectURI, loaded.Route.Path, rkey, ownerID, backlinks.LookupOptions{UsageKey: usageKey, UsagePage: usagePage, UsagePerPage: 25})

	WriteJSON(w, BacklinksResponseJSON{
		EntityNoun: strings.ToLower(loaded.EntityNoun),
		EntityName: name,
		BackURL:    backURL,
		DetailURL:  detailURL,
		Result:     result,
	}, "backlinks")
}
