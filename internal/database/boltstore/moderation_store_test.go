package boltstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"arabica/internal/moderation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestModerationStore(t *testing.T) *ModerationStore {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := Open(Options{Path: dbPath})
	require.NoError(t, err)

	t.Cleanup(func() {
		store.Close()
	})

	return store.ModerationStore()
}

func TestHiddenRecords(t *testing.T) {
	ctx := context.Background()
	store := setupTestModerationStore(t)

	t.Run("hide and check record", func(t *testing.T) {
		entry := moderation.HiddenRecord{
			ATURI:      "at://did:plc:test/app.bsky.feed.post/abc123",
			HiddenAt:   time.Now(),
			HiddenBy:   "did:plc:admin",
			Reason:     "Spam content",
			AutoHidden: false,
		}

		err := store.HideRecord(ctx, entry)
		require.NoError(t, err)

		assert.True(t, store.IsRecordHidden(ctx, entry.ATURI))
		assert.False(t, store.IsRecordHidden(ctx, "at://did:plc:other/app.bsky.feed.post/xyz"))
	})

	t.Run("get hidden record", func(t *testing.T) {
		uri := "at://did:plc:test/social.arabica.alpha.brew/get123"
		entry := moderation.HiddenRecord{
			ATURI:      uri,
			HiddenAt:   time.Now(),
			HiddenBy:   "did:plc:mod",
			Reason:     "Inappropriate",
			AutoHidden: true,
		}

		err := store.HideRecord(ctx, entry)
		require.NoError(t, err)

		retrieved, err := store.GetHiddenRecord(ctx, uri)
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		assert.Equal(t, uri, retrieved.ATURI)
		assert.Equal(t, "did:plc:mod", retrieved.HiddenBy)
		assert.Equal(t, "Inappropriate", retrieved.Reason)
		assert.True(t, retrieved.AutoHidden)
	})

	t.Run("unhide record", func(t *testing.T) {
		uri := "at://did:plc:test/social.arabica.alpha.brew/unhide123"
		entry := moderation.HiddenRecord{
			ATURI:    uri,
			HiddenAt: time.Now(),
			HiddenBy: "did:plc:admin",
		}

		err := store.HideRecord(ctx, entry)
		require.NoError(t, err)
		assert.True(t, store.IsRecordHidden(ctx, uri))

		err = store.UnhideRecord(ctx, uri)
		require.NoError(t, err)
		assert.False(t, store.IsRecordHidden(ctx, uri))
	})

	t.Run("list hidden records", func(t *testing.T) {
		// Clear by unhiding previous test records
		store.UnhideRecord(ctx, "at://did:plc:test/app.bsky.feed.post/abc123")
		store.UnhideRecord(ctx, "at://did:plc:test/social.arabica.alpha.brew/get123")

		// Add fresh records
		for i := 0; i < 3; i++ {
			entry := moderation.HiddenRecord{
				ATURI:    "at://did:plc:list/social.arabica.alpha.brew/list" + string(rune('0'+i)),
				HiddenAt: time.Now(),
				HiddenBy: "did:plc:admin",
			}
			require.NoError(t, store.HideRecord(ctx, entry))
		}

		records, err := store.ListHiddenRecords(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(records), 3)
	})
}

func TestBlacklist(t *testing.T) {
	ctx := context.Background()
	store := setupTestModerationStore(t)

	t.Run("blacklist and check user", func(t *testing.T) {
		entry := moderation.BlacklistedUser{
			DID:           "did:plc:baduser",
			BlacklistedAt: time.Now(),
			BlacklistedBy: "did:plc:admin",
			Reason:        "Repeated violations",
		}

		err := store.BlacklistUser(ctx, entry)
		require.NoError(t, err)

		assert.True(t, store.IsBlacklisted(ctx, "did:plc:baduser"))
		assert.False(t, store.IsBlacklisted(ctx, "did:plc:gooduser"))
	})

	t.Run("get blacklisted user", func(t *testing.T) {
		did := "did:plc:getblacklist"
		entry := moderation.BlacklistedUser{
			DID:           did,
			BlacklistedAt: time.Now(),
			BlacklistedBy: "did:plc:admin",
			Reason:        "Test reason",
		}

		err := store.BlacklistUser(ctx, entry)
		require.NoError(t, err)

		retrieved, err := store.GetBlacklistedUser(ctx, did)
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		assert.Equal(t, did, retrieved.DID)
		assert.Equal(t, "did:plc:admin", retrieved.BlacklistedBy)
		assert.Equal(t, "Test reason", retrieved.Reason)
	})

	t.Run("unblacklist user", func(t *testing.T) {
		did := "did:plc:unblacklist"
		entry := moderation.BlacklistedUser{
			DID:           did,
			BlacklistedAt: time.Now(),
			BlacklistedBy: "did:plc:admin",
		}

		err := store.BlacklistUser(ctx, entry)
		require.NoError(t, err)
		assert.True(t, store.IsBlacklisted(ctx, did))

		err = store.UnblacklistUser(ctx, did)
		require.NoError(t, err)
		assert.False(t, store.IsBlacklisted(ctx, did))
	})

	t.Run("list blacklisted users", func(t *testing.T) {
		users, err := store.ListBlacklistedUsers(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(users), 1)
	})
}

