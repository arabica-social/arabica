package aiaccess

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

const MaxEntries = 1000

var ErrFull = errors.New("AI access grant limit reached")
var ErrInvalidDID = errors.New("invalid DID")

type Entry struct {
	DID       string `json:"did"`
	GrantedBy string `json:"granted_by"`
	CreatedAt string `json:"created_at"`
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS ai_access_grants (
 did TEXT PRIMARY KEY NOT NULL,
 granted_by TEXT NOT NULL,
 created_at TEXT NOT NULL
 )`)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Allows(ctx context.Context, did string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM ai_access_grants WHERE did = ?)`, did).Scan(&exists)
	return exists, err
}

func (s *Store) List(ctx context.Context) ([]Entry, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT did, granted_by, created_at FROM ai_access_grants ORDER BY created_at DESC, did LIMIT ?`, MaxEntries)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := make([]Entry, 0)
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(&entry.DID, &entry.GrantedBy, &entry.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (s *Store) Grant(ctx context.Context, did, grantedBy string) error {
	if _, err := syntax.ParseDID(did); err != nil {
		return ErrInvalidDID
	}
	if _, err := syntax.ParseDID(grantedBy); err != nil {
		return ErrInvalidDID
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO ai_access_grants (did, granted_by, created_at)
 SELECT ?, ?, ? WHERE (SELECT COUNT(*) FROM ai_access_grants) < ?
 ON CONFLICT(did) DO NOTHING`, did, grantedBy, time.Now().UTC().Format(time.RFC3339Nano), MaxEntries)
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
			return ErrFull
		}
	}
	return nil
}

func (s *Store) Revoke(ctx context.Context, did string) error {
	if _, err := syntax.ParseDID(did); err != nil {
		return ErrInvalidDID
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM ai_access_grants WHERE did = ?`, did)
	return err
}
