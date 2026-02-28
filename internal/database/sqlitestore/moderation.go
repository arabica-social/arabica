// Package sqlitestore provides SQLite-backed store implementations.
package sqlitestore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"arabica/internal/moderation"
)

// ModerationStore implements moderation.Store using SQLite.
// It shares the database connection with the firehose FeedIndex.
type ModerationStore struct {
	db *sql.DB
}

// NewModerationStore creates a ModerationStore backed by the given database.
// The database must already have the moderation schema applied.
func NewModerationStore(db *sql.DB) *ModerationStore {
	return &ModerationStore{db: db}
}

// Ensure ModerationStore implements the interface at compile time.
var _ moderation.Store = (*ModerationStore)(nil)

// ========== Hidden Records ==========

func (s *ModerationStore) HideRecord(ctx context.Context, entry moderation.HiddenRecord) error {
	autoHidden := 0
	if entry.AutoHidden {
		autoHidden = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO moderation_hidden_records (uri, hidden_at, hidden_by, reason, auto_hidden)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(uri) DO UPDATE SET
			hidden_at   = excluded.hidden_at,
			hidden_by   = excluded.hidden_by,
			reason      = excluded.reason,
			auto_hidden = excluded.auto_hidden
	`, entry.ATURI, entry.HiddenAt.Format(time.RFC3339Nano), entry.HiddenBy, entry.Reason, autoHidden)
	if err != nil {
		return fmt.Errorf("hide record: %w", err)
	}
	return nil
}

func (s *ModerationStore) UnhideRecord(ctx context.Context, atURI string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM moderation_hidden_records WHERE uri = ?`, atURI)
	return err
}

func (s *ModerationStore) IsRecordHidden(ctx context.Context, atURI string) bool {
	var exists int
	_ = s.db.QueryRowContext(ctx, `SELECT 1 FROM moderation_hidden_records WHERE uri = ?`, atURI).Scan(&exists)
	return exists == 1
}

func (s *ModerationStore) GetHiddenRecord(ctx context.Context, atURI string) (*moderation.HiddenRecord, error) {
	var r moderation.HiddenRecord
	var hiddenAtStr string
	var autoHidden int
	err := s.db.QueryRowContext(ctx, `
		SELECT uri, hidden_at, hidden_by, reason, auto_hidden
		FROM moderation_hidden_records WHERE uri = ?
	`, atURI).Scan(&r.ATURI, &hiddenAtStr, &r.HiddenBy, &r.Reason, &autoHidden)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.HiddenAt, _ = time.Parse(time.RFC3339Nano, hiddenAtStr)
	r.AutoHidden = autoHidden == 1
	return &r, nil
}

func (s *ModerationStore) ListHiddenRecords(ctx context.Context) ([]moderation.HiddenRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT uri, hidden_at, hidden_by, reason, auto_hidden
		FROM moderation_hidden_records ORDER BY hidden_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []moderation.HiddenRecord
	for rows.Next() {
		var r moderation.HiddenRecord
		var hiddenAtStr string
		var autoHidden int
		if err := rows.Scan(&r.ATURI, &hiddenAtStr, &r.HiddenBy, &r.Reason, &autoHidden); err != nil {
			continue
		}
		r.HiddenAt, _ = time.Parse(time.RFC3339Nano, hiddenAtStr)
		r.AutoHidden = autoHidden == 1
		records = append(records, r)
	}
	return records, rows.Err()
}

// ========== Blacklist ==========

func (s *ModerationStore) BlacklistUser(ctx context.Context, entry moderation.BlacklistedUser) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO moderation_blacklist (did, blacklisted_at, blacklisted_by, reason)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(did) DO UPDATE SET
			blacklisted_at = excluded.blacklisted_at,
			blacklisted_by = excluded.blacklisted_by,
			reason         = excluded.reason
	`, entry.DID, entry.BlacklistedAt.Format(time.RFC3339Nano), entry.BlacklistedBy, entry.Reason)
	if err != nil {
		return fmt.Errorf("blacklist user: %w", err)
	}
	return nil
}

func (s *ModerationStore) UnblacklistUser(ctx context.Context, did string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM moderation_blacklist WHERE did = ?`, did)
	return err
}

