package firehose

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
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
	bolt "go.etcd.io/bbolt"
)

// Bucket names for the feed index
var (
	// BucketRecords stores full record data: {at-uri} -> {IndexedRecord JSON}
	BucketRecords = []byte("records")

	// BucketByTime stores records by timestamp for chronological queries: {timestamp:at-uri} -> {}
	BucketByTime = []byte("by_time")

	// BucketByDID stores records by DID for user-specific queries: {did:at-uri} -> {}
	BucketByDID = []byte("by_did")

	// BucketByCollection stores records by type: {collection:timestamp:at-uri} -> {}
	BucketByCollection = []byte("by_collection")

	// BucketProfiles stores cached profile data: {did} -> {CachedProfile JSON}
	BucketProfiles = []byte("profiles")

	// BucketMeta stores metadata like cursor position: {key} -> {value}
	BucketMeta = []byte("meta")

	// BucketKnownDIDs stores all DIDs we've seen with Arabica records
	BucketKnownDIDs = []byte("known_dids")

	// BucketBackfilled stores DIDs that have been backfilled: {did} -> {timestamp}
	BucketBackfilled = []byte("backfilled")

	// BucketLikes stores like mappings: {subject_uri:actor_did} -> {rkey}
	BucketLikes = []byte("likes")

	// BucketLikeCounts stores aggregated like counts: {subject_uri} -> {uint64 count}
	BucketLikeCounts = []byte("like_counts")

	// BucketLikesByActor stores likes by actor for lookup: {actor_did:subject_uri} -> {rkey}
	BucketLikesByActor = []byte("likes_by_actor")

	// BucketComments stores comment data: {subject_uri:timestamp:actor_did} -> {comment JSON}
	BucketComments = []byte("comments")

	// BucketCommentCounts stores aggregated comment counts: {subject_uri} -> {uint64 count}
	BucketCommentCounts = []byte("comment_counts")

	// BucketCommentsByActor stores comments by actor for lookup: {actor_did:rkey} -> {subject_uri}
	BucketCommentsByActor = []byte("comments_by_actor")

	// BucketCommentChildren stores parent-child relationships: {parent_uri:child_rkey} -> {child_actor_did}
	BucketCommentChildren = []byte("comment_children")
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
	CreatedAt  time.Time       `json:"created_at"` // Parsed from record
}

// CachedProfile stores profile data with TTL
type CachedProfile struct {
	Profile   *atproto.Profile `json:"profile"`
	CachedAt  time.Time        `json:"cached_at"`
	ExpiresAt time.Time        `json:"expires_at"`
}

// FeedIndex provides persistent storage for firehose events
type FeedIndex struct {
	db           *bolt.DB
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
	Limit      int                  // Max items to return
	Cursor     string               // Opaque cursor for pagination (base64-encoded time key)
	TypeFilter lexicons.RecordType  // Filter to a specific record type (empty = all)
	Sort       FeedSort             // Sort order (default: recent)
}

// FeedResult contains feed items plus pagination info
type FeedResult struct {
	Items      []*FeedItem
	NextCursor string // Empty if no more results
}

// NewFeedIndex creates a new feed index backed by BoltDB
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

	db, err := bolt.Open(path, 0600, &bolt.Options{
		Timeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open index database: %w", err)
	}

	// Create buckets
	err = db.Update(func(tx *bolt.Tx) error {
		buckets := [][]byte{
			BucketRecords,
			BucketByTime,
			BucketByDID,
			BucketByCollection,
			BucketProfiles,
			BucketMeta,
			BucketKnownDIDs,
			BucketBackfilled,
			BucketLikes,
			BucketLikeCounts,
			BucketLikesByActor,
			BucketComments,
			BucketCommentCounts,
			BucketCommentsByActor,
			BucketCommentChildren,
		}
		for _, bucket := range buckets {
			if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
			}
		}
		return nil
	})
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	idx := &FeedIndex{
		db:           db,
		publicClient: atproto.NewPublicClient(),
		profileTTL:   profileTTL,
		profileCache: make(map[string]*CachedProfile),
	}

	return idx, nil
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
	err := idx.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketMeta)
		v := b.Get([]byte("cursor"))
		if len(v) == 8 {
			cursor = int64(binary.BigEndian.Uint64(v))
		}
		return nil
	})
	return cursor, err
}

