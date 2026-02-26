package moderation

import (
	"context"
	"time"
)

// Store defines the persistence interface for moderation data.
// Implementations must be safe for concurrent use.
type Store interface {
	// Hidden records
	HideRecord(ctx context.Context, entry HiddenRecord) error
	UnhideRecord(ctx context.Context, atURI string) error
	IsRecordHidden(ctx context.Context, atURI string) bool
	GetHiddenRecord(ctx context.Context, atURI string) (*HiddenRecord, error)
	ListHiddenRecords(ctx context.Context) ([]HiddenRecord, error)

	// Blacklist
	BlacklistUser(ctx context.Context, entry BlacklistedUser) error
	UnblacklistUser(ctx context.Context, did string) error
	IsBlacklisted(ctx context.Context, did string) bool
	GetBlacklistedUser(ctx context.Context, did string) (*BlacklistedUser, error)
	ListBlacklistedUsers(ctx context.Context) ([]BlacklistedUser, error)

	// Reports
	CreateReport(ctx context.Context, report Report) error
	GetReport(ctx context.Context, id string) (*Report, error)
	ListPendingReports(ctx context.Context) ([]Report, error)
	ListAllReports(ctx context.Context) ([]Report, error)
	ResolveReport(ctx context.Context, id string, status ReportStatus, resolvedBy string) error
	CountReportsForURI(ctx context.Context, atURI string) (int, error)
	CountReportsForDID(ctx context.Context, did string) (int, error)
	CountReportsForDIDSince(ctx context.Context, did string, since time.Time) (int, error)
	HasReportedURI(ctx context.Context, reporterDID, subjectURI string) (bool, error)
	CountReportsFromUserSince(ctx context.Context, reporterDID string, since time.Time) (int, error)

	// Audit log
	LogAction(ctx context.Context, entry AuditEntry) error
	ListAuditLog(ctx context.Context, limit int) ([]AuditEntry, error)

	// Auto-hide resets
	SetAutoHideReset(ctx context.Context, did string, resetAt time.Time) error
	GetAutoHideReset(ctx context.Context, did string) (time.Time, error)
}