func (s *ModerationStore) IsBlacklisted(ctx context.Context, did string) bool {
	var exists int
	_ = s.db.QueryRowContext(ctx, `SELECT 1 FROM moderation_blacklist WHERE did = ?`, did).Scan(&exists)
	return exists == 1
}

func (s *ModerationStore) GetBlacklistedUser(ctx context.Context, did string) (*moderation.BlacklistedUser, error) {
	var u moderation.BlacklistedUser
	var blacklistedAtStr string
	err := s.db.QueryRowContext(ctx, `
		SELECT did, blacklisted_at, blacklisted_by, reason
		FROM moderation_blacklist WHERE did = ?
	`, did).Scan(&u.DID, &blacklistedAtStr, &u.BlacklistedBy, &u.Reason)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.BlacklistedAt, _ = time.Parse(time.RFC3339Nano, blacklistedAtStr)
	return &u, nil
}

func (s *ModerationStore) ListBlacklistedUsers(ctx context.Context) ([]moderation.BlacklistedUser, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT did, blacklisted_at, blacklisted_by, reason
		FROM moderation_blacklist ORDER BY blacklisted_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []moderation.BlacklistedUser
	for rows.Next() {
		var u moderation.BlacklistedUser
		var blacklistedAtStr string
		if err := rows.Scan(&u.DID, &blacklistedAtStr, &u.BlacklistedBy, &u.Reason); err != nil {
			continue
		}
		u.BlacklistedAt, _ = time.Parse(time.RFC3339Nano, blacklistedAtStr)
		users = append(users, u)
	}
	return users, rows.Err()
}

// ========== Reports ==========

func (s *ModerationStore) CreateReport(ctx context.Context, report moderation.Report) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO moderation_reports
			(id, subject_uri, subject_did, reporter_did, reason, created_at, status, resolved_by, resolved_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, report.ID, report.SubjectURI, report.SubjectDID, report.ReporterDID, report.Reason,
		report.CreatedAt.Format(time.RFC3339Nano), string(report.Status), report.ResolvedBy, nil)
	if err != nil {
		return fmt.Errorf("create report: %w", err)
	}
	return nil
}

func (s *ModerationStore) GetReport(ctx context.Context, id string) (*moderation.Report, error) {
	var r moderation.Report
	var createdAtStr string
	var resolvedAtStr sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, subject_uri, subject_did, reporter_did, reason, created_at, status, resolved_by, resolved_at
		FROM moderation_reports WHERE id = ?
	`, id).Scan(&r.ID, &r.SubjectURI, &r.SubjectDID, &r.ReporterDID, &r.Reason,
		&createdAtStr, &r.Status, &r.ResolvedBy, &resolvedAtStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
	if resolvedAtStr.Valid {
		t, _ := time.Parse(time.RFC3339Nano, resolvedAtStr.String)
		r.ResolvedAt = &t
	}
	return &r, nil
}

func (s *ModerationStore) ListPendingReports(ctx context.Context) ([]moderation.Report, error) {
	return s.listReports(ctx, `WHERE status = 'pending' ORDER BY created_at DESC`)
}

func (s *ModerationStore) ListAllReports(ctx context.Context) ([]moderation.Report, error) {
	return s.listReports(ctx, `ORDER BY created_at DESC`)
}

func (s *ModerationStore) listReports(ctx context.Context, clause string) ([]moderation.Report, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, subject_uri, subject_did, reporter_did, reason, created_at, status, resolved_by, resolved_at
		FROM moderation_reports `+clause)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReports(rows)
}

func scanReports(rows *sql.Rows) ([]moderation.Report, error) {
	var reports []moderation.Report
	for rows.Next() {
		var r moderation.Report
		var createdAtStr string
		var resolvedAtStr sql.NullString
		if err := rows.Scan(&r.ID, &r.SubjectURI, &r.SubjectDID, &r.ReporterDID, &r.Reason,
			&createdAtStr, &r.Status, &r.ResolvedBy, &resolvedAtStr); err != nil {
			continue
		}
		r.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
		if resolvedAtStr.Valid {
			t, _ := time.Parse(time.RFC3339Nano, resolvedAtStr.String)
			r.ResolvedAt = &t
		}
		reports = append(reports, r)
	}
	return reports, rows.Err()
}