// SetCursor stores the cursor position
func (idx *FeedIndex) SetCursor(cursor int64) error {
	return idx.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketMeta)
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, uint64(cursor))
		return b.Put([]byte("cursor"), buf)
	})
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

	indexed := &IndexedRecord{
		URI:        uri,
		DID:        did,
		Collection: collection,
		RKey:       rkey,
		Record:     record,
		CID:        cid,
		IndexedAt:  time.Now(),
		CreatedAt:  createdAt,
	}

	data, err := json.Marshal(indexed)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	return idx.db.Update(func(tx *bolt.Tx) error {
		// Store the record
		records := tx.Bucket(BucketRecords)
		if err := records.Put([]byte(uri), data); err != nil {
			return err
		}

		// Index by time (use createdAt for sorting, not event time)
		byTime := tx.Bucket(BucketByTime)
		timeKey := makeTimeKey(createdAt, uri)
		if err := byTime.Put(timeKey, nil); err != nil {
			return err
		}

		// Index by DID
		byDID := tx.Bucket(BucketByDID)
		didKey := []byte(did + ":" + uri)
		if err := byDID.Put(didKey, nil); err != nil {
			return err
		}

		// Index by collection
		byCollection := tx.Bucket(BucketByCollection)
		collKey := []byte(collection + ":" + string(timeKey))
		if err := byCollection.Put(collKey, nil); err != nil {
			return err
		}

		// Track known DID
		knownDIDs := tx.Bucket(BucketKnownDIDs)
		if err := knownDIDs.Put([]byte(did), []byte("1")); err != nil {
			return err
		}

		return nil
	})
}

// DeleteRecord removes a record from the index
func (idx *FeedIndex) DeleteRecord(did, collection, rkey string) error {
	uri := atproto.BuildATURI(did, collection, rkey)

	return idx.db.Update(func(tx *bolt.Tx) error {
		// Get the existing record to find its timestamp
		records := tx.Bucket(BucketRecords)
		existingData := records.Get([]byte(uri))
		if existingData == nil {
			// Record doesn't exist, nothing to delete
			return nil
		}

		var existing IndexedRecord
		if err := json.Unmarshal(existingData, &existing); err != nil {
			// Can't parse, just delete the main record
			return records.Delete([]byte(uri))
		}

		// Delete from records
		if err := records.Delete([]byte(uri)); err != nil {
			return err
		}

		// Delete from by_time index
		byTime := tx.Bucket(BucketByTime)
		timeKey := makeTimeKey(existing.CreatedAt, uri)
		if err := byTime.Delete(timeKey); err != nil {
			return err
		}

		// Delete from by_did index
		byDID := tx.Bucket(BucketByDID)
		didKey := []byte(did + ":" + uri)
		if err := byDID.Delete(didKey); err != nil {
			return err
		}

		// Delete from by_collection index
		byCollection := tx.Bucket(BucketByCollection)
		collKey := []byte(collection + ":" + string(timeKey))
		if err := byCollection.Delete(collKey); err != nil {
			return err
		}

		return nil
	})
}

