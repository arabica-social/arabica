package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"tangled.org/arabica.social/arabica/internal/moderation"

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

// HandleReport handles content report submissions. The SPA always sends
// Accept: application/json, so this delegates to the JSON handler which
// shares the same validation and automod logic.
func (h *Handler) HandleReport(w http.ResponseWriter, r *http.Request) {
	h.HandleReportJSON(w, r)
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