func TestReports(t *testing.T) {
	ctx := context.Background()
	store := setupTestModerationStore(t)

	t.Run("create and get report", func(t *testing.T) {
		report := moderation.Report{
			ID:          "report001",
			SubjectURI:  "at://did:plc:subject/social.arabica.alpha.brew/abc",
			SubjectDID:  "did:plc:subject",
			ReporterDID: "did:plc:reporter",
			Reason:      "This is spam",
			CreatedAt:   time.Now(),
			Status:      moderation.ReportStatusPending,
		}

		err := store.CreateReport(ctx, report)
		require.NoError(t, err)

		retrieved, err := store.GetReport(ctx, "report001")
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		assert.Equal(t, "report001", retrieved.ID)
		assert.Equal(t, "did:plc:reporter", retrieved.ReporterDID)
		assert.Equal(t, moderation.ReportStatusPending, retrieved.Status)
	})

	t.Run("list pending reports", func(t *testing.T) {
		// Create a mix of pending and resolved reports
		pending := moderation.Report{
			ID:          "report_pending",
			SubjectURI:  "at://did:plc:sub/social.arabica.alpha.brew/p1",
			SubjectDID:  "did:plc:sub",
			ReporterDID: "did:plc:rep1",
			Status:      moderation.ReportStatusPending,
			CreatedAt:   time.Now(),
		}
		require.NoError(t, store.CreateReport(ctx, pending))

		dismissed := moderation.Report{
			ID:          "report_dismissed",
			SubjectURI:  "at://did:plc:sub/social.arabica.alpha.brew/p2",
			SubjectDID:  "did:plc:sub",
			ReporterDID: "did:plc:rep2",
			Status:      moderation.ReportStatusDismissed,
			CreatedAt:   time.Now(),
		}
		require.NoError(t, store.CreateReport(ctx, dismissed))

		reports, err := store.ListPendingReports(ctx)
		require.NoError(t, err)

		// Should only include pending reports
		for _, r := range reports {
			assert.Equal(t, moderation.ReportStatusPending, r.Status)
		}
	})

	t.Run("resolve report", func(t *testing.T) {
		report := moderation.Report{
			ID:          "report_to_resolve",
			SubjectURI:  "at://did:plc:sub/social.arabica.alpha.brew/resolve",
			SubjectDID:  "did:plc:sub",
			ReporterDID: "did:plc:rep",
			Status:      moderation.ReportStatusPending,
			CreatedAt:   time.Now(),
		}
		require.NoError(t, store.CreateReport(ctx, report))

		err := store.ResolveReport(ctx, "report_to_resolve", moderation.ReportStatusActioned, "did:plc:mod")
		require.NoError(t, err)

		retrieved, err := store.GetReport(ctx, "report_to_resolve")
		require.NoError(t, err)

		assert.Equal(t, moderation.ReportStatusActioned, retrieved.Status)
		assert.Equal(t, "did:plc:mod", retrieved.ResolvedBy)
		assert.NotNil(t, retrieved.ResolvedAt)
	})

	t.Run("count reports for URI", func(t *testing.T) {
		uri := "at://did:plc:counted/social.arabica.alpha.brew/count"

		for i := 0; i < 3; i++ {
			report := moderation.Report{
				ID:          "count_uri_" + string(rune('0'+i)),
				SubjectURI:  uri,
				SubjectDID:  "did:plc:counted",
				ReporterDID: "did:plc:reporter" + string(rune('0'+i)),
				Status:      moderation.ReportStatusPending,
				CreatedAt:   time.Now(),
			}
			require.NoError(t, store.CreateReport(ctx, report))
		}

		count, err := store.CountReportsForURI(ctx, uri)
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("count reports for DID", func(t *testing.T) {
		did := "did:plc:counteddid"

		for i := 0; i < 2; i++ {
			report := moderation.Report{
				ID:          "count_did_" + string(rune('0'+i)),
				SubjectURI:  "at://" + did + "/social.arabica.alpha.brew/post" + string(rune('0'+i)),
				SubjectDID:  did,
				ReporterDID: "did:plc:reporter",
				Status:      moderation.ReportStatusPending,
				CreatedAt:   time.Now(),
			}
			require.NoError(t, store.CreateReport(ctx, report))
		}

		count, err := store.CountReportsForDID(ctx, did)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("has reported URI", func(t *testing.T) {
		uri := "at://did:plc:hasreported/social.arabica.alpha.brew/check"
		reporter := "did:plc:checker"

		report := moderation.Report{
			ID:          "has_reported_check",
			SubjectURI:  uri,
			SubjectDID:  "did:plc:hasreported",
			ReporterDID: reporter,
			Status:      moderation.ReportStatusPending,
			CreatedAt:   time.Now(),
		}
		require.NoError(t, store.CreateReport(ctx, report))

		has, err := store.HasReportedURI(ctx, reporter, uri)
		require.NoError(t, err)
		assert.True(t, has)

		has, err = store.HasReportedURI(ctx, "did:plc:other", uri)
		require.NoError(t, err)
		assert.False(t, has)
	})

	t.Run("count reports from user since", func(t *testing.T) {
		reporter := "did:plc:ratelimituser"
		now := time.Now()

		// Create reports at different times
		for i := 0; i < 5; i++ {
			report := moderation.Report{
				ID:          "ratelimit_" + string(rune('a'+i)),
				SubjectURI:  "at://did:plc:target/social.arabica.alpha.brew/rl" + string(rune('0'+i)),
				SubjectDID:  "did:plc:target",
				ReporterDID: reporter,
				Status:      moderation.ReportStatusPending,
				CreatedAt:   now.Add(-time.Duration(i*30) * time.Minute), // 0, -30, -60, -90, -120 mins
			}
			require.NoError(t, store.CreateReport(ctx, report))
		}

		// Count reports in the last hour (should be 2: 0min and -30min)
		oneHourAgo := now.Add(-1 * time.Hour)
		count, err := store.CountReportsFromUserSince(ctx, reporter, oneHourAgo)
		require.NoError(t, err)
		assert.Equal(t, 2, count)

		// Count reports in the last 2 hours (should be 4: 0, -30, -60, -90 mins)
		twoHoursAgo := now.Add(-2 * time.Hour)
		count, err = store.CountReportsFromUserSince(ctx, reporter, twoHoursAgo)
		require.NoError(t, err)
		assert.Equal(t, 4, count)

		// Count reports from a different user (should be 0)
		count, err = store.CountReportsFromUserSince(ctx, "did:plc:otheruser", oneHourAgo)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestAuditLog(t *testing.T) {
	ctx := context.Background()
	store := setupTestModerationStore(t)

	t.Run("log action", func(t *testing.T) {
		entry := moderation.AuditEntry{
			ID:        "audit001",
			Action:    moderation.AuditActionHideRecord,
			ActorDID:  "did:plc:mod",
			TargetURI: "at://did:plc:target/social.arabica.alpha.brew/abc",
			Reason:    "Spam",
			Timestamp: time.Now(),
			AutoMod:   false,
		}

		err := store.LogAction(ctx, entry)
		require.NoError(t, err)
	})

	t.Run("list audit log", func(t *testing.T) {
		// Add several entries with different timestamps
		now := time.Now()
		for i := 0; i < 5; i++ {
			entry := moderation.AuditEntry{
				ID:        "audit_list_" + string(rune('0'+i)),
				Action:    moderation.AuditActionHideRecord,
				ActorDID:  "did:plc:mod",
				TargetURI: "at://did:plc:target/social.arabica.alpha.brew/" + string(rune('0'+i)),
				Timestamp: now.Add(time.Duration(i) * time.Second),
			}
			require.NoError(t, store.LogAction(ctx, entry))
		}

		entries, err := store.ListAuditLog(ctx, 3)
		require.NoError(t, err)
		assert.Len(t, entries, 3)

		// Should be in reverse chronological order (newest first)
		for i := 1; i < len(entries); i++ {
			assert.True(t, entries[i-1].Timestamp.After(entries[i].Timestamp) ||
				entries[i-1].Timestamp.Equal(entries[i].Timestamp))
		}
	})

	t.Run("automod entry", func(t *testing.T) {
		entry := moderation.AuditEntry{
			ID:        "audit_automod",
			Action:    moderation.AuditActionHideRecord,
			ActorDID:  "automod",
			TargetURI: "at://did:plc:auto/social.arabica.alpha.brew/auto",
			Reason:    "Exceeded report threshold",
			Timestamp: time.Now(),
			AutoMod:   true,
		}

		err := store.LogAction(ctx, entry)
		require.NoError(t, err)

		entries, err := store.ListAuditLog(ctx, 100)
		require.NoError(t, err)

		var found bool
		for _, e := range entries {
			if e.ID == "audit_automod" {
				assert.True(t, e.AutoMod)
				found = true
				break
			}
		}
		assert.True(t, found, "automod entry not found")
	})
}

func TestNonExistentRecords(t *testing.T) {
	ctx := context.Background()
	store := setupTestModerationStore(t)

	t.Run("get nonexistent hidden record", func(t *testing.T) {
		record, err := store.GetHiddenRecord(ctx, "at://nonexistent")
		require.NoError(t, err)
		assert.Nil(t, record)
	})

	t.Run("get nonexistent blacklisted user", func(t *testing.T) {
		user, err := store.GetBlacklistedUser(ctx, "did:plc:nonexistent")
		require.NoError(t, err)
		assert.Nil(t, user)
	})

	t.Run("get nonexistent report", func(t *testing.T) {
		report, err := store.GetReport(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, report)
	})

	t.Run("resolve nonexistent report", func(t *testing.T) {
		err := store.ResolveReport(ctx, "nonexistent", moderation.ReportStatusDismissed, "did:plc:mod")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}
