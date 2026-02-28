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

	"arabica/internal/atproto"
	"arabica/internal/lexicons"
	"arabica/internal/models"

	"github.com/rs/zerolog/log"
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
	publicClient *atproto.PublicClient
	profileTTL   time.Duration

	// In-memory cache for hot data
	profileCache   map[string]*CachedProfile
	profileCacheMu sync.RWMutex

	ready   bool
	readyMu sync.RWMutex
}

// FeedSort defines the sort order for feed queries
type FeedSort string

const (
	FeedSortRecent  FeedSort = "recent"
	FeedSortPopular FeedSort = "popular"
)

// FeedQuery specifies filtering, sorting, and pagination for feed queries
type FeedQuery struct {
	Limit      int                 // Max items to return
	Cursor     string              // Opaque cursor for pagination (created_at|uri)
	TypeFilter lexicons.RecordType // Filter to a specific record type (empty = all)
	Sort       FeedSort            // Sort order (default: recent)
}

// FeedResult contains feed items plus pagination info
type FeedResult struct {
	Items      []*FeedItem
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

CREATE TABLE IF NOT EXISTS meta (
    key   TEXT PRIMARY KEY,
    value BLOB
);

CREATE TABLE IF NOT EXISTS known_dids (did TEXT PRIMARY KEY);
CREATE TABLE IF NOT EXISTS backfilled (did TEXT PRIMARY KEY, backfilled_at TEXT NOT NULL);

CREATE TABLE IF NOT EXISTS profiles (
    did        TEXT PRIMARY KEY,
    data       TEXT NOT NULL,
    expires_at TEXT NOT NULL
);

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

	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_pragma=temp_store(MEMORY)&_pragma=mmap_size(134217728)&_pragma=cache_size(-65536)")
	if err != nil {
		return nil, fmt.Errorf("failed to open index database: %w", err)
	}

	// WAL mode allows concurrent reads with a single writer.
	// Allow multiple reader connections but limit to avoid file descriptor exhaustion.
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)

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

	return idx, nil
}

// DB returns the underlying database connection for shared use by other stores.
func (idx *FeedIndex) DB() *sql.DB {
	return idx.db
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
func (idx *FeedIndex) GetCursor() (int64, error) {
	var cursor int64
	err := idx.db.QueryRow(`SELECT value FROM meta WHERE key = 'cursor'`).Scan(&cursor)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return cursor, err
}

// SetCursor stores the cursor position
func (idx *FeedIndex) SetCursor(cursor int64) error {
	_, err := idx.db.Exec(`INSERT OR REPLACE INTO meta (key, value) VALUES ('cursor', ?)`, cursor)
	return err
}

// UpsertRecord adds or updates a record in the index
func (idx *FeedIndex) UpsertRecord(did, collection, rkey, cid string, record json.RawMessage, eventTime int64) error {
	uri := atproto.BuildATURI(did, collection, rkey)

	// Parse createdAt from record
	var recordData map[string]any
	createdAt := time.Now()
	if err := json.Unmarshal(record, &recordData); err == nil {
		if createdAtStr, ok := recordData["createdAt"].(string); ok {
			if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
				createdAt = t
			}
		}
	}

	now := time.Now()

	_, err := idx.db.Exec(`
		INSERT INTO records (uri, did, collection, rkey, record, cid, indexed_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(uri) DO UPDATE SET
			record = excluded.record,
			cid = excluded.cid,
			indexed_at = excluded.indexed_at,
			created_at = excluded.created_at
	`, uri, did, collection, rkey, string(record), cid,
		now.Format(time.RFC3339Nano), createdAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("failed to upsert record: %w", err)
	}

	// Track known DID
	_, err = idx.db.Exec(`INSERT OR IGNORE INTO known_dids (did) VALUES (?)`, did)
	if err != nil {
		return fmt.Errorf("failed to track known DID: %w", err)
	}

	return nil
}

// DeleteRecord removes a record from the index
func (idx *FeedIndex) DeleteRecord(did, collection, rkey string) error {
	uri := atproto.BuildATURI(did, collection, rkey)
	_, err := idx.db.Exec(`DELETE FROM records WHERE uri = ?`, uri)
	return err
}

