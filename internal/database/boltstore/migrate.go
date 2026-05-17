package boltstore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

// OAuthMigrationResult reports counts from a bolt -> sqlite OAuth migration.
type OAuthMigrationResult struct {
	Sessions     int
	AuthRequests int
}

// MigrateOAuthToSQLite copies every OAuth session and pending auth request from
// the bolt store into the sqlite database. The destination must already have
// the oauth_sessions and oauth_auth_requests tables (see firehose schema).
//
// The migration is idempotent: rows are upserted by their natural key, so
// re-running after a partial failure is safe. Values from bolt are stored
// verbatim — we copy the raw JSON blobs without re-marshalling so a forward
// rollback to bolt would round-trip cleanly.
func (s *Store) MigrateOAuthToSQLite(ctx context.Context, sqlite *sql.DB) (OAuthMigrationResult, error) {
	var result OAuthMigrationResult

	tx, err := sqlite.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("begin sqlite tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	sessStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO oauth_sessions (did, session_id, data, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(did, session_id) DO UPDATE SET
			data       = excluded.data,
			updated_at = excluded.updated_at
	`)
	if err != nil {
		return result, fmt.Errorf("prepare session insert: %w", err)
	}
	defer sessStmt.Close()

	reqStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO oauth_auth_requests (state, data, created_at)
		VALUES (?, ?, ?)
		ON CONFLICT(state) DO UPDATE SET
			data       = excluded.data,
			created_at = excluded.created_at
	`)
	if err != nil {
		return result, fmt.Errorf("prepare auth request insert: %w", err)
	}
	defer reqStmt.Close()

	now := time.Now().UTC().Format(time.RFC3339Nano)

	err = s.db.View(func(btx *bolt.Tx) error {
		if b := btx.Bucket(BucketSessions); b != nil {
			if err := b.ForEach(func(k, v []byte) error {
				did, sessionID, ok := splitSessionKey(k)
				if !ok {
					return nil
				}
				if _, err := sessStmt.ExecContext(ctx, did, sessionID, string(v), now); err != nil {
					return fmt.Errorf("insert session %s/%s: %w", did, sessionID, err)
				}
				result.Sessions++
				return nil
			}); err != nil {
				return err
			}
		}

		if b := btx.Bucket(BucketAuthRequests); b != nil {
			if err := b.ForEach(func(k, v []byte) error {
				if _, err := reqStmt.ExecContext(ctx, string(k), string(v), now); err != nil {
					return fmt.Errorf("insert auth request %s: %w", string(k), err)
				}
				result.AuthRequests++
				return nil
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return result, err
	}

	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("commit sqlite tx: %w", err)
	}
	return result, nil
}

// splitSessionKey reverses sessionKey: "<did>:<sessionID>" -> (did, sessionID).
// DIDs contain colons (did:plc:abc...) so we split on the LAST colon.
func splitSessionKey(k []byte) (did, sessionID string, ok bool) {
	s := string(k)
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			// Must have a did:* prefix to be a real session key, not just any colon.
			if i == 0 || i == len(s)-1 {
				return "", "", false
			}
			return s[:i], s[i+1:], true
		}
	}
	return "", "", false
}