// GetRecord retrieves a single record by URI
func (idx *FeedIndex) GetRecord(uri string) (*IndexedRecord, error) {
	var record *IndexedRecord
	err := idx.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketRecords)
		data := b.Get([]byte(uri))
		if data == nil {
			return nil
		}
		record = &IndexedRecord{}
		return json.Unmarshal(data, record)
	})
	return record, err
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
	var records []*IndexedRecord
	err := idx.db.View(func(tx *bolt.Tx) error {
		byTime := tx.Bucket(BucketByTime)
		recordsBucket := tx.Bucket(BucketRecords)

		c := byTime.Cursor()

		// Iterate in reverse (newest first)
		count := 0
		for k, _ := c.First(); k != nil && count < limit*2; k, _ = c.Next() {
			// Extract URI from key (format: timestamp:uri)
			uri := extractURIFromTimeKey(k)
			if uri == "" {
				continue
			}

			data := recordsBucket.Get([]byte(uri))
			if data == nil {
				continue
			}

			var record IndexedRecord
			if err := json.Unmarshal(data, &record); err != nil {
				continue
			}

			records = append(records, &record)
			count++
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Build lookup maps for reference resolution
	recordsByURI := make(map[string]*IndexedRecord)
	for _, r := range records {
		recordsByURI[r.URI] = r
	}

	// Also load additional records we might need for references
	err = idx.db.View(func(tx *bolt.Tx) error {
		recordsBucket := tx.Bucket(BucketRecords)
		return recordsBucket.ForEach(func(k, v []byte) error {
			uri := string(k)
			if _, exists := recordsByURI[uri]; exists {
				return nil
			}
			var record IndexedRecord
			if err := json.Unmarshal(v, &record); err != nil {
				return nil
			}
			// Only load beans, roasters, grinders, brewers for reference resolution
			switch record.Collection {
			case atproto.NSIDBean, atproto.NSIDRoaster, atproto.NSIDGrinder, atproto.NSIDBrewer:
				recordsByURI[uri] = &record
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	// Convert to FeedItems
	items := make([]*FeedItem, 0, len(records))
	for _, record := range records {
		// Skip likes - they're indexed for like counts but not displayed as feed items
		if record.Collection == atproto.NSIDLike {
			continue
		}

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

	// Sort by timestamp descending
	sort.Slice(items, func(i, j int) bool {
		return items[i].Timestamp.After(items[j].Timestamp)
	})

	// Apply limit
	if len(items) > limit {
		items = items[:limit]
	}

	return items, nil
}

// recordTypeToNSID maps a lexicons.RecordType to its NSID collection string
var recordTypeToNSID = map[lexicons.RecordType]string{
	lexicons.RecordTypeBrew:    atproto.NSIDBrew,
	lexicons.RecordTypeBean:    atproto.NSIDBean,
	lexicons.RecordTypeRoaster: atproto.NSIDRoaster,
	lexicons.RecordTypeGrinder: atproto.NSIDGrinder,
	lexicons.RecordTypeBrewer:  atproto.NSIDBrewer,
}

// GetFeedWithQuery returns feed items matching the given query with cursor-based pagination
func (idx *FeedIndex) GetFeedWithQuery(ctx context.Context, q FeedQuery) (*FeedResult, error) {
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Sort == "" {
		q.Sort = FeedSortRecent
	}

	// For type-filtered queries, use BucketByCollection for efficiency
	// For unfiltered queries, use BucketByTime
	var records []*IndexedRecord
	var lastTimeKey []byte

	// Decode cursor if provided
	var cursorBytes []byte
	if q.Cursor != "" {
		var err error
		cursorBytes, err = decodeCursor(q.Cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
	}

	// Fetch more than needed to account for filtering
	fetchLimit := q.Limit + 10

	err := idx.db.View(func(tx *bolt.Tx) error {
		recordsBucket := tx.Bucket(BucketRecords)

		if q.TypeFilter != "" {
			// Use BucketByCollection for filtered queries
			nsid, ok := recordTypeToNSID[q.TypeFilter]
			if !ok {
				return fmt.Errorf("unknown record type: %s", q.TypeFilter)
			}

			byCollection := tx.Bucket(BucketByCollection)
			c := byCollection.Cursor()

			// Collection keys: {collection}:{inverted_timestamp}:{uri}
			prefix := []byte(nsid + ":")

			var k []byte
			if cursorBytes != nil {
				// Seek to cursor position (cursor is the full collection key)
				k, _ = c.Seek(cursorBytes)
				// Skip the cursor key itself (it was the last item of previous page)
				if k != nil && string(k) == string(cursorBytes) {
					k, _ = c.Next()
				}
			} else {
				k, _ = c.Seek(prefix)
			}

			count := 0
			for ; k != nil && count < fetchLimit; k, _ = c.Next() {
				if !bytes.HasPrefix(k, prefix) {
					break
				}

				// Extract URI from collection key: {collection}:{timestamp_bytes}:{uri}
				uri := extractURIFromCollectionKey(k, nsid)
				if uri == "" {
					continue
				}

				data := recordsBucket.Get([]byte(uri))
				if data == nil {
					continue
				}

				var record IndexedRecord
				if err := json.Unmarshal(data, &record); err != nil {
					continue
				}
				records = append(records, &record)
				lastTimeKey = make([]byte, len(k))
				copy(lastTimeKey, k)
				count++
			}
		} else {
			// Use BucketByTime for unfiltered queries
			byTime := tx.Bucket(BucketByTime)
			c := byTime.Cursor()

			var k []byte
			if cursorBytes != nil {
				k, _ = c.Seek(cursorBytes)
				if k != nil && string(k) == string(cursorBytes) {
					k, _ = c.Next()
				}
			} else {
				k, _ = c.First()
			}

			count := 0
			for ; k != nil && count < fetchLimit; k, _ = c.Next() {
				uri := extractURIFromTimeKey(k)
				if uri == "" {
					continue
				}

				data := recordsBucket.Get([]byte(uri))
				if data == nil {
					continue
				}

				var record IndexedRecord
				if err := json.Unmarshal(data, &record); err != nil {
					continue
				}
				records = append(records, &record)
				lastTimeKey = make([]byte, len(k))
				copy(lastTimeKey, k)
				count++
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Build lookup maps for reference resolution
	recordsByURI := make(map[string]*IndexedRecord)
	for _, r := range records {
		recordsByURI[r.URI] = r
	}

	// Load additional records for reference resolution
	err = idx.db.View(func(tx *bolt.Tx) error {
		recordsBucket := tx.Bucket(BucketRecords)
		return recordsBucket.ForEach(func(k, v []byte) error {
			uri := string(k)
			if _, exists := recordsByURI[uri]; exists {
				return nil
			}
			var record IndexedRecord
			if err := json.Unmarshal(v, &record); err != nil {
				return nil
			}
			switch record.Collection {
			case atproto.NSIDBean, atproto.NSIDRoaster, atproto.NSIDGrinder, atproto.NSIDBrewer:
				recordsByURI[uri] = &record
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	// Convert to FeedItems
	items := make([]*FeedItem, 0, len(records))
	for _, record := range records {
		if record.Collection == atproto.NSIDLike || record.Collection == atproto.NSIDComment {
			continue
		}

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

	// Sort based on query
	switch q.Sort {
	case FeedSortPopular:
		sort.Slice(items, func(i, j int) bool {
			scoreI := items[i].LikeCount*3 + items[i].CommentCount*2
			scoreJ := items[j].LikeCount*3 + items[j].CommentCount*2
			if scoreI != scoreJ {
				return scoreI > scoreJ
			}
			return items[i].Timestamp.After(items[j].Timestamp)
		})
	default: // FeedSortRecent
		sort.Slice(items, func(i, j int) bool {
			return items[i].Timestamp.After(items[j].Timestamp)
		})
	}

	// Build result with cursor
	result := &FeedResult{Items: items}

	if len(items) > q.Limit {
		result.Items = items[:q.Limit]
		// Cursor is the last time key we read from the DB
		if lastTimeKey != nil {
			result.NextCursor = encodeCursor(lastTimeKey)
		}
	}

	return result, nil
}

// extractURIFromCollectionKey extracts the URI from a collection key
// Format: {collection}:{inverted_timestamp_8bytes}:{uri}
func extractURIFromCollectionKey(key []byte, collection string) string {
	// prefix is collection + ":"
	prefixLen := len(collection) + 1
	// Then 8 bytes of timestamp + ":"
	minLen := prefixLen + 8 + 1 + 1 // prefix + timestamp + ":" + at least 1 char
	if len(key) < minLen {
		return ""
	}
	return string(key[prefixLen+9:])
}

func encodeCursor(key []byte) string {
	return hex.EncodeToString(key)
}

func decodeCursor(s string) ([]byte, error) {
	return hex.DecodeString(s)
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
		// Use a placeholder profile
		profile = &atproto.Profile{
			DID:    record.DID,
			Handle: record.DID, // Use DID as handle if we can't resolve
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
		// This should never be reached - likes are filtered before calling recordToFeedItem
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
	var cached *CachedProfile
	err := idx.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketProfiles)
		data := b.Get([]byte(did))
		if data == nil {
			return nil
		}
		cached = &CachedProfile{}
		return json.Unmarshal(data, cached)
	})
	if err == nil && cached != nil && time.Now().Before(cached.ExpiresAt) {
		// Update in-memory cache
		idx.profileCacheMu.Lock()
		idx.profileCache[did] = cached
		idx.profileCacheMu.Unlock()
		return cached.Profile, nil
	}

	// Fetch from API
	profile, err := idx.publicClient.GetProfile(ctx, did)
	if err != nil {
		return nil, err
	}

	// Cache the result
	now := time.Now()
	cached = &CachedProfile{
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
	_ = idx.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketProfiles)
		return b.Put([]byte(did), data)
	})

	return profile, nil
}

// GetKnownDIDs returns all DIDs that have created Arabica records
func (idx *FeedIndex) GetKnownDIDs() ([]string, error) {
	var dids []string
	err := idx.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketKnownDIDs)
		return b.ForEach(func(k, v []byte) error {
			dids = append(dids, string(k))
			return nil
		})
	})
	return dids, err
}

// RecordCount returns the total number of indexed records
func (idx *FeedIndex) RecordCount() int {
	var count int
	_ = idx.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketRecords)
		count = b.Stats().KeyN
		return nil
	})
	return count
}

// Helper functions

func makeTimeKey(t time.Time, uri string) []byte {
	// Format: inverted timestamp (for reverse chronological order) + ":" + uri
	// Use nanoseconds for uniqueness
	inverted := ^uint64(t.UnixNano())
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, inverted)
	return append(buf, []byte(":"+uri)...)
}

func extractURIFromTimeKey(key []byte) string {
	if len(key) < 10 { // 8 bytes timestamp + ":" + at least 1 char
		return ""
	}
	// Skip 8 bytes timestamp + 1 byte ":"
	return string(key[9:])
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
	var exists bool
	_ = idx.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketBackfilled)
		exists = b.Get([]byte(did)) != nil
		return nil
	})
	return exists
}

// MarkBackfilled marks a DID as backfilled with current timestamp
func (idx *FeedIndex) MarkBackfilled(did string) error {
	return idx.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketBackfilled)
		timestamp := []byte(time.Now().Format(time.RFC3339))
		return b.Put([]byte(did), timestamp)
	})
}

// BackfillUser fetches all existing records for a DID and adds them to the index
// Returns early if the DID has already been backfilled
func (idx *FeedIndex) BackfillUser(ctx context.Context, did string) error {
	// Check if already backfilled
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
			// Extract rkey from URI
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
			} else {
				recordCount++
			}
		}
	}

	// Mark as backfilled
	if err := idx.MarkBackfilled(did); err != nil {
		log.Warn().Err(err).Str("did", did).Msg("failed to mark DID as backfilled")
	}

	log.Info().Str("did", did).Int("record_count", recordCount).Msg("backfill complete")
	return nil
}

