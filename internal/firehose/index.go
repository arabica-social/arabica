package firehose

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/entities/arabica"
	"tangled.org/arabica.social/arabica/internal/entities/oolong"
	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/lexicons"
	"tangled.org/arabica.social/arabica/internal/tracing"
	"tangled.org/pdewey.com/atp"

	"database/sql/driver"

	"github.com/XSAM/otelsql"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	_ "modernc.org/sqlite"
)

// FeedableRecordTypes are the record types that should appear as feed items.
// Likes, comments, etc. are indexed but not displayed directly in the feed.
var FeedableRecordTypes = map[lexicons.RecordType]bool{
	lexicons.RecordTypeBrew:    true,
	lexicons.RecordTypeBean:    true,
	lexicons.RecordTypeRoaster: true,
	lexicons.RecordTypeGrinder: true,
	lexicons.RecordTypeBrewer:  true,
	lexicons.RecordTypeRecipe:  true,

	lexicons.RecordTypeOolongTea:     true,
	lexicons.RecordTypeOolongBrew:    true,
	lexicons.RecordTypeOolongVessel:  true,
	lexicons.RecordTypeOolongInfuser: true,
	lexicons.RecordTypeOolongVendor:  true,
	// Cafe and Drink are deferred for the v1 oolong launch; once their
	// UI ships, flip these back to true.
	// lexicons.RecordTypeOolongCafe:   true,
	// lexicons.RecordTypeOolongDrink:  true,
}

// IndexedRecord represents a record stored in the index
type IndexedRecord struct {
	URI        string          `json:"uri"`
	DID        string          `json:"did"`
	Collection string          `json:"collection"`
	RKey       string          `json:"rkey"`
	Record     json.RawMessage `json:"record"`
	CID        string          `json:"cid"`
	IndexedAt  time.Time       `json:"indexed_at"`
	CreatedAt  time.Time       `json:"created_at"`
}

// CachedProfile stores profile data with TTL
type CachedProfile struct {
	Profile   *atproto.Profile `json:"profile"`
	CachedAt  time.Time        `json:"cached_at"`
	ExpiresAt time.Time        `json:"expires_at"`
}

// FeedIndex provides persistent storage for firehose events
type FeedIndex struct {
	db           *sql.DB
	publicClient *atp.PublicClient
	profileTTL   time.Duration

	// commentNSID is the comment collection this index's binary serves
	// (e.g. social.arabica.alpha.comment or social.oolong.alpha.comment).
	// Used when rebuilding comment AT-URIs from indexed rows. Falls back
	// to arabica.NSIDComment when unset for backwards-compat with tests
	// that construct a FeedIndex directly via NewFeedIndex.
	commentNSID string

	// In-memory cache for hot data
	profileCache   map[string]*CachedProfile
	profileCacheMu sync.RWMutex

	ready   bool
	readyMu sync.RWMutex
}

// SetCommentNSID configures the comment collection NSID used when
// reconstructing comment AT-URIs from rows in the comments table.
func (idx *FeedIndex) SetCommentNSID(nsid string) {
	idx.commentNSID = nsid
}

func (idx *FeedIndex) commentCollection() string {
	if idx.commentNSID != "" {
		return idx.commentNSID
	}
	return arabica.NSIDComment
}

// FeedSort defines the sort order for feed queries
type FeedSort string

const (
	FeedSortRecent  FeedSort = "recent"
	FeedSortPopular FeedSort = "popular"
)

// FeedQuery specifies filtering, sorting, and pagination for feed queries
type FeedQuery struct {
	Limit       int                   // Max items to return
	Cursor      string                // Opaque cursor for pagination (created_at|uri)
	TypeFilter  lexicons.RecordType   // Filter to a specific record type (empty = all)
	TypeFilters []lexicons.RecordType // Filter to multiple record types (takes precedence over TypeFilter)
	Sort        FeedSort              // Sort order (default: recent)
}

// FeedResult contains feed items plus pagination info
type FeedResult struct {
	Items      []*feed.FeedItem
	NextCursor string // Empty if no more results
}

const schemaNoTrailingPragma = `
CREATE TABLE IF NOT EXISTS records (
    uri         TEXT PRIMARY KEY,
    did         TEXT NOT NULL,
    collection  TEXT NOT NULL,
    rkey        TEXT NOT NULL,
    record      TEXT NOT NULL,
    cid         TEXT NOT NULL DEFAULT '',
    indexed_at  TEXT NOT NULL,
    created_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_records_created ON records(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_records_did ON records(did);
CREATE INDEX IF NOT EXISTS idx_records_coll_created ON records(collection, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_records_did_coll ON records(did, collection, created_at DESC);

CREATE TABLE IF NOT EXISTS meta (
    key   TEXT PRIMARY KEY,
    value BLOB
);

CREATE TABLE IF NOT EXISTS known_dids (did TEXT PRIMARY KEY);
CREATE TABLE IF NOT EXISTS registered_dids (
    did           TEXT PRIMARY KEY,
    registered_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS backfilled (did TEXT PRIMARY KEY, backfilled_at TEXT NOT NULL);

CREATE TABLE IF NOT EXISTS profiles (
    did        TEXT PRIMARY KEY,
    data       TEXT NOT NULL,
    expires_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS did_by_handle (
    handle     TEXT PRIMARY KEY,
    did        TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_did_by_handle_did ON did_by_handle(did);

CREATE TABLE IF NOT EXISTS likes (
    subject_uri TEXT NOT NULL,
    actor_did   TEXT NOT NULL,
    rkey        TEXT NOT NULL,
    PRIMARY KEY (subject_uri, actor_did)
);
CREATE INDEX IF NOT EXISTS idx_likes_actor ON likes(actor_did, subject_uri);

CREATE TABLE IF NOT EXISTS comments (
    actor_did   TEXT NOT NULL,
    rkey        TEXT NOT NULL,
    subject_uri TEXT NOT NULL,
    parent_uri  TEXT NOT NULL DEFAULT '',
    parent_rkey TEXT NOT NULL DEFAULT '',
    cid         TEXT NOT NULL DEFAULT '',
    text        TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    PRIMARY KEY (actor_did, rkey)
);
CREATE INDEX IF NOT EXISTS idx_comments_subject ON comments(subject_uri, created_at);

CREATE TABLE IF NOT EXISTS notifications (
    id          TEXT NOT NULL,
    target_did  TEXT NOT NULL,
    type        TEXT NOT NULL,
    actor_did   TEXT NOT NULL,
    subject_uri TEXT NOT NULL,
    created_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_notif_target ON notifications(target_did, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_notif_dedup ON notifications(target_did, type, actor_did, subject_uri);

CREATE TABLE IF NOT EXISTS notifications_meta (
    target_did TEXT PRIMARY KEY,
    last_read  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS moderation_hidden_records (
    uri         TEXT PRIMARY KEY,
    hidden_at   TEXT NOT NULL,
    hidden_by   TEXT NOT NULL,
    reason      TEXT NOT NULL DEFAULT '',
    auto_hidden INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS moderation_blacklist (
    did            TEXT PRIMARY KEY,
    blacklisted_at TEXT NOT NULL,
    blacklisted_by TEXT NOT NULL,
    reason         TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS moderation_reports (
    id           TEXT PRIMARY KEY,
    subject_uri  TEXT NOT NULL DEFAULT '',
    subject_did  TEXT NOT NULL DEFAULT '',
    reporter_did TEXT NOT NULL,
    reason       TEXT NOT NULL,
    created_at   TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending',
    resolved_by  TEXT NOT NULL DEFAULT '',
    resolved_at  TEXT
);
CREATE INDEX IF NOT EXISTS idx_modreports_uri      ON moderation_reports(subject_uri);
CREATE INDEX IF NOT EXISTS idx_modreports_did      ON moderation_reports(subject_did);
CREATE INDEX IF NOT EXISTS idx_modreports_reporter ON moderation_reports(reporter_did, created_at);
CREATE INDEX IF NOT EXISTS idx_modreports_status   ON moderation_reports(status);

CREATE TABLE IF NOT EXISTS moderation_audit_log (
    id         TEXT PRIMARY KEY,
    action     TEXT NOT NULL,
    actor_did  TEXT NOT NULL,
    target_uri TEXT NOT NULL DEFAULT '',
    reason     TEXT NOT NULL DEFAULT '',
    details    TEXT NOT NULL DEFAULT '{}',
    timestamp  TEXT NOT NULL,
    auto_mod   INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_modaudit_ts ON moderation_audit_log(timestamp DESC);

CREATE TABLE IF NOT EXISTS moderation_autohide_resets (
    did      TEXT PRIMARY KEY,
    reset_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS moderation_labels (
    id          TEXT PRIMARY KEY,
    entity_type TEXT NOT NULL,
    entity_id   TEXT NOT NULL,
    label       TEXT NOT NULL,
    value       TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL,
    created_by  TEXT NOT NULL,
    expires_at  TEXT,
    UNIQUE(entity_type, entity_id, label)
);
CREATE INDEX IF NOT EXISTS idx_modlabels_entity ON moderation_labels(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_modlabels_expires ON moderation_labels(expires_at) WHERE expires_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS user_settings (
    did  TEXT PRIMARY KEY,
    profile_stats_visibility TEXT NOT NULL DEFAULT '{}'
);
`