func (s *ModerationStore) ResolveReport(ctx context.Context, id string, status moderation.ReportStatus, resolvedBy string) error {
	now := time.Now().Format(time.RFC3339Nano)
	res, err := s.db.ExecContext(ctx, `
		UPDATE moderation_reports SET status = ?, resolved_by = ?, resolved_at = ? WHERE id = ?
	`, string(status), resolvedBy, now, id)
	if err != nil {
		return fmt.Errorf("resolve report: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("report not found: %s", id)
	}
	return nil
}

func (s *ModerationStore) CountReportsForURI(ctx context.Context, atURI string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM moderation_reports WHERE subject_uri = ?`, atURI).Scan(&count)
	return count, err
}

func (s *ModerationStore) CountReportsForDID(ctx context.Context, did string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM moderation_reports WHERE subject_did = ?`, did).Scan(&count)
	return count, err
}

func (s *ModerationStore) CountReportsForDIDSince(ctx context.Context, did string, since time.Time) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM moderation_reports WHERE subject_did = ? AND created_at > ?
	`, did, since.Format(time.RFC3339Nano)).Scan(&count)
	return count, err
}

func (s *ModerationStore) HasReportedURI(ctx context.Context, reporterDID, subjectURI string) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, `
		SELECT 1 FROM moderation_reports WHERE reporter_did = ? AND subject_uri = ? LIMIT 1
	`, reporterDID, subjectURI).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return exists == 1, err
}

func (s *ModerationStore) CountReportsFromUserSince(ctx context.Context, reporterDID string, since time.Time) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM moderation_reports WHERE reporter_did = ? AND created_at > ?
	`, reporterDID, since.Format(time.RFC3339Nano)).Scan(&count)
	return count, err
}

// ========== Audit Log ==========

func (s *ModerationStore) LogAction(ctx context.Context, entry moderation.AuditEntry) error {
	details, err := json.Marshal(entry.Details)
	if err != nil {
		details = []byte("{}")
	}
	autoMod := 0
	if entry.AutoMod {
		autoMod = 1
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO moderation_audit_log (id, action, actor_did, target_uri, reason, details, timestamp, auto_mod)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, entry.ID, string(entry.Action), entry.ActorDID, entry.TargetURI, entry.Reason,
		string(details), entry.Timestamp.Format(time.RFC3339Nano), autoMod)
	if err != nil {
		return fmt.Errorf("log action: %w", err)
	}
	return nil
}

func (s *ModerationStore) ListAuditLog(ctx context.Context, limit int) ([]moderation.AuditEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, action, actor_did, target_uri, reason, details, timestamp, auto_mod
		FROM moderation_audit_log ORDER BY timestamp DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []moderation.AuditEntry
	for rows.Next() {
		var e moderation.AuditEntry
		var timestampStr, detailsStr string
		var autoMod int
		if err := rows.Scan(&e.ID, &e.Action, &e.ActorDID, &e.TargetURI, &e.Reason,
			&detailsStr, &timestampStr, &autoMod); err != nil {
			continue
		}
		e.Timestamp, _ = time.Parse(time.RFC3339Nano, timestampStr)
		e.AutoMod = autoMod == 1
		_ = json.Unmarshal([]byte(detailsStr), &e.Details)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ========== Auto-hide Resets ==========

func (s *ModerationStore) SetAutoHideReset(ctx context.Context, did string, resetAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO moderation_autohide_resets (did, reset_at) VALUES (?, ?)
		ON CONFLICT(did) DO UPDATE SET reset_at = excluded.reset_at
	`, did, resetAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("set autohide reset: %w", err)
	}
	return nil
}

func (s *ModerationStore) GetAutoHideReset(ctx context.Context, did string) (time.Time, error) {
	var resetAtStr string
	err := s.db.QueryRowContext(ctx, `SELECT reset_at FROM moderation_autohide_resets WHERE did = ?`, did).Scan(&resetAtStr)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	t, _ := time.Parse(time.RFC3339Nano, resetAtStr)
	return t, nil
}
