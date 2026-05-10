package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/database/boltstore"
	"tangled.org/arabica.social/arabica/internal/metrics"
	"tangled.org/arabica.social/arabica/internal/middleware"
	"tangled.org/arabica.social/arabica/internal/moderation"
	"tangled.org/arabica.social/arabica/internal/web/components"
	"tangled.org/arabica.social/arabica/internal/web/pages"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"
	"tangled.org/pdewey.com/atp"

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
// Auth and permission checks are handled by RequirePermission middleware.
func (h *Handler) HandleHideRecord(w http.ResponseWriter, r *http.Request) {
	userDID, _ := atpmiddleware.GetDID(r.Context())

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
// Auth and permission checks are handled by RequirePermission middleware.
func (h *Handler) HandleUnhideRecord(w http.ResponseWriter, r *http.Request) {
	userDID, _ := atpmiddleware.GetDID(r.Context())

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
	canManageLabels := h.moderationService.HasPermission(userDID, moderation.PermissionManageLabels)

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

	var labels []moderation.Label
	if canManageLabels && h.moderationStore != nil {
		labels, _ = h.moderationStore.ListAllLabels(ctx)
	}

	isAdmin := h.moderationService.IsAdmin(userDID)

	var joinRequests []*boltstore.JoinRequest
	if isAdmin && h.joinStore != nil {
		joinRequests, _ = h.joinStore.ListRequests(ctx)
	}

	// Build stats for admin users
	var stats pages.AdminStats
	if isAdmin {
		stats = h.collectAdminStats(ctx)
	}

	return pages.AdminProps{
		HiddenRecords:    hiddenRecords,
		AuditLog:         auditLog,
		Reports:          enrichedReports,
		BlockedUsers:     blockedUsers,
		Labels:           labels,
		JoinRequests:     joinRequests,
		Stats:            stats,
		CanHide:          canHide,
		CanUnhide:        canUnhide,
		CanViewLogs:      canViewLogs,
		CanViewReports:   canViewReports,
		CanBlock:         canBlock,
		CanUnblock:       canUnblock,
		CanResetAutoHide: canResetAutoHide,
		CanManageLabels:  canManageLabels,
		IsAdmin:          isAdmin,
	}
}

// HandleAdmin renders the moderation dashboard
func (h *Handler) HandleAdmin(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	userDID, ok := atpmiddleware.GetDID(r.Context())
	if !ok {
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
// Auth and moderator checks are handled by RequireModerator middleware.
func (h *Handler) HandleAdminPartial(w http.ResponseWriter, r *http.Request) {
	userDID, _ := atpmiddleware.GetDID(r.Context())
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
	uriParts, err := atp.ParseATURI(atURI)
	if err != nil {
		return ""
	}

	// Fetch the record
	record, err := publicClient.GetRecord(ctx, uriParts.DID, uriParts.Collection, uriParts.RKey)
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
// Auth and permission checks are handled by RequirePermission middleware.
func (h *Handler) HandleBlockUser(w http.ResponseWriter, r *http.Request) {
	userDID, _ := atpmiddleware.GetDID(r.Context())

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
// Auth and permission checks are handled by RequirePermission middleware.
func (h *Handler) HandleUnblockUser(w http.ResponseWriter, r *http.Request) {
	userDID, _ := atpmiddleware.GetDID(r.Context())

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
// Auth and permission checks are handled by RequirePermission middleware.
func (h *Handler) HandleResetAutoHide(w http.ResponseWriter, r *http.Request) {
	userDID, _ := atpmiddleware.GetDID(r.Context())

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
// Auth and permission checks are handled by RequirePermission middleware.
func (h *Handler) HandleDismissReport(w http.ResponseWriter, r *http.Request) {
	userDID, _ := atpmiddleware.GetDID(r.Context())

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

// HandleAddLabel handles POST /_mod/label/add
// Auth and permission checks are handled by RequirePermission middleware.
func (h *Handler) HandleAddLabel(w http.ResponseWriter, r *http.Request) {
	userDID, _ := atpmiddleware.GetDID(r.Context())

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	entityType := r.FormValue("entity_type")
	entityID := r.FormValue("entity_id")
	labelName := r.FormValue("label")

	if entityType == "" || entityID == "" || labelName == "" {
		http.Error(w, "entity_type, entity_id, and label are required", http.StatusBadRequest)
		return
	}
	if entityType != "user" && entityType != "record" {
		http.Error(w, "entity_type must be 'user' or 'record'", http.StatusBadRequest)
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

	// Parse optional TTL
	if ttl := r.FormValue("expires"); ttl != "" {
		if d, err := time.ParseDuration(ttl); err == nil {
			exp := time.Now().Add(d)
			label.ExpiresAt = &exp
		}
	}

	if err := h.moderationStore.AddLabel(r.Context(), label); err != nil {
		log.Error().Err(err).Str("label", labelName).Msg("Failed to add label")
		http.Error(w, "Failed to add label", http.StatusInternalServerError)
		return
	}

	// Log the action
	auditEntry := moderation.AuditEntry{
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
	}
	if err := h.moderationStore.LogAction(r.Context(), auditEntry); err != nil {
		log.Error().Err(err).Msg("Failed to log add-label action")
	}

	log.Info().
		Str("entity_type", entityType).
		Str("entity_id", entityID).
		Str("label", labelName).
		Str("by", userDID).
		Msg("Label added")

	w.Header().Set("HX-Trigger", "mod-action")
	w.WriteHeader(http.StatusOK)
}

// HandleRemoveLabel handles POST /_mod/label/remove
// Auth and permission checks are handled by RequirePermission middleware.
func (h *Handler) HandleRemoveLabel(w http.ResponseWriter, r *http.Request) {
	userDID, _ := atpmiddleware.GetDID(r.Context())

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	entityType := r.FormValue("entity_type")
	entityID := r.FormValue("entity_id")
	labelName := r.FormValue("label")

	if entityType == "" || entityID == "" || labelName == "" {
		http.Error(w, "entity_type, entity_id, and label are required", http.StatusBadRequest)
		return
	}

	if err := h.moderationStore.RemoveLabel(r.Context(), entityType, entityID, labelName); err != nil {
		log.Error().Err(err).Str("label", labelName).Msg("Failed to remove label")
		http.Error(w, "Failed to remove label", http.StatusInternalServerError)
		return
	}

	// Log the action
	auditEntry := moderation.AuditEntry{
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
	}
	if err := h.moderationStore.LogAction(r.Context(), auditEntry); err != nil {
		log.Error().Err(err).Msg("Failed to log remove-label action")
	}

	log.Info().
		Str("entity_type", entityType).
		Str("entity_id", entityID).
		Str("label", labelName).
		Str("by", userDID).
		Msg("Label removed")

	w.Header().Set("HX-Trigger", "mod-action")
	w.WriteHeader(http.StatusOK)
}

// collectAdminStats gathers current system statistics from available data sources.
func (h *Handler) collectAdminStats(ctx context.Context) pages.AdminStats {
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
		if reqs, err := h.joinStore.ListRequests(ctx); err == nil {
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
// Auth and admin checks are handled by RequireAdmin middleware.
func (h *Handler) HandleAdminStats(w http.ResponseWriter, r *http.Request) {
	stats := h.collectAdminStats(r.Context())

	if err := pages.AdminStatsContent(stats).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render admin stats partial")
		http.Error(w, "Failed to render", http.StatusInternalServerError)
	}
}

// exportedRecord is the per-record shape in the witness export payload.
type exportedRecord struct {
	URI       string          `json:"uri"`
	RKey      string          `json:"rkey"`
	CID       string          `json:"cid"`
	CreatedAt time.Time       `json:"createdAt"`
	IndexedAt time.Time       `json:"indexedAt"`
	Record    json.RawMessage `json:"record"`
}

// witnessExport is the top-level payload returned by HandleAdminExportDID.
type witnessExport struct {
	DID         string                      `json:"did"`
	ExportedAt  time.Time                   `json:"exportedAt"`
	Source      string                      `json:"source"`
	Collections map[string][]exportedRecord `json:"collections"`
}

// HandleAdminExportDID exports every witness-cached record for a given DID as
// a single JSON document. Records come from the firehose-backed SQLite index,
// not the user's PDS. Auth and admin checks are handled by RequireAdmin.
func (h *Handler) HandleAdminExportDID(w http.ResponseWriter, r *http.Request) {
	rawDID := strings.TrimSpace(r.URL.Query().Get("did"))
	if rawDID == "" {
		http.Error(w, "missing 'did' query parameter", http.StatusBadRequest)
		return
	}
	did, err := syntax.ParseDID(rawDID)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid DID: %v", err), http.StatusBadRequest)
		return
	}
	if h.witnessCache == nil {
		http.Error(w, "witness cache not configured", http.StatusServiceUnavailable)
		return
	}

	didStr := did.String()
	out := witnessExport{
		DID:         didStr,
		ExportedAt:  time.Now().UTC(),
		Source:      "witness-cache",
		Collections: make(map[string][]exportedRecord, len(h.appNSIDs())),
	}

	for _, collection := range h.appNSIDs() {
		records, err := h.witnessCache.ListWitnessRecords(r.Context(), didStr, collection)
		if err != nil {
			log.Error().Err(err).Str("did", didStr).Str("collection", collection).Msg("witness export: list failed")
			http.Error(w, "failed to read witness cache", http.StatusInternalServerError)
			return
		}
		exported := make([]exportedRecord, 0, len(records))
		for _, rec := range records {
			exported = append(exported, exportedRecord{
				URI:       rec.URI,
				RKey:      rec.RKey,
				CID:       rec.CID,
				CreatedAt: rec.CreatedAt,
				IndexedAt: rec.IndexedAt,
				Record:    rec.Record,
			})
		}
		out.Collections[collection] = exported
	}

	filename := fmt.Sprintf("arabica-witness-%s-%s.json",
		strings.ReplaceAll(didStr, ":", "_"),
		out.ExportedAt.Format("20060102-150405"))

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		log.Error().Err(err).Str("did", didStr).Msg("witness export: encode failed")
	}
}

// HandleAdminPurgeDID removes every trace of a DID from the witness cache:
// records, likes, comments (including ones targeting this DID's records),
// notifications, profile cache, did_by_handle index, known/registered/backfilled
// tracking, and user settings. Moderation tables are preserved as evidence.
//
// Required when an account orphans its data — e.g. the user's PDS goes away
// without the firehose ever emitting a deleted/takendown account event, so the
// stale records sit in the cache forever. Auth and admin checks are handled by
// RequireAdmin.
func (h *Handler) HandleAdminPurgeDID(w http.ResponseWriter, r *http.Request) {
	rawDID := strings.TrimSpace(r.URL.Query().Get("did"))
	if rawDID == "" {
		// Form posts may put it in the body.
		if err := r.ParseForm(); err == nil {
			rawDID = strings.TrimSpace(r.FormValue("did"))
		}
	}
	if rawDID == "" {
		http.Error(w, "missing 'did' parameter", http.StatusBadRequest)
		return
	}
	did, err := syntax.ParseDID(rawDID)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid DID: %v", err), http.StatusBadRequest)
		return
	}
	if h.feedIndex == nil {
		http.Error(w, "feed index not configured", http.StatusServiceUnavailable)
		return
	}

	didStr := did.String()
	actor, _ := atpmiddleware.GetDID(r.Context())

	if err := h.feedIndex.DeleteAllByDID(r.Context(), didStr); err != nil {
		log.Error().Err(err).Str("did", didStr).Str("actor", actor).Msg("admin purge: DeleteAllByDID failed")
		http.Error(w, "purge failed", http.StatusInternalServerError)
		return
	}
	h.feedIndex.InvalidatePublicCachesForDID(didStr)

	log.Warn().Str("did", didStr).Str("actor", actor).Msg("admin purge: removed all witness data for DID")

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"did":      didStr,
		"purged":   true,
		"purgedAt": time.Now().UTC(),
	})
}

// HandleAdminRebuildDID re-pulls every Arabica record for a DID from their PDS
// and writes them into the witness cache via BackfillUser. Pair with
// HandleAdminPurgeDID to fully recycle a user's witness data — purge clears the
// `backfilled` row, so this call will run a fresh pass instead of short-circuiting.
//
// Auth and admin checks are handled by RequireAdmin.
func (h *Handler) HandleAdminRebuildDID(w http.ResponseWriter, r *http.Request) {
	rawDID := strings.TrimSpace(r.URL.Query().Get("did"))
	if rawDID == "" {
		if err := r.ParseForm(); err == nil {
			rawDID = strings.TrimSpace(r.FormValue("did"))
		}
	}
	if rawDID == "" {
		http.Error(w, "missing 'did' parameter", http.StatusBadRequest)
		return
	}
	did, err := syntax.ParseDID(rawDID)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid DID: %v", err), http.StatusBadRequest)
		return
	}
	if h.feedIndex == nil {
		http.Error(w, "feed index not configured", http.StatusServiceUnavailable)
		return
	}

	didStr := did.String()
	actor, _ := atpmiddleware.GetDID(r.Context())

	if err := h.feedIndex.BackfillUser(r.Context(), didStr, nil); err != nil {
		log.Error().Err(err).Str("did", didStr).Str("actor", actor).Msg("admin rebuild: BackfillUser failed")
		http.Error(w, "rebuild failed", http.StatusInternalServerError)
		return
	}

	log.Warn().Str("did", didStr).Str("actor", actor).Msg("admin rebuild: refilled witness cache from PDS")

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"did":       didStr,
		"rebuilt":   true,
		"rebuiltAt": time.Now().UTC(),
	})
}

// HandleAdminRefreshHandles re-fetches every cached profile from the AppView so
// stale handles get corrected. A less-destructive alternative to purge+rebuild
// when the only thing wrong with a profile is a stale handle from an identity-
// event race. Auth and admin checks are handled by RequireAdmin.
func (h *Handler) HandleAdminRefreshHandles(w http.ResponseWriter, r *http.Request) {
	if h.feedIndex == nil {
		http.Error(w, "feed index not configured", http.StatusServiceUnavailable)
		return
	}
	actor, _ := atpmiddleware.GetDID(r.Context())

	start := time.Now()
	refreshed, failed := h.feedIndex.RefreshAllProfiles(r.Context())

	log.Info().
		Str("actor", actor).
		Int("refreshed", refreshed).
		Int("failed", failed).
		Dur("duration", time.Since(start)).
		Msg("admin refresh handles: complete")

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"refreshed":  refreshed,
		"failed":     failed,
		"durationMs": time.Since(start).Milliseconds(),
		"finishedAt": time.Now().UTC(),
	})
}

