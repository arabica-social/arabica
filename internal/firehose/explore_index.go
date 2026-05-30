package firehose

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/explore"
	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/lexicons"

	"github.com/rs/zerolog/log"
)

const (
	exploreIndexVersion    = "1"
	popularSourceRefWeight = 5
	popularLikeWeight      = 3
	popularCommentWeight   = 2
)

type ExploreQuery struct {
	App     string
	Type    lexicons.RecordType
	Q       string
	Sort    string
	Limit   int
	Cursor  string
	Filters map[string]string
}

type ExploreResult struct {
	Items       []*feed.FeedItem
	Documents   map[string]ExploreDocument
	FacetCounts []ExploreFacetCount
	NextCursor  string
}

type ExploreDocument struct {
	URI             string
	DID             string
	RecordType      lexicons.RecordType
	ClusterKey      string
	Title           string
	Summary         string
	OwnRating       sql.NullFloat64
	CommunityRating sql.NullFloat64
	RatingCount     int
	LikeCount       int
	CommentCount    int
	SourceRefCount  int
	PopularScore    float64
	CreatedAt       time.Time
}

type ExploreFacetCount struct {
	Field string
	Value string
	Count int
}

type ExploreHealth struct {
	Ready          bool
	CurrentVersion string
	StoredVersion  string
	Dirty          bool
	LastError      string
	DocumentCount  int
	ValueCount     int
}

func (idx *FeedIndex) ensureExploreIndex(ctx context.Context) {
	var stored string
	_ = idx.db.QueryRowContext(ctx, `SELECT CAST(value AS TEXT) FROM meta WHERE key = 'explore_index_version'`).Scan(&stored)
	if stored == exploreIndexVersion {
		return
	}
	if err := idx.RebuildExploreIndex(ctx); err != nil {
		log.Warn().Err(err).Msg("explore index rebuild failed")
		idx.markExploreDirty(ctx, err)
	}
}

