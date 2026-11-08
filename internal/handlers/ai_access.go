package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"
	arabicaai "tangled.org/arabica.social/arabica/internal/arabica/ai"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"
)

func (h *Handler) SetAIAccess(store *arabicaai.AccessStore) { h.aiAccess = store }

func (h AIHandlers) allowed(r *http.Request) bool {
	did, ok := atpmiddleware.GetDID(r.Context())
	if !ok || did == "" {
		return false
	}
	if h.moderationService != nil && h.moderationService.IsModerator(did) {
		return true
	}
	if h.aiAccess == nil {
		return false
	}
	allowed, err := h.aiAccess.Allows(r.Context(), did)
	return err == nil && allowed
}

func (h AIHandlers) RequireAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if !h.allowed(r) {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h AIHandlers) RequirePageAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if h.allowed(r) {
			next.ServeHTTP(w, r)
			return
		}
		NotFoundPage(next).ServeHTTP(w, r)
	})
}

func (h *Handler) HandleAIAccess(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	actor, ok := atpmiddleware.GetDID(r.Context())
	if !ok || h.moderationService == nil || !h.moderationService.IsModerator(actor) {
		WriteJSONError(w, http.StatusForbidden, "forbidden", "Moderator access required")
		return
	}
	if h.aiAccess == nil {
		WriteJSONError(w, 503, "ai_access_unavailable", "AI access management unavailable")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodPut && r.Method != http.MethodDelete {
		w.Header().Set("Allow", "GET, HEAD, PUT, DELETE")
		WriteJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		entries, err := h.aiAccess.List(r.Context())
		if err != nil {
			WriteJSONError(w, 503, "ai_access_unavailable", "AI access management unavailable")
			return
		}
		type staff struct {
			DID string `json:"did"`
		}
		moderators := make([]staff, 0)
		for _, user := range h.moderationService.ListModerators() {
			moderators = append(moderators, staff{DID: user.DID})
		}
		WriteJSON(w, struct {
			Entries    []arabicaai.AccessEntry `json:"entries"`
			Moderators []staff                 `json:"moderators"`
		}{entries, moderators}, "AI access")
		return
	}
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		WriteJSONError(w, 415, "invalid_content_type", "JSON content type required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var req struct {
		DID string `json:"did"`
	}
	if err := decoder.Decode(&req); err != nil {
		WriteJSONError(w, 400, "invalid_request", "A valid DID is required")
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		WriteJSONError(w, 400, "invalid_request", "A valid DID is required")
		return
	}
	if _, err := syntax.ParseDID(req.DID); err != nil {
		WriteJSONError(w, 400, "invalid_request", "A valid DID is required")
		return
	}
	if r.Method == http.MethodPut {
		err = h.aiAccess.Grant(r.Context(), req.DID, actor)
	} else if r.Method == http.MethodDelete {
		err = h.aiAccess.Revoke(r.Context(), req.DID)
	}
	if errors.Is(err, arabicaai.ErrAccessFull) {
		WriteJSONError(w, 409, "ai_access_full", "AI access grant limit reached")
		return
	}
	if err != nil {
		WriteJSONError(w, 503, "ai_access_unavailable", "AI access management unavailable")
		return
	}
	WriteJSON(w, struct {
		Success bool `json:"success"`
	}{true}, "AI access updated")
}
