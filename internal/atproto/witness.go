package atproto

import (
	"context"
	"encoding/json"
	"time"
)

// WitnessRecord is a record retrieved from the witness cache (firehose index).
type WitnessRecord struct {
	URI        string
	DID        string
	Collection string
	RKey       string
	CID        string
	Record     json.RawMessage
	IndexedAt  time.Time
	CreatedAt  time.Time
}

// WitnessCache is a read-only view of the firehose index that lets AtprotoStore
// serve reads from the locally-indexed SQLite database instead of the PDS.
// Implementations must be safe for concurrent use.
type WitnessCache interface {
	// GetWitnessRecord retrieves a single record by AT-URI.
	// Returns (nil, nil) when the record is not present in the cache.
	GetWitnessRecord(ctx context.Context, uri string) (*WitnessRecord, error)

	// ListWitnessRecords returns all cached records for a DID+collection pair,
	// ordered by created_at descending.
	// Returns an empty (non-nil) slice when none are found.
	ListWitnessRecords(ctx context.Context, did, collection string) ([]*WitnessRecord, error)
}
