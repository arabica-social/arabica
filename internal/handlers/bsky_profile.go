package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/rs/zerolog/log"
)

// blueskyProfileCollection is the NSID of the Bluesky profile record. The
// profile is a singleton: a single record per repo at rkey "self".
const (
	blueskyProfileCollection = "app.bsky.actor.profile"
	blueskyProfileRKey       = "self"
)

// maxAvatarBytes caps avatar uploads. The Bluesky app rejects images larger
// than ~1MiB after server-side processing; we keep the same ceiling at the
// edge so users get an immediate, comprehensible error.
const maxAvatarBytes = 1024 * 1024

// blueskyProfileForm is the data the settings page needs to render the
// Bluesky-profile-edit card. When HasScopes is false, the page shows a
// "grant elevated permissions" CTA instead of the form.
type blueskyProfileForm struct {
	HasScopes      bool
	DisplayName    string
	AvatarURL      string
	LoadError      string
	NeedsAuthAgain bool
}

// loadBlueskyProfileForm gathers everything HandleSettings needs to render
// the Bluesky-profile card. Returns a non-nil form even on error so the
// caller can render a graceful CTA.
func (h *Handler) loadBlueskyProfileForm(ctx context.Context, didStr, sessionID string) *blueskyProfileForm {
	form := &blueskyProfileForm{}
	if h.oauth == nil || didStr == "" || sessionID == "" {
		return form
	}
	did, err := syntax.ParseDID(didStr)
	if err != nil {
		return form
	}

	scopes, err := h.oauth.SessionScopes(ctx, did, sessionID)
	if err != nil {
		log.Warn().Err(err).Str("user_did", didStr).Msg("Failed to read session scopes")
		form.NeedsAuthAgain = true
		return form
	}
	form.HasScopes = domain.HasBlueskyProfileScopes(scopes)
	if !form.HasScopes {
		return form
	}

	client, err := h.atprotoClient.AtpClient(ctx, did, sessionID)
	if err != nil {
		log.Warn().Err(err).Str("user_did", didStr).Msg("Failed to resume session for bsky profile fetch")
		form.LoadError = "Couldn't reach your PDS to load the current profile."
		return form
	}
	rec, err := client.GetRecord(ctx, blueskyProfileCollection, blueskyProfileRKey)
	if err != nil {
		// A missing profile record is not an error — Bluesky users create the
		// record lazily on first edit, so just leave the form blank.
		if isNotFoundErr(err) {
			return form
		}
		log.Warn().Err(err).Str("user_did", didStr).Msg("Failed to fetch bsky profile record")
		form.LoadError = "Couldn't load the current profile from your PDS."
		return form
	}
	form.DisplayName = stringField(rec.Value, "displayName")
	if avatar, ok := rec.Value["avatar"].(map[string]any); ok {
		if cid := cidFromBlobRef(avatar); cid != "" {
			form.AvatarURL = fmt.Sprintf("https://cdn.bsky.app/img/avatar/plain/%s/%s@jpeg", did.String(), cid)
		}
	}
	return form
}

