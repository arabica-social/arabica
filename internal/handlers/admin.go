package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"arabica/internal/atproto"
	"arabica/internal/middleware"
	"arabica/internal/moderation"
	"arabica/internal/web/components"
	"arabica/internal/web/pages"

	"github.com/rs/zerolog/log"
)

// hideRequest is the request body for hiding a record
type hideRequest struct {
	URI    string `json:"uri"`
	Reason string `json:"reason,omitempty"`
}

// HandleHideRecord handles POST /admin/hide
func (h *Handler) HandleHideRecord(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	userDID, err := atproto.GetAuthenticatedDID(r.Context())
	if err != nil || userDID == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Check permission
	if h.moderationService == nil || !h.moderationService.HasPermission(userDID, moderation.PermissionHideRecord) {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	// Parse request - support both JSON and form data
	var req hideRequest
	contentType := r.Header.Get("Content-Type")
	if contentType == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
	} else {
		// Parse as form data (HTMX default)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		req.URI = r.FormValue("uri")
		req.Reason = r.FormValue("reason")
	}

	if req.URI == "" {
		http.Error(w, "URI is required", http.StatusBadRequest)
		return
	}

	// Hide the record
	entry := moderation.HiddenRecord{
		ATURI:      req.URI,
		HiddenAt:   time.Now(),
		HiddenBy:   userDID,
		Reason:     req.Reason,
		AutoHidden: false,
	}

	if err := h.moderationStore.HideRecord(r.Context(), entry); err != nil {
		log.Error().Err(err).Str("uri", req.URI).Msg("Failed to hide record")
		http.Error(w, "Failed to hide record", http.StatusInternalServerError)
		return
	}

	// Log the action
	auditEntry := moderation.AuditEntry{
		ID:        generateTID(),
		Action:    moderation.AuditActionHideRecord,
		ActorDID:  userDID,
		TargetURI: req.URI,
		Reason:    req.Reason,
		Timestamp: time.Now(),
		AutoMod:   false,
	}
	if err := h.moderationStore.LogAction(r.Context(), auditEntry); err != nil {
		log.Error().Err(err).Msg("Failed to log hide action")
		// Don't fail the request, just log the error
	}

	log.Info().
		Str("uri", req.URI).
		Str("by", userDID).
		Msg("Record hidden from feed")

	w.WriteHeader(http.StatusOK)
}

// HandleUnhideRecord handles POST /admin/unhide
func (h *Handler) HandleUnhideRecord(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	userDID, err := atproto.GetAuthenticatedDID(r.Context())
	if err != nil || userDID == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Check permission
	if h.moderationService == nil || !h.moderationService.HasPermission(userDID, moderation.PermissionUnhideRecord) {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	// Parse request - support both JSON and form data
	var req hideRequest
	contentType := r.Header.Get("Content-Type")
	if contentType == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
	} else {
		// Parse as form data (HTMX default)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		req.URI = r.FormValue("uri")
		req.Reason = r.FormValue("reason")
	}

	if req.URI == "" {
		http.Error(w, "URI is required", http.StatusBadRequest)
		return
	}

	// Unhide the record
	if err := h.moderationStore.UnhideRecord(r.Context(), req.URI); err != nil {
		log.Error().Err(err).Str("uri", req.URI).Msg("Failed to unhide record")
		http.Error(w, "Failed to unhide record", http.StatusInternalServerError)
		return
	}

	// Log the action
	auditEntry := moderation.AuditEntry{
		ID:        generateTID(),
		Action:    moderation.AuditActionUnhideRecord,
		ActorDID:  userDID,
		TargetURI: req.URI,
		Reason:    req.Reason,
		Timestamp: time.Now(),
		AutoMod:   false,
	}
	if err := h.moderationStore.LogAction(r.Context(), auditEntry); err != nil {
		log.Error().Err(err).Msg("Failed to log unhide action")
	}

	log.Info().
		Str("uri", req.URI).
		Str("by", userDID).
		Msg("Record unhidden")

	w.WriteHeader(http.StatusOK)
}

// generateTID generates a TID (timestamp-based identifier)
func generateTID() string {
	// Simple implementation using unix nano timestamp
	// In production, you might want a more sophisticated TID generator
	return time.Now().Format("20060102150405.000000000")
}

// HandleAdmin renders the moderation dashboard
func (h *Handler) HandleAdmin(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	userDID, err := atproto.GetAuthenticatedDID(r.Context())
	if err != nil || userDID == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Check if user is a moderator
	if h.moderationService == nil || !h.moderationService.IsModerator(userDID) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get user profile for layout
	userProfile := h.getUserProfile(r.Context(), userDID)

	// Check permissions
	canHide := h.moderationService.HasPermission(userDID, moderation.PermissionHideRecord)
	canUnhide := h.moderationService.HasPermission(userDID, moderation.PermissionUnhideRecord)
	canViewLogs := h.moderationService.HasPermission(userDID, moderation.PermissionViewAuditLog)

	// Fetch data based on permissions
	var hiddenRecords []moderation.HiddenRecord
	var auditLog []moderation.AuditEntry

	if (canHide || canUnhide) && h.moderationStore != nil {
		hiddenRecords, _ = h.moderationStore.ListHiddenRecords(r.Context())
	}

	if canViewLogs && h.moderationStore != nil {
		auditLog, _ = h.moderationStore.ListAuditLog(r.Context(), 50)
	}

	layoutData := &components.LayoutData{
		Title:           "Moderation",
		IsAuthenticated: true,
		UserDID:         userDID,
		UserProfile:     userProfile,
		CSPNonce:        middleware.CSPNonceFromContext(r.Context()),
		IsModerator:     true,
	}

	adminProps := pages.AdminProps{
		HiddenRecords: hiddenRecords,
		AuditLog:      auditLog,
		CanHide:       canHide,
		CanUnhide:     canUnhide,
		CanViewLogs:   canViewLogs,
	}

	if err := pages.Admin(layoutData, adminProps).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render admin page")
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}