// NewFeedIndex creates a new feed index backed by SQLite
func NewFeedIndex(path string, profileTTL time.Duration) (*FeedIndex, error) {
	if path == "" {
		return nil, fmt.Errorf("index path is required")
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create index directory: %w", err)
		}
	}

	db, err := otelsql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_pragma=temp_store(MEMORY)&_pragma=mmap_size(134217728)&_pragma=cache_size(-65536)",
		otelsql.WithAttributes(semconv.DBSystemSqlite),
		otelsql.WithSpanOptions(otelsql.SpanOptions{
			// Only create SQL spans when there's already a parent span (e.g. from an HTTP request).
			// This avoids standalone traces for background work like firehose indexing.
			SpanFilter: func(ctx context.Context, _ otelsql.Method, _ string, _ []driver.NamedValue) bool {
				return trace.SpanFromContext(ctx).SpanContext().IsValid()
			},
			// Suppress noisy low-level driver spans — keep only the statement-level spans.
			OmitConnResetSession: true,
			OmitConnectorConnect: true,
			OmitRows:             true,
			OmitConnPrepare:      true,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open index database: %w", err)
	}

	// WAL mode allows concurrent reads with a single writer.
	// Allow multiple reader connections but limit to avoid file descriptor exhaustion.
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)

	// Record DB connection pool metrics via OTel
	if _, err := otelsql.RegisterDBStatsMetrics(db, otelsql.WithAttributes(semconv.DBSystemSqlite)); err != nil {
		log.Warn().Err(err).Msg("Failed to register OTel DB stats metrics")
	}

	// Execute schema (skip PRAGMAs — already set via DSN)
	if _, err := db.Exec(schemaNoTrailingPragma); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	idx := &FeedIndex{
		db:           db,
		publicClient: atproto.NewPublicClient(),
		profileTTL:   profileTTL,
		profileCache: make(map[string]*CachedProfile),
	}

	// One-time backfill: populate did_by_handle from any pre-existing profile rows
	// so handle resolution works for users observed before this table existed.
	if err := idx.backfillHandleIndex(); err != nil {
		log.Warn().Err(err).Msg("did_by_handle backfill failed; lookups will populate lazily")
	}

	// If the database already has records from a previous run, mark ready immediately
	// so the feed is served from persisted data while the firehose reconnects.
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM records`).Scan(&count); err == nil && count > 0 {
		idx.ready = true
	}

	return idx, nil
}

// backfillHandleIndex populates did_by_handle from the profiles table. Idempotent.
// Iterates every cached profile and inserts (handle, did) — last writer wins,
// matching the live storeProfile semantics, so a handle that existed on multiple
// DIDs resolves to whichever profile was inserted most recently in the iteration.
func (idx *FeedIndex) backfillHandleIndex() error {
	var n int
	if err := idx.db.QueryRow(`SELECT COUNT(*) FROM did_by_handle`).Scan(&n); err == nil && n > 0 {
		return nil
	}

	rows, err := idx.db.Query(`SELECT did, data FROM profiles`)
	if err != nil {
		return err
	}
	defer rows.Close()

	now := time.Now().Format(time.RFC3339Nano)
	for rows.Next() {
		var did, dataStr string
		if err := rows.Scan(&did, &dataStr); err != nil {
			continue
		}
		cached := &CachedProfile{}
		if err := json.Unmarshal([]byte(dataStr), cached); err != nil || cached.Profile == nil || cached.Profile.Handle == "" {
			continue
		}
		_, _ = idx.db.Exec(
			`INSERT OR REPLACE INTO did_by_handle (handle, did, updated_at) VALUES (?, ?, ?)`,
			cached.Profile.Handle, did, now)
	}
	return rows.Err()
}

// Compile-time check: FeedIndex must satisfy the atproto.WitnessCache interface.
var _ atproto.WitnessCache = (*FeedIndex)(nil)

// DB returns the underlying database connection for shared use by other stores.
func (idx *FeedIndex) DB() *sql.DB {
	return idx.db
}

// GetWitnessRecord retrieves a single record by AT-URI from the index.
// Returns (nil, nil) when the record is not found.
func (idx *FeedIndex) GetWitnessRecord(ctx context.Context, uri string) (*atproto.WitnessRecord, error) {
	var rec atproto.WitnessRecord
	var recordStr, indexedAtStr, createdAtStr string

	err := idx.db.QueryRowContext(ctx, `
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

// ListWitnessRecords returns all indexed records for a DID+collection pair,
// ordered by created_at descending. Returns an empty slice when none are found.
func (idx *FeedIndex) ListWitnessRecords(ctx context.Context, did, collection string) ([]*atproto.WitnessRecord, error) {
	return idx.listWitnessRecords(ctx, did, collection, 0, 0)
}

// ListWitnessRecordsPaginated returns a page of cached records for a
// DID+collection pair, ordered by created_at descending.
// When limit <= 0, returns all records.
func (idx *FeedIndex) ListWitnessRecordsPaginated(ctx context.Context, did, collection string, offset, limit int) ([]*atproto.WitnessRecord, error) {
	return idx.listWitnessRecords(ctx, did, collection, offset, limit)
}

