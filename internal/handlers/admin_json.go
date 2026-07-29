package handlers

import (
	"net/http"
	"time"

	"tangled.org/arabica.social/arabica/internal/backup"
	"tangled.org/arabica.social/arabica/internal/metrics"
	"tangled.org/arabica.social/arabica/internal/moderation"
	sharedpages "tangled.org/arabica.social/arabica/internal/web/pages"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// AdminMutationResponseJSON is the JSON response for admin mutation endpoints
// (hide, unhide, block, unblock, dismiss-report, reset-autohide, label add/remove).
type AdminMutationResponseJSON struct {
	OK      bool   `json:"ok"`
	Action  string `json:"action"`
	Message string `json:"message,omitempty"`
}

// AdminStatsResponseJSON is the JSON response for GET /api/_mod/stats.
type AdminStatsResponseJSON struct {
	Stats   sharedpages.AdminStats `json:"stats"`
	Backups []backup.SourceStatus  `json:"backups"`
}

// HandleAdminJSON returns admin dashboard data. Route middleware enforces
// moderator access.
func (h *Handler) HandleAdminJSON(w http.ResponseWriter, r *http.Request) {
	userDID, _ := atpmiddleware.GetDID(r.Context())
	props := h.buildAdminProps(r.Context(), userDID)
	WriteJSON(w, props, "admin")
}

// HandleAdminStatsJSON returns admin stats + backups as JSON.
func (h *Handler) HandleAdminStatsJSON(w http.ResponseWriter, r *http.Request) {
	stats := h.collectAdminStats(r.Context())
	var backups []backup.SourceStatus
	if h.backupService != nil {
		backups = h.backupService.Status()
	}
	WriteJSON(w, AdminStatsResponseJSON{
		Stats:   stats,
		Backups: backups,
	}, "admin-stats")
}

// writeAdminMutationJSON writes a JSON success response for admin mutation endpoints.
func writeAdminMutationJSON(w http.ResponseWriter, action, message string) {
	WriteJSON(w, AdminMutationResponseJSON{
		OK:      true,
		Action:  action,
		Message: message,
	}, "admin-"+action)
}

// HandleHideRecordJSON handles hiding a record and returns JSON.
func (h *Handler) HandleHideRecordJSON(w http.ResponseWriter, r *http.Request) {
	userDID, _ := atpmiddleware.GetDID(r.Context())
	if err := r.ParseForm(); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}
	uri := r.FormValue("uri")
	reason := r.FormValue("reason")
	if uri == "" {
		WriteJSONError(w, http.StatusBadRequest, "validation_failed", "URI is required")
		return
	}
	entry := moderation.HiddenRecord{
		ATURI:    uri,
		HiddenAt: time.Now(),
		HiddenBy: userDID,
		Reason:   reason,
	}
	if err := h.moderationStore.HideRecord(r.Context(), entry); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to hide record")
		return
	}
	h.moderationStore.LogAction(r.Context(), moderation.AuditEntry{
		ID:        generateTID(),
		Action:    moderation.AuditActionHideRecord,
		ActorDID:  userDID,
		TargetURI: uri,
		Reason:    reason,
		Timestamp: time.Now(),
	})
	writeAdminMutationJSON(w, "hide", "Record hidden from feed")
}

// HandleUnhideRecordJSON handles unhiding a record and returns JSON.
func (h *Handler) HandleUnhideRecordJSON(w http.ResponseWriter, r *http.Request) {
	userDID, _ := atpmiddleware.GetDID(r.Context())
	if err := r.ParseForm(); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}
	uri := r.FormValue("uri")
	reason := r.FormValue("reason")
	if uri == "" {
		WriteJSONError(w, http.StatusBadRequest, "validation_failed", "URI is required")
		return
	}
	if err := h.moderationStore.UnhideRecord(r.Context(), uri); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to unhide record")
		return
	}
	h.moderationStore.LogAction(r.Context(), moderation.AuditEntry{
		ID:        generateTID(),
		Action:    moderation.AuditActionUnhideRecord,
		ActorDID:  userDID,
		TargetURI: uri,
		Reason:    reason,
		Timestamp: time.Now(),
	})
	writeAdminMutationJSON(w, "unhide", "Record unhidden")
}

// HandleDismissReportJSON handles dismissing a report and returns JSON.
func (h *Handler) HandleDismissReportJSON(w http.ResponseWriter, r *http.Request) {
	userDID, _ := atpmiddleware.GetDID(r.Context())
	if err := r.ParseForm(); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}
	reportID := r.FormValue("id")
	if reportID == "" {
		WriteJSONError(w, http.StatusBadRequest, "validation_failed", "Report ID is required")
		return
	}
	if err := h.moderationStore.ResolveReport(r.Context(), reportID, moderation.ReportStatusDismissed, userDID); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to dismiss report")
		return
	}
	h.moderationStore.LogAction(r.Context(), moderation.AuditEntry{
		ID:        generateTID(),
		Action:    moderation.AuditActionDismissReport,
		ActorDID:  userDID,
		TargetURI: reportID,
		Timestamp: time.Now(),
	})
	writeAdminMutationJSON(w, "dismiss-report", "Report dismissed")
}

