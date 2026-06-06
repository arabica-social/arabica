package firehose

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type profileIndexStorage struct {
	db *sql.DB
}

func newProfileIndexStorage(db *sql.DB) *profileIndexStorage {
	return &profileIndexStorage{db: db}
}

// backfillHandleIndex populates did_by_handle from the profiles table. Idempotent.
// Iterates every cached profile and inserts (handle, did) — last writer wins,
// matching the live storeProfile semantics, so a handle that existed on multiple
// DIDs resolves to whichever profile was inserted most recently in the iteration.
func (s *profileIndexStorage) backfillHandleIndex() error {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM did_by_handle`).Scan(&n); err == nil && n > 0 {
		return nil
	}

	rows, err := s.db.Query(`SELECT did, data FROM profiles`)
	if err != nil {
		return err
	}
	defer rows.Close()

	now := time.Now().Format(time.RFC3339Nano)
	for rows.Next() {
		var did, dataStr string
		if err := rows.Scan(&did, &dataStr); err != nil {
			continue
		}
		cached := &CachedProfile{}
		if err := json.Unmarshal([]byte(dataStr), cached); err != nil || cached.Profile == nil || cached.Profile.Handle == "" {
			continue
		}
		_, _ = s.db.Exec(
			`INSERT OR REPLACE INTO did_by_handle (handle, did, updated_at) VALUES (?, ?, ?)`,
			cached.Profile.Handle, did, now)
	}
	return rows.Err()
}

func (s *profileIndexStorage) loadProfile(ctx context.Context, did string) (*CachedProfile, bool) {
	var dataStr string
	err := s.db.QueryRowContext(ctx, `SELECT data FROM profiles WHERE did = ?`, did).Scan(&dataStr)
	if err != nil {
		return nil, false
	}
	cached := &CachedProfile{}
	if err := json.Unmarshal([]byte(dataStr), cached); err != nil {
		return nil, false
	}
	return cached, true
}

func (s *profileIndexStorage) storeProfile(ctx context.Context, did string, cached *CachedProfile) {
	data, _ := json.Marshal(cached)
	_, _ = s.db.ExecContext(ctx, `INSERT OR REPLACE INTO profiles (did, data, expires_at) VALUES (?, ?, ?)`,
		did, string(data), cached.ExpiresAt.Format(time.RFC3339Nano))

	if cached.Profile != nil && cached.Profile.Handle != "" {
		// Drop any prior row pointing this DID at a different handle (handle change).
		_, _ = s.db.ExecContext(ctx,
			`DELETE FROM did_by_handle WHERE did = ? AND handle != ?`, did, cached.Profile.Handle)
		// Last writer wins on handle — this naturally resolves handle reassignment
		// from an old DID to a new one, since the new profile's INSERT OR REPLACE
		// overwrites the old DID's mapping.
		_, _ = s.db.ExecContext(ctx,
			`INSERT OR REPLACE INTO did_by_handle (handle, did, updated_at) VALUES (?, ?, ?)`,
			cached.Profile.Handle, did, cached.CachedAt.Format(time.RFC3339Nano))
	}
}

func (s *profileIndexStorage) deleteProfile(did string) {
	_, _ = s.db.Exec(`DELETE FROM profiles WHERE did = ?`, did)
	_, _ = s.db.Exec(`DELETE FROM did_by_handle WHERE did = ?`, did)
}

func (s *profileIndexStorage) didByHandle(ctx context.Context, handle string) (string, bool) {
	var did string
	err := s.db.QueryRowContext(ctx,
		`SELECT did FROM did_by_handle WHERE handle = ?`, handle).Scan(&did)
	if err != nil || did == "" {
		return "", false
	}
	return did, true
}

func (s *profileIndexStorage) didByHandleExcept(ctx context.Context, handle, exceptDID string) (string, bool) {
	var did string
	err := s.db.QueryRowContext(ctx,
		`SELECT did FROM did_by_handle WHERE handle = ? AND did != ?`, handle, exceptDID).Scan(&did)
	if err != nil || did == "" {
		return "", false
	}
	return did, true
}

func (s *profileIndexStorage) profileDIDs(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT did FROM profiles`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dids := make([]string, 0, 128)
	for rows.Next() {
		var did string
		if err := rows.Scan(&did); err == nil {
			dids = append(dids, did)
		}
	}
	return dids, rows.Err()
}

func (s *profileIndexStorage) profileStatsVisibility(ctx context.Context, did string) (string, bool) {
	var raw string
	err := s.db.QueryRowContext(ctx,
		`SELECT profile_stats_visibility FROM user_settings WHERE did = ?`, did,
	).Scan(&raw)
	if err != nil {
		return "", false
	}
	return raw, true
}

func (s *profileIndexStorage) setProfileStatsVisibility(ctx context.Context, did, raw string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO user_settings (did, profile_stats_visibility) VALUES (?, ?)
		 ON CONFLICT(did) DO UPDATE SET profile_stats_visibility = excluded.profile_stats_visibility`,
		did, raw,
	)
	return err
}

func (s *profileIndexStorage) userPreferences(ctx context.Context, did string) (string, bool) {
	var raw string
	err := s.db.QueryRowContext(ctx,
		`SELECT preferences FROM user_settings WHERE did = ?`, did,
	).Scan(&raw)
	if err != nil {
		return "", false
	}
	return raw, true
}

func (s *profileIndexStorage) setUserPreferences(ctx context.Context, did, raw string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO user_settings (did, preferences) VALUES (?, ?)
		 ON CONFLICT(did) DO UPDATE SET preferences = excluded.preferences`,
		did, raw,
	)
	return err
}