// listWitnessRecords is the shared implementation with optional LIMIT/OFFSET.
func (idx *FeedIndex) listWitnessRecords(ctx context.Context, did, collection string, offset, limit int) ([]*atproto.WitnessRecord, error) {
	query := `
		SELECT uri, did, collection, rkey, record, cid, indexed_at, created_at
		FROM records WHERE did = ? AND collection = ?
		ORDER BY created_at DESC
	`
	var args []any
	args = append(args, did, collection)
	if limit > 0 {
		query += ` LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}

	rows, err := idx.db.QueryContext(ctx, query, args...)
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

// CountWitnessRecords returns the total count of cached records for a
// DID+collection pair.
func (idx *FeedIndex) CountWitnessRecords(ctx context.Context, did, collection string) (int, error) {
	var count int
	err := idx.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM records WHERE did = ? AND collection = ?
	`, did, collection).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// Close closes the index database
func (idx *FeedIndex) Close() error {
	if idx.db != nil {
		return idx.db.Close()
	}
	return nil
}

// SetReady marks the index as ready to serve queries
func (idx *FeedIndex) SetReady(ready bool) {
	idx.readyMu.Lock()
	defer idx.readyMu.Unlock()
	idx.ready = ready
}

// IsReady returns true if the index is populated and ready
func (idx *FeedIndex) IsReady() bool {
	idx.readyMu.RLock()
	defer idx.readyMu.RUnlock()
	return idx.ready
}

// GetCursor returns the last processed cursor (microseconds timestamp)
func (idx *FeedIndex) GetCursor(ctx context.Context) (int64, error) {
	var cursor int64
	err := idx.db.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = 'cursor'`).Scan(&cursor)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return cursor, err
}

// SetCursor stores the cursor position
func (idx *FeedIndex) SetCursor(ctx context.Context, cursor int64) error {
	_, err := idx.db.ExecContext(ctx, `INSERT OR REPLACE INTO meta (key, value) VALUES ('cursor', ?)`, cursor)
	return err
}

// UpsertRecord adds or updates a record in the index.
// The context is used for OTel tracing; pass context.Background() for background operations.
func (idx *FeedIndex) UpsertRecord(ctx context.Context, did, collection, rkey, cid string, record json.RawMessage, eventTime int64) error {
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

	// Parse createdAt from record
	var recordData map[string]any
	createdAt := time.Now().UTC()
	if err := json.Unmarshal(record, &recordData); err == nil {
		if createdAtStr, ok := recordData["createdAt"].(string); ok {
			if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
				createdAt = t.UTC()
			}
		}
	}

	now := time.Now().UTC()

	_, err := idx.db.ExecContext(ctx, stmt, uri, did, collection, rkey, string(record), cid,
		now.Format(time.RFC3339Nano), createdAt.Format(time.RFC3339Nano))
	if err != nil {
		tracing.EndWithError(span, err)
		return fmt.Errorf("failed to upsert record: %w", err)
	}

	// Track known DID
	_, err = idx.db.ExecContext(ctx, `INSERT OR IGNORE INTO known_dids (did) VALUES (?)`, did)
	if err != nil {
		tracing.EndWithError(span, err)
		return fmt.Errorf("failed to track known DID: %w", err)
	}

	return nil
}

// DeleteRecord removes a record from the index
func (idx *FeedIndex) DeleteRecord(ctx context.Context, did, collection, rkey string) error {
	uri := atp.BuildATURI(did, collection, rkey)
	_, err := idx.db.ExecContext(ctx, `DELETE FROM records WHERE uri = ?`, uri)
	return err
}

// DeleteAllByDID removes all data associated with a DID from the index.
// Used when a Jetstream account event reports the DID as deleted or takendown.
//
// Removes: records authored by the DID; likes/comments by the DID; likes/comments
// targeting the DID's records; profile cache; notifications to or from the DID;
// known/registered/backfilled tracking; user settings.
//
// Preserves moderation_* tables (reports, audit log, blacklist, labels, hidden
// records, autohide resets) — those are evidence of moderation actions and
// should outlive the account.
func (idx *FeedIndex) DeleteAllByDID(ctx context.Context, did string) error {
	uriPrefix := fmt.Sprintf("at://%s/%%", did)

	tx, err := idx.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	stmts := []struct {
		sql  string
		args []any
	}{
		{`DELETE FROM records WHERE did = ?`, []any{did}},
		{`DELETE FROM likes WHERE actor_did = ?`, []any{did}},
		{`DELETE FROM likes WHERE subject_uri LIKE ?`, []any{uriPrefix}},
		{`DELETE FROM comments WHERE actor_did = ?`, []any{did}},
		{`DELETE FROM comments WHERE subject_uri LIKE ?`, []any{uriPrefix}},
		{`DELETE FROM notifications WHERE target_did = ? OR actor_did = ?`, []any{did, did}},
		{`DELETE FROM notifications_meta WHERE target_did = ?`, []any{did}},
		{`DELETE FROM profiles WHERE did = ?`, []any{did}},
		{`DELETE FROM did_by_handle WHERE did = ?`, []any{did}},
		{`DELETE FROM known_dids WHERE did = ?`, []any{did}},
		{`DELETE FROM registered_dids WHERE did = ?`, []any{did}},
		{`DELETE FROM backfilled WHERE did = ?`, []any{did}},
		{`DELETE FROM user_settings WHERE did = ?`, []any{did}},
	}

	for _, s := range stmts {
		if _, err := tx.ExecContext(ctx, s.sql, s.args...); err != nil {
			return fmt.Errorf("delete by did: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	idx.profileCacheMu.Lock()
	delete(idx.profileCache, did)
	idx.profileCacheMu.Unlock()

	return nil
}

// UpsertWitnessRecord implements atproto.WitnessCache for write-through caching.
func (idx *FeedIndex) UpsertWitnessRecord(ctx context.Context, did, collection, rkey, cid string, record json.RawMessage) error {
	return idx.UpsertRecord(ctx, did, collection, rkey, cid, record, time.Now().UnixMicro())
}

// UpsertWitnessRecordBatch implements atproto.WitnessCache batch upsert.
// All records are inserted in a single transaction for efficiency.
func (idx *FeedIndex) UpsertWitnessRecordBatch(ctx context.Context, records []atproto.WitnessWriteRecord) error {
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

	tx, err := idx.db.Begin()
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

	// Track known DIDs (deduplicated)
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

// DeleteWitnessRecord implements atproto.WitnessCache for write-through caching.
func (idx *FeedIndex) DeleteWitnessRecord(ctx context.Context, did, collection, rkey string) error {
	return idx.DeleteRecord(ctx, did, collection, rkey)
}

// GetRecord retrieves a single record by URI
func (idx *FeedIndex) GetRecord(ctx context.Context, uri string) (*IndexedRecord, error) {
	var rec IndexedRecord
	var recordStr, indexedAtStr, createdAtStr string

	err := idx.db.QueryRowContext(ctx, `
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

// GetRecentFeed returns recent feed items from the index
func (idx *FeedIndex) GetRecentFeed(ctx context.Context, limit int) ([]*feed.FeedItem, error) {
	return idx.getFeedItems(ctx, nil, limit, "")
}

// recordTypeToNSID maps a feedable lexicons.RecordType to its NSID
// collection string. Built from the descriptor registry so every
// registered app contributes its own entries — no app-specific
// imports in this package. Only types listed in FeedableRecordTypes
// participate; entries from sister apps that aren't installed in the
// running binary simply never appear in the registry.
var recordTypeToNSID = func() map[lexicons.RecordType]string {
	m := make(map[lexicons.RecordType]string)
	for _, d := range entities.All() {
		if FeedableRecordTypes[d.Type] && d.NSID != "" {
			m[d.Type] = d.NSID
		}
	}
	return m
}()

// feedableCollections is the set of collection NSIDs that appear in the feed
var feedableCollections = func() []string {
	out := make([]string, 0, len(recordTypeToNSID))
	for _, nsid := range recordTypeToNSID {
		out = append(out, nsid)
	}
	return out
}()

// GetFeedWithQuery returns feed items matching the given query with cursor-based pagination
func (idx *FeedIndex) GetFeedWithQuery(ctx context.Context, q FeedQuery) (*FeedResult, error) {
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Sort == "" {
		q.Sort = FeedSortRecent
	}

	// Determine collection filters
	var collectionFilters []string
	if len(q.TypeFilters) > 0 {
		for _, tf := range q.TypeFilters {
			nsid, ok := recordTypeToNSID[tf]
			if !ok {
				return nil, fmt.Errorf("unknown record type: %s", tf)
			}
			collectionFilters = append(collectionFilters, nsid)
		}
	} else if q.TypeFilter != "" {
		nsid, ok := recordTypeToNSID[q.TypeFilter]
		if !ok {
			return nil, fmt.Errorf("unknown record type: %s", q.TypeFilter)
		}
		collectionFilters = []string{nsid}
	}

	// For popular sort, fetch more candidates to re-rank by score
	fetchLimit := q.Limit + 1
	if q.Sort == FeedSortPopular {
		fetchLimit = q.Limit * 5
	}

	items, err := idx.getFeedItems(ctx, collectionFilters, fetchLimit, q.Cursor)
	if err != nil {
		return nil, err
	}

	if q.Sort == FeedSortPopular {
		sort.Slice(items, func(i, j int) bool {
			scoreI := items[i].LikeCount*3 + items[i].CommentCount*2
			scoreJ := items[j].LikeCount*3 + items[j].CommentCount*2
			if scoreI != scoreJ {
				return scoreI > scoreJ
			}
			return items[i].Timestamp.After(items[j].Timestamp)
		})
	}

	result := &FeedResult{Items: items}
	if len(items) > q.Limit {
		result.Items = items[:q.Limit]
		last := result.Items[q.Limit-1]
		result.NextCursor = last.Timestamp.Format(time.RFC3339Nano) + "|" + last.SubjectURI
	}

	return result, nil
}

// getFeedItems fetches records from SQLite, resolves references, and returns FeedItems.
func (idx *FeedIndex) getFeedItems(ctx context.Context, collectionFilters []string, limit int, cursor string) ([]*feed.FeedItem, error) {
	// Build query for feedable records
	var args []any
	query := `SELECT uri, did, collection, rkey, record, cid, indexed_at, created_at FROM records WHERE `

	if len(collectionFilters) == 1 {
		query += `collection = ? `
		args = append(args, collectionFilters[0])
	} else if len(collectionFilters) > 1 {
		placeholders := make([]string, len(collectionFilters))
		for i, c := range collectionFilters {
			placeholders[i] = "?"
			args = append(args, c)
		}
		query += `collection IN (` + strings.Join(placeholders, ",") + `) `
	} else {
		// Only feedable collections
		placeholders := make([]string, len(feedableCollections))
		for i, c := range feedableCollections {
			placeholders[i] = "?"
			args = append(args, c)
		}
		query += `collection IN (` + strings.Join(placeholders, ",") + `) `
	}

	// Cursor-based pagination: cursor format is "created_at|uri"
	if cursor != "" {
		parts := strings.SplitN(cursor, "|", 2)
		if len(parts) == 2 {
			query += `AND (created_at < ? OR (created_at = ? AND uri < ?)) `
			args = append(args, parts[0], parts[0], parts[1])
		}
	}

	query += `ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := idx.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*IndexedRecord
	refURIs := make(map[string]bool) // URIs we need to resolve

	for rows.Next() {
		var rec IndexedRecord
		var recordStr, indexedAtStr, createdAtStr string
		if err := rows.Scan(&rec.URI, &rec.DID, &rec.Collection, &rec.RKey,
			&recordStr, &rec.CID, &indexedAtStr, &createdAtStr); err != nil {
			continue
		}
		rec.Record = json.RawMessage(recordStr)
		rec.IndexedAt, _ = time.Parse(time.RFC3339Nano, indexedAtStr)
		rec.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
		records = append(records, &rec)

		// Collect reference URIs from the record data
		var recordData map[string]any
		if err := json.Unmarshal(rec.Record, &recordData); err == nil {
			for _, key := range []string{"beanRef", "roasterRef", "grinderRef", "brewerRef", "teaRef", "vendorRef", "vesselRef", "infuserRef"} {
				if ref, ok := recordData[key].(string); ok && ref != "" {
					refURIs[ref] = true
				}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Build lookup map starting with the fetched records
	recordsByURI := make(map[string]*IndexedRecord, len(records))
	for _, r := range records {
		recordsByURI[r.URI] = r
	}

	// Fetch referenced records that we don't already have
	var missingURIs []string
	for uri := range refURIs {
		if _, ok := recordsByURI[uri]; !ok {
			missingURIs = append(missingURIs, uri)
		}
	}

	if len(missingURIs) > 0 {
		placeholders := make([]string, len(missingURIs))
		refArgs := make([]any, len(missingURIs))
		for i, uri := range missingURIs {
			placeholders[i] = "?"
			refArgs[i] = uri
		}
		refQuery := `SELECT uri, did, collection, rkey, record, cid, indexed_at, created_at FROM records WHERE uri IN (` + strings.Join(placeholders, ",") + `)`
		refRows, err := idx.db.QueryContext(ctx, refQuery, refArgs...)
		if err == nil {
			defer refRows.Close()
			for refRows.Next() {
				var rec IndexedRecord
				var recordStr, indexedAtStr, createdAtStr string
				if err := refRows.Scan(&rec.URI, &rec.DID, &rec.Collection, &rec.RKey,
					&recordStr, &rec.CID, &indexedAtStr, &createdAtStr); err != nil {
					continue
				}
				rec.Record = json.RawMessage(recordStr)
				rec.IndexedAt, _ = time.Parse(time.RFC3339Nano, indexedAtStr)
				rec.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
				recordsByURI[rec.URI] = &rec

				// If this is an oolong tea, check if it references a vendor we also need
				if rec.Collection == oolong.NSIDTea {
					var teaData map[string]any
					if err := json.Unmarshal(rec.Record, &teaData); err == nil {
						if vendorRef, ok := teaData["vendorRef"].(string); ok && vendorRef != "" {
							if _, ok := recordsByURI[vendorRef]; !ok {
								var vRec IndexedRecord
								var vStr, vIdxAt, vCreAt string
								err := idx.db.QueryRowContext(ctx,
									`SELECT uri, did, collection, rkey, record, cid, indexed_at, created_at FROM records WHERE uri = ?`,
									vendorRef).Scan(&vRec.URI, &vRec.DID, &vRec.Collection, &vRec.RKey,
									&vStr, &vRec.CID, &vIdxAt, &vCreAt)
								if err == nil {
									vRec.Record = json.RawMessage(vStr)
									vRec.IndexedAt, _ = time.Parse(time.RFC3339Nano, vIdxAt)
									vRec.CreatedAt, _ = time.Parse(time.RFC3339Nano, vCreAt)
									recordsByURI[vRec.URI] = &vRec
								}
							}
						}
					}
				}

				// If this is a bean, check if it references a roaster we also need
				if rec.Collection == arabica.NSIDBean {
					var beanData map[string]any
					if err := json.Unmarshal(rec.Record, &beanData); err == nil {
						if roasterRef, ok := beanData["roasterRef"].(string); ok && roasterRef != "" {
							if _, ok := recordsByURI[roasterRef]; !ok {
								// Fetch this roaster too
								var rRec IndexedRecord
								var rStr, rIdxAt, rCreAt string
								err := idx.db.QueryRowContext(ctx,
									`SELECT uri, did, collection, rkey, record, cid, indexed_at, created_at FROM records WHERE uri = ?`,
									roasterRef).Scan(&rRec.URI, &rRec.DID, &rRec.Collection, &rRec.RKey,
									&rStr, &rRec.CID, &rIdxAt, &rCreAt)
								if err == nil {
									rRec.Record = json.RawMessage(rStr)
									rRec.IndexedAt, _ = time.Parse(time.RFC3339Nano, rIdxAt)
									rRec.CreatedAt, _ = time.Parse(time.RFC3339Nano, rCreAt)
									recordsByURI[rRec.URI] = &rRec
								}
							}
						}
					}
				}
			}
		}
	}

	// Batch-fetch social data for all records
	recordURIs := make([]string, 0, len(records))
	didSet := make(map[string]struct{}, len(records))
	for _, r := range records {
		recordURIs = append(recordURIs, r.URI)
		didSet[r.DID] = struct{}{}
	}
	likeCounts := idx.GetLikeCountsBatch(ctx, recordURIs)
	commentCounts := idx.GetCommentCountsBatch(ctx, recordURIs)

	// Pre-warm profile cache for all unique DIDs
	profiles := make(map[string]*atproto.Profile, len(didSet))
	for did := range didSet {
		if p, err := idx.GetProfile(ctx, did); err == nil {
			profiles[did] = p
		}
	}

	// Convert to FeedItems
	items := make([]*feed.FeedItem, 0, len(records))
	for _, record := range records {
		item, err := idx.recordToFeedItem(ctx, record, recordsByURI, profiles)
		if err != nil {
			log.Warn().Err(err).Str("uri", record.URI).Msg("failed to convert record to feed item")
			continue
		}
		if !FeedableRecordTypes[item.RecordType] {
			continue
		}
		item.LikeCount = likeCounts[record.URI]
		item.CommentCount = commentCounts[record.URI]
		items = append(items, item)
	}

	return items, nil
}

// recordToFeedItem converts an IndexedRecord to a FeedItem.
// The profiles map provides pre-fetched profiles keyed by DID; if nil or missing,
// the profile is fetched individually as a fallback.
func (idx *FeedIndex) recordToFeedItem(ctx context.Context, record *IndexedRecord, refMap map[string]*IndexedRecord, profiles map[string]*atproto.Profile) (*feed.FeedItem, error) {
	var recordData map[string]any
	if err := json.Unmarshal(record.Record, &recordData); err != nil {
		return nil, err
	}

	item := &feed.FeedItem{
		Timestamp: record.CreatedAt,
		TimeAgo:   formatTimeAgo(record.CreatedAt),
	}

	// Get author profile from pre-fetched map or fallback to individual fetch
	profile, ok := profiles[record.DID]
	if !ok || profile == nil {
		var err error
		profile, err = idx.GetProfile(ctx, record.DID)
		if err != nil {
			log.Warn().Err(err).Str("did", record.DID).Msg("failed to get profile")
			profile = &atproto.Profile{
				DID:    record.DID,
				Handle: record.DID,
			}
		}
	}
	item.Author = profile

	if strings.HasSuffix(record.Collection, ".like") {
		return nil, fmt.Errorf("unexpected: likes should be filtered before conversion")
	}

	desc := entities.GetByNSID(record.Collection)
	if desc == nil || desc.RecordToModel == nil {
		return nil, fmt.Errorf("unknown collection: %s", record.Collection)
	}
	model, err := desc.RecordToModel(recordData, record.URI)
	if err != nil {
		return nil, err
	}

	item.RecordType = desc.Type
	item.Action = "added a new " + desc.Noun
	item.Record = model

	// Per-entity reference resolution. The ref shape is genuinely
	// entity-specific (brew has four refs, bean has one, recipe has one) and
	// the resolved values land on per-entity fields. Keep these inline
	// until a future descriptor extension generalises ref resolution.
	switch m := model.(type) {
	case *arabica.Brew:
		resolveBrewFeedRefs(m, recordData, refMap)
	case *arabica.Bean:
		resolveBeanFeedRef(m, recordData, refMap)
	case *arabica.Recipe:
		resolveRecipeFeedRef(m, recordData, refMap)
	case *oolong.Brew:
		resolveOolongBrewFeedRefs(m, recordData, refMap)
	case *oolong.Tea:
		resolveOolongTeaFeedRef(m, recordData, refMap)
	}

	// Populate subject fields (like/comment counts are set by caller via batch)
	item.SubjectURI = record.URI
	item.SubjectCID = record.CID

	return item, nil
}

// resolveBrewFeedRefs hydrates bean/grinder/brewer/recipe references on a
// brew using already-fetched indexed records in refMap. Missing refs are
// silently skipped — feed cards render fine with partial reference data.
func resolveBrewFeedRefs(brew *arabica.Brew, recordData map[string]any, refMap map[string]*IndexedRecord) {
	if beanRef, ok := recordData["beanRef"].(string); ok && beanRef != "" {
		if beanRecord, found := refMap[beanRef]; found {
			var beanData map[string]any
			if err := json.Unmarshal(beanRecord.Record, &beanData); err == nil {
				bean, _ := arabica.RecordToBean(beanData, beanRef)
				brew.Bean = bean

				// Resolve roaster reference for the bean.
				if roasterRef, ok := beanData["roasterRef"].(string); ok && roasterRef != "" {
					if roasterRecord, found := refMap[roasterRef]; found {
						var roasterData map[string]any
						if err := json.Unmarshal(roasterRecord.Record, &roasterData); err == nil {
							roaster, _ := arabica.RecordToRoaster(roasterData, roasterRef)
							brew.Bean.Roaster = roaster
						}
					}
				}
			}
		}
	}

	if grinderRef, ok := recordData["grinderRef"].(string); ok && grinderRef != "" {
		if grinderRecord, found := refMap[grinderRef]; found {
			var grinderData map[string]any
			if err := json.Unmarshal(grinderRecord.Record, &grinderData); err == nil {
				grinder, _ := arabica.RecordToGrinder(grinderData, grinderRef)
				brew.GrinderObj = grinder
			}
		}
	}

	if brewerRef, ok := recordData["brewerRef"].(string); ok && brewerRef != "" {
		if brewerRecord, found := refMap[brewerRef]; found {
			var brewerData map[string]any
			if err := json.Unmarshal(brewerRecord.Record, &brewerData); err == nil {
				brewer, _ := arabica.RecordToBrewer(brewerData, brewerRef)
				brew.BrewerObj = brewer
			}
		}
	}

	if recipeRef, ok := recordData["recipeRef"].(string); ok && recipeRef != "" {
		if rkey := atp.RKeyFromURI(recipeRef); rkey != "" {
			brew.RecipeRKey = rkey
		}
		if recipeRecord, found := refMap[recipeRef]; found {
			var recipeData map[string]any
			if err := json.Unmarshal(recipeRecord.Record, &recipeData); err == nil {
				recipe, _ := arabica.RecordToRecipe(recipeData, recipeRef)
				brew.RecipeObj = recipe
			}
		}
	}
}

// resolveBeanFeedRef hydrates a bean's roaster reference from refMap.
func resolveBeanFeedRef(bean *arabica.Bean, recordData map[string]any, refMap map[string]*IndexedRecord) {
	roasterRef, ok := recordData["roasterRef"].(string)
	if !ok || roasterRef == "" {
		return
	}
	roasterRecord, found := refMap[roasterRef]
	if !found {
		return
	}
	var roasterData map[string]any
	if err := json.Unmarshal(roasterRecord.Record, &roasterData); err != nil {
		return
	}
	roaster, _ := arabica.RecordToRoaster(roasterData, roasterRef)
	bean.Roaster = roaster
}

// resolveOolongBrewFeedRefs hydrates tea (with nested vendor), vessel and
// infuser references on an oolong brew using refMap.
func resolveOolongBrewFeedRefs(brew *oolong.Brew, recordData map[string]any, refMap map[string]*IndexedRecord) {
	if teaRef, ok := recordData["teaRef"].(string); ok && teaRef != "" {
		if teaRecord, found := refMap[teaRef]; found {
			var teaData map[string]any
			if err := json.Unmarshal(teaRecord.Record, &teaData); err == nil {
				tea, _ := oolong.RecordToTea(teaData, teaRef)
				brew.Tea = tea

				if tea != nil {
					if vendorRef, ok := teaData["vendorRef"].(string); ok && vendorRef != "" {
						if vendorRecord, found := refMap[vendorRef]; found {
							var vendorData map[string]any
							if err := json.Unmarshal(vendorRecord.Record, &vendorData); err == nil {
								vendor, _ := oolong.RecordToVendor(vendorData, vendorRef)
								brew.Tea.Vendor = vendor
							}
						}
					}
				}
			}
		}
	}

	if vesselRef, ok := recordData["vesselRef"].(string); ok && vesselRef != "" {
		if vesselRecord, found := refMap[vesselRef]; found {
			var vesselData map[string]any
			if err := json.Unmarshal(vesselRecord.Record, &vesselData); err == nil {
				vessel, _ := oolong.RecordToVessel(vesselData, vesselRef)
				brew.Vessel = vessel
			}
		}
	}

	if infuserRef, ok := recordData["infuserRef"].(string); ok && infuserRef != "" {
		if infuserRecord, found := refMap[infuserRef]; found {
			var infuserData map[string]any
			if err := json.Unmarshal(infuserRecord.Record, &infuserData); err == nil {
				infuser, _ := oolong.RecordToInfuser(infuserData, infuserRef)
				brew.Infuser = infuser
			}
		}
	}
}

// resolveOolongTeaFeedRef hydrates a tea's vendor reference from refMap.
func resolveOolongTeaFeedRef(tea *oolong.Tea, recordData map[string]any, refMap map[string]*IndexedRecord) {
	vendorRef, ok := recordData["vendorRef"].(string)
	if !ok || vendorRef == "" {
		return
	}
	vendorRecord, found := refMap[vendorRef]
	if !found {
		return
	}
	var vendorData map[string]any
	if err := json.Unmarshal(vendorRecord.Record, &vendorData); err != nil {
		return
	}
	vendor, _ := oolong.RecordToVendor(vendorData, vendorRef)
	tea.Vendor = vendor
}

// resolveRecipeFeedRef hydrates a recipe's brewer reference from refMap.
func resolveRecipeFeedRef(recipe *arabica.Recipe, recordData map[string]any, refMap map[string]*IndexedRecord) {
	brewerRef, ok := recordData["brewerRef"].(string)
	if !ok || brewerRef == "" {
		return
	}
	brewerRecord, found := refMap[brewerRef]
	if !found {
		return
	}
	var brewerData map[string]any
	if err := json.Unmarshal(brewerRecord.Record, &brewerData); err != nil {
		return
	}
	brewer, _ := arabica.RecordToBrewer(brewerData, brewerRef)
	recipe.BrewerObj = brewer
}

// GetProfile fetches a profile, using cache when possible. The persistent
// SQLite store has no TTL — the profile watcher keeps it fresh via the
// firehose. Only a completely unknown DID triggers an API fetch.
func (idx *FeedIndex) GetProfile(ctx context.Context, did string) (*atproto.Profile, error) {
	// Check in-memory cache first (TTL used only for memory management)
	idx.profileCacheMu.RLock()
	if cached, ok := idx.profileCache[did]; ok && time.Now().Before(cached.ExpiresAt) {
		idx.profileCacheMu.RUnlock()
		return cached.Profile, nil
	}
	idx.profileCacheMu.RUnlock()

	// Check persistent store — no TTL, firehose keeps it fresh
	var dataStr string
	err := idx.db.QueryRowContext(ctx, `SELECT data FROM profiles WHERE did = ?`, did).Scan(&dataStr)
	if err == nil {
		cached := &CachedProfile{}
		if err := json.Unmarshal([]byte(dataStr), cached); err == nil {
			// Promote to in-memory cache
			cached.ExpiresAt = time.Now().Add(idx.profileTTL)
			idx.profileCacheMu.Lock()
			idx.profileCache[did] = cached
			idx.profileCacheMu.Unlock()
			return cached.Profile, nil
		}
	}

	// Unknown DID — fetch from API
	profile, err := idx.publicClient.GetProfile(ctx, did)
	if err != nil {
		return nil, err
	}

	idx.storeProfile(ctx, did, profile)
	return profile, nil
}

// StoreProfile writes a profile to both in-memory and persistent caches and
// maintains the did_by_handle index. Use this when you've already fetched a
// profile (backfill workers, tests, externally-provided data) and want to seed
// the cache without going through the public API.
func (idx *FeedIndex) StoreProfile(ctx context.Context, did string, profile *atproto.Profile) {
	idx.storeProfile(ctx, did, profile)
}

// storeProfile writes a profile to both in-memory and persistent caches, and
// maintains the did_by_handle index so handle lookups stay accurate across
// handle changes and handle reassignment between DIDs.
func (idx *FeedIndex) storeProfile(ctx context.Context, did string, profile *atproto.Profile) {
	now := time.Now()
	cached := &CachedProfile{
		Profile:   profile,
		CachedAt:  now,
		ExpiresAt: now.Add(idx.profileTTL),
	}

	idx.profileCacheMu.Lock()
	idx.profileCache[did] = cached
	idx.profileCacheMu.Unlock()

	data, _ := json.Marshal(cached)
	_, _ = idx.db.ExecContext(ctx, `INSERT OR REPLACE INTO profiles (did, data, expires_at) VALUES (?, ?, ?)`,
		did, string(data), cached.ExpiresAt.Format(time.RFC3339Nano))

	if profile != nil && profile.Handle != "" {
		// Drop any prior row pointing this DID at a different handle (handle change).
		_, _ = idx.db.ExecContext(ctx,
			`DELETE FROM did_by_handle WHERE did = ? AND handle != ?`, did, profile.Handle)
		// Last writer wins on handle — this naturally resolves handle reassignment
		// from an old DID to a new one, since the new profile's INSERT OR REPLACE
		// overwrites the old DID's mapping.
		_, _ = idx.db.ExecContext(ctx,
			`INSERT OR REPLACE INTO did_by_handle (handle, did, updated_at) VALUES (?, ?, ?)`,
			profile.Handle, did, now.Format(time.RFC3339Nano))
	}
}

// GetDIDByHandle looks up a DID from the handle index. Returns the DID and
// true if found, or empty string and false if not indexed.
//
// Backed by the did_by_handle table — last-writer-wins, so a handle that has
// been reassigned to a new DID resolves to that new DID once the new profile
// is observed (via the firehose profile watcher or a GetProfile call).
func (idx *FeedIndex) GetDIDByHandle(ctx context.Context, handle string) (string, bool) {
	var did string
	err := idx.db.QueryRowContext(ctx,
		`SELECT did FROM did_by_handle WHERE handle = ?`, handle).Scan(&did)
	if err != nil || did == "" {
		return "", false
	}
	return did, true
}

// InvalidateProfile removes a DID's profile from both the in-memory and persistent
// caches. The next GetProfile call will re-fetch from the API.
func (idx *FeedIndex) InvalidateProfile(did string) {
	idx.profileCacheMu.Lock()
	delete(idx.profileCache, did)
	idx.profileCacheMu.Unlock()

	_, _ = idx.db.Exec(`DELETE FROM profiles WHERE did = ?`, did)
	_, _ = idx.db.Exec(`DELETE FROM did_by_handle WHERE did = ?`, did)
}

// ProfileCachedInMemory reports whether the in-memory profile cache currently
// holds an entry for did. Test-only helper for asserting eviction behavior
// without going through GetProfile (which would fall through to the public API
// on a cache miss).
func (idx *FeedIndex) ProfileCachedInMemory(did string) bool {
	idx.profileCacheMu.RLock()
	defer idx.profileCacheMu.RUnlock()
	_, ok := idx.profileCache[did]
	return ok
}

// RefreshProfile fetches a profile from the API and stores it in both caches.
// Used by the profile watcher to keep the cache warm on firehose events.
func (idx *FeedIndex) RefreshProfile(ctx context.Context, did string) {
	profile, err := idx.publicClient.GetProfile(ctx, did)
	if err != nil {
		log.Warn().Err(err).Str("did", did).Msg("profile refresh: failed to fetch, invalidating instead")
		idx.InvalidateProfile(did)
		return
	}

	idx.storeProfile(ctx, did, profile)
}

// RefreshAllProfiles re-fetches every cached profile from the AppView and
// rewrites the result to both the in-memory and persistent caches. Used as a
// non-destructive recovery for stale handle data — anything an identity-event
// race left behind gets corrected since AppView has caught up by the time the
// admin runs this. Returns refreshed and failed counts. Honours ctx cancel.
func (idx *FeedIndex) RefreshAllProfiles(ctx context.Context) (refreshed, failed int) {
	rows, err := idx.db.QueryContext(ctx, `SELECT did FROM profiles`)
	if err != nil {
		log.Warn().Err(err).Msg("refresh all profiles: query failed")
		return 0, 0
	}
	dids := make([]string, 0, 128)
	for rows.Next() {
		var did string
		if err := rows.Scan(&did); err == nil {
			dids = append(dids, did)
		}
	}
	rows.Close()

	log.Info().Int("count", len(dids)).Msg("refresh all profiles: starting")

	for _, did := range dids {
		if err := ctx.Err(); err != nil {
			log.Warn().Err(err).Int("refreshed", refreshed).Int("failed", failed).Int("remaining", len(dids)-refreshed-failed).Msg("refresh all profiles: cancelled")
			return
		}

		var oldHandle string
		idx.profileCacheMu.RLock()
		if cached, ok := idx.profileCache[did]; ok && cached.Profile != nil {
			oldHandle = cached.Profile.Handle
		}
		idx.profileCacheMu.RUnlock()

		profile, err := idx.publicClient.GetProfile(ctx, did)
		if err != nil {
			log.Warn().Err(err).Str("did", did).Msg("refresh all profiles: fetch failed")
			failed++
			continue
		}
		idx.storeProfile(ctx, did, profile)
		refreshed++

		newHandle := ""
		if profile != nil {
			newHandle = profile.Handle
		}
		if oldHandle != newHandle {
			log.Info().
				Str("did", did).
				Str("old_handle", oldHandle).
				Str("new_handle", newHandle).
				Msg("refresh all profiles: handle updated")
		} else {
			log.Debug().
				Str("did", did).
				Str("handle", newHandle).
				Msg("refresh all profiles: refreshed")
		}
	}
	log.Info().Int("refreshed", refreshed).Int("failed", failed).Msg("refresh all profiles: complete")
	return refreshed, failed
}

// InvalidatePublicCachesForDID drops the public client's cached PDS endpoint
// and any handle→DID mappings pointing at this DID. Used when an account is
// deleted/takendown so subsequent lookups don't keep hitting the tombstoned DID.
func (idx *FeedIndex) InvalidatePublicCachesForDID(did string) {
	if idx.publicClient != nil {
		idx.publicClient.InvalidateDID(did)
	}
}

// OnIdentityEvent reconciles caches when a Jetstream identity event reports
// that a DID's handle has changed. It is the only path through which a handle
// can be reassigned from one DID to another (handle release + reclaim by a
// different account), so this is where stale mappings must be evicted.
//
// Steps:
//  1. Look up this DID's previously cached handle (the old handle).
//  2. Find any *other* DID whose cached profile still claims the new handle —
//     that's the prior owner; invalidate its profile and resolver entries.
//  3. Drop the old handle from the resolver cache (it may now resolve to
//     someone else, or to nothing).
//  4. Drop the new handle from the resolver cache so the next ResolveHandle
//     re-fetches from the directory.
//  5. Refresh this DID's profile via the API; storeProfile then writes the
//     authoritative did_by_handle row. The AppView's getProfile lags the relay
//     during a handle change, so we overwrite the returned Handle with the
//     value from the Jetstream event (which the relay verified bidirectionally
//     before emitting) to avoid caching a stale handle indefinitely.
func (idx *FeedIndex) OnIdentityEvent(ctx context.Context, did, newHandle string) {
	var oldHandle string
	idx.profileCacheMu.RLock()
	if cached, ok := idx.profileCache[did]; ok && cached.Profile != nil {
		oldHandle = cached.Profile.Handle
	}
	idx.profileCacheMu.RUnlock()
	if oldHandle == "" {
		// Fall back to persistent store.
		var dataStr string
		if err := idx.db.QueryRowContext(ctx, `SELECT data FROM profiles WHERE did = ?`, did).Scan(&dataStr); err == nil {
			cached := &CachedProfile{}
			if err := json.Unmarshal([]byte(dataStr), cached); err == nil && cached.Profile != nil {
				oldHandle = cached.Profile.Handle
			}
		}
	}

	if newHandle != "" {
		// Evict any prior owner of newHandle (other than `did` itself).
		var priorDID string
		err := idx.db.QueryRowContext(ctx,
			`SELECT did FROM did_by_handle WHERE handle = ? AND did != ?`, newHandle, did).Scan(&priorDID)
		if err == nil && priorDID != "" {
			log.Warn().
				Str("handle", newHandle).
				Str("prior_did", priorDID).
				Str("new_did", did).
				Msg("identity event: handle reassigned, invalidating prior owner")
			idx.InvalidateProfile(priorDID)
			idx.publicClient.InvalidateDID(priorDID)
		}
	}

	if oldHandle != "" && oldHandle != newHandle {
		idx.publicClient.InvalidateHandle(oldHandle)
	}
	if newHandle != "" {
		idx.publicClient.InvalidateHandle(newHandle)
	}
	idx.publicClient.InvalidateDID(did)

	profile, err := idx.publicClient.GetProfile(ctx, did)
	if err != nil {
		log.Warn().Err(err).Str("did", did).Msg("identity event: profile refresh failed, invalidating instead")
		idx.InvalidateProfile(did)
		return
	}
	if profile != nil && newHandle != "" && profile.Handle != newHandle {
		log.Info().
			Str("did", did).
			Str("appview_handle", profile.Handle).
			Str("event_handle", newHandle).
			Msg("identity event: overriding stale appview handle with relay-verified value")
		profile.Handle = newHandle
	}
	idx.storeProfile(ctx, did, profile)
}

// GetKnownDIDs returns all DIDs that have created Arabica records
func (idx *FeedIndex) GetKnownDIDs(ctx context.Context) ([]string, error) {
	rows, err := idx.db.QueryContext(ctx, `SELECT did FROM known_dids`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dids []string
	for rows.Next() {
		var did string
		if err := rows.Scan(&did); err != nil {
			continue
		}
		dids = append(dids, did)
	}
	return dids, rows.Err()
}

// ListRecordsByCollection returns all indexed records for a given collection,
// ordered by created_at DESC (newest first).
func (idx *FeedIndex) ListRecordsByCollection(ctx context.Context, collection string) ([]IndexedRecord, error) {
	return idx.listRecordsByCollection(ctx, collection, "DESC")
}

// ListRecordsByCollectionOldest returns all indexed records for a given collection,
// ordered by created_at ASC (oldest first).
func (idx *FeedIndex) ListRecordsByCollectionOldest(ctx context.Context, collection string) ([]IndexedRecord, error) {
	return idx.listRecordsByCollection(ctx, collection, "ASC")
}

func (idx *FeedIndex) listRecordsByCollection(ctx context.Context, collection string, dir string) ([]IndexedRecord, error) {
	rows, err := idx.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT uri, did, collection, rkey, record, cid, indexed_at, created_at
		FROM records WHERE collection = ? ORDER BY created_at %s
	`, dir), collection)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []IndexedRecord
	for rows.Next() {
		var rec IndexedRecord
		var recordStr, indexedAtStr, createdAtStr string
		if err := rows.Scan(&rec.URI, &rec.DID, &rec.Collection, &rec.RKey,
			&recordStr, &rec.CID, &indexedAtStr, &createdAtStr); err != nil {
			continue
		}
		rec.Record = json.RawMessage(recordStr)
		rec.IndexedAt, _ = time.Parse(time.RFC3339Nano, indexedAtStr)
		rec.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
		records = append(records, rec)
	}
	return records, rows.Err()
}

