package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"tangled.org/arabica.social/arabica/internal/firehose"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/rs/zerolog/log"
)

// EntityViewJSONResponse is the JSON envelope returned by entity view JSON
// endpoints (GET /api/{entity}/{actor}/{id}). It carries the typed record
// plus all the social, author, and backlink context the SPA needs to render
// a full entity detail page. See docs/api/entities.md for the contract.
type EntityViewJSONResponse struct {
	Record          any             `json:"record"`
	SubjectURI      string          `json:"subject_uri"`
	SubjectCID      string          `json:"subject_cid"`
	Author          *AuthorSummary  `json:"author,omitempty"`
	Social          SocialDataJSON  `json:"social"`
	Backlinks       json.RawMessage `json:"backlinks,omitempty"`
	IsOwnProfile    bool            `json:"is_own_profile"`
	IsAuthenticated bool            `json:"is_authenticated"`
	ShareURL        string          `json:"share_url"`
	EntityType      string          `json:"entity_type"`
	EntityCount     int             `json:"entity_count"`
}

// RenderEntityViewJSON is the JSON counterpart to RenderEntityView. It reuses
// the same EntityViewLoader.Load pipeline (own-store -> witness -> PDS + ref
// resolution), FetchSocialData, and fetchBacklinks, then serializes the
// assembled data to JSON instead of rendering a templ page.
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
	}

	sd := h.FetchSocialData(r.Context(), loaded.SubjectURI, didStr, isAuthenticated)
	bl, _ := h.fetchBacklinks(r.Context(), loaded.SubjectURI, loaded.Route.Path, rkey, owner)

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

	resp := EntityViewJSONResponse{
		Record:     loaded.Record,
		SubjectURI: loaded.SubjectURI,
		SubjectCID: loaded.SubjectCID,
		Author:     author,
		Social: SocialDataJSON{
			IsLiked:        sd.IsLiked,
			LikeCount:      sd.LikeCount,
			CommentCount:   sd.CommentCount,
			Comments:       commentsToMaps(sd.Comments),
			IsModerator:    sd.IsModerator,
			CanHideRecord:  sd.CanHideRecord,
			CanBlockUser:   sd.CanBlockUser,
			IsRecordHidden: sd.IsRecordHidden,
		},
		Backlinks:       backlinksJSON,
		IsOwnProfile:    loaded.IsOwnProfile,
		IsAuthenticated: isAuthenticated,
		ShareURL:        shareURL,
		EntityType:      strings.ToLower(loaded.EntityNoun),
		EntityCount:     entityCount,
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

// commentsToMaps converts IndexedComment slices to a JSON-friendly shape that
// includes the computed profile fields (Handle, DisplayName, Avatar) that the
// firehose.IndexedComment struct marks with json:"-".
func commentsToMaps(comments []firehose.IndexedComment) []map[string]any {
	if len(comments) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(comments))
	for _, c := range comments {
		m := map[string]any{
			"rkey":        c.RKey,
			"subject_uri": c.SubjectURI,
			"text":        c.Text,
			"actor_did":   c.ActorDID,
			"created_at":  c.CreatedAt,
			"depth":       c.Depth,
			"like_count":  c.LikeCount,
			"is_liked":    c.IsLiked,
		}
		if c.ParentURI != "" {
			m["parent_uri"] = c.ParentURI
		}
		if c.ParentRKey != "" {
			m["parent_rkey"] = c.ParentRKey
		}
		if c.CID != "" {
			m["cid"] = c.CID
		}
		if c.Handle != "" {
			m["handle"] = c.Handle
		}
		if c.DisplayName != nil {
			m["display_name"] = *c.DisplayName
		}
		if c.Avatar != nil {
			m["avatar"] = *c.Avatar
		}
		out = append(out, m)
	}
	return out
}
