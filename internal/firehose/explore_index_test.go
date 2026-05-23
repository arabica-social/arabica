package firehose

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/explore"
	"tangled.org/arabica.social/arabica/internal/lexicons"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newExploreTestIndex(t *testing.T) *FeedIndex {
	t.Helper()
	idx, err := NewFeedIndex(t.TempDir()+"/test.db", time.Hour)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, idx.Close()) })
	return idx
}

func rawRecord(t *testing.T, fields map[string]any) json.RawMessage {
	t.Helper()
	if _, ok := fields["createdAt"]; !ok {
		fields["createdAt"] = "2026-05-23T12:00:00Z"
	}
	b, err := json.Marshal(fields)
	require.NoError(t, err)
	return b
}

func storeExploreProfile(idx *FeedIndex, did, handle string) {
	dn := handle
	idx.StoreProfile(context.Background(), did, &atproto.Profile{DID: did, Handle: handle, DisplayName: &dn})
}

func upsertExploreRecord(t *testing.T, idx *FeedIndex, did, collection, rkey string, fields map[string]any, seconds int) string {
	t.Helper()
	ctx := context.Background()
	storeExploreProfile(idx, did, did+".test")
	if seconds != 0 {
		fields["createdAt"] = time.Date(2026, 5, 23, 12, 0, seconds, 0, time.UTC).Format(time.RFC3339)
	}
	rec := rawRecord(t, fields)
	require.NoError(t, idx.UpsertRecord(ctx, did, collection, rkey, "cid-"+rkey, rec, time.Now().Unix()))
	return fmt.Sprintf("at://%s/%s/%s", did, collection, rkey)
}

func TestExploreIndexesConfiguredArabicaTypesAndValues(t *testing.T) {
	idx := newExploreTestIndex(t)
	ctx := context.Background()
	roasterURI := upsertExploreRecord(t, idx, "did:plc:roaster", arabica.NSIDRoaster, "r1", map[string]any{"$type": arabica.NSIDRoaster, "name": "Pilot", "location": "Toronto"}, 1)
	upsertExploreRecord(t, idx, "did:plc:user", arabica.NSIDBean, "b1", map[string]any{"$type": arabica.NSIDBean, "name": "Ardi", "origin": "Ethiopia", "variety": "Heirloom", "process": "Washed", "roastLevel": "Light", "rating": 9, "closed": false, "roasterRef": roasterURI, "description": "floral"}, 2)
	upsertExploreRecord(t, idx, "did:plc:user", arabica.NSIDGrinder, "g1", map[string]any{"$type": arabica.NSIDGrinder, "name": "C40", "grinderType": "hand", "burrType": "conical", "notes": "travel"}, 3)
	upsertExploreRecord(t, idx, "did:plc:user", arabica.NSIDBrewer, "br1", map[string]any{"$type": arabica.NSIDBrewer, "name": "V60", "brewerType": "pourover", "description": "cone"}, 4)
	upsertExploreRecord(t, idx, "did:plc:user", arabica.NSIDRecipe, "rec1", map[string]any{"$type": arabica.NSIDRecipe, "name": "Morning V60", "brewerType": "pourover", "coffeeAmount": 200, "waterAmount": 3200, "notes": "sweet"}, 5)

	res, err := idx.GetExplore(ctx, ExploreQuery{App: "arabica", Limit: 10})
	require.NoError(t, err)
	assert.Len(t, res.Items, 5)

	res, err = idx.GetExplore(ctx, ExploreQuery{App: "arabica", Type: lexicons.RecordTypeBean, Q: "ethiopia", Filters: map[string]string{"roaster": "Pilot", "origin": "Ethiopia", "min_rating": "8", "closed": "false"}})
	require.NoError(t, err)
	require.Len(t, res.Items, 1)
	assert.Equal(t, lexicons.RecordTypeBean, res.Items[0].RecordType)
	assert.Equal(t, 9.0, res.Documents[res.Items[0].SubjectURI].OwnRating.Float64)

	res, err = idx.GetExplore(ctx, ExploreQuery{App: "arabica", Type: lexicons.RecordTypeRecipe, Filters: map[string]string{"brewer_type": "pourover", "ratio_min": "15", "ratio_max": "17"}})
	require.NoError(t, err)
	assert.Len(t, res.Items, 1)
}