// CountReferencesToURI returns how many records have a sourceRef pointing to the given URI.
// This searches the JSON record field across all collections.
func (idx *FeedIndex) CountReferencesToURI(ctx context.Context, uri string) (int, error) {
	var count int
	err := idx.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM records
		WHERE json_extract(record, '$.sourceRef') = ?
	`, uri).Scan(&count)
	return count, err
}

// RecordCount returns the total number of indexed records
func (idx *FeedIndex) RecordCount() int {
	var count int
	_ = idx.db.QueryRow(`SELECT COUNT(*) FROM records`).Scan(&count)
	return count
}

// KnownDIDCount returns the number of unique DIDs in the index
func (idx *FeedIndex) KnownDIDCount() int {
	var count int
	_ = idx.db.QueryRow(`SELECT COUNT(*) FROM known_dids`).Scan(&count)
	return count
}

// TotalLikeCount returns the total number of likes indexed
func (idx *FeedIndex) TotalLikeCount() int {
	var count int
	_ = idx.db.QueryRow(`SELECT COUNT(*) FROM likes`).Scan(&count)
	return count
}

// TotalCommentCount returns the total number of comments indexed
func (idx *FeedIndex) TotalCommentCount() int {
	var count int
	_ = idx.db.QueryRow(`SELECT COUNT(*) FROM comments`).Scan(&count)
	return count
}

// RecordCountByCollection returns a breakdown of record counts by collection type
func (idx *FeedIndex) RecordCountByCollection() map[string]int {
	counts := make(map[string]int)
	rows, err := idx.db.Query(`SELECT collection, COUNT(*) FROM records GROUP BY collection`)
	if err != nil {
		return counts
	}
	defer rows.Close()
	for rows.Next() {
		var collection string
		var count int
		if err := rows.Scan(&collection, &count); err == nil {
			counts[collection] = count
		}
	}
	return counts
}

// BrewCountsByRecipeURI returns a map of recipe AT-URI -> number of brews referencing that recipe.
// Uses SQLite json_extract to efficiently query the recipeRef field in brew records.
func (idx *FeedIndex) BrewCountsByRecipeURI(ctx context.Context) map[string]int {
	counts := make(map[string]int)
	rows, err := idx.db.QueryContext(ctx, `
		SELECT json_extract(record, '$.recipeRef') as recipe_uri, COUNT(*) as cnt
		FROM records
		WHERE collection = 'social.arabica.alpha.brew'
		  AND recipe_uri IS NOT NULL AND recipe_uri != ''
		GROUP BY recipe_uri
	`)
	if err != nil {
		return counts
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		var count int
		if err := rows.Scan(&uri, &count); err == nil {
			counts[uri] = count
		}
	}
	return counts
}

// refCounts returns a map of ref AT-URI -> count of records in the given collection
// that reference it via the specified JSON field. If did is non-empty, only records
// owned by that DID are counted.
func (idx *FeedIndex) refCounts(ctx context.Context, collection, jsonField, did string) map[string]int {
	counts := make(map[string]int)
	var rows *sql.Rows
	var err error
	if did != "" {
		rows, err = idx.db.QueryContext(ctx, fmt.Sprintf(`
			SELECT json_extract(record, '$.%s') as ref_uri, COUNT(*) as cnt
			FROM records
			WHERE collection = ? AND did = ?
			  AND ref_uri IS NOT NULL AND ref_uri != ''
			GROUP BY ref_uri
		`, jsonField), collection, did)
	} else {
		rows, err = idx.db.QueryContext(ctx, fmt.Sprintf(`
			SELECT json_extract(record, '$.%s') as ref_uri, COUNT(*) as cnt
			FROM records
			WHERE collection = ?
			  AND ref_uri IS NOT NULL AND ref_uri != ''
			GROUP BY ref_uri
		`, jsonField), collection)
	}
	if err != nil {
		return counts
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		var count int
		if err := rows.Scan(&uri, &count); err == nil {
			counts[uri] = count
		}
	}
	return counts
}

// BrewCountsByBeanURI returns a map of bean AT-URI -> number of brews referencing that bean.
// If did is non-empty, only brews owned by that DID are counted.
func (idx *FeedIndex) BrewCountsByBeanURI(ctx context.Context, did string) map[string]int {
	return idx.refCounts(ctx, "social.arabica.alpha.brew", "beanRef", did)
}

// BrewCountsByGrinderURI returns a map of grinder AT-URI -> number of brews referencing that grinder.
// If did is non-empty, only brews owned by that DID are counted.
func (idx *FeedIndex) BrewCountsByGrinderURI(ctx context.Context, did string) map[string]int {
	return idx.refCounts(ctx, "social.arabica.alpha.brew", "grinderRef", did)
}

// BrewCountsByBrewerURI returns a map of brewer AT-URI -> number of brews referencing that brewer.
// If did is non-empty, only brews owned by that DID are counted.
func (idx *FeedIndex) BrewCountsByBrewerURI(ctx context.Context, did string) map[string]int {
	return idx.refCounts(ctx, "social.arabica.alpha.brew", "brewerRef", did)
}

// BeanCountsByRoasterURI returns a map of roaster AT-URI -> number of beans referencing that roaster.
// If did is non-empty, only beans owned by that DID are counted.
func (idx *FeedIndex) BeanCountsByRoasterURI(ctx context.Context, did string) map[string]int {
	return idx.refCounts(ctx, "social.arabica.alpha.bean", "roasterRef", did)
}

// RatingStats holds aggregated rating statistics for an entity.
type RatingStats struct {
	Average float64
	Count   int
}

// refAvgRatings returns a map of ref URI -> RatingStats for brew records,
// grouped by the given JSON reference field. If did is non-empty, only brews
// owned by that DID are included.
func (idx *FeedIndex) refAvgRatings(ctx context.Context, jsonField, did string) map[string]RatingStats {
	stats := make(map[string]RatingStats)
	var rows *sql.Rows
	var err error
	if did != "" {
		rows, err = idx.db.QueryContext(ctx, fmt.Sprintf(`
			SELECT json_extract(record, '$.%s') as ref_uri,
			       AVG(json_extract(record, '$.rating')) as avg_rating,
			       COUNT(*) as cnt
			FROM records
			WHERE collection = 'social.arabica.alpha.brew'
			  AND did = ?
			  AND ref_uri IS NOT NULL AND ref_uri != ''
			  AND json_extract(record, '$.rating') IS NOT NULL
			GROUP BY ref_uri
		`, jsonField), did)
	} else {
		rows, err = idx.db.QueryContext(ctx, fmt.Sprintf(`
			SELECT json_extract(record, '$.%s') as ref_uri,
			       AVG(json_extract(record, '$.rating')) as avg_rating,
			       COUNT(*) as cnt
			FROM records
			WHERE collection = 'social.arabica.alpha.brew'
			  AND ref_uri IS NOT NULL AND ref_uri != ''
			  AND json_extract(record, '$.rating') IS NOT NULL
			GROUP BY ref_uri
		`, jsonField))
	}
	if err != nil {
		return stats
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		var avg float64
		var count int
		if err := rows.Scan(&uri, &avg, &count); err == nil {
			stats[uri] = RatingStats{Average: avg, Count: count}
		}
	}
	return stats
}

// AvgBrewRatingByBeanURI returns a map of bean AT-URI -> RatingStats from brew ratings.
// If did is non-empty, only brews owned by that DID are included.
func (idx *FeedIndex) AvgBrewRatingByBeanURI(ctx context.Context, did string) map[string]RatingStats {
	return idx.refAvgRatings(ctx, "beanRef", did)
}

// AvgBrewRatingByRoasterURI returns a map of roaster AT-URI -> RatingStats,
// aggregated from brew ratings through the bean's roaster reference.
// If did is non-empty, only brews owned by that DID are included.
func (idx *FeedIndex) AvgBrewRatingByRoasterURI(ctx context.Context, did string) map[string]RatingStats {
	stats := make(map[string]RatingStats)
	var rows *sql.Rows
	var err error
	if did != "" {
		rows, err = idx.db.QueryContext(ctx, `
			SELECT json_extract(beans.record, '$.roasterRef') as roaster_uri,
			       AVG(json_extract(brews.record, '$.rating')) as avg_rating,
			       COUNT(*) as cnt
			FROM records brews
			JOIN records beans
			  ON beans.uri = json_extract(brews.record, '$.beanRef')
			  AND beans.collection = 'social.arabica.alpha.bean'
			WHERE brews.collection = 'social.arabica.alpha.brew'
			  AND brews.did = ?
			  AND json_extract(brews.record, '$.rating') IS NOT NULL
			  AND roaster_uri IS NOT NULL AND roaster_uri != ''
			GROUP BY roaster_uri
		`, did)
	} else {
		rows, err = idx.db.QueryContext(ctx, `
			SELECT json_extract(beans.record, '$.roasterRef') as roaster_uri,
			       AVG(json_extract(brews.record, '$.rating')) as avg_rating,
			       COUNT(*) as cnt
			FROM records brews
			JOIN records beans
			  ON beans.uri = json_extract(brews.record, '$.beanRef')
			  AND beans.collection = 'social.arabica.alpha.bean'
			WHERE brews.collection = 'social.arabica.alpha.brew'
			  AND json_extract(brews.record, '$.rating') IS NOT NULL
			  AND roaster_uri IS NOT NULL AND roaster_uri != ''
			GROUP BY roaster_uri
		`)
	}
	if err != nil {
		return stats
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		var avg float64
		var count int
		if err := rows.Scan(&uri, &avg, &count); err == nil {
			stats[uri] = RatingStats{Average: avg, Count: count}
		}
	}
	return stats
}

// GetProfileStatsVisibility returns the profile stats visibility settings for a user.
// Returns default (all public) if no settings are stored.
func (idx *FeedIndex) GetProfileStatsVisibility(ctx context.Context, did string) arabica.ProfileStatsVisibility {
	defaults := arabica.DefaultProfileStatsVisibility()
	if did == "" {
		return defaults
	}
	var raw string
	err := idx.db.QueryRowContext(ctx,
		`SELECT profile_stats_visibility FROM user_settings WHERE did = ?`, did,
	).Scan(&raw)
	if err != nil {
		return defaults
	}
	var settings arabica.ProfileStatsVisibility
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return defaults
	}
	// Fill in defaults for any empty fields
	if settings.BeanAvgRating == "" {
		settings.BeanAvgRating = arabica.VisibilityPublic
	}
	if settings.RoasterAvgRating == "" {
		settings.RoasterAvgRating = arabica.VisibilityPublic
	}
	return settings
}

// SetProfileStatsVisibility saves the profile stats visibility settings for a user.
func (idx *FeedIndex) SetProfileStatsVisibility(ctx context.Context, did string, settings arabica.ProfileStatsVisibility) error {
	raw, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	_, err = idx.db.ExecContext(ctx,
		`INSERT INTO user_settings (did, profile_stats_visibility) VALUES (?, ?)
		 ON CONFLICT(did) DO UPDATE SET profile_stats_visibility = excluded.profile_stats_visibility`,
		did, string(raw),
	)
	return err
}

func formatTimeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 48*time.Hour:
		return "yesterday"
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	default:
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
}

// IsBackfilled checks if a DID has already been backfilled
func (idx *FeedIndex) IsBackfilled(ctx context.Context, did string) bool {
	var exists int
	err := idx.db.QueryRowContext(ctx, `SELECT 1 FROM backfilled WHERE did = ?`, did).Scan(&exists)
	return err == nil
}

// BackfilledDIDs returns the set of all DIDs that have been backfilled.
func (idx *FeedIndex) BackfilledDIDs(ctx context.Context) (map[string]struct{}, error) {
	rows, err := idx.db.QueryContext(ctx, `SELECT did FROM backfilled`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]struct{})
	for rows.Next() {
		var did string
		if err := rows.Scan(&did); err != nil {
			return nil, err
		}
		result[did] = struct{}{}
	}
	return result, rows.Err()
}

// MarkBackfilled marks a DID as backfilled with current timestamp
func (idx *FeedIndex) MarkBackfilled(ctx context.Context, did string) error {
	_, err := idx.db.ExecContext(ctx, `INSERT OR IGNORE INTO backfilled (did, backfilled_at) VALUES (?, ?)`,
		did, time.Now().Format(time.RFC3339))
	return err
}

// BackfillUser fetches all existing records for a DID and adds them to the index
// BackfillUser fetches every record this user has in the supplied
// collections and indexes them into the witness cache. Collections
// typically come from app.NSIDs() so backfill tracks the running app's
// entity set.
func (idx *FeedIndex) BackfillUser(ctx context.Context, did string, collections []string) error {
	if idx.IsBackfilled(ctx, did) {
		log.Debug().Str("did", did).Msg("DID already backfilled, skipping")
		return nil
	}

	ctx, span := tracing.HandlerSpan(ctx, "backfill.user",
		attribute.String("backfill.did", did),
	)
	defer span.End()

	log.Info().Str("did", did).Msg("backfilling user records")

	recordCount := 0
	for _, collection := range collections {
		recs, _, err := idx.publicClient.ListPublicRecords(ctx, did, collection, atp.ListPublicRecordsOpts{Limit: 100, Reverse: true})
		if err != nil {
			log.Warn().Err(err).Str("did", did).Str("collection", collection).Msg("failed to list records for backfill")
			continue
		}

		for _, record := range recs {
			parts := strings.Split(record.URI, "/")
			if len(parts) < 3 {
				continue
			}
			rkey := parts[len(parts)-1]

			recordJSON, err := json.Marshal(record.Value)
			if err != nil {
				continue
			}

			if err := idx.UpsertRecord(ctx, did, collection, rkey, record.CID, recordJSON, 0); err != nil {
				log.Warn().Err(err).Str("uri", record.URI).Msg("failed to upsert record during backfill")
				continue
			}
			recordCount++

			switch {
			case strings.HasSuffix(collection, ".like"):
				if subject, ok := record.Value["subject"].(map[string]any); ok {
					if subjectURI, ok := subject["uri"].(string); ok {
						if err := idx.UpsertLike(ctx, did, rkey, subjectURI); err != nil {
							log.Warn().Err(err).Str("uri", record.URI).Msg("failed to index like during backfill")
						}
					}
				}
			case strings.HasSuffix(collection, ".comment"):
				if subject, ok := record.Value["subject"].(map[string]any); ok {
					if subjectURI, ok := subject["uri"].(string); ok {
						text, _ := record.Value["text"].(string)
						var createdAt time.Time
						if createdAtStr, ok := record.Value["createdAt"].(string); ok {
							if parsed, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
								createdAt = parsed
							} else {
								createdAt = time.Now()
							}
						} else {
							createdAt = time.Now()
						}
						var parentURI string
						if parent, ok := record.Value["parent"].(map[string]any); ok {
							parentURI, _ = parent["uri"].(string)
						}
						if err := idx.UpsertComment(ctx, did, rkey, subjectURI, parentURI, record.CID, text, createdAt); err != nil {
							log.Warn().Err(err).Str("uri", record.URI).Msg("failed to index comment during backfill")
						}
					}
				}
			}
		}
	}

	if err := idx.MarkBackfilled(ctx, did); err != nil {
		log.Warn().Err(err).Str("did", did).Msg("failed to mark DID as backfilled")
	}

	log.Info().Str("did", did).Int("record_count", recordCount).Msg("backfill complete")
	return nil
}

// ========== Like Indexing Methods ==========

// UpsertLike adds or updates a like in the index
func (idx *FeedIndex) UpsertLike(ctx context.Context, actorDID, rkey, subjectURI string) error {
	_, err := idx.db.ExecContext(ctx, `INSERT OR IGNORE INTO likes (subject_uri, actor_did, rkey) VALUES (?, ?, ?)`,
		subjectURI, actorDID, rkey)
	return err
}

// DeleteLike removes a like from the index
func (idx *FeedIndex) DeleteLike(ctx context.Context, actorDID, subjectURI string) error {
	_, err := idx.db.ExecContext(ctx, `DELETE FROM likes WHERE subject_uri = ? AND actor_did = ?`,
		subjectURI, actorDID)
	return err
}

// GetLikeCount returns the number of likes for a record
func (idx *FeedIndex) GetLikeCount(ctx context.Context, subjectURI string) int {
	var count int
	_ = idx.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM likes WHERE subject_uri = ?`, subjectURI).Scan(&count)
	return count
}

