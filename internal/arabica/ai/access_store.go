package ai

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

const MaxAccessEntries = 1000

var ErrAccessFull = errors.New("AI access grant limit reached")
var ErrInvalidDID = errors.New("invalid DID")

type AccessEntry struct {
	DID       string `json:"did"`
	GrantedBy string `json:"granted_by"`
	CreatedAt string `json:"created_at"`
}

type AccessStore struct{ db *sql.DB }

func NewAccessStore(db *sql.DB) (*AccessStore, error) {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS ai_access_grants (
 did TEXT PRIMARY KEY NOT NULL,
 granted_by TEXT NOT NULL,
 created_at TEXT NOT NULL
 )`)
	if err != nil {
		return nil, err
	}
	return &AccessStore{db: db}, nil
}

func (s *AccessStore) Allows(ctx context.Context, did string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM ai_access_grants WHERE did = ?)`, did).Scan(&exists)
	return exists, err
}

func (s *AccessStore) List(ctx context.Context) ([]AccessEntry, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT did, granted_by, created_at FROM ai_access_grants ORDER BY created_at DESC, did LIMIT ?`, MaxAccessEntries)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := make([]AccessEntry, 0)
	for rows.Next() {
		var entry AccessEntry
		if err := rows.Scan(&entry.DID, &entry.GrantedBy, &entry.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (s *AccessStore) Grant(ctx context.Context, did, grantedBy string) error {
	if _, err := syntax.ParseDID(did); err != nil {
		return ErrInvalidDID
	}
	if _, err := syntax.ParseDID(grantedBy); err != nil {
		return ErrInvalidDID
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO ai_access_grants (did, granted_by, created_at)
 SELECT ?, ?, ? WHERE (SELECT COUNT(*) FROM ai_access_grants) < ?
 ON CONFLICT(did) DO NOTHING`, did, grantedBy, time.Now().UTC().Format(time.RFC3339Nano), MaxAccessEntries)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		allowed, err := s.Allows(ctx, did)
		if err != nil {
			return err
		}
		if !allowed {
			return ErrAccessFull
		}
	}
	return nil
}

func (s *AccessStore) Revoke(ctx context.Context, did string) error {
	if _, err := syntax.ParseDID(did); err != nil {
		return ErrInvalidDID
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM ai_access_grants WHERE did = ?`, did)
	return err
}
