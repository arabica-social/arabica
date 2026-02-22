package boltstore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"arabica/internal/moderation"

	bolt "go.etcd.io/bbolt"
)

// ModerationStore provides persistent storage for moderation data.
type ModerationStore struct {
	db *bolt.DB
}

// HideRecord stores a hidden record entry.
func (s *ModerationStore) HideRecord(ctx context.Context, entry moderation.HiddenRecord) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketModerationHiddenRecords)
		if bucket == nil {
			return fmt.Errorf("bucket not found: %s", BucketModerationHiddenRecords)
		}

		data, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal hidden record: %w", err)
		}

		return bucket.Put([]byte(entry.ATURI), data)
	})
}

// UnhideRecord removes a record from the hidden list.
func (s *ModerationStore) UnhideRecord(ctx context.Context, atURI string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketModerationHiddenRecords)
		if bucket == nil {
			return nil
		}

		return bucket.Delete([]byte(atURI))
	})
}

// IsRecordHidden checks if a record is hidden.
func (s *ModerationStore) IsRecordHidden(ctx context.Context, atURI string) bool {
	var hidden bool

	s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketModerationHiddenRecords)
		if bucket == nil {
			return nil
		}

		hidden = bucket.Get([]byte(atURI)) != nil
		return nil
	})

	return hidden
}

// GetHiddenRecord retrieves a hidden record entry by AT-URI.
func (s *ModerationStore) GetHiddenRecord(ctx context.Context, atURI string) (*moderation.HiddenRecord, error) {
	var record *moderation.HiddenRecord

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketModerationHiddenRecords)
		if bucket == nil {
			return nil
		}

		data := bucket.Get([]byte(atURI))
		if data == nil {
			return nil
		}

		record = &moderation.HiddenRecord{}
		return json.Unmarshal(data, record)
	})

	return record, err
}

// ListHiddenRecords returns all hidden records.
func (s *ModerationStore) ListHiddenRecords(ctx context.Context) ([]moderation.HiddenRecord, error) {
	var records []moderation.HiddenRecord

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketModerationHiddenRecords)
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			var record moderation.HiddenRecord
			if err := json.Unmarshal(v, &record); err != nil {
				return err
			}
			records = append(records, record)
			return nil
		})
	})

	return records, err
}

// BlacklistUser adds a user to the blacklist.
func (s *ModerationStore) BlacklistUser(ctx context.Context, entry moderation.BlacklistedUser) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketModerationBlacklist)
		if bucket == nil {
			return fmt.Errorf("bucket not found: %s", BucketModerationBlacklist)
		}

		data, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal blacklisted user: %w", err)
		}

		return bucket.Put([]byte(entry.DID), data)
	})
}

// UnblacklistUser removes a user from the blacklist.
func (s *ModerationStore) UnblacklistUser(ctx context.Context, did string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketModerationBlacklist)
		if bucket == nil {
			return nil
		}

		return bucket.Delete([]byte(did))
	})
}

// IsBlacklisted checks if a user is blacklisted.
func (s *ModerationStore) IsBlacklisted(ctx context.Context, did string) bool {
	var blacklisted bool

	s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketModerationBlacklist)
		if bucket == nil {
			return nil
		}

		blacklisted = bucket.Get([]byte(did)) != nil
		return nil
	})

	return blacklisted
}

// GetBlacklistedUser retrieves a blacklisted user entry by DID.
func (s *ModerationStore) GetBlacklistedUser(ctx context.Context, did string) (*moderation.BlacklistedUser, error) {
	var user *moderation.BlacklistedUser

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketModerationBlacklist)
		if bucket == nil {
			return nil
		}

		data := bucket.Get([]byte(did))
		if data == nil {
			return nil
		}

		user = &moderation.BlacklistedUser{}
		return json.Unmarshal(data, user)
	})

	return user, err
}

// ListBlacklistedUsers returns all blacklisted users.
func (s *ModerationStore) ListBlacklistedUsers(ctx context.Context) ([]moderation.BlacklistedUser, error) {
	var users []moderation.BlacklistedUser

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketModerationBlacklist)
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			var user moderation.BlacklistedUser
			if err := json.Unmarshal(v, &user); err != nil {
				return err
			}
			users = append(users, user)
			return nil
		})
	})

	return users, err
}