// HasUserLiked checks if a user has liked a specific record
func (idx *FeedIndex) HasUserLiked(ctx context.Context, actorDID, subjectURI string) bool {
	var exists int
	err := idx.db.QueryRowContext(ctx, `SELECT 1 FROM likes WHERE actor_did = ? AND subject_uri = ? LIMIT 1`,
		actorDID, subjectURI).Scan(&exists)
	return err == nil
}

// GetUserLikeRKey returns the rkey of a user's like for a specific record, or empty string if not found
func (idx *FeedIndex) GetUserLikeRKey(ctx context.Context, actorDID, subjectURI string) string {
	var rkey string
	err := idx.db.QueryRowContext(ctx, `SELECT rkey FROM likes WHERE actor_did = ? AND subject_uri = ?`,
		actorDID, subjectURI).Scan(&rkey)
	if err != nil {
		return ""
	}
	return rkey
}

// ========== Batch Query Methods ==========

// placeholders returns a string of "?,?,?" for n items and a corresponding []any slice.
func placeholders(uris []string) (string, []any) {
	ph := make([]string, len(uris))
	args := make([]any, len(uris))
	for i, u := range uris {
		ph[i] = "?"
		args[i] = u
	}
	return strings.Join(ph, ","), args
}

// GetLikeCountsBatch returns like counts for multiple subject URIs in a single query.
func (idx *FeedIndex) GetLikeCountsBatch(ctx context.Context, uris []string) map[string]int {
	counts := make(map[string]int, len(uris))
	if len(uris) == 0 {
		return counts
	}
	ph, args := placeholders(uris)
	rows, err := idx.db.QueryContext(ctx,
		`SELECT subject_uri, COUNT(*) FROM likes WHERE subject_uri IN (`+ph+`) GROUP BY subject_uri`, args...)
	if err != nil {
		return counts
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		var count int
		if err := rows.Scan(&uri, &count); err == nil {
			counts[uri] = count
		}
	}
	return counts
}

