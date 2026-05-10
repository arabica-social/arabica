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

// WitnessRecordToMap unmarshals a WitnessRecord's raw JSON into the map format
// expected by the Record* conversion functions.
func WitnessRecordToMap(wr *WitnessRecord) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(wr.Record, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// WitnessCache is a view of the firehose index that lets AtprotoStore
// serve reads from the locally-indexed SQLite database instead of the PDS.
// Write methods enable write-through caching so the witness stays fresh
// without waiting for the firehose to re-index after a PDS mutation.
// Implementations must be safe for concurrent use.
type WitnessCache interface {
	// GetWitnessRecord retrieves a single record by AT-URI.
	// Returns (nil, nil) when the record is not present in the cache.
	GetWitnessRecord(ctx context.Context, uri string) (*WitnessRecord, error)

	// ListWitnessRecords returns all cached records for a DID+collection pair,
	// ordered by created_at descending.
	// Returns an empty (non-nil) slice when none are found.
	ListWitnessRecords(ctx context.Context, did, collection string) ([]*WitnessRecord, error)

	// ListWitnessRecordsPaginated returns a page of cached records for a
	// DID+collection pair, ordered by created_at descending.
	// When limit <= 0, returns all records (same as ListWitnessRecords).
	ListWitnessRecordsPaginated(ctx context.Context, did, collection string, offset, limit int) ([]*WitnessRecord, error)

	// CountWitnessRecords returns the total count of cached records for a
	// DID+collection pair. Returns 0 when none are found or on error.
	CountWitnessRecords(ctx context.Context, did, collection string) (int, error)

	// UpsertWitnessRecord inserts or updates a record in the cache.
	// Used for write-through caching after successful PDS mutations.
	UpsertWitnessRecord(ctx context.Context, did, collection, rkey, cid string, record json.RawMessage) error

	// UpsertWitnessRecordBatch inserts or updates multiple records in a single
	// transaction. Used for bulk operations like refresh/backfill.
	UpsertWitnessRecordBatch(ctx context.Context, records []WitnessWriteRecord) error

	// DeleteWitnessRecord removes a record from the cache.
	// Used for write-through caching after successful PDS deletions.
	DeleteWitnessRecord(ctx context.Context, did, collection, rkey string) error
}

// WitnessWriteRecord holds the fields needed to upsert a record into the witness cache.
type WitnessWriteRecord struct {
	DID        string
	Collection string
	RKey       string
	CID        string
	Record     json.RawMessage
}