// ========== Like Indexing Methods ==========

// UpsertLike adds or updates a like in the index
func (idx *FeedIndex) UpsertLike(actorDID, rkey, subjectURI string) error {
	return idx.db.Update(func(tx *bolt.Tx) error {
		likes := tx.Bucket(BucketLikes)
		likeCounts := tx.Bucket(BucketLikeCounts)
		likesByActor := tx.Bucket(BucketLikesByActor)

		// Key format: {subject_uri}:{actor_did}
		likeKey := []byte(subjectURI + ":" + actorDID)

		// Check if this like already exists
		existingRKey := likes.Get(likeKey)
		if existingRKey != nil {
			// Already exists, nothing to do
			return nil
		}

		// Store the like mapping
		if err := likes.Put(likeKey, []byte(rkey)); err != nil {
			return err
		}

		// Store by actor for reverse lookup
		actorKey := []byte(actorDID + ":" + subjectURI)
		if err := likesByActor.Put(actorKey, []byte(rkey)); err != nil {
			return err
		}

		// Increment the like count
		countKey := []byte(subjectURI)
		currentCount := uint64(0)
		if countData := likeCounts.Get(countKey); len(countData) == 8 {
			currentCount = binary.BigEndian.Uint64(countData)
		}
		currentCount++
		countBuf := make([]byte, 8)
		binary.BigEndian.PutUint64(countBuf, currentCount)
		return likeCounts.Put(countKey, countBuf)
	})
}

