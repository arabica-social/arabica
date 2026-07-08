package handlers

import (
	"net/http"
	"strings"
)

// WantsJSON reports whether the client prefers a JSON response over the
// default HTML/HTMX rendering. The SPA migration uses Accept-header
// content negotiation on shared paths (e.g. GET /api/feed) so a single
// route can serve both HTMX partials (existing templ clients) and JSON
// (SvelteKit frontend) without a parallel URL namespace.
//
// A request is treated as JSON when:
//   - the Accept header lists application/json (anywhere, any q-value), or
//   - the request carries an explicit X-Requested-With: JSON marker.
//
// HTMX requests (HX-Request: true) without a JSON Accept are served HTML.
func WantsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	if accept != "" && strings.Contains(accept, "application/json") {
		return true
	}
	return r.Header.Get("X-Requested-With") == "JSON"
}

// AuthorSummary is the author profile slice included in JSON API responses
// for feed items and entity views. It mirrors the fields of atproto.Profile
// (a.k.a. atp.PublicProfile) but flattens the pointer fields so the JSON
// shape is stable for TypeScript consumers.
type AuthorSummary struct {
	DID         string `json:"did"`
	Handle      string `json:"handle"`
	DisplayName string `json:"display_name,omitempty"`
	Avatar      string `json:"avatar,omitempty"`
}

// SocialDataJSON is the JSON-serializable view of SocialData (likes,
// comments, and moderation state) returned by entity view endpoints.
type SocialDataJSON struct {
	IsLiked        bool             `json:"is_liked"`
	LikeCount      int              `json:"like_count"`
	CommentCount   int              `json:"comment_count"`
	Comments       []map[string]any `json:"comments"`
	IsModerator    bool             `json:"is_moderator"`
	CanHideRecord  bool             `json:"can_hide_record"`
	CanBlockUser   bool             `json:"can_block_user"`
	IsRecordHidden bool             `json:"is_record_hidden"`
}