// HasUserLikedBatch checks if a user has liked multiple records in a single query.
func (idx *FeedIndex) HasUserLikedBatch(ctx context.Context, actorDID string, uris []string) map[string]bool {
	liked := make(map[string]bool, len(uris))
	if len(uris) == 0 || actorDID == "" {
		return liked
	}
	ph, args := placeholders(uris)
	// Prepend actorDID to args
	allArgs := make([]any, 0, len(args)+1)
	allArgs = append(allArgs, actorDID)
	allArgs = append(allArgs, args...)
	rows, err := idx.db.QueryContext(ctx,
		`SELECT subject_uri FROM likes WHERE actor_did = ? AND subject_uri IN (`+ph+`)`, allArgs...)
	if err != nil {
		return liked
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		if err := rows.Scan(&uri); err == nil {
			liked[uri] = true
		}
	}
	return liked
}

// GetCommentCountsBatch returns comment counts for multiple subject URIs in a single query.
func (idx *FeedIndex) GetCommentCountsBatch(ctx context.Context, uris []string) map[string]int {
	counts := make(map[string]int, len(uris))
	if len(uris) == 0 {
		return counts
	}
	ph, args := placeholders(uris)
	rows, err := idx.db.QueryContext(ctx,
		`SELECT subject_uri, COUNT(*) FROM comments WHERE subject_uri IN (`+ph+`) GROUP BY subject_uri`, args...)
	if err != nil {
		return counts
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		var count int
		if err := rows.Scan(&uri, &count); err == nil {
			counts[uri] = count
		}
	}
	return counts
}

