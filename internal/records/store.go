// Package records defines the app-agnostic record-store boundary used by
// handlers that only need generic AT Protocol record CRUD.
package records

import "context"

// RawRecord is one decoded repository record plus its AT Protocol metadata.
type RawRecord struct {
	URI    string
	RKey   string
	CID    string
	Record map[string]any
}

// Store is the minimal backend surface shared by app-generic handlers.
// Implementations are responsible for any cache/witness/PDS fallback policy.
type Store interface {
	DID() string
	FetchRecord(ctx context.Context, nsid, rkey string) (record map[string]any, uri, cid string, err error)
	FetchAllRecords(ctx context.Context, nsid string) ([]RawRecord, error)
	PutRecord(ctx context.Context, nsid, rkey string, record any) (resultRKey, cid string, err error)
	RemoveRecord(ctx context.Context, nsid, rkey string) error
}
