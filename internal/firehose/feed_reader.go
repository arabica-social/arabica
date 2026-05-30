package firehose

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/lexicons"

	"github.com/rs/zerolog/log"
)

// GetRecentFeed returns recent feed items from the index
func (idx *FeedIndex) GetRecentFeed(ctx context.Context, limit int) ([]*feed.FeedItem, error) {
	return idx.getFeedItems(ctx, nil, limit, "")
}

func feedableCollectionsForDescriptors(descriptors []*entities.Descriptor) (map[lexicons.RecordType]string, []string) {
	m := make(map[lexicons.RecordType]string)
	collections := make([]string, 0, len(descriptors))
	for _, d := range descriptors {
		if d != nil && d.Type != "" && d.NSID != "" {
			m[d.Type] = d.NSID
			collections = append(collections, d.NSID)
		}
	}
	sort.Strings(collections)
	return m, collections
}

// GetFeedWithQuery returns feed items matching the given query with cursor-based pagination
func (idx *FeedIndex) GetFeedWithQuery(ctx context.Context, q feed.FeedQuery) (*feed.FeedResult, error) {
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Sort == "" {
		q.Sort = feed.FeedSortRecent
	}

	// Determine collection filters
	var collectionFilters []string
	if len(q.TypeFilters) > 0 {
		for _, tf := range q.TypeFilters {
			nsid, ok := idx.recordTypeToNSID[tf]
			if !ok {
				return nil, fmt.Errorf("unknown record type: %s", tf)
			}
			collectionFilters = append(collectionFilters, nsid)
		}
	} else if q.TypeFilter != "" {
		nsid, ok := idx.recordTypeToNSID[q.TypeFilter]
		if !ok {
			return nil, fmt.Errorf("unknown record type: %s", q.TypeFilter)
		}
		collectionFilters = []string{nsid}
	}

	// For popular sort, fetch more candidates to re-rank by score
	fetchLimit := q.Limit + 1
	if q.Sort == feed.FeedSortPopular {
		fetchLimit = q.Limit * 5
	}

	items, err := idx.getFeedItems(ctx, collectionFilters, fetchLimit, q.Cursor)
	if err != nil {
		return nil, err
	}

	if q.Sort == feed.FeedSortPopular {
		sort.Slice(items, func(i, j int) bool {
			scoreI := items[i].LikeCount*3 + items[i].CommentCount*2
			scoreJ := items[j].LikeCount*3 + items[j].CommentCount*2
			if scoreI != scoreJ {
				return scoreI > scoreJ
			}
			return items[i].Timestamp.After(items[j].Timestamp)
		})
	}

	result := &feed.FeedResult{Items: items}
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
		if len(idx.feedableCollections) == 0 {
			return nil, nil
		}
		placeholders := make([]string, len(idx.feedableCollections))
		for i, c := range idx.feedableCollections {
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

		var recordData map[string]any
		if err := json.Unmarshal(rec.Record, &recordData); err == nil {
			collectRecordRefs(refURIs, rec.Collection, recordData)
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

	idx.fetchReferenceRecords(ctx, recordsByURI, refURIs)

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
		if _, ok := idx.recordTypeToNSID[item.RecordType]; !ok {
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
	behavior := entities.BehaviorByNSID(record.Collection)
	if desc == nil || behavior == nil || behavior.RecordToModel == nil {
		return nil, fmt.Errorf("unknown collection: %s", record.Collection)
	}
	model, err := behavior.RecordToModel(recordData, record.URI)
	if err != nil {
		return nil, err
	}

	item.RecordType = desc.Type
	item.Action = "added a new " + strings.ToLower(desc.DisplayName)
	item.Record = model

	// Per-entity reference resolution. The ref shape is genuinely
	// entity-specific; per-app record behaviors register a ResolveRefs hook
	// that hydrates their typed fields from refMap.
	if behavior.ResolveRefs != nil {
		lookup := func(refURI string) (map[string]any, bool) {
			rec, found := refMap[refURI]
			if !found {
				return nil, false
			}
			var data map[string]any
			if err := json.Unmarshal(rec.Record, &data); err != nil {
				return nil, false
			}
			return data, true
		}
		behavior.ResolveRefs(model, recordData, lookup)
	}

	// Populate subject fields (like/comment counts are set by caller via batch)
	item.SubjectURI = record.URI
	item.SubjectCID = record.CID

	return item, nil
}

func collectRecordRefs(refURIs map[string]bool, collection string, recordData map[string]any) {
	behavior := entities.BehaviorByNSID(collection)
	if behavior == nil {
		return
	}
	for _, key := range behavior.ReferenceFields {
		if ref, ok := recordData[key].(string); ok && ref != "" {
			refURIs[ref] = true
		}
	}
}

func (idx *FeedIndex) fetchReferenceRecords(ctx context.Context, recordsByURI map[string]*IndexedRecord, refURIs map[string]bool) {
	attempted := make(map[string]bool)
	for {
		missingURIs := make([]string, 0, len(refURIs))
		for uri := range refURIs {
			if _, found := recordsByURI[uri]; found || attempted[uri] {
				continue
			}
			attempted[uri] = true
			missingURIs = append(missingURIs, uri)
		}
		if len(missingURIs) == 0 {
			return
		}

		placeholders := make([]string, len(missingURIs))
		refArgs := make([]any, len(missingURIs))
		for i, uri := range missingURIs {
			placeholders[i] = "?"
			refArgs[i] = uri
		}
		refQuery := `SELECT uri, did, collection, rkey, record, cid, indexed_at, created_at FROM records WHERE uri IN (` + strings.Join(placeholders, ",") + `)`
		refRows, err := idx.db.QueryContext(ctx, refQuery, refArgs...)
		if err != nil {
			return
		}

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

			var recordData map[string]any
			if err := json.Unmarshal(rec.Record, &recordData); err == nil {
				collectRecordRefs(refURIs, rec.Collection, recordData)
			}
		}
		refRows.Close()
	}
}
