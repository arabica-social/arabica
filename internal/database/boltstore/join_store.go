package boltstore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"arabica/internal/tracing"

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
func (s *JoinStore) SaveRequest(ctx context.Context, req *JoinRequest) error {
	_, span := tracing.BoltSpan(ctx, "SaveRequest", "join_requests")
	defer span.End()

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

// GetRequest returns a join request by ID.
func (s *JoinStore) GetRequest(ctx context.Context, id string) (*JoinRequest, error) {
	_, span := tracing.BoltSpan(ctx, "GetRequest", "join_requests")
	defer span.End()

	var req JoinRequest
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketJoinRequests)
		if bucket == nil {
			return fmt.Errorf("bucket not found: %s", BucketJoinRequests)
		}
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("join request not found: %s", id)
		}
		return json.Unmarshal(data, &req)
	})
	if err != nil {
		return nil, err
	}
	return &req, nil
}

// DeleteRequest removes a join request by ID.
func (s *JoinStore) DeleteRequest(ctx context.Context, id string) error {
	_, span := tracing.BoltSpan(ctx, "DeleteRequest", "join_requests")
	defer span.End()

	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketJoinRequests)
		if bucket == nil {
			return nil
		}
		return bucket.Delete([]byte(id))
	})
}

// ListRequests returns all stored join requests.
func (s *JoinStore) ListRequests(ctx context.Context) ([]*JoinRequest, error) {
	_, span := tracing.BoltSpan(ctx, "ListRequests", "join_requests")
	defer span.End()

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