// HandleUpdateBlueskyProfile handles the form submit from /settings to
// update the user's Bluesky profile record. Expects multipart/form-data:
//
//	displayName (optional text field)
//	avatar      (optional image file; replaces existing avatar)
//
// The session must have the elevated Bluesky profile scopes; if not, the
// handler responds 403 so the page can prompt the user to upgrade scopes.
func (h *Handler) HandleUpdateBlueskyProfile(w http.ResponseWriter, r *http.Request) {
	didStr, ok := atpmiddleware.GetDID(r.Context())
	if !ok || didStr == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	sessionID, ok := atpmiddleware.GetSessionID(r.Context())
	if !ok || sessionID == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	did, err := syntax.ParseDID(didStr)
	if err != nil {
		http.Error(w, "Invalid session", http.StatusBadRequest)
		return
	}

	scopes, err := h.oauth.SessionScopes(r.Context(), did, sessionID)
	if err != nil {
		http.Error(w, "Session expired", http.StatusUnauthorized)
		return
	}
	if !domain.HasBlueskyProfileScopes(scopes) {
		http.Error(w, "Bluesky profile scopes not granted; re-authorize and try again.", http.StatusForbidden)
		return
	}

	if err := r.ParseMultipartForm(maxAvatarBytes + 64*1024); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	displayName := strings.TrimSpace(r.FormValue("displayName"))

	client, err := h.atprotoClient.AtpClient(r.Context(), did, sessionID)
	if err != nil {
		log.Error().Err(err).Str("user_did", didStr).Msg("Failed to resume session for bsky profile write")
		http.Error(w, "Failed to reach your PDS", http.StatusBadGateway)
		return
	}

	// Read existing record so we preserve fields we don't touch (banner,
	// labels, pinnedPost, etc.). 404 is fine — start from scratch.
	var record map[string]any
	if existing, err := client.GetRecord(r.Context(), blueskyProfileCollection, blueskyProfileRKey); err != nil {
		if !isNotFoundErr(err) {
			log.Error().Err(err).Str("user_did", didStr).Msg("Failed to fetch existing bsky profile")
			http.Error(w, "Failed to load existing profile", http.StatusBadGateway)
			return
		}
		record = map[string]any{}
	} else {
		record = existing.Value
		if record == nil {
			record = map[string]any{}
		}
	}
	record["$type"] = blueskyProfileCollection

	// Apply text edits. Empty values clear the field (Bluesky treats absence
	// of the key as no display name); use omit-on-empty to keep the record
	// tidy rather than writing an empty string.
	if displayName == "" {
		delete(record, "displayName")
	} else {
		record["displayName"] = displayName
	}

	// Optional avatar upload. We accept image/* and rely on the PDS to
	// validate MIME — the OAuth scope is blob:image/* so non-image uploads
	// would be rejected upstream regardless.
	if file, header, err := r.FormFile("avatar"); err == nil {
		defer file.Close()
		if header.Size > maxAvatarBytes {
			http.Error(w, fmt.Sprintf("Avatar must be %d bytes or smaller", maxAvatarBytes), http.StatusRequestEntityTooLarge)
			return
		}
		data, err := io.ReadAll(io.LimitReader(file, maxAvatarBytes+1))
		if err != nil {
			http.Error(w, "Failed to read avatar", http.StatusBadRequest)
			return
		}
		if len(data) > maxAvatarBytes {
			http.Error(w, fmt.Sprintf("Avatar must be %d bytes or smaller", maxAvatarBytes), http.StatusRequestEntityTooLarge)
			return
		}
		mime := header.Header.Get("Content-Type")
		if mime == "" {
			mime = "image/jpeg"
		}
		blob, err := client.UploadBlob(r.Context(), data, mime)
		if err != nil {
			log.Error().Err(err).Str("user_did", didStr).Msg("Avatar upload failed")
			http.Error(w, "Failed to upload avatar", http.StatusBadGateway)
			return
		}
		record["avatar"] = blob
	} else if !errors.Is(err, http.ErrMissingFile) {
		// Real error (not just "no file") — surface it.
		log.Warn().Err(err).Msg("Unexpected error reading avatar form field")
	}

	if _, _, err := client.PutRecord(r.Context(), blueskyProfileCollection, blueskyProfileRKey, record); err != nil {
		log.Error().Err(err).Str("user_did", didStr).Msg("Failed to putRecord bsky profile")
		http.Error(w, "Failed to save profile", http.StatusBadGateway)
		return
	}

	// HTMX-friendly inline success ack. The caller swaps a small status
	// span; full redirect not needed.
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<span class="text-sm text-green-700 dark:text-green-400">Saved</span>`))
}

// stringField pulls a string out of a record value map, tolerating missing
// or wrong-typed entries.
func stringField(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}

// cidFromBlobRef extracts the CID string from a blob ref shaped like:
//
//	{ "$type": "blob", "ref": { "$link": "bafy..." }, "mimeType": "...", "size": 1234 }
//
// or the legacy {"cid": "..."} form. Returns "" if it can't find one.
func cidFromBlobRef(blob map[string]any) string {
	if ref, ok := blob["ref"].(map[string]any); ok {
		if link, _ := ref["$link"].(string); link != "" {
			return link
		}
	}
	// Indigo's atp.BlobRef when marshalled through map[string]any flattens
	// the CIDLink type; handle the case where the JSON decoder yielded a
	// plain string under "ref" or "cid".
	if ref, _ := blob["ref"].(string); ref != "" {
		return ref
	}
	if ref, _ := blob["cid"].(string); ref != "" {
		return ref
	}
	return ""
}

// isNotFoundErr reports whether an indigo PDS error indicates the record
// doesn't exist. The PDS returns "RecordNotFound" in the error name; we
// match leniently in case the wrapper text changes.
func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "RecordNotFound") || strings.Contains(s, "not found")
}

