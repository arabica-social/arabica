package boltstore

import (
	"encoding/json"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

// JoinRequest represents a request to join the PDS.
type JoinRequest struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Message   string    `json:"message,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	IP        string    `json:"ip"`
}

// JoinStore provides persistent storage for join requests.
type JoinStore struct {
	db *bolt.DB
}

// SaveRequest stores a join request in BoltDB.
func (s *JoinStore) SaveRequest(req *JoinRequest) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketJoinRequests)
		if bucket == nil {
			return fmt.Errorf("bucket not found: %s", BucketJoinRequests)
		}

		data, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("failed to marshal join request: %w", err)
		}

		return bucket.Put([]byte(req.ID), data)
	})
}

// DeleteRequest removes a join request by ID.
func (s *JoinStore) DeleteRequest(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketJoinRequests)
		if bucket == nil {
			return nil
		}
		return bucket.Delete([]byte(id))
	})
}

// ListRequests returns all stored join requests.
func (s *JoinStore) ListRequests() ([]*JoinRequest, error) {
	var requests []*JoinRequest

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketJoinRequests)
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			var req JoinRequest
			if err := json.Unmarshal(v, &req); err != nil {
				return fmt.Errorf("failed to unmarshal join request: %w", err)
			}
			requests = append(requests, &req)
			return nil
		})
	})

	return requests, err
}