// DeleteLike removes a like from the index
func (idx *FeedIndex) DeleteLike(actorDID, subjectURI string) error {
	return idx.db.Update(func(tx *bolt.Tx) error {
		likes := tx.Bucket(BucketLikes)
		likeCounts := tx.Bucket(BucketLikeCounts)
		likesByActor := tx.Bucket(BucketLikesByActor)

		// Key format: {subject_uri}:{actor_did}
		likeKey := []byte(subjectURI + ":" + actorDID)

		// Check if like exists
		if likes.Get(likeKey) == nil {
			// Doesn't exist, nothing to do
			return nil
		}

		// Delete the like mapping
		if err := likes.Delete(likeKey); err != nil {
			return err
		}

		// Delete by actor lookup
		actorKey := []byte(actorDID + ":" + subjectURI)
		if err := likesByActor.Delete(actorKey); err != nil {
			return err
		}

		// Decrement the like count
		countKey := []byte(subjectURI)
		currentCount := uint64(0)
		if countData := likeCounts.Get(countKey); len(countData) == 8 {
			currentCount = binary.BigEndian.Uint64(countData)
		}
		if currentCount > 0 {
			currentCount--
		}
		if currentCount == 0 {
			return likeCounts.Delete(countKey)
		}
		countBuf := make([]byte, 8)
		binary.BigEndian.PutUint64(countBuf, currentCount)
		return likeCounts.Put(countKey, countBuf)
	})
}

