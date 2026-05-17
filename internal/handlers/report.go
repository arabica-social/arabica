package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"tangled.org/arabica.social/arabica/internal/metrics"
	"tangled.org/arabica.social/arabica/internal/moderation"
	"tangled.org/arabica.social/arabica/internal/web/components"
	"tangled.org/pdewey.com/atp"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

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

// HandleReport handles content report submissions from the in-app dialog.
// Renders HTML partials for htmx: the form re-rendered with an inline error,
// or a success partial plus an HX-Trigger that toasts and auto-closes the
// dialog.
func (h *Handler) HandleReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		writeReportError(ctx, w, "", "", "", "", "Invalid form data")
		return
	}
	subjectURI := r.FormValue("subject_uri")
	subjectCID := r.FormValue("subject_cid")
	rawReason := r.FormValue("reason")
	dialogID := r.FormValue("dialog_id")

	reporterDID, ok := atpmiddleware.GetDID(ctx)
	if !ok {
		writeReportError(ctx, w, dialogID, subjectURI, subjectCID, rawReason, "Authentication required")
		return
	}

	if h.moderationStore == nil {
		log.Error().Msg("moderation: store not configured")
		writeReportError(ctx, w, dialogID, subjectURI, subjectCID, rawReason, "Reports are not enabled")
		return
	}

	if subjectURI == "" {
		writeReportError(ctx, w, dialogID, subjectURI, subjectCID, rawReason, "subject_uri is required")
		return
	}

	uriParts, err := atp.ParseATURI(subjectURI)
	if err != nil {
		writeReportError(ctx, w, dialogID, subjectURI, subjectCID, rawReason, "Invalid subject_uri format")
		return
	}
	subjectDID := uriParts.DID

	if subjectDID == reporterDID {
		writeReportError(ctx, w, dialogID, subjectURI, subjectCID, rawReason, "You cannot report your own content")
		return
	}

	reason := strings.TrimSpace(rawReason)
	if reason == "" {
		reason = "No reason provided"
	}
	if len(reason) > MaxReportReasonLength {
		reason = reason[:MaxReportReasonLength]
	}

	oneHourAgo := time.Now().Add(-1 * time.Hour)
	recentCount, err := h.moderationStore.CountReportsFromUserSince(ctx, reporterDID, oneHourAgo)
	if err != nil {
		log.Error().Err(err).Str("reporter", reporterDID).Msg("moderation: failed to check rate limit")
		writeReportError(ctx, w, dialogID, subjectURI, subjectCID, rawReason, "Failed to process report")
		return
	}
	if recentCount >= ReportRateLimitPerHour {
		writeReportError(ctx, w, dialogID, subjectURI, subjectCID, rawReason, "Rate limit exceeded. Please try again later.")
		return
	}

	alreadyReported, err := h.moderationStore.HasReportedURI(ctx, reporterDID, subjectURI)
	if err != nil {
		log.Error().Err(err).Str("reporter", reporterDID).Msg("moderation: failed to check duplicate")
		writeReportError(ctx, w, dialogID, subjectURI, subjectCID, rawReason, "Failed to process report")
		return
	}
	if alreadyReported {
		writeReportError(ctx, w, dialogID, subjectURI, subjectCID, rawReason, "You have already reported this content")
		return
	}

	report := moderation.Report{
		ID:          generateTID(),
		SubjectURI:  subjectURI,
		SubjectDID:  subjectDID,
		ReporterDID: reporterDID,
		Reason:      reason,
		CreatedAt:   time.Now(),
		Status:      moderation.ReportStatusPending,
	}

	if err := h.moderationStore.CreateReport(ctx, report); err != nil {
		log.Error().Err(err).Str("reporter", reporterDID).Msg("moderation: failed to create report")
		writeReportError(ctx, w, dialogID, subjectURI, subjectCID, rawReason, "Failed to save report")
		return
	}

	metrics.ReportsTotal.Inc()

	log.Info().
		Str("report_id", report.ID).
		Str("subject_uri", report.SubjectURI).
		Str("subject_did", report.SubjectDID).
		Str("reporter_did", report.ReporterDID).
		Str("reason", report.Reason).
		Msg("moderation: report created")

	h.checkAutomod(ctx, report)

	// Toast + delayed close. The success partial stays visible for the delay
	// so the user sees the confirmation before the dialog disappears.
	trigger := fmt.Sprintf(
		`{"notify":{"message":"Report submitted"},"close-dialog":{"id":%q,"delay":2000}}`,
		dialogID,
	)
	w.Header().Set("HX-Trigger", trigger)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := components.ReportSuccess().Render(ctx, w); err != nil {
		log.Error().Err(err).Msg("moderation: failed to render report success partial")
	}
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

	// Check total report count for content by this user (respecting any reset)
	var didReportCount int
	resetAt, err := h.moderationStore.GetAutoHideReset(ctx, report.SubjectDID)
	if err != nil {
		log.Error().Err(err).Str("did", report.SubjectDID).Msg("moderation: failed to get auto-hide reset for automod")
		return
	}
	if !resetAt.IsZero() {
		didReportCount, err = h.moderationStore.CountReportsForDIDSince(ctx, report.SubjectDID, resetAt)
	} else {
		didReportCount, err = h.moderationStore.CountReportsForDID(ctx, report.SubjectDID)
	}
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

// writeReportError re-renders the report form with an inline error message.
// Always returns 200 OK — htmx ignores 4xx/5xx by default and won't swap. The
// reason field is preserved so the user doesn't lose their typing.
func writeReportError(ctx context.Context, w http.ResponseWriter, dialogID, subjectURI, subjectCID, reason, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = components.ReportForm(components.ReportFormProps{
		DialogID:     dialogID,
		SubjectURI:   subjectURI,
		SubjectCID:   subjectCID,
		Reason:       reason,
		ErrorMessage: message,
	}).Render(ctx, w)
}
