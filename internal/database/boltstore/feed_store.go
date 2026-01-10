package boltstore

import (
	"encoding/json"
	"time"

	bolt "go.etcd.io/bbolt"
)

// FeedUser represents a registered user in the feed registry.
type FeedUser struct {
	DID          string    `json:"did"`
	RegisteredAt time.Time `json:"registered_at"`
}

// FeedStore provides persistent storage for the feed registry.
// It stores DIDs of users who have logged in and should appear in the community feed.
type FeedStore struct {
	db *bolt.DB
}

// Register adds a DID to the feed registry.
// If the DID already exists, this is a no-op.
func (s *FeedStore) Register(did string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketFeedRegistry)
		if bucket == nil {
			return nil
		}

		// Check if already registered
		existing := bucket.Get([]byte(did))
		if existing != nil {
			// Already registered, no-op
			return nil
		}

		// Create new registration
		user := FeedUser{
			DID:          did,
			RegisteredAt: time.Now(),
		}

		data, err := json.Marshal(user)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(did), data)
	})
}

// Unregister removes a DID from the feed registry.
func (s *FeedStore) Unregister(did string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketFeedRegistry)
		if bucket == nil {
			return nil
		}

		return bucket.Delete([]byte(did))
	})
}

// IsRegistered checks if a DID is in the feed registry.
func (s *FeedStore) IsRegistered(did string) bool {
	var registered bool

	s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketFeedRegistry)
		if bucket == nil {
			return nil
		}

		registered = bucket.Get([]byte(did)) != nil
		return nil
	})

	return registered
}

// List returns all registered DIDs.
func (s *FeedStore) List() []string {
	var dids []string

	s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketFeedRegistry)
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			dids = append(dids, string(k))
			return nil
		})
	})

	return dids
}

// ListWithMetadata returns all registered users with their metadata.
func (s *FeedStore) ListWithMetadata() []FeedUser {
	var users []FeedUser

	s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketFeedRegistry)
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			var user FeedUser
			if err := json.Unmarshal(v, &user); err != nil {
				// Fallback for simple keys without metadata
				user = FeedUser{DID: string(k)}
			}
			users = append(users, user)
			return nil
		})
	})

	return users
}

// Count returns the number of registered users.
func (s *FeedStore) Count() int {
	var count int

	s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketFeedRegistry)
		if bucket == nil {
			return nil
		}

		count = bucket.Stats().KeyN
		return nil
	})

	return count
}

// Clear removes all entries from the feed registry.
// Use with caution - primarily for testing.
func (s *FeedStore) Clear() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		// Delete and recreate the bucket
		if err := tx.DeleteBucket(BucketFeedRegistry); err != nil {
			// Bucket might not exist, that's ok
			if err != bolt.ErrBucketNotFound {
				return err
			}
		}

		_, err := tx.CreateBucket(BucketFeedRegistry)
		return err
	})
}
