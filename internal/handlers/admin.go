package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/backup"
	"tangled.org/arabica.social/arabica/internal/metrics"
	"tangled.org/arabica.social/arabica/internal/moderation"
	sharedpages "tangled.org/arabica.social/arabica/internal/web/pages"
	"tangled.org/pdewey.com/atp"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

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

// blockRequest is the request body for blocking a user
type blockRequest struct {
	DID    string `json:"did"`
	Reason string `json:"reason,omitempty"`
}

// generateTID generates a TID (timestamp-based identifier) using the AT Protocol TID format.
func generateTID() string {
	return syntax.NewTIDNow(0).String()
}

// buildAdminProps builds the admin dashboard props for the given moderator.
func (h *Handler) buildAdminProps(ctx context.Context, userDID string) sharedpages.AdminProps {
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
	var enrichedReports []sharedpages.EnrichedReport
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

	// Build stats for admin users
	var stats sharedpages.AdminStats
	var backups []backup.SourceStatus
	if isAdmin {
		stats = h.collectAdminStats(ctx)
		if h.backupService != nil {
			backups = h.backupService.Status()
		}
	}

	// Normalize nil slices/maps to empty so JSON serialization emits []/{} instead
	// of null. The Svelte admin page accesses .length / Object.keys() directly
	// on these fields, which throws on null.
	if hiddenRecords == nil {
		hiddenRecords = []moderation.HiddenRecord{}
	}
	if auditLog == nil {
		auditLog = []moderation.AuditEntry{}
	}
	if enrichedReports == nil {
		enrichedReports = []sharedpages.EnrichedReport{}
	}
	if blockedUsers == nil {
		blockedUsers = []moderation.BlacklistedUser{}
	}
	if labels == nil {
		labels = []moderation.Label{}
	}
	if backups == nil {
		backups = []backup.SourceStatus{}
	}
	if stats.RecordsByCollection == nil {
		stats.RecordsByCollection = map[string]int{}
	}

	return sharedpages.AdminProps{
		HiddenRecords:    hiddenRecords,
		AuditLog:         auditLog,
		Reports:          enrichedReports,
		BlockedUsers:     blockedUsers,
		Labels:           labels,
		Stats:            stats,
		Backups:          backups,
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

// enrichReports resolves handles and fetches post content for reports
func (h *Handler) enrichReports(ctx context.Context, reports []moderation.Report) []sharedpages.EnrichedReport {
	if len(reports) == 0 {
		return nil
	}

	publicClient := atproto.NewPublicClient()
	enriched := make([]sharedpages.EnrichedReport, 0, len(reports))

	for _, report := range reports {
		er := sharedpages.EnrichedReport{
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
func (h *Handler) getPostContentSummary(ctx context.Context, publicClient *atp.PublicClient, atURI string) string {
	// Parse AT-URI to get DID, collection, and rkey
	uriParts, err := atp.ParseATURI(atURI)
	if err != nil {
		return ""
	}

	// Fetch the record
	record, err := publicClient.GetPublicRecord(ctx, uriParts.DID, uriParts.Collection, uriParts.RKey)
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

// Moderation action handlers. The SPA always sends Accept: application/json,
// so each delegates to its JSON counterpart. Auth and permission checks are
// handled by RequirePermission middleware in routing.go.

func (h *Handler) HandleHideRecord(w http.ResponseWriter, r *http.Request) {
	h.HandleHideRecordJSON(w, r)
}

func (h *Handler) HandleUnhideRecord(w http.ResponseWriter, r *http.Request) {
	h.HandleUnhideRecordJSON(w, r)
}

func (h *Handler) HandleBlockUser(w http.ResponseWriter, r *http.Request) {
	h.HandleBlockUserJSON(w, r)
}

func (h *Handler) HandleUnblockUser(w http.ResponseWriter, r *http.Request) {
	h.HandleUnblockUserJSON(w, r)
}

func (h *Handler) HandleResetAutoHide(w http.ResponseWriter, r *http.Request) {
	h.HandleResetAutoHideJSON(w, r)
}

func (h *Handler) HandleDismissReport(w http.ResponseWriter, r *http.Request) {
	h.HandleDismissReportJSON(w, r)
}

func (h *Handler) HandleAddLabel(w http.ResponseWriter, r *http.Request) {
	h.HandleAddLabelJSON(w, r)
}

func (h *Handler) HandleRemoveLabel(w http.ResponseWriter, r *http.Request) {
	h.HandleRemoveLabelJSON(w, r)
}

// collectAdminStats gathers current system statistics from available data sources.
func (h *Handler) collectAdminStats(ctx context.Context) sharedpages.AdminStats {
	var stats sharedpages.AdminStats

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

	filename := fmt.Sprintf("%s-witness-%s-%s.json",
		h.appName(),
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
	actorInput := strings.TrimSpace(r.URL.Query().Get("did"))
	if actorInput == "" {
		if err := r.ParseForm(); err == nil {
			actorInput = strings.TrimSpace(r.FormValue("did"))
		}
	}
	if actorInput == "" {
		http.Error(w, "missing 'did' parameter", http.StatusBadRequest)
		return
	}
	if h.feedIndex == nil {
		http.Error(w, "feed index not configured", http.StatusServiceUnavailable)
		return
	}

	var didStr, handle string
	if strings.HasPrefix(actorInput, "did:") {
		did, err := syntax.ParseDID(actorInput)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid DID: %v", err), http.StatusBadRequest)
			return
		}
		didStr = did.String()
	} else {
		handle = atp.NormalizeHandle(actorInput)
		if handle == "" {
			http.Error(w, "invalid handle", http.StatusBadRequest)
			return
		}
		resolved, err := atproto.NewPublicClient().ResolveHandle(r.Context(), handle)
		if err != nil {
			log.Warn().Err(err).Str("handle", handle).Msg("admin rebuild: ResolveHandle failed")
			http.Error(w, fmt.Sprintf("could not resolve handle %q: %v", handle, err), http.StatusNotFound)
			return
		}
		didStr = resolved
	}

	actor, _ := atpmiddleware.GetDID(r.Context())

	if err := h.feedIndex.BackfillUser(r.Context(), didStr, nil); err != nil {
		log.Error().Err(err).Str("did", didStr).Str("handle", handle).Str("actor", actor).Msg("admin rebuild: BackfillUser failed")
		http.Error(w, "rebuild failed", http.StatusInternalServerError)
		return
	}

	log.Warn().Str("did", didStr).Str("handle", handle).Str("actor", actor).Msg("admin rebuild: refilled witness cache from PDS")

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"did":       didStr,
		"handle":    handle,
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

	filename := fmt.Sprintf("%s-pds-%s-%s.json",
		h.appName(),
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
