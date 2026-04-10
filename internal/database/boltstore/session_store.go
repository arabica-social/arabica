package boltstore

import (
	"context"
	"encoding/json"
	"fmt"

	"tangled.org/arabica.social/arabica/internal/tracing"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	bolt "go.etcd.io/bbolt"
)

// SessionStore implements oauth.ClientAuthStore using BoltDB for persistence.
// It stores OAuth sessions and auth request data, allowing sessions to survive
// server restarts.
type SessionStore struct {
	db *bolt.DB
}

// Ensure SessionStore implements oauth.ClientAuthStore
var _ oauth.ClientAuthStore = (*SessionStore)(nil)

// sessionKey generates a composite key for session storage: "did:sessionID"
func sessionKey(did syntax.DID, sessionID string) []byte {
	return []byte(did.String() + ":" + sessionID)
}

// GetSession retrieves a session by DID and session ID.
// Returns an error if the session is not found.
func (s *SessionStore) GetSession(ctx context.Context, did syntax.DID, sessionID string) (*oauth.ClientSessionData, error) {
	ctx, span := tracing.BoltSpan(ctx, "GetSession", "oauth_sessions")
	defer span.End()

	var session oauth.ClientSessionData

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketSessions)
		if bucket == nil {
			return fmt.Errorf("session bucket not found")
		}

		data := bucket.Get(sessionKey(did, sessionID))
		if data == nil {
			return fmt.Errorf("session not found")
		}

		return json.Unmarshal(data, &session)
	})

	if err != nil {
		return nil, err
	}

	return &session, nil
}

// SaveSession persists a session (upsert operation).
// If a session with the same DID and sessionID exists, it will be updated.
func (s *SessionStore) SaveSession(ctx context.Context, sess oauth.ClientSessionData) error {
	_, span := tracing.BoltSpan(ctx, "SaveSession", "oauth_sessions")
	defer span.End()

	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketSessions)
		if bucket == nil {
			return fmt.Errorf("session bucket not found")
		}

		return bucket.Put(sessionKey(sess.AccountDID, sess.SessionID), data)
	})
}

// DeleteSession removes a session by DID and session ID.
func (s *SessionStore) DeleteSession(ctx context.Context, did syntax.DID, sessionID string) error {
	_, span := tracing.BoltSpan(ctx, "DeleteSession", "oauth_sessions")
	defer span.End()

	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketSessions)
		if bucket == nil {
			return fmt.Errorf("session bucket not found")
		}

		return bucket.Delete(sessionKey(did, sessionID))
	})
}

// GetAuthRequestInfo retrieves pending auth request data by state token.
func (s *SessionStore) GetAuthRequestInfo(ctx context.Context, state string) (*oauth.AuthRequestData, error) {
	ctx, span := tracing.BoltSpan(ctx, "GetAuthRequestInfo", "oauth_auth_requests")
	defer span.End()

	var info oauth.AuthRequestData

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketAuthRequests)
		if bucket == nil {
			return fmt.Errorf("auth requests bucket not found")
		}

		data := bucket.Get([]byte(state))
		if data == nil {
			return fmt.Errorf("auth request not found")
		}

		return json.Unmarshal(data, &info)
	})

	if err != nil {
		return nil, err
	}

	return &info, nil
}

// SaveAuthRequestInfo stores auth request data keyed by state token.
// This is a create-only operation per the oauth.ClientAuthStore contract.
func (s *SessionStore) SaveAuthRequestInfo(ctx context.Context, info oauth.AuthRequestData) error {
	_, span := tracing.BoltSpan(ctx, "SaveAuthRequestInfo", "oauth_auth_requests")
	defer span.End()

	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal auth request: %w", err)
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketAuthRequests)
		if bucket == nil {
			return fmt.Errorf("auth requests bucket not found")
		}

		return bucket.Put([]byte(info.State), data)
	})
}

// DeleteAuthRequestInfo removes auth request data by state token.
func (s *SessionStore) DeleteAuthRequestInfo(ctx context.Context, state string) error {
	_, span := tracing.BoltSpan(ctx, "DeleteAuthRequestInfo", "oauth_auth_requests")
	defer span.End()

	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketAuthRequests)
		if bucket == nil {
			return fmt.Errorf("auth requests bucket not found")
		}

		return bucket.Delete([]byte(state))
	})
}

// ListSessions returns all sessions (for debugging/admin purposes).
func (s *SessionStore) ListSessions(ctx context.Context) ([]oauth.ClientSessionData, error) {
	ctx, span := tracing.BoltSpan(ctx, "ListSessions", "oauth_sessions")
	defer span.End()

	var sessions []oauth.ClientSessionData

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketSessions)
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			var session oauth.ClientSessionData
			if err := json.Unmarshal(v, &session); err != nil {
				// Skip malformed entries
				return nil
			}
			sessions = append(sessions, session)
			return nil
		})
	})

	return sessions, err
}

// CountSessions returns the number of stored sessions.
func (s *SessionStore) CountSessions(ctx context.Context) (int, error) {
	ctx, span := tracing.BoltSpan(ctx, "CountSessions", "oauth_sessions")
	defer span.End()

	var count int

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketSessions)
		if bucket == nil {
			return nil
		}

		count = bucket.Stats().KeyN
		return nil
	})

	return count, err
}