// HandleBlockUserJSON handles blocking a user and returns JSON.
func (h *Handler) HandleBlockUserJSON(w http.ResponseWriter, r *http.Request) {
	userDID, _ := atpmiddleware.GetDID(r.Context())
	if err := r.ParseForm(); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}
	did := r.FormValue("did")
	reason := r.FormValue("reason")
	if did == "" {
		WriteJSONError(w, http.StatusBadRequest, "validation_failed", "DID is required")
		return
	}
	entry := moderation.BlacklistedUser{
		DID:           did,
		BlacklistedAt: time.Now(),
		BlacklistedBy: userDID,
		Reason:        reason,
	}
	if err := h.moderationStore.BlacklistUser(r.Context(), entry); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to block user")
		return
	}
	h.moderationStore.LogAction(r.Context(), moderation.AuditEntry{
		ID:        generateTID(),
		Action:    moderation.AuditActionBlacklistUser,
		ActorDID:  userDID,
		TargetURI: did,
		Reason:    reason,
		Timestamp: time.Now(),
	})
	writeAdminMutationJSON(w, "block", "User blocked")
}

// HandleUnblockUserJSON handles unblocking a user and returns JSON.
func (h *Handler) HandleUnblockUserJSON(w http.ResponseWriter, r *http.Request) {
	userDID, _ := atpmiddleware.GetDID(r.Context())
	if err := r.ParseForm(); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}
	did := r.FormValue("did")
	if did == "" {
		WriteJSONError(w, http.StatusBadRequest, "validation_failed", "DID is required")
		return
	}
	if err := h.moderationStore.UnblacklistUser(r.Context(), did); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to unblock user")
		return
	}
	h.moderationStore.LogAction(r.Context(), moderation.AuditEntry{
		ID:        generateTID(),
		Action:    moderation.AuditActionUnblacklistUser,
		ActorDID:  userDID,
		TargetURI: did,
		Timestamp: time.Now(),
	})
	writeAdminMutationJSON(w, "unblock", "User unblocked")
}

// HandleResetAutoHideJSON handles resetting auto-hide counter and returns JSON.
func (h *Handler) HandleResetAutoHideJSON(w http.ResponseWriter, r *http.Request) {
	userDID, _ := atpmiddleware.GetDID(r.Context())
	if err := r.ParseForm(); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}
	targetDID := r.FormValue("did")
	if targetDID == "" {
		WriteJSONError(w, http.StatusBadRequest, "validation_failed", "DID is required")
		return
	}
	now := time.Now()
	if err := h.moderationStore.SetAutoHideReset(r.Context(), targetDID, now); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to reset auto-hide")
		return
	}
	h.moderationStore.LogAction(r.Context(), moderation.AuditEntry{
		ID:        generateTID(),
		Action:    moderation.AuditActionResetAutoHide,
		ActorDID:  userDID,
		TargetURI: targetDID,
		Reason:    "Auto-hide report counter reset",
		Timestamp: now,
	})
	writeAdminMutationJSON(w, "reset-autohide", "Auto-hide counter reset")
}

// HandleAddLabelJSON handles adding a label and returns JSON.
func (h *Handler) HandleAddLabelJSON(w http.ResponseWriter, r *http.Request) {
	userDID, _ := atpmiddleware.GetDID(r.Context())
	if err := r.ParseForm(); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}
	entityType := r.FormValue("entity_type")
	entityID := r.FormValue("entity_id")
	labelName := r.FormValue("label")
	if entityType == "" || entityID == "" || labelName == "" {
		WriteJSONError(w, http.StatusBadRequest, "validation_failed", "entity_type, entity_id, and label are required")
		return
	}
	if entityType != "user" && entityType != "record" {
		WriteJSONError(w, http.StatusBadRequest, "validation_failed", "entity_type must be 'user' or 'record'")
		return
	}
	label := moderation.Label{
		ID:         generateTID(),
		EntityType: entityType,
		EntityID:   entityID,
		Name:       labelName,
		Value:      r.FormValue("value"),
		CreatedAt:  time.Now(),
		CreatedBy:  userDID,
	}
	if ttl := r.FormValue("expires"); ttl != "" {
		if d, err := time.ParseDuration(ttl); err == nil {
			exp := time.Now().Add(d)
			label.ExpiresAt = &exp
		}
	}
	if err := h.moderationStore.AddLabel(r.Context(), label); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to add label")
		return
	}
	h.moderationStore.LogAction(r.Context(), moderation.AuditEntry{
		ID:        generateTID(),
		Action:    moderation.AuditActionAddLabel,
		ActorDID:  userDID,
		TargetURI: entityID,
		Reason:    labelName,
		Details: map[string]string{
			"entity_type": entityType,
			"label":       labelName,
		},
		Timestamp: time.Now(),
	})
	writeAdminMutationJSON(w, "add-label", "Label added")
}

// HandleRemoveLabelJSON handles removing a label and returns JSON.
func (h *Handler) HandleRemoveLabelJSON(w http.ResponseWriter, r *http.Request) {
	userDID, _ := atpmiddleware.GetDID(r.Context())
	if err := r.ParseForm(); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}
	entityType := r.FormValue("entity_type")
	entityID := r.FormValue("entity_id")
	labelName := r.FormValue("label")
	if entityType == "" || entityID == "" || labelName == "" {
		WriteJSONError(w, http.StatusBadRequest, "validation_failed", "entity_type, entity_id, and label are required")
		return
	}
	if err := h.moderationStore.RemoveLabel(r.Context(), entityType, entityID, labelName); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to remove label")
		return
	}
	h.moderationStore.LogAction(r.Context(), moderation.AuditEntry{
		ID:        generateTID(),
		Action:    moderation.AuditActionRemoveLabel,
		ActorDID:  userDID,
		TargetURI: entityID,
		Reason:    labelName,
		Details: map[string]string{
			"entity_type": entityType,
			"label":       labelName,
		},
		Timestamp: time.Now(),
	})
	writeAdminMutationJSON(w, "remove-label", "Label removed")
}

// Keep imports used.
var _ = metrics.ReportsTotal
var _ prometheus.Gauge
var _ = dto.Metric{}
