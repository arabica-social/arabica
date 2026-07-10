package handlers

import (
	"errors"
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

// ParseFormOrMultipart parses the request body for form values regardless of
// Content-Type. The SvelteKit SPA submits forms as multipart/form-data (via
// FormData), while legacy HTMX clients submit application/x-www-form-urlencoded.
//
// Request.ParseForm alone only handles url-encoded bodies: for multipart it
// leaves PostForm empty and, once r.Form is non-nil, subsequent FormValue calls
// never invoke ParseMultipartForm — so every multipart field reads as empty.
// ParseMultipartForm handles both encodings (it calls ParseForm internally and
// then parses the multipart body when present), returning ErrNotMultipart for
// non-multipart requests, which we ignore.
//
// maxMemory is the bytes of multipart file data to keep in memory (the rest
// spills to disk); pass 0 for form-only requests. It mirrors the argument to
// http.Request.ParseMultipartForm.
func ParseFormOrMultipart(r *http.Request, maxMemory int64) error {
	if err := r.ParseMultipartForm(maxMemory); err != nil && !errors.Is(err, http.ErrNotMultipart) {
		return err
	}
	return nil
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