// GetLikeCount returns the number of likes for a record
func (idx *FeedIndex) GetLikeCount(subjectURI string) int {
	var count uint64
	_ = idx.db.View(func(tx *bolt.Tx) error {
		likeCounts := tx.Bucket(BucketLikeCounts)
		countData := likeCounts.Get([]byte(subjectURI))
		if len(countData) == 8 {
			count = binary.BigEndian.Uint64(countData)
		}
		return nil
	})
	return int(count)
}

// HasUserLiked checks if a user has liked a specific record
func (idx *FeedIndex) HasUserLiked(actorDID, subjectURI string) bool {
	var exists bool
	_ = idx.db.View(func(tx *bolt.Tx) error {
		likesByActor := tx.Bucket(BucketLikesByActor)
		actorKey := []byte(actorDID + ":" + subjectURI)
		exists = likesByActor.Get(actorKey) != nil
		return nil
	})
	return exists
}

// GetUserLikeRKey returns the rkey of a user's like for a specific record, or empty string if not found
func (idx *FeedIndex) GetUserLikeRKey(actorDID, subjectURI string) string {
	var rkey string
	_ = idx.db.View(func(tx *bolt.Tx) error {
		likesByActor := tx.Bucket(BucketLikesByActor)
		actorKey := []byte(actorDID + ":" + subjectURI)
		if data := likesByActor.Get(actorKey); data != nil {
			rkey = string(data)
		}
		return nil
	})
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
	return idx.db.Update(func(tx *bolt.Tx) error {
		comments := tx.Bucket(BucketComments)
		commentCounts := tx.Bucket(BucketCommentCounts)
		commentsByActor := tx.Bucket(BucketCommentsByActor)
		commentChildren := tx.Bucket(BucketCommentChildren)

		// Key format: {subject_uri}:{timestamp}:{actor_did}:{rkey}
		// Using timestamp for chronological ordering
		commentKey := []byte(subjectURI + ":" + createdAt.Format(time.RFC3339Nano) + ":" + actorDID + ":" + rkey)

		// Check if this comment already exists (by actor key)
		actorKey := []byte(actorDID + ":" + rkey)
		existingSubject := commentsByActor.Get(actorKey)
		isNew := existingSubject == nil

		// Extract parent rkey from parent URI if present
		var parentRKey string
		if parentURI != "" {
			parts := strings.Split(parentURI, "/")
			if len(parts) > 0 {
				parentRKey = parts[len(parts)-1]
			}
		}

		// Store comment data as JSON
		commentData := IndexedComment{
			RKey:       rkey,
			SubjectURI: subjectURI,
			Text:       text,
			ActorDID:   actorDID,
			CreatedAt:  createdAt,
			ParentURI:  parentURI,
			ParentRKey: parentRKey,
			CID:        cid,
		}
		commentJSON, err := json.Marshal(commentData)
		if err != nil {
			return fmt.Errorf("failed to marshal comment: %w", err)
		}

		// Store comment
		if err := comments.Put(commentKey, commentJSON); err != nil {
			return fmt.Errorf("failed to store comment: %w", err)
		}

		// Store actor lookup
		if err := commentsByActor.Put(actorKey, []byte(subjectURI)); err != nil {
			return fmt.Errorf("failed to store comment by actor: %w", err)
		}

		// Store parent-child relationship if this is a reply
		if parentURI != "" {
			childKey := []byte(parentURI + ":" + rkey)
			if err := commentChildren.Put(childKey, []byte(actorDID)); err != nil {
				return fmt.Errorf("failed to store comment child: %w", err)
			}
		}

		// Increment count only if this is a new comment
		if isNew {
			countKey := []byte(subjectURI)
			var count uint64
			if countData := commentCounts.Get(countKey); len(countData) == 8 {
				count = binary.BigEndian.Uint64(countData)
			}
			count++
			countBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(countBytes, count)
			if err := commentCounts.Put(countKey, countBytes); err != nil {
				return fmt.Errorf("failed to update comment count: %w", err)
			}
		}

		return nil
	})
}