// GetRecordsBatch retrieves multiple records by URI in a single query.
func (idx *FeedIndex) GetRecordsBatch(ctx context.Context, uris []string) map[string]*IndexedRecord {
	records := make(map[string]*IndexedRecord, len(uris))
	if len(uris) == 0 {
		return records
	}
	ph, args := placeholders(uris)
	rows, err := idx.db.QueryContext(ctx,
		`SELECT uri, did, collection, rkey, record, cid, indexed_at, created_at FROM records WHERE uri IN (`+ph+`)`, args...)
	if err != nil {
		return records
	}
	defer rows.Close()
	for rows.Next() {
		var rec IndexedRecord
		var recordStr, indexedAtStr, createdAtStr string
		if err := rows.Scan(&rec.URI, &rec.DID, &rec.Collection, &rec.RKey,
			&recordStr, &rec.CID, &indexedAtStr, &createdAtStr); err != nil {
			continue
		}
		rec.Record = json.RawMessage(recordStr)
		rec.IndexedAt, _ = time.Parse(time.RFC3339Nano, indexedAtStr)
		rec.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
		records[rec.URI] = &rec
	}
	return records
}

// IndexedComment represents a comment stored in the index
type IndexedComment struct {
	RKey       string    `json:"rkey"`
	SubjectURI string    `json:"subject_uri"`
	Text       string    `json:"text"`
	ActorDID   string    `json:"actor_did"`
	CreatedAt  time.Time `json:"created_at"`
	// Parent fields for threading (stored)
	ParentURI  string `json:"parent_uri,omitempty"`
	ParentRKey string `json:"parent_rkey,omitempty"`
	CID        string `json:"cid,omitempty"`
	// Computed fields (populated on retrieval, not stored)
	Depth   int              `json:"-"` // Nesting depth (0 = top-level, 1 = reply, 2+ = nested reply)
	Replies []IndexedComment `json:"-"` // Child comments (for tree building)
	// Profile fields (populated on retrieval, not stored)
	Handle      string  `json:"-"`
	DisplayName *string `json:"-"`
	Avatar      *string `json:"-"`
	// Like fields (populated on retrieval, not stored)
	LikeCount int  `json:"-"`
	IsLiked   bool `json:"-"`
}