// CreateReport stores a new report.
func (s *ModerationStore) CreateReport(ctx context.Context, report moderation.Report) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		// Store the report
		bucket := tx.Bucket(BucketModerationReports)
		if bucket == nil {
			return fmt.Errorf("bucket not found: %s", BucketModerationReports)
		}

		data, err := json.Marshal(report)
		if err != nil {
			return fmt.Errorf("failed to marshal report: %w", err)
		}

		if err := bucket.Put([]byte(report.ID), data); err != nil {
			return err
		}

		// Index by subject URI
		uriIndex := tx.Bucket(BucketModerationReportsByURI)
		if uriIndex != nil {
			// Store report ID in a list for this URI
			key := []byte(report.SubjectURI + ":" + report.ID)
			if err := uriIndex.Put(key, []byte(report.ID)); err != nil {
				return err
			}
		}

		// Index by subject DID
		didIndex := tx.Bucket(BucketModerationReportsByDID)
		if didIndex != nil {
			// Store report ID in a list for this DID
			key := []byte(report.SubjectDID + ":" + report.ID)
			if err := didIndex.Put(key, []byte(report.ID)); err != nil {
				return err
			}
		}

		return nil
	})
}

// GetReport retrieves a report by ID.
func (s *ModerationStore) GetReport(ctx context.Context, id string) (*moderation.Report, error) {
	var report *moderation.Report

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketModerationReports)
		if bucket == nil {
			return nil
		}

		data := bucket.Get([]byte(id))
		if data == nil {
			return nil
		}

		report = &moderation.Report{}
		return json.Unmarshal(data, report)
	})

	return report, err
}

// ListPendingReports returns all reports with pending status.
func (s *ModerationStore) ListPendingReports(ctx context.Context) ([]moderation.Report, error) {
	var reports []moderation.Report

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketModerationReports)
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			var report moderation.Report
			if err := json.Unmarshal(v, &report); err != nil {
				return err
			}
			if report.Status == moderation.ReportStatusPending {
				reports = append(reports, report)
			}
			return nil
		})
	})

	return reports, err
}

// ListAllReports returns all reports regardless of status.
func (s *ModerationStore) ListAllReports(ctx context.Context) ([]moderation.Report, error) {
	var reports []moderation.Report

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketModerationReports)
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			var report moderation.Report
			if err := json.Unmarshal(v, &report); err != nil {
				return err
			}
			reports = append(reports, report)
			return nil
		})
	})

	return reports, err
}

// ResolveReport updates a report's status and resolution info.
func (s *ModerationStore) ResolveReport(ctx context.Context, id string, status moderation.ReportStatus, resolvedBy string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketModerationReports)
		if bucket == nil {
			return fmt.Errorf("bucket not found: %s", BucketModerationReports)
		}

		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("report not found: %s", id)
		}

		var report moderation.Report
		if err := json.Unmarshal(data, &report); err != nil {
			return err
		}

		report.Status = status
		report.ResolvedBy = resolvedBy
		now := time.Now()
		report.ResolvedAt = &now

		newData, err := json.Marshal(report)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(id), newData)
	})
}

// CountReportsForURI returns the number of reports for a given AT-URI.
func (s *ModerationStore) CountReportsForURI(ctx context.Context, atURI string) (int, error) {
	var count int

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketModerationReportsByURI)
		if bucket == nil {
			return nil
		}

		cursor := bucket.Cursor()
		prefix := []byte(atURI + ":")

		for k, _ := cursor.Seek(prefix); k != nil && hasPrefix(k, prefix); k, _ = cursor.Next() {
			count++
		}

		return nil
	})

	return count, err
}

// CountReportsForDID returns the number of reports for content by a given DID.
func (s *ModerationStore) CountReportsForDID(ctx context.Context, did string) (int, error) {
	var count int

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketModerationReportsByDID)
		if bucket == nil {
			return nil
		}

		cursor := bucket.Cursor()
		prefix := []byte(did + ":")

		for k, _ := cursor.Seek(prefix); k != nil && hasPrefix(k, prefix); k, _ = cursor.Next() {
			count++
		}

		return nil
	})

	return count, err
}

// HasReportedURI checks if a user has already reported a specific URI.
func (s *ModerationStore) HasReportedURI(ctx context.Context, reporterDID, subjectURI string) (bool, error) {
	var found bool

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketModerationReports)
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			var report moderation.Report
			if err := json.Unmarshal(v, &report); err != nil {
				return nil // Skip malformed entries
			}
			if report.ReporterDID == reporterDID && report.SubjectURI == subjectURI {
				found = true
			}
			return nil
		})
	})

	return found, err
}

