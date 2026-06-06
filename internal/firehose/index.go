package firehose

import (
	"context"
	"database/sql"
	_ "embed"
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
	"tangled.org/arabica.social/arabica/internal/lexicons"
	"tangled.org/arabica.social/arabica/internal/profileprefs"
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
	db             *sql.DB
	publicClient   *atp.PublicClient
	profileTTL     time.Duration
	profileStorage *profileIndexStorage
	notifications  *notificationIndexStorage
	social         *socialIndexStorage
	witness        *witnessRecordStorage

	// commentNSID is the comment collection this index's binary serves
	// (e.g. social.arabica.alpha.comment or social.oolong.alpha.comment).
	// Used when rebuilding comment AT-URIs from indexed rows. Falls back
	// to the Arabica comment collection when unset for backwards-compat with tests
	// that construct a FeedIndex directly via NewFeedIndex.
	commentNSID string

	// App-scoped feed collections. The running app passes its descriptors at
	// construction time so feed queries don't depend on package-global entity
	// registration side effects or include sister-app collections.
	recordTypeToNSID    map[lexicons.RecordType]string
	feedableCollections []string

	// In-memory cache for hot data
	profileCache   map[string]*CachedProfile
	profileCacheMu sync.RWMutex

	ready   bool
	readyMu sync.RWMutex
}

type FeedIndexOption func(*feedIndexConfig)

type feedIndexConfig struct {
	feedableDescriptors []*entities.Descriptor
}

// WithFeedableDescriptors configures which app-owned entity descriptors should
// appear in feed queries. Passing app.Descriptors keeps one FeedIndex scoped to
// the app whose SQLite database it serves.
func WithFeedableDescriptors(descriptors []*entities.Descriptor) FeedIndexOption {
	return func(cfg *feedIndexConfig) {
		cfg.feedableDescriptors = descriptors
	}
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
	return "social.arabica.alpha.comment"
}

// schemaNoTrailingPragma is the firehose SQLite schema. PRAGMAs stay in the
// connection DSN so the embedded schema remains portable SQL.
//
//go:embed sql/schema.sql
var schemaNoTrailingPragma string

const feedIndexSQLiteParams = "?_pragma=busy_timeout(5000)" +
	"&_pragma=journal_mode(WAL)" +
	"&_pragma=synchronous(NORMAL)" +
	"&_pragma=foreign_keys(ON)" +
	"&_pragma=temp_store(MEMORY)" +
	"&_pragma=mmap_size(134217728)" +
	"&_pragma=cache_size(-65536)"