// UpsertComment adds or updates a comment in the index
func (idx *FeedIndex) UpsertComment(ctx context.Context, actorDID, rkey, subjectURI, parentURI, cid, text string, createdAt time.Time) error {
	// Extract parent rkey from parent URI if present
	var parentRKey string
	if parentURI != "" {
		parts := strings.Split(parentURI, "/")
		if len(parts) > 0 {
			parentRKey = parts[len(parts)-1]
		}
	}

	_, err := idx.db.ExecContext(ctx, `
		INSERT INTO comments (actor_did, rkey, subject_uri, parent_uri, parent_rkey, cid, text, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(actor_did, rkey) DO UPDATE SET
			subject_uri = excluded.subject_uri,
			parent_uri = excluded.parent_uri,
			parent_rkey = excluded.parent_rkey,
			cid = excluded.cid,
			text = excluded.text,
			created_at = excluded.created_at
	`, actorDID, rkey, subjectURI, parentURI, parentRKey, cid, text, createdAt.Format(time.RFC3339Nano))
	return err
}

// DeleteComment removes a comment from the index
func (idx *FeedIndex) DeleteComment(ctx context.Context, actorDID, rkey, subjectURI string) error {
	_, err := idx.db.ExecContext(ctx, `DELETE FROM comments WHERE actor_did = ? AND rkey = ?`, actorDID, rkey)
	return err
}

// GetCommentCount returns the number of comments on a record
func (idx *FeedIndex) GetCommentCount(ctx context.Context, subjectURI string) int {
	var count int
	_ = idx.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM comments WHERE subject_uri = ?`, subjectURI).Scan(&count)
	return count
}

// GetCommentsForSubject returns all comments for a specific record, ordered by creation time
func (idx *FeedIndex) GetCommentsForSubject(ctx context.Context, subjectURI string, limit int, viewerDID string) []IndexedComment {
	query := `SELECT actor_did, rkey, subject_uri, parent_uri, parent_rkey, cid, text, created_at
		FROM comments WHERE subject_uri = ? ORDER BY created_at`
	var args []any
	args = append(args, subjectURI)
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := idx.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var comments []IndexedComment
	for rows.Next() {
		var c IndexedComment
		var createdAtStr string
		if err := rows.Scan(&c.ActorDID, &c.RKey, &c.SubjectURI, &c.ParentURI, &c.ParentRKey,
			&c.CID, &c.Text, &createdAtStr); err != nil {
			continue
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
		comments = append(comments, c)
	}

	// Batch-fetch profiles and social data for all comments
	commentURIs := make([]string, len(comments))
	didSet := make(map[string]struct{}, len(comments))
	for i, c := range comments {
		commentURIs[i] = fmt.Sprintf("at://%s/%s/%s", c.ActorDID, idx.commentCollection(), c.RKey)
		didSet[c.ActorDID] = struct{}{}
	}
	likeCounts := idx.GetLikeCountsBatch(ctx, commentURIs)
	var likedByViewer map[string]bool
	if viewerDID != "" {
		likedByViewer = idx.HasUserLikedBatch(ctx, viewerDID, commentURIs)
	}

	for i := range comments {
		profile, err := idx.GetProfile(ctx, comments[i].ActorDID)
		if err != nil {
			comments[i].Handle = comments[i].ActorDID
		} else {
			comments[i].Handle = profile.Handle
			comments[i].DisplayName = profile.DisplayName
			comments[i].Avatar = profile.Avatar
		}

		comments[i].LikeCount = likeCounts[commentURIs[i]]
		if likedByViewer != nil {
			comments[i].IsLiked = likedByViewer[commentURIs[i]]
		}
	}

	return comments
}

// GetThreadedCommentsForSubject returns comments for a record in threaded order with depth
func (idx *FeedIndex) GetThreadedCommentsForSubject(ctx context.Context, subjectURI string, limit int, viewerDID string) []IndexedComment {
	allComments := idx.GetCommentsForSubject(ctx, subjectURI, 0, viewerDID)

	if len(allComments) == 0 {
		return nil
	}

	// Build a map of comment rkey -> comment for quick lookup
	commentMap := make(map[string]*IndexedComment)
	for i := range allComments {
		commentMap[allComments[i].RKey] = &allComments[i]
	}

	// Build parent -> children map
	childrenMap := make(map[string][]*IndexedComment)
	var topLevel []*IndexedComment

	for i := range allComments {
		comment := &allComments[i]
		if comment.ParentRKey == "" {
			topLevel = append(topLevel, comment)
		} else {
			childrenMap[comment.ParentRKey] = append(childrenMap[comment.ParentRKey], comment)
		}
	}

	// Sort top-level comments by creation time (oldest first)
	sort.Slice(topLevel, func(i, j int) bool {
		return topLevel[i].CreatedAt.Before(topLevel[j].CreatedAt)
	})

	// Sort children within each parent by creation time
	for _, children := range childrenMap {
		sort.Slice(children, func(i, j int) bool {
			return children[i].CreatedAt.Before(children[j].CreatedAt)
		})
	}

	// Flatten the tree in depth-first order
	var result []IndexedComment
	var flatten func(comment *IndexedComment, depth int)
	flatten = func(comment *IndexedComment, depth int) {
		if limit > 0 && len(result) >= limit {
			return
		}
		visualDepth := min(depth, 2)
		comment.Depth = visualDepth
		result = append(result, *comment)

		if children, ok := childrenMap[comment.RKey]; ok {
			for _, child := range children {
				flatten(child, depth+1)
			}
		}
	}

	for _, comment := range topLevel {
		flatten(comment, 0)
	}

	return result
}