func (idx *FeedIndex) RebuildExploreIndex(ctx context.Context) error {
	tx, err := idx.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err := tx.ExecContext(ctx, `DELETE FROM explore_values`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM explore_documents`); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	rows, err := idx.db.QueryContext(ctx, `SELECT uri FROM records ORDER BY created_at ASC`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		if err := rows.Scan(&uri); err != nil {
			return err
		}
		if err := idx.reindexExploreRecord(ctx, uri); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = idx.db.ExecContext(ctx, `INSERT INTO meta(key,value) VALUES('explore_index_version', ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, exploreIndexVersion)
	if err == nil {
		_, _ = idx.db.ExecContext(ctx, `DELETE FROM meta WHERE key IN ('explore_dirty','explore_last_error')`)
	}
	return err
}

func (idx *FeedIndex) markExploreDirty(ctx context.Context, cause error) {
	_, _ = idx.db.ExecContext(ctx, `INSERT INTO meta(key,value) VALUES('explore_dirty','1') ON CONFLICT(key) DO UPDATE SET value='1'`)
	if cause != nil {
		_, _ = idx.db.ExecContext(ctx, `INSERT INTO meta(key,value) VALUES('explore_last_error',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, cause.Error())
	}
}

func (idx *FeedIndex) ExploreHealth(ctx context.Context) ExploreHealth {
	h := ExploreHealth{CurrentVersion: exploreIndexVersion}
	_ = idx.db.QueryRowContext(ctx, `SELECT CAST(value AS TEXT) FROM meta WHERE key='explore_index_version'`).Scan(&h.StoredVersion)
	var dirty string
	_ = idx.db.QueryRowContext(ctx, `SELECT CAST(value AS TEXT) FROM meta WHERE key='explore_dirty'`).Scan(&dirty)
	h.Dirty = dirty == "1"
	_ = idx.db.QueryRowContext(ctx, `SELECT CAST(value AS TEXT) FROM meta WHERE key='explore_last_error'`).Scan(&h.LastError)
	_ = idx.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM explore_documents`).Scan(&h.DocumentCount)
	_ = idx.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM explore_values`).Scan(&h.ValueCount)
	h.Ready = h.StoredVersion == exploreIndexVersion && !h.Dirty
	return h
}

func (idx *FeedIndex) reindexExploreRecord(ctx context.Context, uri string) error {
	var rec IndexedRecord
	var recordStr, createdAtStr, indexedAtStr string
	err := idx.db.QueryRowContext(ctx, `SELECT uri,did,collection,rkey,record,cid,indexed_at,created_at FROM records WHERE uri=?`, uri).Scan(&rec.URI, &rec.DID, &rec.Collection, &rec.RKey, &recordStr, &rec.CID, &indexedAtStr, &createdAtStr)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	rec.Record = json.RawMessage(recordStr)
	rec.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
	reg := explore.NewArabicaRegistry(idx.recordTypeToNSID)
	typ, ok := reg.TypeByNSID(rec.Collection)
	if !ok {
		_, _ = idx.db.ExecContext(ctx, `DELETE FROM explore_values WHERE uri=?`, uri)
		_, _ = idx.db.ExecContext(ctx, `DELETE FROM explore_documents WHERE uri=?`, uri)
		return nil
	}
	var data map[string]any
	if err := json.Unmarshal(rec.Record, &data); err != nil {
		return err
	}
	refs := func(refURI string) (string, map[string]any, bool) {
		if refURI == "" {
			return "", nil, false
		}
		var coll, raw string
		if err := idx.db.QueryRowContext(ctx, `SELECT collection, record FROM records WHERE uri=?`, refURI).Scan(&coll, &raw); err != nil {
			return "", nil, false
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			return "", nil, false
		}
		return coll, m, true
	}
	doc := typ.Extract(data, refs)
	clusterKey := doc.SourceRef
	if clusterKey == "" {
		clusterKey = uri
	}
	canonicalRank := 1.0
	if uri == clusterKey {
		canonicalRank = 2
	}
	likeCount, commentCount := idx.GetLikeCount(ctx, uri), idx.GetCommentCount(ctx, uri)
	sourceRefCount := idx.countSourceRefs(ctx, uri)
	popular := float64(sourceRefCount*popularSourceRefWeight + likeCount*popularLikeWeight + commentCount*popularCommentWeight)
	communityRating, ratingCount := idx.exploreCommunityRating(ctx, typ.RecordType, uri)
	tx, err := idx.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err := tx.ExecContext(ctx, `DELETE FROM explore_values WHERE uri=?`, uri); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO explore_documents(uri,did,app,record_type,cluster_key,canonical_rank,title,summary,search_text,own_rating,community_rating,rating_count,like_count,comment_count,source_ref_count,popular_score,created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(uri) DO UPDATE SET did=excluded.did, app=excluded.app, record_type=excluded.record_type, cluster_key=excluded.cluster_key, canonical_rank=excluded.canonical_rank, title=excluded.title, summary=excluded.summary, search_text=excluded.search_text, own_rating=excluded.own_rating, community_rating=excluded.community_rating, rating_count=excluded.rating_count, like_count=excluded.like_count, comment_count=excluded.comment_count, source_ref_count=excluded.source_ref_count, popular_score=excluded.popular_score, created_at=excluded.created_at`,
		uri, rec.DID, typ.App, string(typ.RecordType), clusterKey, canonicalRank, doc.Title, doc.Summary, doc.SearchText, doc.OwnRating, communityRating, ratingCount, likeCount, commentCount, sourceRefCount, popular, createdAtStr); err != nil {
		return err
	}
	for _, v := range doc.Values {
		var n any
		if v.Num != nil {
			n = *v.Num
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO explore_values(uri,did,app,record_type,field,value_text,value_num,created_at) VALUES(?,?,?,?,?,?,?,?)`, uri, rec.DID, typ.App, string(typ.RecordType), v.Field, v.Text, n, createdAtStr); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (idx *FeedIndex) countSourceRefs(ctx context.Context, uri string) int {
	var count int
	_ = idx.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM records WHERE json_extract(record,'$.sourceRef') = ?`, uri).Scan(&count)
	return count
}

func (idx *FeedIndex) exploreCommunityRating(ctx context.Context, recordType lexicons.RecordType, uri string) (any, int) {
	brewNSID := idx.recordTypeToNSID[lexicons.RecordTypeBrew]
	if brewNSID == "" {
		return nil, 0
	}
	var field string
	switch recordType {
	case lexicons.RecordTypeBean:
		field = "beanRef"
	case lexicons.RecordTypeGrinder:
		field = "grinderRef"
	case lexicons.RecordTypeBrewer:
		field = "brewerRef"
	case lexicons.RecordTypeRecipe:
		field = "recipeRef"
	case lexicons.RecordTypeRoaster:
		return idx.exploreRoasterCommunityRating(ctx, uri)
	default:
		return nil, 0
	}
	return idx.exploreAverageBrewRating(ctx, brewNSID, field, uri)
}

func (idx *FeedIndex) exploreAverageBrewRating(ctx context.Context, brewNSID, refField, refURI string) (any, int) {
	var avg sql.NullFloat64
	var count int
	_ = idx.db.QueryRowContext(ctx, `SELECT AVG(CAST(json_extract(record, '$.rating') AS REAL)), COUNT(*) FROM records WHERE collection = ? AND json_extract(record, '$.`+refField+`') = ? AND json_type(record, '$.rating') IS NOT NULL`, brewNSID, refURI).Scan(&avg, &count)
	if !avg.Valid || count == 0 {
		return nil, 0
	}
	return avg.Float64, count
}

func (idx *FeedIndex) exploreRoasterCommunityRating(ctx context.Context, roasterURI string) (any, int) {
	brewNSID := idx.recordTypeToNSID[lexicons.RecordTypeBrew]
	beanNSID := idx.recordTypeToNSID[lexicons.RecordTypeBean]
	if brewNSID == "" || beanNSID == "" {
		return nil, 0
	}
	var avg sql.NullFloat64
	var count int
	_ = idx.db.QueryRowContext(ctx, `SELECT AVG(CAST(json_extract(brew.record, '$.rating') AS REAL)), COUNT(*) FROM records brew WHERE brew.collection = ? AND json_type(brew.record, '$.rating') IS NOT NULL AND json_extract(brew.record, '$.beanRef') IN (SELECT bean.uri FROM records bean WHERE bean.collection = ? AND json_extract(bean.record, '$.roasterRef') = ?)`, brewNSID, beanNSID, roasterURI).Scan(&avg, &count)
	if !avg.Valid || count == 0 {
		return nil, 0
	}
	return avg.Float64, count
}

func exploreSourceRef(record json.RawMessage) string {
	var data map[string]any
	if err := json.Unmarshal(record, &data); err != nil {
		return ""
	}
	v, _ := data["sourceRef"].(string)
	return strings.TrimSpace(v)
}

func (idx *FeedIndex) reindexExploreBrewReferences(ctx context.Context, record json.RawMessage) error {
	var data map[string]any
	if err := json.Unmarshal(record, &data); err != nil {
		return err
	}
	seen := make(map[string]struct{})
	for _, field := range []string{"beanRef", "grinderRef", "brewerRef", "recipeRef"} {
		uri, _ := data[field].(string)
		if uri == "" {
			continue
		}
		if _, ok := seen[uri]; !ok {
			seen[uri] = struct{}{}
			if err := idx.reindexExploreRecord(ctx, uri); err != nil {
				return err
			}
		}
		if field == "beanRef" {
			roasterURI := idx.roasterRefForBean(ctx, uri)
			if roasterURI != "" {
				if err := idx.reindexExploreRecord(ctx, roasterURI); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (idx *FeedIndex) roasterRefForBean(ctx context.Context, beanURI string) string {
	var raw string
	if err := idx.db.QueryRowContext(ctx, `SELECT record FROM records WHERE uri = ?`, beanURI).Scan(&raw); err != nil {
		return ""
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return ""
	}
	uri, _ := data["roasterRef"].(string)
	return uri
}

func (idx *FeedIndex) refreshExploreStats(ctx context.Context, uri string) error {
	likeCount, commentCount := idx.GetLikeCount(ctx, uri), idx.GetCommentCount(ctx, uri)
	sourceRefCount := idx.countSourceRefs(ctx, uri)
	popular := float64(sourceRefCount*popularSourceRefWeight + likeCount*popularLikeWeight + commentCount*popularCommentWeight)
	_, err := idx.db.ExecContext(ctx, `UPDATE explore_documents SET like_count=?, comment_count=?, source_ref_count=?, popular_score=? WHERE uri=?`, likeCount, commentCount, sourceRefCount, popular, uri)
	return err
}

func (idx *FeedIndex) exploreSubjectsAffectedByDID(ctx context.Context, did, uriPrefix string) map[string]struct{} {
	affected := make(map[string]struct{})
	queries := []struct {
		sql  string
		args []any
	}{
		{`SELECT subject_uri FROM likes WHERE actor_did = ?`, []any{did}},
		{`SELECT subject_uri FROM likes WHERE subject_uri LIKE ?`, []any{uriPrefix}},
		{`SELECT subject_uri FROM comments WHERE actor_did = ?`, []any{did}},
		{`SELECT subject_uri FROM comments WHERE subject_uri LIKE ?`, []any{uriPrefix}},
	}
	for _, q := range queries {
		rows, err := idx.db.QueryContext(ctx, q.sql, q.args...)
		if err != nil {
			continue
		}
		for rows.Next() {
			var subject string
			if err := rows.Scan(&subject); err == nil && subject != "" {
				affected[subject] = struct{}{}
			}
		}
		_ = rows.Close()
	}
	return affected
}

func (idx *FeedIndex) exploreSourceRefsByDID(ctx context.Context, did string) map[string]struct{} {
	refs := make(map[string]struct{})
	rows, err := idx.db.QueryContext(ctx, `SELECT json_extract(record, '$.sourceRef') FROM records WHERE did = ? AND json_type(record, '$.sourceRef') IS NOT NULL`, did)
	if err != nil {
		return refs
	}
	defer rows.Close()
	for rows.Next() {
		var ref string
		if err := rows.Scan(&ref); err == nil && ref != "" {
			refs[ref] = struct{}{}
		}
	}
	return refs
}

func (idx *FeedIndex) reindexExploreDependents(ctx context.Context, uri, collection string) error {
	// V1 only refreshes narrow first-hop dependencies: roaster -> beans, brewer -> recipes.
	var field string
	switch collection {
	case idx.recordTypeToNSID[lexicons.RecordTypeRoaster]:
		field = "roasterRef"
	case idx.recordTypeToNSID[lexicons.RecordTypeBrewer]:
		field = "brewerRef"
	default:
		return nil
	}
	rows, err := idx.db.QueryContext(ctx, `SELECT uri FROM records WHERE json_extract(record, '$.`+field+`') = ?`, uri)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var dep string
		if err := rows.Scan(&dep); err != nil {
			return err
		}
		if err := idx.reindexExploreRecord(ctx, dep); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (idx *FeedIndex) GetExplore(ctx context.Context, q ExploreQuery) (*ExploreResult, error) {
	reg := explore.NewArabicaRegistry(idx.recordTypeToNSID)
	if q.App == "" {
		q.App = "arabica"
	}
	if q.Limit <= 0 || q.Limit > 50 {
		q.Limit = 20
	}
	q.Sort = reg.ValidateSort(q.Sort)
	var wh []string
	args := []any{q.App}
	wh = append(wh, "app = ?")
	if q.Type != "" {
		if _, ok := reg.Type(q.Type); !ok {
			return nil, fmt.Errorf("unknown explore type: %s", q.Type)
		}
		wh = append(wh, "record_type = ?")
		args = append(args, string(q.Type))
	} else {
		var ph []string
		for _, t := range reg.Types() {
			ph = append(ph, "?")
			args = append(args, string(t.RecordType))
		}
		wh = append(wh, "record_type IN ("+strings.Join(ph, ",")+")")
	}
	if s := strings.TrimSpace(q.Q); s != "" {
		wh = append(wh, "search_text LIKE ?")
		args = append(args, "%"+strings.ToLower(s)+"%")
	}
	for name, val := range q.Filters {
		if strings.TrimSpace(val) == "" {
			continue
		}
		def, ok := reg.ValidateFilter(q.Type, name)
		if !ok {
			continue
		}
		alias := "v_" + strings.ReplaceAll(name, "-", "_")
		switch def.Kind {
		case explore.FilterFacetText:
			wh = append(wh, fmt.Sprintf("EXISTS (SELECT 1 FROM explore_values %s WHERE %s.uri = d.uri AND %s.field = ? AND lower(%s.value_text) LIKE ? ESCAPE '\\')", alias, alias, alias, alias))
			args = append(args, def.Field, exploreContainsPattern(val))
		case explore.FilterBool:
			wh = append(wh, fmt.Sprintf("EXISTS (SELECT 1 FROM explore_values %s WHERE %s.uri = d.uri AND %s.field = ? AND lower(%s.value_text) = lower(?))", alias, alias, alias, alias))
			args = append(args, def.Field, val)
		case explore.FilterNumberMin:
			f, err := strconv.ParseFloat(val, 64)
			if err != nil {
				continue
			}
			wh = append(wh, fmt.Sprintf("EXISTS (SELECT 1 FROM explore_values %s WHERE %s.uri = d.uri AND %s.field = ? AND %s.value_num >= ?)", alias, alias, alias, alias))
			args = append(args, def.Field, f)
		case explore.FilterNumberMax:
			f, err := strconv.ParseFloat(val, 64)
			if err != nil {
				continue
			}
			wh = append(wh, fmt.Sprintf("EXISTS (SELECT 1 FROM explore_values %s WHERE %s.uri = d.uri AND %s.field = ? AND %s.value_num <= ?)", alias, alias, alias, alias))
			args = append(args, def.Field, f)
		}
	}
	order, cursorWhere, cursorArgs := exploreSortSQL(q.Sort, q.Cursor)
	canonicalWhere := "rn=1"
	if cursorWhere != "" {
		canonicalWhere += " AND " + cursorWhere
		args = append(args, cursorArgs...)
	}
	query := `WITH filtered AS (SELECT d.*, COALESCE(d.community_rating, d.own_rating, -1) AS rating_sort FROM explore_documents d WHERE ` + strings.Join(wh, " AND ") + `), ranked AS (SELECT *, ROW_NUMBER() OVER (PARTITION BY cluster_key ORDER BY canonical_rank DESC, popular_score DESC, created_at DESC, uri ASC) rn FROM filtered) SELECT uri,did,record_type,cluster_key,title,summary,own_rating,community_rating,rating_count,like_count,comment_count,source_ref_count,popular_score,created_at FROM ranked WHERE ` + canonicalWhere + ` ` + order + ` LIMIT ?`
	args = append(args, q.Limit+1)
	rows, err := idx.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var docs []ExploreDocument
	var uris []string
	for rows.Next() {
		var d ExploreDocument
		var created string
		if err := rows.Scan(&d.URI, &d.DID, &d.RecordType, &d.ClusterKey, &d.Title, &d.Summary, &d.OwnRating, &d.CommunityRating, &d.RatingCount, &d.LikeCount, &d.CommentCount, &d.SourceRefCount, &d.PopularScore, &created); err != nil {
			return nil, err
		}
		d.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		docs = append(docs, d)
		uris = append(uris, d.URI)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	res := &ExploreResult{Documents: make(map[string]ExploreDocument)}
	if len(docs) > q.Limit {
		last := docs[q.Limit-1]
		res.NextCursor = encodeExploreCursor(q.Sort, last)
		docs = docs[:q.Limit]
		uris = uris[:q.Limit]
	}
	items, err := idx.feedItemsForURIs(ctx, uris)
	if err != nil {
		return nil, err
	}
	res.Items = items
	for _, d := range docs {
		res.Documents[d.URI] = d
	}
	res.FacetCounts = idx.exploreFacetCounts(ctx, q)
	return res, nil
}

func exploreSortSQL(sort, cursor string) (string, string, []any) {
	parts := decodeExploreCursor(cursor)
	switch sort {
	case explore.SortPopular:
		if len(parts) == 4 {
			score, _ := strconv.ParseFloat(parts[0], 64)
			return `ORDER BY popular_score DESC, created_at DESC, cluster_key ASC, uri ASC`, `((popular_score < ?) OR (popular_score = ? AND created_at < ?) OR (popular_score = ? AND created_at = ? AND cluster_key > ?) OR (popular_score = ? AND created_at = ? AND cluster_key = ? AND uri > ?))`, []any{score, score, parts[1], score, parts[1], parts[2], score, parts[1], parts[2], parts[3]}
		}
		return `ORDER BY popular_score DESC, created_at DESC, cluster_key ASC, uri ASC`, "", nil
	case explore.SortRatingHigh:
		if len(parts) == 4 {
			rating, _ := strconv.ParseFloat(parts[0], 64)
			return `ORDER BY rating_sort DESC, created_at DESC, cluster_key ASC, uri ASC`, `((rating_sort < ?) OR (rating_sort = ? AND created_at < ?) OR (rating_sort = ? AND created_at = ? AND cluster_key > ?) OR (rating_sort = ? AND created_at = ? AND cluster_key = ? AND uri > ?))`, []any{rating, rating, parts[1], rating, parts[1], parts[2], rating, parts[1], parts[2], parts[3]}
		}
		return `ORDER BY rating_sort DESC, created_at DESC, cluster_key ASC, uri ASC`, "", nil
	default:
		if len(parts) == 3 {
			return `ORDER BY created_at DESC, cluster_key ASC, uri ASC`, `((created_at < ?) OR (created_at = ? AND cluster_key > ?) OR (created_at = ? AND cluster_key = ? AND uri > ?))`, []any{parts[0], parts[0], parts[1], parts[0], parts[1], parts[2]}
		}
		return `ORDER BY created_at DESC, cluster_key ASC, uri ASC`, "", nil
	}
}
func encodeExploreCursor(sort string, d ExploreDocument) string {
	var raw string
	if sort == explore.SortPopular {
		raw = fmt.Sprintf("%f|%s|%s|%s", d.PopularScore, d.CreatedAt.Format(time.RFC3339Nano), d.ClusterKey, d.URI)
	} else if sort == explore.SortRatingHigh {
		rating := -1.0
		if d.CommunityRating.Valid {
			rating = d.CommunityRating.Float64
		} else if d.OwnRating.Valid {
			rating = d.OwnRating.Float64
		}
		raw = fmt.Sprintf("%f|%s|%s|%s", rating, d.CreatedAt.Format(time.RFC3339Nano), d.ClusterKey, d.URI)
	} else {
		raw = fmt.Sprintf("%s|%s|%s", d.CreatedAt.Format(time.RFC3339Nano), d.ClusterKey, d.URI)
	}
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func ExploreCursor(sort string, d ExploreDocument) string {
	return encodeExploreCursor(sort, d)
}

func exploreContainsPattern(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return "%" + s + "%"
}

func decodeExploreCursor(s string) []string {
	if s == "" {
		return nil
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil
	}
	return strings.Split(string(b), "|")
}

func (idx *FeedIndex) feedItemsForURIs(ctx context.Context, uris []string) ([]*feed.FeedItem, error) {
	recs := idx.GetRecordsBatch(ctx, uris)
	refMap := make(map[string]*IndexedRecord, len(recs))
	profiles := make(map[string]*atproto.Profile)
	for _, r := range recs {
		refMap[r.URI] = r
		var data map[string]any
		if err := json.Unmarshal(r.Record, &data); err == nil {
			for _, k := range []string{"roasterRef", "brewerRef"} {
				if u, _ := data[k].(string); u != "" {
					if rr := idx.GetRecordsBatch(ctx, []string{u})[u]; rr != nil {
						refMap[u] = rr
					}
				}
			}
		}
		if _, ok := profiles[r.DID]; !ok {
			p, _ := idx.GetProfile(ctx, r.DID)
			profiles[r.DID] = p
		}
	}
	items := make([]*feed.FeedItem, 0, len(uris))
	likeCounts, commentCounts := idx.GetLikeCountsBatch(ctx, uris), idx.GetCommentCountsBatch(ctx, uris)
	for _, uri := range uris {
		r := recs[uri]
		if r == nil {
			continue
		}
		item, err := idx.recordToFeedItem(ctx, r, refMap, profiles)
		if err != nil {
			continue
		}
		item.Action = ""
		item.LikeCount = likeCounts[uri]
		item.CommentCount = commentCounts[uri]
		items = append(items, item)
	}
	return items, nil
}

func (idx *FeedIndex) exploreFacetCounts(ctx context.Context, q ExploreQuery) []ExploreFacetCount {
	// V1 counts are raw and unmoderated; visible results are filtered later by the shared moderation filter.
	args := []any{q.App}
	where := `app = ?`
	if q.Type != "" {
		where += ` AND record_type = ?`
		args = append(args, string(q.Type))
	}
	rows, err := idx.db.QueryContext(ctx, `SELECT field, value_text, COUNT(DISTINCT uri) FROM explore_values WHERE `+where+` AND value_text IS NOT NULL AND value_text <> '' GROUP BY field, value_text ORDER BY field, COUNT(DISTINCT uri) DESC, value_text LIMIT 200`, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []ExploreFacetCount
	for rows.Next() {
		var f ExploreFacetCount
		if err := rows.Scan(&f.Field, &f.Value, &f.Count); err == nil {
			out = append(out, f)
		}
	}
	return out
}