// NewFeedIndex creates a new feed index backed by SQLite.
func NewFeedIndex(path string, profileTTL time.Duration, opts ...FeedIndexOption) (*FeedIndex, error) {

	if path == "" {
		return nil, fmt.Errorf("index path is required")
	}
	cfg := feedIndexConfig{feedableDescriptors: entities.All()}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	recordTypeToNSID, feedableCollections := feedableCollectionsForDescriptors(cfg.feedableDescriptors)

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create index directory: %w", err)
		}
	}

	dsn := fmt.Sprintf("file:%s%s", path, feedIndexSQLiteParams)
	db, err := otelsql.Open("sqlite", dsn,
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
	if _, err := db.Exec(`ALTER TABLE user_settings ADD COLUMN preferences TEXT NOT NULL DEFAULT '{}'`); err != nil {
		// Existing databases already have this column. SQLite reports that as an
		// error, so only fail for genuinely unexpected migration problems.
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			_ = db.Close()
			return nil, fmt.Errorf("failed to migrate user settings: %w", err)
		}
	}

	idx := &FeedIndex{
		db:                  db,
		publicClient:        atproto.NewPublicClient(),
		profileTTL:          profileTTL,
		profileStorage:      newProfileIndexStorage(db),
		notifications:       newNotificationIndexStorage(db),
		social:              newSocialIndexStorage(db),
		witness:             newWitnessRecordStorage(db),
		recordTypeToNSID:    recordTypeToNSID,
		feedableCollections: feedableCollections,
		profileCache:        make(map[string]*CachedProfile),
	}

	// One-time backfill: populate did_by_handle from any pre-existing profile rows
	// so handle resolution works for users observed before this table existed.
	if err := idx.profileStorage.backfillHandleIndex(); err != nil {
		log.Warn().Err(err).Msg("did_by_handle backfill failed; lookups will populate lazily")
	}
	idx.ensureExploreIndex(context.Background())

	// If the database already has records from a previous run, mark ready immediately
	// so the feed is served from persisted data while the firehose reconnects.
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM records`).Scan(&count); err == nil && count > 0 {
		idx.ready = true
	}

	return idx, nil
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
	return idx.witness.get(ctx, uri)
}

// ListWitnessRecords returns all indexed records for a DID+collection pair,
// ordered by created_at descending. Returns an empty slice when none are found.
func (idx *FeedIndex) ListWitnessRecords(ctx context.Context, did, collection string) ([]*atproto.WitnessRecord, error) {
	return idx.witness.list(ctx, did, collection, 0, 0)
}

// ListWitnessRecordsPaginated returns a page of cached records for a
// DID+collection pair, ordered by created_at descending.
// When limit <= 0, returns all records.
func (idx *FeedIndex) ListWitnessRecordsPaginated(ctx context.Context, did, collection string, offset, limit int) ([]*atproto.WitnessRecord, error) {
	return idx.witness.list(ctx, did, collection, offset, limit)
}

// CountWitnessRecords returns the total count of cached records for a
// DID+collection pair.
func (idx *FeedIndex) CountWitnessRecords(ctx context.Context, did, collection string) (int, error) {
	return idx.witness.count(ctx, did, collection)
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
	err := idx.witness.upsert(ctx, did, collection, rkey, cid, record, eventTime)
	if err != nil {
		return err
	}

	uri := atp.BuildATURI(did, collection, rkey)
	if err := idx.reindexExploreRecord(ctx, uri); err != nil {
		log.Warn().Err(err).Str("uri", uri).Msg("failed to refresh explore document")
		idx.markExploreDirty(ctx, err)
	} else if err := idx.reindexExploreDependents(ctx, uri, collection); err != nil {
		log.Warn().Err(err).Str("uri", uri).Msg("failed to refresh explore dependents")
		idx.markExploreDirty(ctx, err)
	} else if sourceRef := exploreSourceRef(record); sourceRef != "" {
		if err := idx.refreshExploreStats(ctx, sourceRef); err != nil {
			idx.markExploreDirty(ctx, err)
		}
	}
	if collection == idx.recordTypeToNSID[lexicons.RecordTypeBrew] {
		if err := idx.reindexExploreBrewReferences(ctx, record); err != nil {
			log.Warn().Err(err).Str("uri", uri).Msg("failed to refresh explore brew ratings")
			idx.markExploreDirty(ctx, err)
		}
	}

	return nil

}

// DeleteRecord removes a record from the index
func (idx *FeedIndex) DeleteRecord(ctx context.Context, did, collection, rkey string) error {
	uri := atp.BuildATURI(did, collection, rkey)
	var deletedRecord json.RawMessage
	var raw string
	if err := idx.db.QueryRowContext(ctx, `SELECT record FROM records WHERE uri = ?`, uri).Scan(&raw); err == nil {
		deletedRecord = json.RawMessage(raw)
	}

	err := idx.witness.delete(ctx, did, collection, rkey)
	if err == nil {
		if sourceRef := exploreSourceRef(deletedRecord); sourceRef != "" {
			if refreshErr := idx.refreshExploreStats(ctx, sourceRef); refreshErr != nil {
				idx.markExploreDirty(ctx, refreshErr)
			}
		}
		if collection == idx.recordTypeToNSID[lexicons.RecordTypeBrew] && len(deletedRecord) > 0 {
			if refreshErr := idx.reindexExploreBrewReferences(ctx, deletedRecord); refreshErr != nil {
				idx.markExploreDirty(ctx, refreshErr)
			}
		}
	}
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
	affectedExploreSubjects := idx.exploreSubjectsAffectedByDID(ctx, did, uriPrefix)
	for sourceRef := range idx.exploreSourceRefsByDID(ctx, did) {
		affectedExploreSubjects[sourceRef] = struct{}{}
	}

	tx, err := idx.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := idx.social.deleteAllForDID(ctx, tx, did, uriPrefix); err != nil {
		return fmt.Errorf("delete social data by did: %w", err)
	}

	stmts := []struct {
		sql  string
		args []any
	}{
		{`DELETE FROM records WHERE did = ?`, []any{did}},
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

	for subject := range affectedExploreSubjects {
		if err := idx.refreshExploreStats(ctx, subject); err != nil {
			idx.markExploreDirty(ctx, err)
		}
	}

	return nil
}

// UpsertWitnessRecord implements atproto.WitnessCache for write-through caching.
func (idx *FeedIndex) UpsertWitnessRecord(ctx context.Context, did, collection, rkey, cid string, record json.RawMessage) error {
	return idx.UpsertRecord(ctx, did, collection, rkey, cid, record, time.Now().UnixMicro())
}

// UpdateWitnessRecord updates an existing row's record body and indexed_at
// without touching cid. No-op when the row does not yet exist — the firehose
// event for this commit will INSERT it with the real cid.
func (idx *FeedIndex) UpdateWitnessRecord(ctx context.Context, did, collection, rkey string, record json.RawMessage) error {
	return idx.witness.update(ctx, did, collection, rkey, record)
}

// UpsertWitnessRecordBatch implements atproto.WitnessCache batch upsert.
// All records are inserted in a single transaction for efficiency.
func (idx *FeedIndex) UpsertWitnessRecordBatch(ctx context.Context, records []atproto.WitnessWriteRecord) error {
	return idx.witness.upsertBatch(ctx, records)
}

// DeleteWitnessRecord implements atproto.WitnessCache for write-through caching.
func (idx *FeedIndex) DeleteWitnessRecord(ctx context.Context, did, collection, rkey string) error {
	return idx.DeleteRecord(ctx, did, collection, rkey)
}

// GetRecord retrieves a single record by URI
func (idx *FeedIndex) GetRecord(ctx context.Context, uri string) (*IndexedRecord, error) {
	return idx.witness.getIndexed(ctx, uri)
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
	if cached, ok := idx.profileStorage.loadProfile(ctx, did); ok {
		// Promote to in-memory cache
		cached.ExpiresAt = time.Now().Add(idx.profileTTL)
		idx.profileCacheMu.Lock()
		idx.profileCache[did] = cached
		idx.profileCacheMu.Unlock()
		return cached.Profile, nil
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

	idx.profileStorage.storeProfile(ctx, did, cached)
}

// GetDIDByHandle looks up a DID from the handle index. Returns the DID and
// true if found, or empty string and false if not indexed.
//
// Backed by the did_by_handle table — last-writer-wins, so a handle that has
// been reassigned to a new DID resolves to that new DID once the new profile
// is observed (via the firehose profile watcher or a GetProfile call).
func (idx *FeedIndex) GetDIDByHandle(ctx context.Context, handle string) (string, bool) {
	return idx.profileStorage.didByHandle(ctx, handle)
}

// InvalidateProfile removes a DID's profile from both the in-memory and persistent
// caches. The next GetProfile call will re-fetch from the API.
func (idx *FeedIndex) InvalidateProfile(did string) {
	idx.profileCacheMu.Lock()
	delete(idx.profileCache, did)
	idx.profileCacheMu.Unlock()

	idx.profileStorage.deleteProfile(did)
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
	dids, err := idx.profileStorage.profileDIDs(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("refresh all profiles: query failed")
		return 0, 0
	}

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
		if cached, ok := idx.profileStorage.loadProfile(ctx, did); ok && cached.Profile != nil {
			oldHandle = cached.Profile.Handle
		}
	}

	if newHandle != "" {
		// Evict any prior owner of newHandle (other than `did` itself).
		if priorDID, ok := idx.profileStorage.didByHandleExcept(ctx, newHandle, did); ok {
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

// ListDirectSourceRefs returns records whose sourceRef points directly at uri.
func (idx *FeedIndex) ListDirectSourceRefs(ctx context.Context, uri string) ([]IndexedRecord, error) {
	rows, err := idx.db.QueryContext(ctx, `
		SELECT uri, did, collection, rkey, record, cid, indexed_at, created_at
		FROM records
		WHERE json_extract(record, '$.sourceRef') = ?
		ORDER BY created_at DESC
	`, uri)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIndexedRecords(rows)
}

// ListSourceRefChain walks sourceRef backlinks breadth-first from uri.
func (idx *FeedIndex) ListSourceRefChain(ctx context.Context, uri string, maxDepth, maxRecords int) ([]IndexedRecord, error) {
	if maxDepth <= 0 {
		maxDepth = 5
	}
	if maxRecords <= 0 {
		maxRecords = 500
	}
	seen := map[string]struct{}{uri: {}}
	frontier := []string{uri}
	var out []IndexedRecord
	for depth := 0; depth < maxDepth && len(frontier) > 0 && len(out) < maxRecords; depth++ {
		var next []string
		for _, u := range frontier {
			recs, err := idx.ListDirectSourceRefs(ctx, u)
			if err != nil {
				return nil, err
			}
			for _, rec := range recs {
				if _, ok := seen[rec.URI]; ok {
					continue
				}
				seen[rec.URI] = struct{}{}
				out = append(out, rec)
				next = append(next, rec.URI)
				if len(out) >= maxRecords {
					return out, nil
				}
			}
		}
		frontier = next
	}
	return out, nil
}

// ListUsageBacklinks returns records in fromCollection whose JSON field equals uri.
func (idx *FeedIndex) ListUsageBacklinks(ctx context.Context, uri, fromCollection, fieldName string) ([]IndexedRecord, error) {
	if fieldName == "" || strings.ContainsAny(fieldName, "'\"$.[] ") {
		return nil, fmt.Errorf("invalid JSON field name: %q", fieldName)
	}
	rows, err := idx.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT uri, did, collection, rkey, record, cid, indexed_at, created_at
		FROM records
		WHERE collection = ? AND json_extract(record, '$.%s') = ?
		ORDER BY created_at DESC
	`, fieldName), fromCollection, uri)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIndexedRecords(rows)
}

