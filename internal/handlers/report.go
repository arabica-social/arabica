package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"arabica/internal/atproto"
	"arabica/internal/moderation"

	"github.com/rs/zerolog/log"
)

// Automod thresholds for automatic content hiding
const (
	// AutoHideThreshold is the number of reports on a single record before auto-hiding
	AutoHideThreshold = 3
	// AutoHideUserThreshold is the total reports across a user's records before auto-hiding new reports
	AutoHideUserThreshold = 5
	// ReportRateLimitPerHour is the maximum reports a user can submit per hour
	ReportRateLimitPerHour = 10
	// MaxReportReasonLength is the maximum length of a report reason
	MaxReportReasonLength = 500
)

// ReportRequest represents the JSON request for submitting a report
type ReportRequest struct {
	SubjectURI string `json:"subject_uri"`
	SubjectCID string `json:"subject_cid"`
	Reason     string `json:"reason"`
}

// ReportResponse represents the JSON response from report submission
type ReportResponse struct {
	ID      string `json:"id,omitempty"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// HandleReport handles content report submissions.
// Requires authentication, validates input, checks rate limits and duplicates,
// persists the report, and triggers automod if thresholds are reached.
func (h *Handler) HandleReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Require authentication
	reporterDID, err := atproto.GetAuthenticatedDID(ctx)
	if err != nil || reporterDID == "" {
		writeReportError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Check if moderation store is configured
	if h.moderationStore == nil {
		log.Error().Msg("moderation: store not configured")
		writeReportError(w, "Reports are not enabled", http.StatusServiceUnavailable)
		return
	}

	// Parse request (supports both JSON and form data)
	var req ReportRequest
	if isJSONRequest(r) {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeReportError(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			writeReportError(w, "Invalid form data", http.StatusBadRequest)
			return
		}
		req.SubjectURI = r.FormValue("subject_uri")
		req.SubjectCID = r.FormValue("subject_cid")
		req.Reason = r.FormValue("reason")
	}

	// Validate subject URI
	if req.SubjectURI == "" {
		writeReportError(w, "subject_uri is required", http.StatusBadRequest)
		return
	}

	// Parse the subject URI to get the content owner's DID
	uriComponents, err := atproto.ResolveATURI(req.SubjectURI)
	if err != nil {
		writeReportError(w, "Invalid subject_uri format", http.StatusBadRequest)
		return
	}
	subjectDID := uriComponents.DID

	// Prevent self-reporting
	if subjectDID == reporterDID {
		writeReportError(w, "You cannot report your own content", http.StatusBadRequest)
		return
	}

	// Validate and sanitize reason
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "No reason provided"
	}
	if len(reason) > MaxReportReasonLength {
		reason = reason[:MaxReportReasonLength]
	}

	// Check rate limit (10 reports per hour per user)
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	recentCount, err := h.moderationStore.CountReportsFromUserSince(ctx, reporterDID, oneHourAgo)
	if err != nil {
		log.Error().Err(err).Str("reporter", reporterDID).Msg("moderation: failed to check rate limit")
		writeReportError(w, "Failed to process report", http.StatusInternalServerError)
		return
	}
	if recentCount >= ReportRateLimitPerHour {
		writeReportError(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
		return
	}

	// Check for duplicate report
	alreadyReported, err := h.moderationStore.HasReportedURI(ctx, reporterDID, req.SubjectURI)
	if err != nil {
		log.Error().Err(err).Str("reporter", reporterDID).Msg("moderation: failed to check duplicate")
		writeReportError(w, "Failed to process report", http.StatusInternalServerError)
		return
	}
	if alreadyReported {
		writeReportError(w, "You have already reported this content", http.StatusConflict)
		return
	}

	// Create the report
	report := moderation.Report{
		ID:          generateTID(),
		SubjectURI:  req.SubjectURI,
		SubjectDID:  subjectDID,
		ReporterDID: reporterDID,
		Reason:      reason,
		CreatedAt:   time.Now(),
		Status:      moderation.ReportStatusPending,
	}

	// Persist the report
	if err := h.moderationStore.CreateReport(ctx, report); err != nil {
		log.Error().Err(err).Str("reporter", reporterDID).Msg("moderation: failed to create report")
		writeReportError(w, "Failed to save report", http.StatusInternalServerError)
		return
	}

	log.Info().
		Str("report_id", report.ID).
		Str("subject_uri", report.SubjectURI).
		Str("subject_did", report.SubjectDID).
		Str("reporter_did", report.ReporterDID).
		Str("reason", report.Reason).
		Msg("moderation: report created")

	// Check automod thresholds and potentially auto-hide
	h.checkAutomod(ctx, report)

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ReportResponse{
		ID:      report.ID,
		Status:  "received",
		Message: "Thank you for your report. It will be reviewed by a moderator.",
	})
}

// checkAutomod checks if automod thresholds are met and auto-hides content if needed.
func (h *Handler) checkAutomod(ctx context.Context, report moderation.Report) {
	// Skip if record is already hidden
	if h.moderationStore.IsRecordHidden(ctx, report.SubjectURI) {
		return
	}

	// Check report count for this specific URI
	uriReportCount, err := h.moderationStore.CountReportsForURI(ctx, report.SubjectURI)
	if err != nil {
		log.Error().Err(err).Str("uri", report.SubjectURI).Msg("moderation: failed to count URI reports for automod")
		return
	}

	// Check total report count for content by this user
	didReportCount, err := h.moderationStore.CountReportsForDID(ctx, report.SubjectDID)
	if err != nil {
		log.Error().Err(err).Str("did", report.SubjectDID).Msg("moderation: failed to count DID reports for automod")
		return
	}

	// Determine if we should auto-hide
	shouldAutoHide := false
	autoHideReason := ""

	if uriReportCount >= AutoHideThreshold {
		shouldAutoHide = true
		autoHideReason = fmt.Sprintf("Auto-hidden: %d reports on this record", uriReportCount)
	} else if didReportCount >= AutoHideUserThreshold {
		shouldAutoHide = true
		autoHideReason = fmt.Sprintf("Auto-hidden: %d total reports against user's content", didReportCount)
	}

	if shouldAutoHide {
		// Auto-hide the record
		hiddenRecord := moderation.HiddenRecord{
			ATURI:      report.SubjectURI,
			HiddenAt:   time.Now(),
			HiddenBy:   "automod",
			Reason:     autoHideReason,
			AutoHidden: true,
		}

		if err := h.moderationStore.HideRecord(ctx, hiddenRecord); err != nil {
			log.Error().Err(err).Str("uri", report.SubjectURI).Msg("moderation: automod failed to hide record")
			return
		}

		// Log the automod action
		auditEntry := moderation.AuditEntry{
			ID:        generateTID(),
			Action:    moderation.AuditActionHideRecord,
			ActorDID:  "automod",
			TargetURI: report.SubjectURI,
			Reason:    autoHideReason,
			Timestamp: time.Now(),
			AutoMod:   true,
		}

		if err := h.moderationStore.LogAction(ctx, auditEntry); err != nil {
			log.Error().Err(err).Msg("moderation: failed to log automod action")
		}

		log.Warn().
			Str("uri", report.SubjectURI).
			Str("did", report.SubjectDID).
			Int("uri_reports", uriReportCount).
			Int("did_reports", didReportCount).
			Str("reason", autoHideReason).
			Msg("moderation: automod triggered - record hidden")
	}
}

// writeReportError writes a JSON error response for report endpoints
func writeReportError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ReportResponse{
		Status:  "error",
		Message: message,
	})
}
