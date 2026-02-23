package handlers

import (
	"context"
	"net/http"
	"time"

	"arabica/internal/atproto"
	"arabica/internal/database/boltstore"
	"arabica/internal/metrics"
	"arabica/internal/middleware"
	"arabica/internal/moderation"
	"arabica/internal/web/components"
	"arabica/internal/web/pages"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
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
		log.Warn().Str("did", userDID).Str("endpoint", "/_mod/hide").Msg("Denied: insufficient permissions")
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	// Parse form data only (JSON is rejected to prevent CSRF bypass)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	var req hideRequest
	req.URI = r.FormValue("uri")
	req.Reason = r.FormValue("reason")

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

	w.Header().Set("HX-Trigger", "mod-action")
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
		log.Warn().Str("did", userDID).Str("endpoint", "/_mod/unhide").Msg("Denied: insufficient permissions")
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	// Parse form data only (JSON is rejected to prevent CSRF bypass)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	var req hideRequest
	req.URI = r.FormValue("uri")
	req.Reason = r.FormValue("reason")

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

	w.Header().Set("HX-Trigger", "mod-action")
	w.WriteHeader(http.StatusOK)
}

// generateTID generates a TID (timestamp-based identifier) using the AT Protocol TID format.
func generateTID() string {
	return syntax.NewTIDNow(0).String()
}

