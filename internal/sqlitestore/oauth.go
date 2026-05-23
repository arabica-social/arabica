package sqlitestore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/rs/zerolog/log"
)

// OAuthStore implements oauth.ClientAuthStore using SQLite.
// Shares the database connection with the firehose FeedIndex.
//
// Not wired into the server yet; staged here so we can swap BoltDB out.
// Schema (oauth_sessions, oauth_auth_requests) lives alongside the rest
// of the firehose schema in internal/firehose/index.go.
type OAuthStore struct {
	db *sql.DB
}

func NewOAuthStore(db *sql.DB) *OAuthStore {
	return &OAuthStore{db: db}
}

var _ oauth.ClientAuthStore = (*OAuthStore)(nil)

func (s *OAuthStore) GetSession(ctx context.Context, did syntax.DID, sessionID string) (*oauth.ClientSessionData, error) {
	var data string
	err := s.db.QueryRowContext(ctx,
		`SELECT data FROM oauth_sessions WHERE did = ? AND session_id = ?`,
		did.String(), sessionID,
	).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, err
	}
	var sess oauth.ClientSessionData
	if err := json.Unmarshal([]byte(data), &sess); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return &sess, nil
}

func (s *OAuthStore) SaveSession(ctx context.Context, sess oauth.ClientSessionData) error {
	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO oauth_sessions (did, session_id, data, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(did, session_id) DO UPDATE SET
			data       = excluded.data,
			updated_at = excluded.updated_at
	`, sess.AccountDID.String(), sess.SessionID, string(data), time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *OAuthStore) DeleteSession(ctx context.Context, did syntax.DID, sessionID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM oauth_sessions WHERE did = ? AND session_id = ?`,
		did.String(), sessionID,
	)
	return err
}

func (s *OAuthStore) GetAuthRequestInfo(ctx context.Context, state string) (*oauth.AuthRequestData, error) {
	var data string
	err := s.db.QueryRowContext(ctx,
		`SELECT data FROM oauth_auth_requests WHERE state = ?`, state,
	).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("auth request not found")
	}
	if err != nil {
		return nil, err
	}
	var info oauth.AuthRequestData
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return nil, fmt.Errorf("unmarshal auth request: %w", err)
	}
	return &info, nil
}

func (s *OAuthStore) SaveAuthRequestInfo(ctx context.Context, info oauth.AuthRequestData) error {
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("marshal auth request: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO oauth_auth_requests (state, data, created_at)
		VALUES (?, ?, ?)
	`, info.State, string(data), time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *OAuthStore) DeleteAuthRequestInfo(ctx context.Context, state string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM oauth_auth_requests WHERE state = ?`, state,
	)
	return err
}

func (s *OAuthStore) ListSessions(ctx context.Context) ([]oauth.ClientSessionData, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM oauth_sessions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []oauth.ClientSessionData
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}
		var sess oauth.ClientSessionData
		if err := json.Unmarshal([]byte(data), &sess); err != nil {
			continue
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

func (s *OAuthStore) CountSessions(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM oauth_sessions`).Scan(&count)
	return count, err
}

// CleanupExpired removes stale rows from both OAuth tables. Sessions whose
// updated_at is older than sessionMaxAge are deleted; auth requests older than
// authRequestMaxAge are deleted. Auth requests are short-lived state tokens
// from incomplete OAuth callbacks; sessions are bounded by the upstream
// refresh-token lifetime (~90 days on bsky PDS).
func (s *OAuthStore) CleanupExpired(ctx context.Context, sessionMaxAge, authRequestMaxAge time.Duration) (sessions, authRequests int64, err error) {
	now := time.Now().UTC()
	sessCutoff := now.Add(-sessionMaxAge).Format(time.RFC3339Nano)
	reqCutoff := now.Add(-authRequestMaxAge).Format(time.RFC3339Nano)

	res, err := s.db.ExecContext(ctx, `DELETE FROM oauth_sessions WHERE updated_at < ?`, sessCutoff)
	if err != nil {
		return 0, 0, fmt.Errorf("cleanup sessions: %w", err)
	}
	sessions, _ = res.RowsAffected()

	res, err = s.db.ExecContext(ctx, `DELETE FROM oauth_auth_requests WHERE created_at < ?`, reqCutoff)
	if err != nil {
		return sessions, 0, fmt.Errorf("cleanup auth requests: %w", err)
	}
	authRequests, _ = res.RowsAffected()
	return sessions, authRequests, nil
}

// StartCleanup launches a goroutine that calls CleanupExpired on a ticker.
// Runs an initial pass immediately, then every interval until ctx is cancelled.
func (s *OAuthStore) StartCleanup(ctx context.Context, interval, sessionMaxAge, authRequestMaxAge time.Duration) {
	run := func() {
		sess, reqs, err := s.CleanupExpired(ctx, sessionMaxAge, authRequestMaxAge)
		if err != nil {
			log.Warn().Err(err).Msg("OAuth cleanup failed")
			return
		}
		if sess > 0 || reqs > 0 {
			log.Info().
				Int64("sessions", sess).
				Int64("auth_requests", reqs).
				Msg("Pruned expired OAuth rows")
		}
	}

	run()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
	log.Info().
		Dur("interval", interval).
		Dur("session_max_age", sessionMaxAge).
		Dur("auth_request_max_age", authRequestMaxAge).
		Msg("OAuth cleanup started")
}
