package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/rs/zerolog/log"
)

// EntityViewJSONResponse is the JSON envelope returned by entity view JSON
// endpoints (GET /api/{entity}/{actor}/{id}). It carries the typed record
// plus all the social, author, and backlink context the SPA needs to render
// a full entity detail page. See docs/api/entities.md for the contract.
type EntityViewJSONResponse struct {
	Record             any             `json:"record"`
	SubjectURI         string          `json:"subject_uri"`
	SubjectCID         string          `json:"subject_cid"`
	Author             *AuthorSummary  `json:"author,omitempty"`
	Social             SocialDataJSON  `json:"social"`
	Backlinks          json.RawMessage `json:"backlinks,omitempty"`
	BacklinksDetailURL string          `json:"backlinks_detail_url,omitempty"`
	IsOwnProfile       bool            `json:"is_own_profile"`
	IsAuthenticated    bool            `json:"is_authenticated"`
	ShareURL           string          `json:"share_url"`
	EntityType         string          `json:"entity_type"`
	EntityCount        int             `json:"entity_count"`
	// Extras carries entity-specific view fields that are not part of the
	// record model (e.g. a recipe's resolved forked-from URL + author).
	// Populated via cfg.ViewExtras; nil when the entity has no extras.
	Extras map[string]any `json:"extras,omitempty"`
}

// RenderEntityViewJSON loads records own-store -> witness -> PDS, resolves
// references, and adds social and backlink data.
//
// cfg.CountLookup is invoked when available to populate EntityCount (e.g.
// brew count for a bean, bean count for a roaster).
func (h *Handler) RenderEntityViewJSON(w http.ResponseWriter, r *http.Request, cfg EntityViewConfig) {
	rkey := ValidateRKeyJSON(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	owner := r.URL.Query().Get("owner")
	didStr, _ := atpmiddleware.GetDID(r.Context())
	isAuthenticated := didStr != ""

	loaded, err := h.EntityViewLoader().Load(r, rkey, cfg.loadConfig())
	if err != nil {
		writeEntityLoadJSONError(w, err)
		return
	}

	var shareURL string
	if owner != "" && loaded.Route.Path != "" {
		shareURL = fmt.Sprintf("/%s/%s/%s", loaded.Route.Path, owner, rkey)
	} else if loaded.IsOwnProfile && didStr != "" && loaded.Route.Path != "" {
		// Owner viewing their own record without ?owner= param: build
		// the share URL from their profile handle, mirroring the HTML path.
		if ap := h.GetUserProfile(r.Context(), didStr); ap != nil && ap.Handle != "" {
			shareURL = fmt.Sprintf("/%s/%s/%s", loaded.Route.Path, ap.Handle, rkey)
		}
	}

	sd := h.FetchSocialData(r.Context(), loaded.SubjectURI, didStr, isAuthenticated)
	bl, blDetailURL := h.fetchBacklinks(r.Context(), loaded.SubjectURI, loaded.Route.Path, rkey, owner)

	authorDID := loaded.OwnerDID
	if authorDID == "" {
		authorDID = didStr
	}

	var author *AuthorSummary
	if ap := h.GetUserProfile(r.Context(), authorDID); ap != nil {
		author = &AuthorSummary{
			DID:         authorDID,
			Handle:      ap.Handle,
			DisplayName: ap.DisplayName,
			Avatar:      ap.Avatar,
		}
	} else if authorDID != "" {
		author = &AuthorSummary{DID: authorDID}
	}

	var entityCount int
	if cfg.CountLookup != nil && loaded.SubjectURI != "" {
		entityCount = cfg.CountLookup(r.Context(), authorDID, loaded.SubjectURI)
	}

	var backlinksJSON json.RawMessage
	if bl != nil {
		if data, err := json.Marshal(bl); err == nil {
			backlinksJSON = data
		} else {
			log.Warn().Err(err).Msg("failed to marshal backlinks for entity view JSON")
		}
	}

	var extras map[string]any
	if cfg.ViewExtras != nil {
		extras = cfg.ViewExtras(r.Context(), loaded.Record)
	}

	resp := EntityViewJSONResponse{
		Record:     loaded.Record,
		SubjectURI: loaded.SubjectURI,
		SubjectCID: loaded.SubjectCID,
		Author:     author,
		Social: SocialDataJSON{
			IsLiked:        sd.IsLiked,
			LikeCount:      sd.LikeCount,
			CommentCount:   sd.CommentCount,
			Comments:       NewCommentsJSON(sd.Comments),
			IsModerator:    sd.IsModerator,
			CanHideRecord:  sd.CanHideRecord,
			CanBlockUser:   sd.CanBlockUser,
			IsRecordHidden: sd.IsRecordHidden,
		},
		Backlinks:          backlinksJSON,
		BacklinksDetailURL: blDetailURL,
		IsOwnProfile:       loaded.IsOwnProfile,
		IsAuthenticated:    isAuthenticated,
		ShareURL:           shareURL,
		EntityType:         strings.ToLower(loaded.EntityNoun),
		EntityCount:        entityCount,
		Extras:             extras,
	}

	WriteJSON(w, resp, loaded.EntityNoun+"-json")
}

func writeEntityLoadJSONError(w http.ResponseWriter, err error) {
	loadErr, ok := err.(*EntityLoadError)
	if !ok {
		WriteJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to load record")
		return
	}

	status := loadErr.HTTPStatus()
	code := "internal_error"
	message := "Failed to load record"
	switch status {
	case http.StatusBadRequest:
		code = "invalid_request"
		message = loadErr.Msg
	case http.StatusNotFound:
		code = "not_found"
		message = loadErr.Msg
	}
	WriteJSONError(w, status, code, message)
}