// pdsRecord is the per-record shape in the PDS fetch payload.
type pdsRecord struct {
	URI    string         `json:"uri"`
	RKey   string         `json:"rkey"`
	CID    string         `json:"cid"`
	Record map[string]any `json:"record"`
}

// pdsExport is the top-level payload returned by HandleAdminFetchPDSRecords.
type pdsExport struct {
	DID         string                 `json:"did"`
	Handle      string                 `json:"handle,omitempty"`
	FetchedAt   time.Time              `json:"fetchedAt"`
	Source      string                 `json:"source"`
	Collections map[string][]pdsRecord `json:"collections"`
}

// HandleAdminFetchPDSRecords fetches every Arabica record for an account
// directly from the user's PDS and returns it as a single JSON document.
// Accepts `?actor=did:plc:...` or `?actor=handle.example` — handles are
// resolved via the public directory, not the local witness cache, so this
// works even for users who've never appeared on the firehose.
//
// This is the moderator-side counterpart to /_mod/export: where export reads
// the local witness cache, this one reads the canonical PDS state. Useful for
// investigating reports, comparing against the cache, or capturing a snapshot
// before purging. Auth checks are handled by RequireModerator.
func (h *Handler) HandleAdminFetchPDSRecords(w http.ResponseWriter, r *http.Request) {
	actor := strings.TrimSpace(r.URL.Query().Get("actor"))
	if actor == "" {
		http.Error(w, "missing 'actor' query parameter (DID or handle)", http.StatusBadRequest)
		return
	}

	publicClient := atproto.NewPublicClient()

	var didStr, handle string
	if strings.HasPrefix(actor, "did:") {
		did, err := syntax.ParseDID(actor)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid DID: %v", err), http.StatusBadRequest)
			return
		}
		didStr = did.String()
	} else {
		resolved, err := publicClient.ResolveHandle(r.Context(), actor)
		if err != nil {
			log.Warn().Err(err).Str("handle", actor).Msg("PDS fetch: ResolveHandle failed")
			http.Error(w, fmt.Sprintf("could not resolve handle %q: %v", actor, err), http.StatusNotFound)
			return
		}
		didStr = resolved
		handle = actor
	}

	out := pdsExport{
		DID:         didStr,
		Handle:      handle,
		FetchedAt:   time.Now().UTC(),
		Source:      "pds",
		Collections: make(map[string][]pdsRecord, len(h.appNSIDs())),
	}

	requester, _ := atpmiddleware.GetDID(r.Context())

	for _, collection := range h.appNSIDs() {
		records, err := publicClient.ListAllRecords(r.Context(), didStr, collection)
		if err != nil {
			// One collection failing shouldn't sink the whole fetch — record an
			// empty list and continue. The collection key is preserved so the
			// caller can see which slots came up empty.
			log.Warn().Err(err).
				Str("did", didStr).
				Str("collection", collection).
				Str("actor", requester).
				Msg("PDS fetch: ListAllRecords failed for collection")
			out.Collections[collection] = []pdsRecord{}
			continue
		}
		entries := make([]pdsRecord, 0, len(records))
		for _, rec := range records {
			var rkey string
			if rk := atp.RKeyFromURI(rec.URI); rk != "" {
				rkey = rk
			}
			entries = append(entries, pdsRecord{
				URI:    rec.URI,
				RKey:   rkey,
				CID:    rec.CID,
				Record: rec.Value,
			})
		}
		out.Collections[collection] = entries
	}

	log.Info().
		Str("did", didStr).
		Str("handle", handle).
		Str("actor", requester).
		Msg("PDS fetch: returned records")

	filename := fmt.Sprintf("arabica-pds-%s-%s.json",
		strings.ReplaceAll(didStr, ":", "_"),
		out.FetchedAt.Format("20060102-150405"))

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		log.Error().Err(err).Str("did", didStr).Msg("PDS fetch: encode failed")
	}
}