// LogAction stores a moderation action in the audit log.
func (s *ModerationStore) LogAction(ctx context.Context, entry moderation.AuditEntry) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketModerationAuditLog)
		if bucket == nil {
			return fmt.Errorf("bucket not found: %s", BucketModerationAuditLog)
		}

		data, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal audit entry: %w", err)
		}

		// Use timestamp-based key for chronological ordering
		// Format: timestamp:id for uniqueness
		key := fmt.Sprintf("%d:%s", entry.Timestamp.UnixNano(), entry.ID)

		return bucket.Put([]byte(key), data)
	})
}

// ListAuditLog returns the most recent audit log entries.
// Entries are returned in reverse chronological order (newest first).
func (s *ModerationStore) ListAuditLog(ctx context.Context, limit int) ([]moderation.AuditEntry, error) {
	var entries []moderation.AuditEntry

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketModerationAuditLog)
		if bucket == nil {
			return nil
		}

		// Collect all entries first (BoltDB cursors iterate in key order)
		var all []moderation.AuditEntry
		err := bucket.ForEach(func(k, v []byte) error {
			var entry moderation.AuditEntry
			if err := json.Unmarshal(v, &entry); err != nil {
				return nil // Skip malformed entries
			}
			all = append(all, entry)
			return nil
		})
		if err != nil {
			return err
		}

		// Reverse to get newest first
		for i := len(all) - 1; i >= 0 && len(entries) < limit; i-- {
			entries = append(entries, all[i])
		}

		return nil
	})

	return entries, err
}

// CountReportsFromUserSince counts reports submitted by a user since a given time.
// Used for rate limiting report submissions.
func (s *ModerationStore) CountReportsFromUserSince(ctx context.Context, reporterDID string, since time.Time) (int, error) {
	var count int

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketModerationReports)
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			var report moderation.Report
			if err := json.Unmarshal(v, &report); err != nil {
				return nil // Skip malformed entries
			}
			if report.ReporterDID == reporterDID && report.CreatedAt.After(since) {
				count++
			}
			return nil
		})
	})

	return count, err
}

// SetAutoHideReset stores a reset timestamp for a user's auto-hide counter.
// Reports created before this timestamp are ignored when checking the per-user auto-hide threshold.
func (s *ModerationStore) SetAutoHideReset(ctx context.Context, did string, resetAt time.Time) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketModerationAutoHideResets)
		if bucket == nil {
			return fmt.Errorf("bucket not found: %s", BucketModerationAutoHideResets)
		}

		data, err := resetAt.MarshalBinary()
		if err != nil {
			return fmt.Errorf("failed to marshal reset time: %w", err)
		}

		return bucket.Put([]byte(did), data)
	})
}

// GetAutoHideReset returns the auto-hide reset timestamp for a user, or zero time if none set.
func (s *ModerationStore) GetAutoHideReset(ctx context.Context, did string) (time.Time, error) {
	var resetAt time.Time

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketModerationAutoHideResets)
		if bucket == nil {
			return nil
		}

		data := bucket.Get([]byte(did))
		if data == nil {
			return nil
		}

		return resetAt.UnmarshalBinary(data)
	})

	return resetAt, err
}

// CountReportsForDIDSince returns the number of reports for content by a given DID
// created after the specified time.
func (s *ModerationStore) CountReportsForDIDSince(ctx context.Context, did string, since time.Time) (int, error) {
	var count int

	err := s.db.View(func(tx *bolt.Tx) error {
		didIndex := tx.Bucket(BucketModerationReportsByDID)
		if didIndex == nil {
			return nil
		}

		reportsBucket := tx.Bucket(BucketModerationReports)
		if reportsBucket == nil {
			return nil
		}

		cursor := didIndex.Cursor()
		prefix := []byte(did + ":")

		for k, v := cursor.Seek(prefix); k != nil && hasPrefix(k, prefix); k, v = cursor.Next() {
			// v is the report ID
			reportData := reportsBucket.Get(v)
			if reportData == nil {
				continue
			}

			var report moderation.Report
			if err := json.Unmarshal(reportData, &report); err != nil {
				continue
			}

			if report.CreatedAt.After(since) {
				count++
			}
		}

		return nil
	})

	return count, err
}

// hasPrefix checks if a byte slice has a given prefix.
func hasPrefix(s, prefix []byte) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i, b := range prefix {
		if s[i] != b {
			return false
		}
	}
	return true
}