// buildAdminProps builds the admin dashboard props for the given moderator.
func (h *Handler) buildAdminProps(ctx context.Context, userDID string) pages.AdminProps {
	canHide := h.moderationService.HasPermission(userDID, moderation.PermissionHideRecord)
	canUnhide := h.moderationService.HasPermission(userDID, moderation.PermissionUnhideRecord)
	canViewLogs := h.moderationService.HasPermission(userDID, moderation.PermissionViewAuditLog)
	canViewReports := h.moderationService.HasPermission(userDID, moderation.PermissionViewReports)
	canBlock := h.moderationService.HasPermission(userDID, moderation.PermissionBlacklistUser)
	canUnblock := h.moderationService.HasPermission(userDID, moderation.PermissionUnblacklistUser)
	canResetAutoHide := h.moderationService.HasPermission(userDID, moderation.PermissionResetAutoHide)

	var hiddenRecords []moderation.HiddenRecord
	var auditLog []moderation.AuditEntry
	var enrichedReports []pages.EnrichedReport
	var blockedUsers []moderation.BlacklistedUser

	if (canHide || canUnhide) && h.moderationStore != nil {
		hiddenRecords, _ = h.moderationStore.ListHiddenRecords(ctx)
	}

	if canViewLogs && h.moderationStore != nil {
		auditLog, _ = h.moderationStore.ListAuditLog(ctx, 50)
	}

	if canViewReports && h.moderationStore != nil {
		reports, _ := h.moderationStore.ListPendingReports(ctx)
		enrichedReports = h.enrichReports(ctx, reports)
	}

	if (canBlock || canUnblock) && h.moderationStore != nil {
		blockedUsers, _ = h.moderationStore.ListBlacklistedUsers(ctx)
	}

	isAdmin := h.moderationService.IsAdmin(userDID)

	var joinRequests []*boltstore.JoinRequest
	if isAdmin && h.joinStore != nil {
		joinRequests, _ = h.joinStore.ListRequests()
	}

	// Build stats for admin users
	var stats pages.AdminStats
	if isAdmin {
		stats = h.collectAdminStats()
	}

	return pages.AdminProps{
		HiddenRecords:  hiddenRecords,
		AuditLog:       auditLog,
		Reports:        enrichedReports,
		BlockedUsers:   blockedUsers,
		JoinRequests:   joinRequests,
		Stats:          stats,
		CanHide:        canHide,
		CanUnhide:      canUnhide,
		CanViewLogs:    canViewLogs,
		CanViewReports: canViewReports,
		CanBlock:          canBlock,
		CanUnblock:        canUnblock,
		CanResetAutoHide:  canResetAutoHide,
		IsAdmin:           isAdmin,
	}
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
		log.Warn().Str("did", userDID).Str("endpoint", "/_mod").Msg("Denied: not a moderator")
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	userProfile := h.getUserProfile(r.Context(), userDID)
	adminProps := h.buildAdminProps(r.Context(), userDID)

	layoutData := &components.LayoutData{
		Title:           "Moderation",
		IsAuthenticated: true,
		UserDID:         userDID,
		UserProfile:     userProfile,
		CSPNonce:        middleware.CSPNonceFromContext(r.Context()),
		IsModerator:     true,
	}

	if err := pages.Admin(layoutData, adminProps).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render admin page")
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// HandleAdminPartial renders just the admin dashboard content (for HTMX refresh)
func (h *Handler) HandleAdminPartial(w http.ResponseWriter, r *http.Request) {
	userDID, err := atproto.GetAuthenticatedDID(r.Context())
	if err != nil || userDID == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if h.moderationService == nil || !h.moderationService.IsModerator(userDID) {
		log.Warn().Str("did", userDID).Str("endpoint", "/_mod/content").Msg("Denied: not a moderator")
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	adminProps := h.buildAdminProps(r.Context(), userDID)

	if err := pages.AdminDashboardBody(adminProps).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render admin partial")
		http.Error(w, "Failed to render", http.StatusInternalServerError)
	}
}

// enrichReports resolves handles and fetches post content for reports
func (h *Handler) enrichReports(ctx context.Context, reports []moderation.Report) []pages.EnrichedReport {
	if len(reports) == 0 {
		return nil
	}

	publicClient := atproto.NewPublicClient()
	enriched := make([]pages.EnrichedReport, 0, len(reports))

	for _, report := range reports {
		er := pages.EnrichedReport{
			Report: report,
		}

		// Resolve owner handle
		if profile, err := publicClient.GetProfile(ctx, report.SubjectDID); err == nil {
			er.OwnerHandle = profile.Handle
		}

		// Resolve reporter handle
		if profile, err := publicClient.GetProfile(ctx, report.ReporterDID); err == nil {
			er.ReporterHandle = profile.Handle
		}

		// Fetch post content summary
		er.PostContent = h.getPostContentSummary(ctx, publicClient, report.SubjectURI)

		enriched = append(enriched, er)
	}

	return enriched
}

// getPostContentSummary fetches a summary of post content from an AT-URI
func (h *Handler) getPostContentSummary(ctx context.Context, publicClient *atproto.PublicClient, atURI string) string {
	// Parse AT-URI to get DID, collection, and rkey
	components, err := atproto.ResolveATURI(atURI)
	if err != nil {
		return ""
	}

	// Fetch the record
	record, err := publicClient.GetRecord(ctx, components.DID, components.Collection, components.RKey)
	if err != nil {
		return ""
	}

	// Build summary based on record type
	var summary string

	// Check for brew records
	if method, ok := record.Value["method"].(string); ok {
		summary = "Brew: " + method
	}
	if tastingNotes, ok := record.Value["tastingNotes"].(string); ok && tastingNotes != "" {
		if summary != "" {
			summary += "\n"
		}
		// Truncate long tasting notes
		if len(tastingNotes) > 200 {
			summary += tastingNotes[:200] + "..."
		} else {
			summary += tastingNotes
		}
	}

	// Check for bean records
	if name, ok := record.Value["name"].(string); ok {
		if summary == "" {
			summary = "Bean: " + name
		}
	}

	// If no specific fields found, return a generic message
	if summary == "" {
		summary = "(Record content not available)"
	}

	return summary
}

// blockRequest is the request body for blocking a user
type blockRequest struct {
	DID    string `json:"did"`
	Reason string `json:"reason,omitempty"`
}

// HandleBlockUser handles POST /_mod/block
func (h *Handler) HandleBlockUser(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	userDID, err := atproto.GetAuthenticatedDID(r.Context())
	if err != nil || userDID == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Check permission
	if h.moderationService == nil || !h.moderationService.HasPermission(userDID, moderation.PermissionBlacklistUser) {
		log.Warn().Str("did", userDID).Str("endpoint", "/_mod/block").Msg("Denied: insufficient permissions")
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	// Parse form data only (JSON is rejected to prevent CSRF bypass)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	var req blockRequest
	req.DID = r.FormValue("did")
	req.Reason = r.FormValue("reason")

	if req.DID == "" {
		http.Error(w, "DID is required", http.StatusBadRequest)
		return
	}

	// Block the user
	entry := moderation.BlacklistedUser{
		DID:           req.DID,
		BlacklistedAt: time.Now(),
		BlacklistedBy: userDID,
		Reason:        req.Reason,
	}

	if err := h.moderationStore.BlacklistUser(r.Context(), entry); err != nil {
		log.Error().Err(err).Str("did", req.DID).Msg("Failed to block user")
		http.Error(w, "Failed to block user", http.StatusInternalServerError)
		return
	}

	// Log the action
	auditEntry := moderation.AuditEntry{
		ID:        generateTID(),
		Action:    moderation.AuditActionBlacklistUser,
		ActorDID:  userDID,
		TargetURI: req.DID,
		Reason:    req.Reason,
		Timestamp: time.Now(),
		AutoMod:   false,
	}
	if err := h.moderationStore.LogAction(r.Context(), auditEntry); err != nil {
		log.Error().Err(err).Msg("Failed to log block action")
	}

	log.Info().
		Str("did", req.DID).
		Str("by", userDID).
		Msg("User blocked")

	w.Header().Set("HX-Trigger", "mod-action")
	w.WriteHeader(http.StatusOK)
}

// HandleUnblockUser handles POST /_mod/unblock
func (h *Handler) HandleUnblockUser(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	userDID, err := atproto.GetAuthenticatedDID(r.Context())
	if err != nil || userDID == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Check permission
	if h.moderationService == nil || !h.moderationService.HasPermission(userDID, moderation.PermissionUnblacklistUser) {
		log.Warn().Str("did", userDID).Str("endpoint", "/_mod/unblock").Msg("Denied: insufficient permissions")
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	// Parse form data only (JSON is rejected to prevent CSRF bypass)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	var req blockRequest
	req.DID = r.FormValue("did")

	if req.DID == "" {
		http.Error(w, "DID is required", http.StatusBadRequest)
		return
	}

	// Unblock the user
	if err := h.moderationStore.UnblacklistUser(r.Context(), req.DID); err != nil {
		log.Error().Err(err).Str("did", req.DID).Msg("Failed to unblock user")
		http.Error(w, "Failed to unblock user", http.StatusInternalServerError)
		return
	}

	// Log the action
	auditEntry := moderation.AuditEntry{
		ID:        generateTID(),
		Action:    moderation.AuditActionUnblacklistUser,
		ActorDID:  userDID,
		TargetURI: req.DID,
		Timestamp: time.Now(),
		AutoMod:   false,
	}
	if err := h.moderationStore.LogAction(r.Context(), auditEntry); err != nil {
		log.Error().Err(err).Msg("Failed to log unblock action")
	}

	log.Info().
		Str("did", req.DID).
		Str("by", userDID).
		Msg("User unblocked")

	w.Header().Set("HX-Trigger", "mod-action")
	w.WriteHeader(http.StatusOK)
}

// HandleResetAutoHide handles POST /_mod/reset-autohide
// Resets the per-user auto-hide report counter so that only future reports count toward the threshold.
func (h *Handler) HandleResetAutoHide(w http.ResponseWriter, r *http.Request) {
	userDID, err := atproto.GetAuthenticatedDID(r.Context())
	if err != nil || userDID == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if h.moderationService == nil || !h.moderationService.HasPermission(userDID, moderation.PermissionResetAutoHide) {
		log.Warn().Str("did", userDID).Str("endpoint", "/_mod/reset-autohide").Msg("Denied: insufficient permissions")
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	targetDID := r.FormValue("did")
	if targetDID == "" {
		http.Error(w, "DID is required", http.StatusBadRequest)
		return
	}

	now := time.Now()
	if err := h.moderationStore.SetAutoHideReset(r.Context(), targetDID, now); err != nil {
		log.Error().Err(err).Str("did", targetDID).Msg("Failed to reset auto-hide")
		http.Error(w, "Failed to reset auto-hide", http.StatusInternalServerError)
		return
	}

	auditEntry := moderation.AuditEntry{
		ID:        generateTID(),
		Action:    moderation.AuditActionResetAutoHide,
		ActorDID:  userDID,
		TargetURI: targetDID,
		Reason:    "Auto-hide report counter reset",
		Timestamp: now,
		AutoMod:   false,
	}
	if err := h.moderationStore.LogAction(r.Context(), auditEntry); err != nil {
		log.Error().Err(err).Msg("Failed to log reset-autohide action")
	}

	log.Info().
		Str("did", targetDID).
		Str("by", userDID).
		Msg("Auto-hide counter reset for user")

	w.Header().Set("HX-Trigger", "mod-action")
	w.WriteHeader(http.StatusOK)
}

// HandleDismissReport handles POST /_mod/dismiss-report
func (h *Handler) HandleDismissReport(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	userDID, err := atproto.GetAuthenticatedDID(r.Context())
	if err != nil || userDID == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Check permission
	if h.moderationService == nil || !h.moderationService.HasPermission(userDID, moderation.PermissionDismissReport) {
		log.Warn().Str("did", userDID).Str("endpoint", "/_mod/dismiss-report").Msg("Denied: insufficient permissions")
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	// Parse request
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	reportID := r.FormValue("id")
	if reportID == "" {
		http.Error(w, "Report ID is required", http.StatusBadRequest)
		return
	}

	// Dismiss the report
	if err := h.moderationStore.ResolveReport(r.Context(), reportID, moderation.ReportStatusDismissed, userDID); err != nil {
		log.Error().Err(err).Str("reportID", reportID).Msg("Failed to dismiss report")
		http.Error(w, "Failed to dismiss report", http.StatusInternalServerError)
		return
	}

	// Log the action
	auditEntry := moderation.AuditEntry{
		ID:        generateTID(),
		Action:    moderation.AuditActionDismissReport,
		ActorDID:  userDID,
		TargetURI: reportID,
		Timestamp: time.Now(),
		AutoMod:   false,
	}
	if err := h.moderationStore.LogAction(r.Context(), auditEntry); err != nil {
		log.Error().Err(err).Msg("Failed to log dismiss action")
	}

	log.Info().
		Str("reportID", reportID).
		Str("by", userDID).
		Msg("Report dismissed")

	w.Header().Set("HX-Trigger", "mod-action")
	w.WriteHeader(http.StatusOK)
}

// collectAdminStats gathers current system statistics from available data sources.
func (h *Handler) collectAdminStats() pages.AdminStats {
	var stats pages.AdminStats

	if h.feedIndex != nil {
		stats.KnownUsers = h.feedIndex.KnownDIDCount()
		stats.IndexedRecords = h.feedIndex.RecordCount()
		stats.TotalLikes = h.feedIndex.TotalLikeCount()
		stats.TotalComments = h.feedIndex.TotalCommentCount()
		stats.RecordsByCollection = h.feedIndex.RecordCountByCollection()
	}

	if h.feedRegistry != nil {
		stats.RegisteredUsers = h.feedRegistry.Count()
	}

	if h.joinStore != nil {
		if reqs, err := h.joinStore.ListRequests(); err == nil {
			stats.PendingJoinRequests = len(reqs)
		}
	}

	// Read firehose connection state from the Prometheus gauge
	stats.FirehoseConnected = getGaugeValue(metrics.FirehoseConnectionState) == 1

	return stats
}

// getGaugeValue reads the current value of a prometheus.Gauge.
func getGaugeValue(g prometheus.Gauge) float64 {
	m := &dto.Metric{}
	if err := g.Write(m); err != nil {
		return 0
	}
	if m.Gauge != nil {
		return m.GetGauge().GetValue()
	}
	return 0
}

// HandleAdminStats renders the stats partial for HTMX refresh.
func (h *Handler) HandleAdminStats(w http.ResponseWriter, r *http.Request) {
	userDID, err := atproto.GetAuthenticatedDID(r.Context())
	if err != nil || userDID == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if h.moderationService == nil || !h.moderationService.IsAdmin(userDID) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	stats := h.collectAdminStats()

	if err := pages.AdminStatsContent(stats).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render admin stats partial")
		http.Error(w, "Failed to render", http.StatusInternalServerError)
	}
}