func TestExploreClustersBySourceRefAndFallsBackWhenCanonicalDeleted(t *testing.T) {
	idx := newExploreTestIndex(t)
	ctx := context.Background()
	original := upsertExploreRecord(t, idx, "did:plc:one", arabica.NSIDBean, "b1", map[string]any{"$type": arabica.NSIDBean, "name": "Original", "origin": "Kenya"}, 1)
	fork := upsertExploreRecord(t, idx, "did:plc:two", arabica.NSIDBean, "b2", map[string]any{"$type": arabica.NSIDBean, "name": "Fork", "origin": "Kenya", "sourceRef": original}, 2)

	res, err := idx.GetExplore(ctx, ExploreQuery{App: "arabica", Type: lexicons.RecordTypeBean})
	require.NoError(t, err)
	require.Len(t, res.Items, 1)
	assert.Equal(t, original, res.Items[0].SubjectURI)
	assert.Equal(t, 1, res.Documents[original].SourceRefCount)

	require.NoError(t, idx.DeleteRecord(ctx, "did:plc:one", arabica.NSIDBean, "b1"))
	res, err = idx.GetExplore(ctx, ExploreQuery{App: "arabica", Type: lexicons.RecordTypeBean})
	require.NoError(t, err)
	require.Len(t, res.Items, 1)
	assert.Equal(t, fork, res.Items[0].SubjectURI)
}

func TestExplorePopularAndSocialStatsRefresh(t *testing.T) {
	idx := newExploreTestIndex(t)
	ctx := context.Background()
	first := upsertExploreRecord(t, idx, "did:plc:one", arabica.NSIDBean, "b1", map[string]any{"$type": arabica.NSIDBean, "name": "Quiet"}, 1)
	second := upsertExploreRecord(t, idx, "did:plc:two", arabica.NSIDBean, "b2", map[string]any{"$type": arabica.NSIDBean, "name": "Social"}, 2)

	require.NoError(t, idx.UpsertLike(ctx, "did:plc:fan", "l1", second))
	require.NoError(t, idx.UpsertComment(ctx, "did:plc:fan", "c1", second, "", "cid", "nice", time.Now()))
	res, err := idx.GetExplore(ctx, ExploreQuery{App: "arabica", Type: lexicons.RecordTypeBean, Sort: explore.SortPopular})
	require.NoError(t, err)
	require.Len(t, res.Items, 2)
	assert.Equal(t, second, res.Items[0].SubjectURI)
	assert.Equal(t, 1, res.Items[0].LikeCount)
	assert.Equal(t, 1, res.Items[0].CommentCount)
	assert.Equal(t, float64(popularLikeWeight+popularCommentWeight), res.Documents[second].PopularScore)

	require.NoError(t, idx.DeleteLike(ctx, "did:plc:fan", second))
	require.NoError(t, idx.DeleteComment(ctx, "did:plc:fan", "c1", second))
	res, err = idx.GetExplore(ctx, ExploreQuery{App: "arabica", Type: lexicons.RecordTypeBean, Sort: explore.SortPopular})
	require.NoError(t, err)
	assert.Equal(t, 0, res.Documents[second].LikeCount)
	assert.Equal(t, 0, res.Documents[second].CommentCount)
	assert.NotEmpty(t, first)
}

func TestExploreRatingSortAndCursor(t *testing.T) {
	idx := newExploreTestIndex(t)
	ctx := context.Background()
	low := upsertExploreRecord(t, idx, "did:plc:one", arabica.NSIDBean, "b1", map[string]any{"$type": arabica.NSIDBean, "name": "Low", "rating": 3}, 1)
	high := upsertExploreRecord(t, idx, "did:plc:two", arabica.NSIDBean, "b2", map[string]any{"$type": arabica.NSIDBean, "name": "High", "rating": 9}, 2)
	unrated := upsertExploreRecord(t, idx, "did:plc:three", arabica.NSIDBean, "b3", map[string]any{"$type": arabica.NSIDBean, "name": "Unrated"}, 3)

	res, err := idx.GetExplore(ctx, ExploreQuery{App: "arabica", Type: lexicons.RecordTypeBean, Sort: explore.SortRatingHigh, Limit: 2})
	require.NoError(t, err)
	require.Len(t, res.Items, 2)
	assert.Equal(t, high, res.Items[0].SubjectURI)
	assert.Equal(t, low, res.Items[1].SubjectURI)
	assert.NotEmpty(t, res.NextCursor)

	res, err = idx.GetExplore(ctx, ExploreQuery{App: "arabica", Type: lexicons.RecordTypeBean, Sort: explore.SortRatingHigh, Cursor: res.NextCursor, Limit: 2})
	require.NoError(t, err)
	require.Len(t, res.Items, 1)
	assert.Equal(t, unrated, res.Items[0].SubjectURI)
}