// ListUsageBacklinksPage returns one page of usage backlinks plus the total count.
func (idx *FeedIndex) ListUsageBacklinksPage(ctx context.Context, uri, fromCollection, fieldName string, limit, offset int) ([]IndexedRecord, int, error) {
	if fieldName == "" || strings.ContainsAny(fieldName, "'\"$.[] ") {
		return nil, 0, fmt.Errorf("invalid JSON field name: %q", fieldName)
	}
	if limit <= 0 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}
	var count int
	if err := idx.db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT COUNT(*)
		FROM records
		WHERE collection = ? AND json_extract(record, '$.%s') = ?
	`, fieldName), fromCollection, uri).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := idx.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT uri, did, collection, rkey, record, cid, indexed_at, created_at
		FROM records
		WHERE collection = ? AND json_extract(record, '$.%s') = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, fieldName), fromCollection, uri, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	recs, err := scanIndexedRecords(rows)
	return recs, count, err
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

	return scanIndexedRecords(rows)
}

func scanIndexedRecords(rows *sql.Rows) ([]IndexedRecord, error) {
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
	return idx.social.totalLikeCount()
}

// TotalCommentCount returns the total number of comments indexed
func (idx *FeedIndex) TotalCommentCount() int {
	return idx.social.totalCommentCount()
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
func (idx *FeedIndex) GetProfileStatsVisibility(ctx context.Context, did string) profileprefs.ProfileStatsVisibility {
	defaults := profileprefs.DefaultProfileStatsVisibility()
	if did == "" {
		return defaults
	}
	raw, ok := idx.profileStorage.profileStatsVisibility(ctx, did)
	if !ok {
		return defaults
	}
	var settings profileprefs.ProfileStatsVisibility
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return defaults
	}
	// Fill in defaults for any empty fields
	if settings.BeanAvgRating == "" {
		settings.BeanAvgRating = profileprefs.VisibilityPublic
	}
	if settings.RoasterAvgRating == "" {
		settings.RoasterAvgRating = profileprefs.VisibilityPublic
	}
	return settings
}

// SetProfileStatsVisibility saves the profile stats visibility settings for a user.
func (idx *FeedIndex) SetProfileStatsVisibility(ctx context.Context, did string, settings profileprefs.ProfileStatsVisibility) error {
	raw, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	return idx.profileStorage.setProfileStatsVisibility(ctx, did, string(raw))
}

func (idx *FeedIndex) GetUserPreferences(ctx context.Context, did string) profileprefs.UserPreferences {
	defaults := profileprefs.DefaultUserPreferences()
	if did == "" {
		return defaults
	}
	raw, ok := idx.profileStorage.userPreferences(ctx, did)
	if !ok {
		return defaults
	}
	var prefs profileprefs.UserPreferences
	if err := json.Unmarshal([]byte(raw), &prefs); err != nil {
		return defaults
	}
	return prefs.WithDefaults()
}

func (idx *FeedIndex) SetUserPreferences(ctx context.Context, did string, prefs profileprefs.UserPreferences) error {
	raw, err := json.Marshal(prefs.WithDefaults())
	if err != nil {
		return fmt.Errorf("marshal preferences: %w", err)
	}
	return idx.profileStorage.setUserPreferences(ctx, did, string(raw))
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
	err := idx.social.upsertLike(ctx, actorDID, rkey, subjectURI)
	if err == nil {
		if refreshErr := idx.refreshExploreStats(ctx, subjectURI); refreshErr != nil {
			idx.markExploreDirty(ctx, refreshErr)
		}
	}
	return err

}

// DeleteLike removes a like from the index
func (idx *FeedIndex) DeleteLike(ctx context.Context, actorDID, subjectURI string) error {
	err := idx.social.deleteLike(ctx, actorDID, subjectURI)
	if err == nil {
		if refreshErr := idx.refreshExploreStats(ctx, subjectURI); refreshErr != nil {
			idx.markExploreDirty(ctx, refreshErr)
		}
	}
	return err

}

// GetLikeCount returns the number of likes for a record
func (idx *FeedIndex) GetLikeCount(ctx context.Context, subjectURI string) int {
	return idx.social.likeCount(ctx, subjectURI)
}

// HasUserLiked checks if a user has liked a specific record
func (idx *FeedIndex) HasUserLiked(ctx context.Context, actorDID, subjectURI string) bool {
	return idx.social.hasUserLiked(ctx, actorDID, subjectURI)
}

// GetUserLikeRKey returns the rkey of a user's like for a specific record, or empty string if not found
func (idx *FeedIndex) GetUserLikeRKey(ctx context.Context, actorDID, subjectURI string) string {
	return idx.social.userLikeRKey(ctx, actorDID, subjectURI)
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
	return idx.social.likeCountsBatch(ctx, uris)
}

// HasUserLikedBatch checks if a user has liked multiple records in a single query.
func (idx *FeedIndex) HasUserLikedBatch(ctx context.Context, actorDID string, uris []string) map[string]bool {
	return idx.social.hasUserLikedBatch(ctx, actorDID, uris)
}

// GetCommentCountsBatch returns comment counts for multiple subject URIs in a single query.
func (idx *FeedIndex) GetCommentCountsBatch(ctx context.Context, uris []string) map[string]int {
	return idx.social.commentCountsBatch(ctx, uris)
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
	err := idx.social.upsertComment(ctx, actorDID, rkey, subjectURI, parentURI, cid, text, createdAt)
	if err == nil {
		if refreshErr := idx.refreshExploreStats(ctx, subjectURI); refreshErr != nil {
			idx.markExploreDirty(ctx, refreshErr)
		}
	}
	return err

}

// DeleteComment removes a comment from the index
func (idx *FeedIndex) DeleteComment(ctx context.Context, actorDID, rkey, subjectURI string) error {
	err := idx.social.deleteComment(ctx, actorDID, rkey)
	if err == nil {
		if refreshErr := idx.refreshExploreStats(ctx, subjectURI); refreshErr != nil {
			idx.markExploreDirty(ctx, refreshErr)
		}
	}
	return err

}

// GetCommentCount returns the number of comments on a record
func (idx *FeedIndex) GetCommentCount(ctx context.Context, subjectURI string) int {
	return idx.social.commentCount(ctx, subjectURI)
}

// GetCommentsForSubject returns all comments for a specific record, ordered by creation time
func (idx *FeedIndex) GetCommentsForSubject(ctx context.Context, subjectURI string, limit int, viewerDID string) []IndexedComment {
	comments := idx.social.commentsForSubject(ctx, subjectURI, limit)

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
