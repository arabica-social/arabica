package pages

import (
	"tangled.org/arabica.social/arabica/internal/backup"
	"tangled.org/arabica.social/arabica/internal/moderation"
)

// EnrichedReport wraps a report with resolved profile info. Used by the
// admin JSON handlers (buildAdminProps/enrichReports) to serialize report
// rows with their owner/reporter handles and a content summary.
type EnrichedReport struct {
	Report         moderation.Report
	OwnerHandle    string
	ReporterHandle string
	PostContent    string // Summary of the reported content
}

// AdminStats holds aggregate statistics for the admin dashboard. Populated by
// Handler.collectAdminStats and serialized by HandleAdminStatsJSON.
type AdminStats struct {
	KnownUsers          int
	RegisteredUsers     int
	IndexedRecords      int
	TotalLikes          int
	TotalComments       int
	FirehoseConnected   bool
	RecordsByCollection map[string]int
}

// AdminProps is the full admin dashboard payload assembled by buildAdminProps
// and serialized by HandleAdminJSON. The permission flags drive which
// moderation actions the admin UI may perform.
type AdminProps struct {
	HiddenRecords    []moderation.HiddenRecord
	AuditLog         []moderation.AuditEntry
	Reports          []EnrichedReport
	BlockedUsers     []moderation.BlacklistedUser
	Labels           []moderation.Label
	Stats            AdminStats
	Backups          []backup.SourceStatus
	CanHide          bool
	CanUnhide        bool
	CanViewLogs      bool
	CanViewReports   bool
	CanBlock         bool
	CanUnblock       bool
	CanResetAutoHide bool
	CanManageLabels  bool
	IsAdmin          bool
}
