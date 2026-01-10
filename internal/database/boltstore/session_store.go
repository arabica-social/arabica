package boltstore

import (
	"context"
	"encoding/json"
	"fmt"

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

// DeleteAllSessionsForDID removes all sessions for a given DID.
// Useful for "logout from all devices" functionality.
func (s *SessionStore) DeleteAllSessionsForDID(ctx context.Context, did syntax.DID) error {
	prefix := []byte(did.String() + ":")

	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketSessions)
		if bucket == nil {
			return nil
		}

		// Collect keys to delete (can't delete while iterating)
		var keysToDelete [][]byte
		c := bucket.Cursor()
		for k, _ := c.Seek(prefix); k != nil && len(k) >= len(prefix) && string(k[:len(prefix)]) == string(prefix); k, _ = c.Next() {
			keysToDelete = append(keysToDelete, append([]byte{}, k...))
		}

		// Delete collected keys
		for _, k := range keysToDelete {
			if err := bucket.Delete(k); err != nil {
				return err
			}
		}

		return nil
	})
}