func TestExploreCascadeDeleteAndVersionedRebuild(t *testing.T) {
	idx := newExploreTestIndex(t)
	ctx := context.Background()
	uri := upsertExploreRecord(t, idx, "did:plc:one", arabica.NSIDRoaster, "r1", map[string]any{"$type": arabica.NSIDRoaster, "name": "Cascade", "location": "Portland"}, 1)
	var docs int
	require.NoError(t, idx.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM explore_documents WHERE uri=?`, uri).Scan(&docs))
	assert.Equal(t, 1, docs)
	require.NoError(t, idx.DeleteRecord(ctx, "did:plc:one", arabica.NSIDRoaster, "r1"))
	require.NoError(t, idx.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM explore_documents WHERE uri=?`, uri).Scan(&docs))
	assert.Equal(t, 0, docs)

	upsertExploreRecord(t, idx, "did:plc:one", arabica.NSIDRoaster, "r2", map[string]any{"$type": arabica.NSIDRoaster, "name": "Rebuild", "location": "Portland"}, 2)
	require.NoError(t, idx.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM explore_documents`).Scan(&docs))
	assert.Equal(t, 1, docs)
	_, err := idx.DB().ExecContext(ctx, `DELETE FROM explore_documents`)
	require.NoError(t, err)
	require.NoError(t, idx.RebuildExploreIndex(ctx))
	require.NoError(t, idx.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM explore_documents`).Scan(&docs))
	assert.Equal(t, 1, docs)
	assert.True(t, idx.ExploreHealth(ctx).Ready)
}

func TestExploreDeleteAllByDIDCascadesRecords(t *testing.T) {
	idx := newExploreTestIndex(t)
	ctx := context.Background()
	upsertExploreRecord(t, idx, "did:plc:one", arabica.NSIDBrewer, "br1", map[string]any{"$type": arabica.NSIDBrewer, "name": "Origami", "brewerType": "pourover"}, 1)
	upsertExploreRecord(t, idx, "did:plc:two", arabica.NSIDBrewer, "br2", map[string]any{"$type": arabica.NSIDBrewer, "name": "Switch", "brewerType": "immersion"}, 2)
	require.NoError(t, idx.DeleteAllByDID(ctx, "did:plc:one"))
	res, err := idx.GetExplore(ctx, ExploreQuery{App: "arabica", Type: lexicons.RecordTypeBrewer})
	require.NoError(t, err)
	require.Len(t, res.Items, 1)
	assert.Equal(t, "did:plc:two", res.Items[0].Author.DID)
}

func TestExploreDeleteAllByDIDRefreshesExternalSocialStats(t *testing.T) {
	idx := newExploreTestIndex(t)
	ctx := context.Background()
	target := upsertExploreRecord(t, idx, "did:plc:target", arabica.NSIDBean, "b1", map[string]any{"$type": arabica.NSIDBean, "name": "Target"}, 1)
	require.NoError(t, idx.UpsertLike(ctx, "did:plc:deleted", "l1", target))
	require.NoError(t, idx.UpsertComment(ctx, "did:plc:deleted", "c1", target, "", "cid", "nice", time.Now()))
	require.NoError(t, idx.DeleteAllByDID(ctx, "did:plc:deleted"))

	res, err := idx.GetExplore(ctx, ExploreQuery{App: "arabica", Type: lexicons.RecordTypeBean})
	require.NoError(t, err)
	require.Len(t, res.Items, 1)
	assert.Equal(t, 0, res.Documents[target].LikeCount)
	assert.Equal(t, 0, res.Documents[target].CommentCount)
	assert.Equal(t, 0.0, res.Documents[target].PopularScore)
}
