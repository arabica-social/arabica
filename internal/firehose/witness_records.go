package firehose

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/tracing"
	"tangled.org/pdewey.com/atp"

	"go.opentelemetry.io/otel/attribute"
)

// witnessRecordStorage owns the records table behavior used by both the
// firehose ingestion path and the ATProto witness-cache interface. Keeping the
// SQL here stops FeedIndex from exposing table mechanics as its implementation.
type witnessRecordStorage struct {
	db *sql.DB
}

func newWitnessRecordStorage(db *sql.DB) *witnessRecordStorage {
	return &witnessRecordStorage{db: db}
}

func (s *witnessRecordStorage) get(ctx context.Context, uri string) (*atproto.WitnessRecord, error) {
	rec, err := s.getIndexed(ctx, uri)
	if err != nil || rec == nil {
		return nil, err
	}
	return &atproto.WitnessRecord{
		URI:        rec.URI,
		DID:        rec.DID,
		Collection: rec.Collection,
		RKey:       rec.RKey,
		Record:     rec.Record,
		CID:        rec.CID,
		IndexedAt:  rec.IndexedAt,
		CreatedAt:  rec.CreatedAt,
	}, nil
}

func (s *witnessRecordStorage) list(ctx context.Context, did, collection string, offset, limit int) ([]*atproto.WitnessRecord, error) {
	query := `
		SELECT uri, did, collection, rkey, record, cid, indexed_at, created_at
		FROM records WHERE did = ? AND collection = ?
		ORDER BY created_at DESC
	`
	args := []any{did, collection}
	if limit > 0 {
		query += ` LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]*atproto.WitnessRecord, 0)
	for rows.Next() {
		var rec atproto.WitnessRecord
		var recordStr, indexedAtStr, createdAtStr string
		if err := rows.Scan(&rec.URI, &rec.DID, &rec.Collection, &rec.RKey,
			&recordStr, &rec.CID, &indexedAtStr, &createdAtStr); err != nil {
			continue
		}
		rec.Record = json.RawMessage(recordStr)
		rec.IndexedAt, _ = time.Parse(time.RFC3339Nano, indexedAtStr)
		rec.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
		records = append(records, &rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func (s *witnessRecordStorage) count(ctx context.Context, did, collection string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM records WHERE did = ? AND collection = ?
	`, did, collection).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *witnessRecordStorage) upsert(ctx context.Context, did, collection, rkey, cid string, record json.RawMessage, _ int64) error {
	const stmt = `INSERT INTO records (uri, did, collection, rkey, record, cid, indexed_at, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(uri) DO UPDATE SET
	record = excluded.record,
	cid = excluded.cid,
	indexed_at = excluded.indexed_at,
	created_at = excluded.created_at`

	ctx, span := tracing.SqliteSpan(ctx, "upsert", "records")
	span.SetAttributes(
		attribute.String("record.did", did),
		attribute.String("record.collection", collection),
		attribute.String("record.rkey", rkey),
		attribute.String("db.statement", stmt),
	)
	defer span.End()

	uri := atp.BuildATURI(did, collection, rkey)

	createdAt := time.Now().UTC()
	var recordData map[string]any
	if err := json.Unmarshal(record, &recordData); err == nil {
		if createdAtStr, ok := recordData["createdAt"].(string); ok {
			if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
				createdAt = t.UTC()
			}
		}
	}

	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, stmt, uri, did, collection, rkey, string(record), cid,
		now.Format(time.RFC3339Nano), createdAt.Format(time.RFC3339Nano))
	if err != nil {
		tracing.EndWithError(span, err)
		return fmt.Errorf("failed to upsert record: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `INSERT OR IGNORE INTO known_dids (did) VALUES (?)`, did)
	if err != nil {
		tracing.EndWithError(span, err)
		return fmt.Errorf("failed to track known DID: %w", err)
	}

	return nil
}

func (s *witnessRecordStorage) update(ctx context.Context, did, collection, rkey string, record json.RawMessage) error {
	uri := atp.BuildATURI(did, collection, rkey)
	ctx, span := tracing.SqliteSpan(ctx, "update", "records")
	span.SetAttributes(
		attribute.String("record.did", did),
		attribute.String("record.collection", collection),
		attribute.String("record.rkey", rkey),
	)
	defer span.End()

	_, err := s.db.ExecContext(ctx,
		`UPDATE records SET record = ?, indexed_at = ? WHERE uri = ?`,
		string(record), time.Now().UTC().Format(time.RFC3339Nano), uri)
	if err != nil {
		tracing.EndWithError(span, err)
		return fmt.Errorf("failed to update record: %w", err)
	}
	return nil
}

func (s *witnessRecordStorage) upsertBatch(ctx context.Context, records []atproto.WitnessWriteRecord) error {
	if len(records) == 0 {
		return nil
	}

	const upsertSQL = `INSERT INTO records (uri, did, collection, rkey, record, cid, indexed_at, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(uri) DO UPDATE SET
	record = excluded.record,
	cid = excluded.cid,
	indexed_at = excluded.indexed_at,
	created_at = excluded.created_at`

	ctx, span := tracing.SqliteSpan(ctx, "upsert_batch", "records")
	span.SetAttributes(
		attribute.Int("batch.size", len(records)),
		attribute.String("db.statement", upsertSQL),
	)
	defer span.End()

	tx, err := s.db.Begin()
	if err != nil {
		tracing.EndWithError(span, err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.Prepare(upsertSQL)
	if err != nil {
		tracing.EndWithError(span, err)
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC()
	seenDIDs := make(map[string]struct{})

	for _, rec := range records {
		uri := atp.BuildATURI(rec.DID, rec.Collection, rec.RKey)

		createdAt := now
		var recordData map[string]any
		if err := json.Unmarshal(rec.Record, &recordData); err == nil {
			if createdAtStr, ok := recordData["createdAt"].(string); ok {
				if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
					createdAt = t.UTC()
				}
			}
		}

		if _, err := stmt.Exec(uri, rec.DID, rec.Collection, rec.RKey,
			string(rec.Record), rec.CID,
			now.Format(time.RFC3339Nano), createdAt.Format(time.RFC3339Nano)); err != nil {
			tracing.EndWithError(span, err)
			return fmt.Errorf("failed to upsert record %s: %w", uri, err)
		}
		seenDIDs[rec.DID] = struct{}{}
	}

	for did := range seenDIDs {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO known_dids (did) VALUES (?)`, did); err != nil {
			tracing.EndWithError(span, err)
			return fmt.Errorf("failed to track known DID: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		tracing.EndWithError(span, err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (s *witnessRecordStorage) delete(ctx context.Context, did, collection, rkey string) error {
	uri := atp.BuildATURI(did, collection, rkey)
	_, err := s.db.ExecContext(ctx, `DELETE FROM records WHERE uri = ?`, uri)
	return err
}

func (s *witnessRecordStorage) getIndexed(ctx context.Context, uri string) (*IndexedRecord, error) {
	var rec IndexedRecord
	var recordStr, indexedAtStr, createdAtStr string

	err := s.db.QueryRowContext(ctx, `
		SELECT uri, did, collection, rkey, record, cid, indexed_at, created_at
		FROM records WHERE uri = ?
	`, uri).Scan(&rec.URI, &rec.DID, &rec.Collection, &rec.RKey,
		&recordStr, &rec.CID, &indexedAtStr, &createdAtStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rec.Record = json.RawMessage(recordStr)
	rec.IndexedAt, _ = time.Parse(time.RFC3339Nano, indexedAtStr)
	rec.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)

	return &rec, nil
}