// DeleteComment removes a comment from the index
func (idx *FeedIndex) DeleteComment(actorDID, rkey, subjectURI string) error {
	return idx.db.Update(func(tx *bolt.Tx) error {
		comments := tx.Bucket(BucketComments)
		commentCounts := tx.Bucket(BucketCommentCounts)
		commentsByActor := tx.Bucket(BucketCommentsByActor)
		commentChildren := tx.Bucket(BucketCommentChildren)

		actorKey := []byte(actorDID + ":" + rkey)

		// Check if comment exists
		existingSubject := commentsByActor.Get(actorKey)
		if existingSubject == nil {
			return nil // Comment doesn't exist, nothing to do
		}

		// Find and delete the comment by iterating over comments with matching subject
		var parentURI string
		prefix := []byte(subjectURI + ":")
		c := comments.Cursor()
		for k, v := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, v = c.Next() {
			// Check if this key contains our actor and rkey
			if strings.HasSuffix(string(k), ":"+actorDID+":"+rkey) {
				// Parse the comment to get parent URI for cleanup
				var comment IndexedComment
				if err := json.Unmarshal(v, &comment); err == nil {
					parentURI = comment.ParentURI
				}
				if err := comments.Delete(k); err != nil {
					return fmt.Errorf("failed to delete comment: %w", err)
				}
				break
			}
		}

		// Delete actor lookup
		if err := commentsByActor.Delete(actorKey); err != nil {
			return fmt.Errorf("failed to delete comment by actor: %w", err)
		}

		// Delete parent-child relationship if this was a reply
		if parentURI != "" {
			childKey := []byte(parentURI + ":" + rkey)
			if err := commentChildren.Delete(childKey); err != nil {
				return fmt.Errorf("failed to delete comment child: %w", err)
			}
		}

		// Decrement count
		countKey := []byte(subjectURI)
		var count uint64
		if countData := commentCounts.Get(countKey); len(countData) == 8 {
			count = binary.BigEndian.Uint64(countData)
		}
		if count > 0 {
			count--
		}
		countBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(countBytes, count)
		if err := commentCounts.Put(countKey, countBytes); err != nil {
			return fmt.Errorf("failed to update comment count: %w", err)
		}

		return nil
	})
}

// GetCommentCount returns the number of comments on a record
func (idx *FeedIndex) GetCommentCount(subjectURI string) int {
	var count uint64
	_ = idx.db.View(func(tx *bolt.Tx) error {
		commentCounts := tx.Bucket(BucketCommentCounts)
		countData := commentCounts.Get([]byte(subjectURI))
		if len(countData) == 8 {
			count = binary.BigEndian.Uint64(countData)
		}
		return nil
	})
	return int(count)
}

// GetCommentsForSubject returns all comments for a specific record, ordered by creation time
// This returns a flat list of comments without threading
func (idx *FeedIndex) GetCommentsForSubject(ctx context.Context, subjectURI string, limit int, viewerDID string) []IndexedComment {
	var comments []IndexedComment
	_ = idx.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketComments)
		prefix := []byte(subjectURI + ":")
		c := bucket.Cursor()

		for k, v := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, v = c.Next() {
			var comment IndexedComment
			if err := json.Unmarshal(v, &comment); err != nil {
				continue
			}
			comments = append(comments, comment)
			if limit > 0 && len(comments) >= limit {
				break
			}
		}
		return nil
	})

	// Populate profile and like info for each comment
	for i := range comments {
		profile, err := idx.GetProfile(ctx, comments[i].ActorDID)
		if err != nil {
			// Use DID as fallback handle
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
// Comments are returned in depth-first order (parent followed by children)
// Visual depth is capped at 2 levels for display purposes
func (idx *FeedIndex) GetThreadedCommentsForSubject(ctx context.Context, subjectURI string, limit int, viewerDID string) []IndexedComment {
	// First get all comments for this subject
	allComments := idx.GetCommentsForSubject(ctx, subjectURI, 0, viewerDID) // Get all, we'll limit after threading

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
			// Top-level comment
			topLevel = append(topLevel, comment)
		} else {
			// Reply - add to parent's children
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
		// Cap visual depth at 2 for display
		visualDepth := depth
		if visualDepth > 2 {
			visualDepth = 2
		}
		comment.Depth = visualDepth
		result = append(result, *comment)

		// Add children (if any)
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