// GetRecord retrieves a single record by URI
func (idx *FeedIndex) GetRecord(uri string) (*IndexedRecord, error) {
	var rec IndexedRecord
	var recordStr, indexedAtStr, createdAtStr string

	err := idx.db.QueryRow(`
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

// FeedItem represents an item in the feed (matches feed.FeedItem structure)
type FeedItem struct {
	RecordType lexicons.RecordType
	Action     string

	Brew    *models.Brew
	Bean    *models.Bean
	Roaster *models.Roaster
	Grinder *models.Grinder
	Brewer  *models.Brewer

	Author    *atproto.Profile
	Timestamp time.Time
	TimeAgo   string

	// Like-related fields
	LikeCount  int    // Number of likes on this record
	SubjectURI string // AT-URI of this record (for like button)
	SubjectCID string // CID of this record (for like button)

	// Comment-related fields
	CommentCount int // Number of comments on this record
}

// GetRecentFeed returns recent feed items from the index
func (idx *FeedIndex) GetRecentFeed(ctx context.Context, limit int) ([]*FeedItem, error) {
	return idx.getFeedItems(ctx, "", limit, "")
}

// recordTypeToNSID maps a lexicons.RecordType to its NSID collection string
var recordTypeToNSID = map[lexicons.RecordType]string{
	lexicons.RecordTypeBrew:    atproto.NSIDBrew,
	lexicons.RecordTypeBean:    atproto.NSIDBean,
	lexicons.RecordTypeRoaster: atproto.NSIDRoaster,
	lexicons.RecordTypeGrinder: atproto.NSIDGrinder,
	lexicons.RecordTypeBrewer:  atproto.NSIDBrewer,
}

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

	var collectionFilter string
	if q.TypeFilter != "" {
		nsid, ok := recordTypeToNSID[q.TypeFilter]
		if !ok {
			return nil, fmt.Errorf("unknown record type: %s", q.TypeFilter)
		}
		collectionFilter = nsid
	}

	items, err := idx.getFeedItems(ctx, collectionFilter, q.Limit+1, q.Cursor)
	if err != nil {
		return nil, err
	}

	// Sort based on query
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
func (idx *FeedIndex) getFeedItems(ctx context.Context, collectionFilter string, limit int, cursor string) ([]*FeedItem, error) {
	// Build query for feedable records
	var args []any
	query := `SELECT uri, did, collection, rkey, record, cid, indexed_at, created_at FROM records WHERE `

	if collectionFilter != "" {
		query += `collection = ? `
		args = append(args, collectionFilter)
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
			for _, key := range []string{"beanRef", "roasterRef", "grinderRef", "brewerRef"} {
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

				// If this is a bean, check if it references a roaster we also need
				if rec.Collection == atproto.NSIDBean {
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

	// Convert to FeedItems
	items := make([]*FeedItem, 0, len(records))
	for _, record := range records {
		item, err := idx.recordToFeedItem(ctx, record, recordsByURI)
		if err != nil {
			log.Warn().Err(err).Str("uri", record.URI).Msg("failed to convert record to feed item")
			continue
		}
		if !FeedableRecordTypes[item.RecordType] {
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

// recordToFeedItem converts an IndexedRecord to a FeedItem
func (idx *FeedIndex) recordToFeedItem(ctx context.Context, record *IndexedRecord, refMap map[string]*IndexedRecord) (*FeedItem, error) {
	var recordData map[string]any
	if err := json.Unmarshal(record.Record, &recordData); err != nil {
		return nil, err
	}

	item := &FeedItem{
		Timestamp: record.CreatedAt,
		TimeAgo:   formatTimeAgo(record.CreatedAt),
	}

	// Get author profile
	profile, err := idx.GetProfile(ctx, record.DID)
	if err != nil {
		log.Warn().Err(err).Str("did", record.DID).Msg("failed to get profile")
		profile = &atproto.Profile{
			DID:    record.DID,
			Handle: record.DID,
		}
	}
	item.Author = profile

	switch record.Collection {
	case atproto.NSIDBrew:
		brew, err := atproto.RecordToBrew(recordData, record.URI)
		if err != nil {
			return nil, err
		}

		// Resolve bean reference
		if beanRef, ok := recordData["beanRef"].(string); ok && beanRef != "" {
			if beanRecord, found := refMap[beanRef]; found {
				var beanData map[string]any
				if err := json.Unmarshal(beanRecord.Record, &beanData); err == nil {
					bean, _ := atproto.RecordToBean(beanData, beanRef)
					brew.Bean = bean

					// Resolve roaster reference for bean
					if roasterRef, ok := beanData["roasterRef"].(string); ok && roasterRef != "" {
						if roasterRecord, found := refMap[roasterRef]; found {
							var roasterData map[string]any
							if err := json.Unmarshal(roasterRecord.Record, &roasterData); err == nil {
								roaster, _ := atproto.RecordToRoaster(roasterData, roasterRef)
								brew.Bean.Roaster = roaster
							}
						}
					}
				}
			}
		}

		// Resolve grinder reference
		if grinderRef, ok := recordData["grinderRef"].(string); ok && grinderRef != "" {
			if grinderRecord, found := refMap[grinderRef]; found {
				var grinderData map[string]any
				if err := json.Unmarshal(grinderRecord.Record, &grinderData); err == nil {
					grinder, _ := atproto.RecordToGrinder(grinderData, grinderRef)
					brew.GrinderObj = grinder
				}
			}
		}

		// Resolve brewer reference
		if brewerRef, ok := recordData["brewerRef"].(string); ok && brewerRef != "" {
			if brewerRecord, found := refMap[brewerRef]; found {
				var brewerData map[string]any
				if err := json.Unmarshal(brewerRecord.Record, &brewerData); err == nil {
					brewer, _ := atproto.RecordToBrewer(brewerData, brewerRef)
					brew.BrewerObj = brewer
				}
			}
		}

		item.RecordType = lexicons.RecordTypeBrew
		item.Action = "added a new brew"
		item.Brew = brew

	case atproto.NSIDBean:
		bean, err := atproto.RecordToBean(recordData, record.URI)
		if err != nil {
			return nil, err
		}

		// Resolve roaster reference
		if roasterRef, ok := recordData["roasterRef"].(string); ok && roasterRef != "" {
			if roasterRecord, found := refMap[roasterRef]; found {
				var roasterData map[string]any
				if err := json.Unmarshal(roasterRecord.Record, &roasterData); err == nil {
					roaster, _ := atproto.RecordToRoaster(roasterData, roasterRef)
					bean.Roaster = roaster
				}
			}
		}

		item.RecordType = lexicons.RecordTypeBean
		item.Action = "added a new bean"
		item.Bean = bean

	case atproto.NSIDRoaster:
		roaster, err := atproto.RecordToRoaster(recordData, record.URI)
		if err != nil {
			return nil, err
		}
		item.RecordType = lexicons.RecordTypeRoaster
		item.Action = "added a new roaster"
		item.Roaster = roaster

	case atproto.NSIDGrinder:
		grinder, err := atproto.RecordToGrinder(recordData, record.URI)
		if err != nil {
			return nil, err
		}
		item.RecordType = lexicons.RecordTypeGrinder
		item.Action = "added a new grinder"
		item.Grinder = grinder

	case atproto.NSIDBrewer:
		brewer, err := atproto.RecordToBrewer(recordData, record.URI)
		if err != nil {
			return nil, err
		}
		item.RecordType = lexicons.RecordTypeBrewer
		item.Action = "added a new brewer"
		item.Brewer = brewer

	case atproto.NSIDLike:
		return nil, fmt.Errorf("unexpected: likes should be filtered before conversion")

	default:
		return nil, fmt.Errorf("unknown collection: %s", record.Collection)
	}

	// Populate like-related fields for all record types
	item.SubjectURI = record.URI
	item.SubjectCID = record.CID
	item.LikeCount = idx.GetLikeCount(record.URI)
	item.CommentCount = idx.GetCommentCount(record.URI)

	return item, nil
}

// GetProfile fetches a profile, using cache when possible
func (idx *FeedIndex) GetProfile(ctx context.Context, did string) (*atproto.Profile, error) {
	// Check in-memory cache first
	idx.profileCacheMu.RLock()
	if cached, ok := idx.profileCache[did]; ok && time.Now().Before(cached.ExpiresAt) {
		idx.profileCacheMu.RUnlock()
		return cached.Profile, nil
	}
	idx.profileCacheMu.RUnlock()

	// Check persistent cache
	var dataStr, expiresAtStr string
	err := idx.db.QueryRow(`SELECT data, expires_at FROM profiles WHERE did = ?`, did).Scan(&dataStr, &expiresAtStr)
	if err == nil {
		expiresAt, _ := time.Parse(time.RFC3339Nano, expiresAtStr)
		if time.Now().Before(expiresAt) {
			cached := &CachedProfile{}
			if err := json.Unmarshal([]byte(dataStr), cached); err == nil {
				idx.profileCacheMu.Lock()
				idx.profileCache[did] = cached
				idx.profileCacheMu.Unlock()
				return cached.Profile, nil
			}
		}
	}

	// Fetch from API
	profile, err := idx.publicClient.GetProfile(ctx, did)
	if err != nil {
		return nil, err
	}

	// Cache the result
	now := time.Now()
	cached := &CachedProfile{
		Profile:   profile,
		CachedAt:  now,
		ExpiresAt: now.Add(idx.profileTTL),
	}

	// Update in-memory cache
	idx.profileCacheMu.Lock()
	idx.profileCache[did] = cached
	idx.profileCacheMu.Unlock()

	// Persist to database
	data, _ := json.Marshal(cached)
	_, _ = idx.db.Exec(`INSERT OR REPLACE INTO profiles (did, data, expires_at) VALUES (?, ?, ?)`,
		did, string(data), cached.ExpiresAt.Format(time.RFC3339Nano))

	return profile, nil
}

// GetKnownDIDs returns all DIDs that have created Arabica records
func (idx *FeedIndex) GetKnownDIDs() ([]string, error) {
	rows, err := idx.db.Query(`SELECT did FROM known_dids`)
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

// ListRecordsByCollection returns all indexed records for a given collection.
func (idx *FeedIndex) ListRecordsByCollection(collection string) ([]IndexedRecord, error) {
	rows, err := idx.db.Query(`
		SELECT uri, did, collection, rkey, record, cid, indexed_at, created_at
		FROM records WHERE collection = ? ORDER BY created_at DESC
	`, collection)
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
func (idx *FeedIndex) IsBackfilled(did string) bool {
	var exists int
	err := idx.db.QueryRow(`SELECT 1 FROM backfilled WHERE did = ?`, did).Scan(&exists)
	return err == nil
}

// MarkBackfilled marks a DID as backfilled with current timestamp
func (idx *FeedIndex) MarkBackfilled(did string) error {
	_, err := idx.db.Exec(`INSERT OR IGNORE INTO backfilled (did, backfilled_at) VALUES (?, ?)`,
		did, time.Now().Format(time.RFC3339))
	return err
}

// BackfillUser fetches all existing records for a DID and adds them to the index
func (idx *FeedIndex) BackfillUser(ctx context.Context, did string) error {
	if idx.IsBackfilled(did) {
		log.Debug().Str("did", did).Msg("DID already backfilled, skipping")
		return nil
	}

	log.Info().Str("did", did).Msg("backfilling user records")

	recordCount := 0
	for _, collection := range ArabicaCollections {
		records, err := idx.publicClient.ListRecords(ctx, did, collection, 100)
		if err != nil {
			log.Warn().Err(err).Str("did", did).Str("collection", collection).Msg("failed to list records for backfill")
			continue
		}

		for _, record := range records.Records {
			parts := strings.Split(record.URI, "/")
			if len(parts) < 3 {
				continue
			}
			rkey := parts[len(parts)-1]

			recordJSON, err := json.Marshal(record.Value)
			if err != nil {
				continue
			}

			if err := idx.UpsertRecord(did, collection, rkey, record.CID, recordJSON, 0); err != nil {
				log.Warn().Err(err).Str("uri", record.URI).Msg("failed to upsert record during backfill")
				continue
			}
			recordCount++

			switch collection {
			case atproto.NSIDLike:
				if subject, ok := record.Value["subject"].(map[string]interface{}); ok {
					if subjectURI, ok := subject["uri"].(string); ok {
						if err := idx.UpsertLike(did, rkey, subjectURI); err != nil {
							log.Warn().Err(err).Str("uri", record.URI).Msg("failed to index like during backfill")
						}
					}
				}
			case atproto.NSIDComment:
				if subject, ok := record.Value["subject"].(map[string]interface{}); ok {
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
						if parent, ok := record.Value["parent"].(map[string]interface{}); ok {
							parentURI, _ = parent["uri"].(string)
						}
						if err := idx.UpsertComment(did, rkey, subjectURI, parentURI, record.CID, text, createdAt); err != nil {
							log.Warn().Err(err).Str("uri", record.URI).Msg("failed to index comment during backfill")
						}
					}
				}
			}
		}
	}

	if err := idx.MarkBackfilled(did); err != nil {
		log.Warn().Err(err).Str("did", did).Msg("failed to mark DID as backfilled")
	}

	log.Info().Str("did", did).Int("record_count", recordCount).Msg("backfill complete")
	return nil
}

// ========== Like Indexing Methods ==========

// UpsertLike adds or updates a like in the index
func (idx *FeedIndex) UpsertLike(actorDID, rkey, subjectURI string) error {
	_, err := idx.db.Exec(`INSERT OR IGNORE INTO likes (subject_uri, actor_did, rkey) VALUES (?, ?, ?)`,
		subjectURI, actorDID, rkey)
	return err
}

// DeleteLike removes a like from the index
func (idx *FeedIndex) DeleteLike(actorDID, subjectURI string) error {
	_, err := idx.db.Exec(`DELETE FROM likes WHERE subject_uri = ? AND actor_did = ?`,
		subjectURI, actorDID)
	return err
}

// GetLikeCount returns the number of likes for a record
func (idx *FeedIndex) GetLikeCount(subjectURI string) int {
	var count int
	_ = idx.db.QueryRow(`SELECT COUNT(*) FROM likes WHERE subject_uri = ?`, subjectURI).Scan(&count)
	return count
}

// HasUserLiked checks if a user has liked a specific record
func (idx *FeedIndex) HasUserLiked(actorDID, subjectURI string) bool {
	var exists int
	err := idx.db.QueryRow(`SELECT 1 FROM likes WHERE actor_did = ? AND subject_uri = ? LIMIT 1`,
		actorDID, subjectURI).Scan(&exists)
	return err == nil
}

// GetUserLikeRKey returns the rkey of a user's like for a specific record, or empty string if not found
func (idx *FeedIndex) GetUserLikeRKey(actorDID, subjectURI string) string {
	var rkey string
	err := idx.db.QueryRow(`SELECT rkey FROM likes WHERE actor_did = ? AND subject_uri = ?`,
		actorDID, subjectURI).Scan(&rkey)
	if err != nil {
		return ""
	}
	return rkey
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
func (idx *FeedIndex) UpsertComment(actorDID, rkey, subjectURI, parentURI, cid, text string, createdAt time.Time) error {
	// Extract parent rkey from parent URI if present
	var parentRKey string
	if parentURI != "" {
		parts := strings.Split(parentURI, "/")
		if len(parts) > 0 {
			parentRKey = parts[len(parts)-1]
		}
	}

	_, err := idx.db.Exec(`
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
func (idx *FeedIndex) DeleteComment(actorDID, rkey, subjectURI string) error {
	_, err := idx.db.Exec(`DELETE FROM comments WHERE actor_did = ? AND rkey = ?`, actorDID, rkey)
	return err
}

// GetCommentCount returns the number of comments on a record
func (idx *FeedIndex) GetCommentCount(subjectURI string) int {
	var count int
	_ = idx.db.QueryRow(`SELECT COUNT(*) FROM comments WHERE subject_uri = ?`, subjectURI).Scan(&count)
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

	// Populate profile and like info for each comment
	for i := range comments {
		profile, err := idx.GetProfile(ctx, comments[i].ActorDID)
		if err != nil {
			comments[i].Handle = comments[i].ActorDID
		} else {
			comments[i].Handle = profile.Handle
			comments[i].DisplayName = profile.DisplayName
			comments[i].Avatar = profile.Avatar
		}

		commentURI := fmt.Sprintf("at://%s/social.arabica.alpha.comment/%s", comments[i].ActorDID, comments[i].RKey)
		comments[i].LikeCount = idx.GetLikeCount(commentURI)
		if viewerDID != "" {
			comments[i].IsLiked = idx.HasUserLiked(viewerDID, commentURI)
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
		visualDepth := depth
		if visualDepth > 2 {
			visualDepth = 2
		}
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
