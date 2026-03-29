package firehose

import (
	"context"
	"fmt"
	"testing"
	"time"

	"arabica/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestBackfillTracking(t *testing.T) {
	// Create temporary index
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}
	defer idx.Close()

	ctx := context.Background()
	testDID := "did:plc:test123abc"

	// Initially should not be backfilled
	if idx.IsBackfilled(ctx, testDID) {
		t.Error("DID should not be backfilled initially")
	}

	// Mark as backfilled
	if err := idx.MarkBackfilled(ctx, testDID); err != nil {
		t.Fatalf("Failed to mark DID as backfilled: %v", err)
	}

	// Now should be backfilled
	if !idx.IsBackfilled(ctx, testDID) {
		t.Error("DID should be marked as backfilled")
	}

	// Different DID should not be backfilled
	otherDID := "did:plc:other456def"
	if idx.IsBackfilled(ctx, otherDID) {
		t.Error("Other DID should not be backfilled")
	}
}

func TestBackfillTracking_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"
	testDID := "did:plc:persist123"

	// Create index and mark DID as backfilled
	{
		idx, err := NewFeedIndex(dbPath, 1*time.Hour)
		if err != nil {
			t.Fatalf("Failed to create index: %v", err)
		}

		if err := idx.MarkBackfilled(context.Background(), testDID); err != nil {
			t.Fatalf("Failed to mark DID as backfilled: %v", err)
		}

		idx.Close()
	}

	// Reopen index and verify DID is still marked as backfilled
	{
		idx, err := NewFeedIndex(dbPath, 1*time.Hour)
		if err != nil {
			t.Fatalf("Failed to reopen index: %v", err)
		}
		defer idx.Close()

		if !idx.IsBackfilled(context.Background(), testDID) {
			t.Error("DID should still be marked as backfilled after reopening")
		}
	}
}

func TestBackfillTracking_MultipleDIDs(t *testing.T) {
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}
	defer idx.Close()

	dids := []string{
		"did:plc:user1",
		"did:plc:user2",
		"did:web:example.com",
		"did:plc:user3",
	}

	ctx := context.Background()

	// Mark all as backfilled
	for _, did := range dids {
		if err := idx.MarkBackfilled(ctx, did); err != nil {
			t.Fatalf("Failed to mark DID %s as backfilled: %v", did, err)
		}
	}

	// Verify all are marked
	for _, did := range dids {
		if !idx.IsBackfilled(ctx, did) {
			t.Errorf("DID %s should be marked as backfilled", did)
		}
	}
}

func TestCommentThreading(t *testing.T) {
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	assert.NoError(t, err)
	defer idx.Close()

	ctx := context.Background()
	subjectURI := "at://did:plc:user1/social.arabica.alpha.brew/abc123"
	actorDID := "did:plc:commenter1"

	// Create a top-level comment
	now := time.Now()
	err = idx.UpsertComment(ctx, actorDID, "comment1", subjectURI, "", "cid1", "Top level comment", now)
	assert.NoError(t, err)

	// Create a reply to the top-level comment
	parentURI := "at://did:plc:commenter1/social.arabica.alpha.comment/comment1"
	err = idx.UpsertComment(ctx, "did:plc:commenter2", "comment2", subjectURI, parentURI, "cid2", "Reply to comment", now.Add(time.Second))
	assert.NoError(t, err)

	// Create a nested reply (depth 2)
	parentURI2 := "at://did:plc:commenter2/social.arabica.alpha.comment/comment2"
	err = idx.UpsertComment(ctx, "did:plc:commenter3", "comment3", subjectURI, parentURI2, "cid3", "Nested reply", now.Add(2*time.Second))
	assert.NoError(t, err)

	// Get threaded comments
	comments := idx.GetThreadedCommentsForSubject(ctx, subjectURI, 100, "")
	assert.Len(t, comments, 3)

	// Verify ordering and depth
	// Order should be: top-level (depth 0) -> reply (depth 1) -> nested reply (depth 2)
	assert.Equal(t, "comment1", comments[0].RKey)
	assert.Equal(t, 0, comments[0].Depth)

	assert.Equal(t, "comment2", comments[1].RKey)
	assert.Equal(t, 1, comments[1].Depth)

	assert.Equal(t, "comment3", comments[2].RKey)
	assert.Equal(t, 2, comments[2].Depth)

	// Verify comment count
	count := idx.GetCommentCount(ctx, subjectURI)
	assert.Equal(t, 3, count)
}

func TestCommentThreading_DepthCap(t *testing.T) {
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	assert.NoError(t, err)
	defer idx.Close()

	ctx := context.Background()
	subjectURI := "at://did:plc:user1/social.arabica.alpha.brew/abc123"

	// Create a chain of comments: depth 0 -> 1 -> 2 -> 3 -> 4
	now := time.Now()
	parentURI := ""
	for i := 0; i < 5; i++ {
		rkey := "comment" + string(rune('A'+i))
		err = idx.UpsertComment(ctx, "did:plc:user", rkey, subjectURI, parentURI, "cid"+rkey, "Comment", now.Add(time.Duration(i)*time.Second))
		assert.NoError(t, err)
		parentURI = "at://did:plc:user/social.arabica.alpha.comment/" + rkey
	}

	// Get threaded comments
	comments := idx.GetThreadedCommentsForSubject(ctx, subjectURI, 100, "")
	assert.Len(t, comments, 5)

	// Verify depth is capped at 2
	assert.Equal(t, 0, comments[0].Depth) // commentA
	assert.Equal(t, 1, comments[1].Depth) // commentB
	assert.Equal(t, 2, comments[2].Depth) // commentC (capped)
	assert.Equal(t, 2, comments[3].Depth) // commentD (capped at 2)
	assert.Equal(t, 2, comments[4].Depth) // commentE (capped at 2)
}

func TestCommentThreading_MultipleTopLevel(t *testing.T) {
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	assert.NoError(t, err)
	defer idx.Close()

	ctx := context.Background()
	subjectURI := "at://did:plc:user1/social.arabica.alpha.brew/abc123"

	now := time.Now()

	// Create two top-level comments
	err = idx.UpsertComment(ctx, "did:plc:user1", "topA", subjectURI, "", "cidA", "First top comment", now)
	assert.NoError(t, err)
	err = idx.UpsertComment(ctx, "did:plc:user2", "topB", subjectURI, "", "cidB", "Second top comment", now.Add(5*time.Second))
	assert.NoError(t, err)

	// Reply to first top-level comment
	err = idx.UpsertComment(ctx, "did:plc:user3", "replyA1", subjectURI, "at://did:plc:user1/social.arabica.alpha.comment/topA", "cidA1", "Reply to first", now.Add(2*time.Second))
	assert.NoError(t, err)

	// Reply to second top-level comment
	err = idx.UpsertComment(ctx, "did:plc:user4", "replyB1", subjectURI, "at://did:plc:user2/social.arabica.alpha.comment/topB", "cidB1", "Reply to second", now.Add(6*time.Second))
	assert.NoError(t, err)

	// Get threaded comments
	comments := idx.GetThreadedCommentsForSubject(ctx, subjectURI, 100, "")
	assert.Len(t, comments, 4)

	// Order should be: topA (oldest) -> replyA1 -> topB -> replyB1
	assert.Equal(t, "topA", comments[0].RKey)
	assert.Equal(t, 0, comments[0].Depth)

	assert.Equal(t, "replyA1", comments[1].RKey)
	assert.Equal(t, 1, comments[1].Depth)

	assert.Equal(t, "topB", comments[2].RKey)
	assert.Equal(t, 0, comments[2].Depth)

	assert.Equal(t, "replyB1", comments[3].RKey)
	assert.Equal(t, 1, comments[3].Depth)
}

func TestAvgBrewRatingByBeanURI(t *testing.T) {
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	assert.NoError(t, err)
	defer idx.Close()

	ctx := context.Background()
	did := "did:plc:user1"
	now := time.Now().Unix()
	beanURI := "at://did:plc:user1/social.arabica.alpha.bean/bean1"

	// Insert brews with ratings referencing the same bean
	for i, rating := range []int{7, 8, 9} {
		record := []byte(`{"$type":"social.arabica.alpha.brew","beanRef":"` + beanURI + `","rating":` + fmt.Sprintf("%d", rating) + `,"createdAt":"2025-01-0` + fmt.Sprintf("%d", i+1) + `T00:00:00Z"}`)
		err := idx.UpsertRecord(ctx, did, "social.arabica.alpha.brew", fmt.Sprintf("brew%d", i), "cid", record, now)
		assert.NoError(t, err)
	}

	// Per-user average
	stats := idx.AvgBrewRatingByBeanURI(ctx, did)
	assert.Len(t, stats, 1)
	assert.Equal(t, 3, stats[beanURI].Count)
	assert.InDelta(t, 8.0, stats[beanURI].Average, 0.01)

	// Cross-user average (empty DID)
	stats = idx.AvgBrewRatingByBeanURI(ctx, "")
	assert.Len(t, stats, 1)
	assert.Equal(t, 3, stats[beanURI].Count)
	assert.InDelta(t, 8.0, stats[beanURI].Average, 0.01)
}

func TestAvgBrewRatingByBeanURI_MultipleBeansAndUsers(t *testing.T) {
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	assert.NoError(t, err)
	defer idx.Close()

	ctx := context.Background()
	now := time.Now().Unix()
	bean1 := "at://did:plc:user1/social.arabica.alpha.bean/bean1"
	bean2 := "at://did:plc:user1/social.arabica.alpha.bean/bean2"

	// User1 rates bean1: 6, 8
	for i, rating := range []int{6, 8} {
		record := []byte(fmt.Sprintf(`{"$type":"social.arabica.alpha.brew","beanRef":"%s","rating":%d,"createdAt":"2025-01-01T00:00:00Z"}`, bean1, rating))
		assert.NoError(t, idx.UpsertRecord(ctx, "did:plc:user1", "social.arabica.alpha.brew", fmt.Sprintf("u1b1_%d", i), "cid", record, now))
	}

	// User2 rates bean1: 10
	record := []byte(fmt.Sprintf(`{"$type":"social.arabica.alpha.brew","beanRef":"%s","rating":10,"createdAt":"2025-01-01T00:00:00Z"}`, bean1))
	assert.NoError(t, idx.UpsertRecord(ctx, "did:plc:user2", "social.arabica.alpha.brew", "u2b1_0", "cid", record, now))

	// User1 rates bean2: 4
	record = []byte(fmt.Sprintf(`{"$type":"social.arabica.alpha.brew","beanRef":"%s","rating":4,"createdAt":"2025-01-01T00:00:00Z"}`, bean2))
	assert.NoError(t, idx.UpsertRecord(ctx, "did:plc:user1", "social.arabica.alpha.brew", "u1b2_0", "cid", record, now))

	// Per-user1: bean1 avg=7, bean2 avg=4
	stats := idx.AvgBrewRatingByBeanURI(ctx, "did:plc:user1")
	assert.Len(t, stats, 2)
	assert.InDelta(t, 7.0, stats[bean1].Average, 0.01)
	assert.Equal(t, 2, stats[bean1].Count)
	assert.InDelta(t, 4.0, stats[bean2].Average, 0.01)
	assert.Equal(t, 1, stats[bean2].Count)

	// Cross-user: bean1 avg=(6+8+10)/3=8, bean2 avg=4
	stats = idx.AvgBrewRatingByBeanURI(ctx, "")
	assert.Len(t, stats, 2)
	assert.InDelta(t, 8.0, stats[bean1].Average, 0.01)
	assert.Equal(t, 3, stats[bean1].Count)
	assert.InDelta(t, 4.0, stats[bean2].Average, 0.01)
}

func TestAvgBrewRatingByBeanURI_SkipsBrewsWithoutRating(t *testing.T) {
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	assert.NoError(t, err)
	defer idx.Close()

	ctx := context.Background()
	now := time.Now().Unix()
	beanURI := "at://did:plc:user1/social.arabica.alpha.bean/bean1"

	// Brew with rating
	record := []byte(fmt.Sprintf(`{"$type":"social.arabica.alpha.brew","beanRef":"%s","rating":7,"createdAt":"2025-01-01T00:00:00Z"}`, beanURI))
	assert.NoError(t, idx.UpsertRecord(ctx, "did:plc:user1", "social.arabica.alpha.brew", "brew1", "cid", record, now))

	// Brew without rating
	record = []byte(fmt.Sprintf(`{"$type":"social.arabica.alpha.brew","beanRef":"%s","createdAt":"2025-01-02T00:00:00Z"}`, beanURI))
	assert.NoError(t, idx.UpsertRecord(ctx, "did:plc:user1", "social.arabica.alpha.brew", "brew2", "cid", record, now))

	stats := idx.AvgBrewRatingByBeanURI(ctx, "")
	assert.Len(t, stats, 1)
	assert.Equal(t, 1, stats[beanURI].Count)
	assert.InDelta(t, 7.0, stats[beanURI].Average, 0.01)
}

func TestAvgBrewRatingByRoasterURI(t *testing.T) {
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	assert.NoError(t, err)
	defer idx.Close()

	ctx := context.Background()
	did := "did:plc:user1"
	now := time.Now().Unix()
	beanURI := "at://did:plc:user1/social.arabica.alpha.bean/bean1"
	roasterURI := "at://did:plc:user1/social.arabica.alpha.roaster/roaster1"

	// Insert the bean record with roaster reference
	beanRecord := []byte(fmt.Sprintf(`{"$type":"social.arabica.alpha.bean","name":"Ethiopia Yirgacheffe","roasterRef":"%s","createdAt":"2025-01-01T00:00:00Z"}`, roasterURI))
	assert.NoError(t, idx.UpsertRecord(ctx, did, "social.arabica.alpha.bean", "bean1", "cid", beanRecord, now))

	// Insert brews referencing that bean with ratings
	for i, rating := range []int{6, 8, 10} {
		record := []byte(fmt.Sprintf(`{"$type":"social.arabica.alpha.brew","beanRef":"%s","rating":%d,"createdAt":"2025-01-0%dT00:00:00Z"}`, beanURI, rating, i+1))
		assert.NoError(t, idx.UpsertRecord(ctx, did, "social.arabica.alpha.brew", fmt.Sprintf("brew%d", i), "cid", record, now))
	}

	// Per-user average for roaster
	stats := idx.AvgBrewRatingByRoasterURI(ctx, did)
	assert.Len(t, stats, 1)
	assert.Equal(t, 3, stats[roasterURI].Count)
	assert.InDelta(t, 8.0, stats[roasterURI].Average, 0.01)

	// Cross-user
	stats = idx.AvgBrewRatingByRoasterURI(ctx, "")
	assert.Len(t, stats, 1)
	assert.Equal(t, 3, stats[roasterURI].Count)
	assert.InDelta(t, 8.0, stats[roasterURI].Average, 0.01)
}

func TestAvgBrewRatingByRoasterURI_MultipleBeansSameRoaster(t *testing.T) {
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	assert.NoError(t, err)
	defer idx.Close()

	ctx := context.Background()
	did := "did:plc:user1"
	now := time.Now().Unix()
	roasterURI := "at://did:plc:user1/social.arabica.alpha.roaster/roaster1"
	bean1URI := "at://did:plc:user1/social.arabica.alpha.bean/bean1"
	bean2URI := "at://did:plc:user1/social.arabica.alpha.bean/bean2"

	// Two beans from the same roaster
	for _, b := range []struct{ uri, rkey string }{{bean1URI, "bean1"}, {bean2URI, "bean2"}} {
		record := []byte(fmt.Sprintf(`{"$type":"social.arabica.alpha.bean","name":"Bean","roasterRef":"%s","createdAt":"2025-01-01T00:00:00Z"}`, roasterURI))
		assert.NoError(t, idx.UpsertRecord(ctx, did, "social.arabica.alpha.bean", b.rkey, "cid", record, now))
	}

	// Brews: bean1 rated 6, bean2 rated 10
	record := []byte(fmt.Sprintf(`{"$type":"social.arabica.alpha.brew","beanRef":"%s","rating":6,"createdAt":"2025-01-01T00:00:00Z"}`, bean1URI))
	assert.NoError(t, idx.UpsertRecord(ctx, did, "social.arabica.alpha.brew", "brew1", "cid", record, now))
	record = []byte(fmt.Sprintf(`{"$type":"social.arabica.alpha.brew","beanRef":"%s","rating":10,"createdAt":"2025-01-01T00:00:00Z"}`, bean2URI))
	assert.NoError(t, idx.UpsertRecord(ctx, did, "social.arabica.alpha.brew", "brew2", "cid", record, now))

	stats := idx.AvgBrewRatingByRoasterURI(ctx, "")
	assert.Len(t, stats, 1)
	assert.Equal(t, 2, stats[roasterURI].Count)
	assert.InDelta(t, 8.0, stats[roasterURI].Average, 0.01)
}

func TestAvgBrewRatingByBeanURI_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	assert.NoError(t, err)
	defer idx.Close()

	stats := idx.AvgBrewRatingByBeanURI(context.Background(), "")
	assert.Empty(t, stats)
}

func TestProfileStatsVisibility_Defaults(t *testing.T) {
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	assert.NoError(t, err)
	defer idx.Close()

	// No settings stored — should return all public
	vis := idx.GetProfileStatsVisibility(context.Background(), "did:plc:nobody")
	assert.Equal(t, models.VisibilityPublic, vis.BeanAvgRating)
	assert.Equal(t, models.VisibilityPublic, vis.RoasterAvgRating)
}

func TestProfileStatsVisibility_SetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	assert.NoError(t, err)
	defer idx.Close()

	ctx := context.Background()
	did := "did:plc:user1"

	// Set bean to private, roaster stays public
	err = idx.SetProfileStatsVisibility(ctx, did, models.ProfileStatsVisibility{
		BeanAvgRating:    models.VisibilityPrivate,
		RoasterAvgRating: models.VisibilityPublic,
	})
	assert.NoError(t, err)

	vis := idx.GetProfileStatsVisibility(ctx, did)
	assert.Equal(t, models.VisibilityPrivate, vis.BeanAvgRating)
	assert.Equal(t, models.VisibilityPublic, vis.RoasterAvgRating)

	// Update to both private
	err = idx.SetProfileStatsVisibility(ctx, did, models.ProfileStatsVisibility{
		BeanAvgRating:    models.VisibilityPrivate,
		RoasterAvgRating: models.VisibilityPrivate,
	})
	assert.NoError(t, err)

	vis = idx.GetProfileStatsVisibility(ctx, did)
	assert.Equal(t, models.VisibilityPrivate, vis.BeanAvgRating)
	assert.Equal(t, models.VisibilityPrivate, vis.RoasterAvgRating)
}

func TestProfileStatsVisibility_IsolatedPerUser(t *testing.T) {
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	assert.NoError(t, err)
	defer idx.Close()

	ctx := context.Background()

	// User1 sets private, User2 has defaults
	err = idx.SetProfileStatsVisibility(ctx, "did:plc:user1", models.ProfileStatsVisibility{
		BeanAvgRating:    models.VisibilityPrivate,
		RoasterAvgRating: models.VisibilityPrivate,
	})
	assert.NoError(t, err)

	vis1 := idx.GetProfileStatsVisibility(ctx, "did:plc:user1")
	vis2 := idx.GetProfileStatsVisibility(ctx, "did:plc:user2")

	assert.Equal(t, models.VisibilityPrivate, vis1.BeanAvgRating)
	assert.Equal(t, models.VisibilityPublic, vis2.BeanAvgRating)
}

func TestDeleteRecord(t *testing.T) {
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	assert.NoError(t, err)
	defer idx.Close()

	did := "did:plc:testuser"
	collection := "social.arabica.alpha.bean"
	rkey := "bean123"

	// Index a record
	record := []byte(`{"$type":"social.arabica.alpha.bean","name":"Test Bean","origin":"Ethiopia","createdAt":"2025-01-01T00:00:00Z"}`)
	err = idx.UpsertRecord(context.Background(), did, collection, rkey, "cid123", record, time.Now().Unix())
	assert.NoError(t, err)

	ctx := context.Background()

	// Verify it exists
	uri := "at://" + did + "/" + collection + "/" + rkey
	rec, err := idx.GetRecord(ctx, uri)
	assert.NoError(t, err)
	assert.NotNil(t, rec, "record should exist after upsert")

	// Verify it appears in collection listing
	records, err := idx.ListRecordsByCollection(ctx, collection)
	assert.NoError(t, err)
	assert.Len(t, records, 1)

	// Delete the record
	err = idx.DeleteRecord(ctx, did, collection, rkey)
	assert.NoError(t, err)

	// Verify it no longer exists via GetRecord
	rec, err = idx.GetRecord(ctx, uri)
	assert.NoError(t, err)
	assert.Nil(t, rec, "record should not exist after delete")

	// Verify it no longer appears in collection listing
	records, err = idx.ListRecordsByCollection(ctx, collection)
	assert.NoError(t, err)
	assert.Len(t, records, 0, "deleted record should not appear in collection listing")

	// Verify record count is zero
	assert.Equal(t, 0, idx.RecordCount(), "record count should be zero after delete")
}

func TestDeleteRecord_DoesNotAffectOtherRecords(t *testing.T) {
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	assert.NoError(t, err)
	defer idx.Close()

	did := "did:plc:testuser"
	collection := "social.arabica.alpha.bean"

	// Index two records
	record1 := []byte(`{"$type":"social.arabica.alpha.bean","name":"Bean One","createdAt":"2025-01-01T00:00:00Z"}`)
	record2 := []byte(`{"$type":"social.arabica.alpha.bean","name":"Bean Two","createdAt":"2025-01-02T00:00:00Z"}`)
	err = idx.UpsertRecord(context.Background(), did, collection, "bean1", "cid1", record1, time.Now().Unix())
	assert.NoError(t, err)
	err = idx.UpsertRecord(context.Background(), did, collection, "bean2", "cid2", record2, time.Now().Unix())
	assert.NoError(t, err)

	assert.Equal(t, 2, idx.RecordCount())

	ctx := context.Background()

	// Delete only the first record
	err = idx.DeleteRecord(ctx, did, collection, "bean1")
	assert.NoError(t, err)

	// Second record should still exist
	uri2 := "at://" + did + "/" + collection + "/bean2"
	rec, err := idx.GetRecord(ctx, uri2)
	assert.NoError(t, err)
	assert.NotNil(t, rec, "second record should still exist after deleting first")

	// Only one record should remain
	assert.Equal(t, 1, idx.RecordCount())

	records, err := idx.ListRecordsByCollection(ctx, collection)
	assert.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "bean2", records[0].RKey)
}

func TestDeleteRecord_NonexistentIsNoOp(t *testing.T) {
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	assert.NoError(t, err)
	defer idx.Close()

	// Deleting a record that doesn't exist should not error
	err = idx.DeleteRecord(context.Background(), "did:plc:nobody", "social.arabica.alpha.bean", "nonexistent")
	assert.NoError(t, err)
}

func TestDeleteRecord_AllEntityTypes(t *testing.T) {
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	assert.NoError(t, err)
	defer idx.Close()

	did := "did:plc:testuser"
	now := time.Now().Unix()

	// Index one record of each entity type
	types := []struct {
		collection string
		rkey       string
		record     string
	}{
		{"social.arabica.alpha.bean", "bean1", `{"$type":"social.arabica.alpha.bean","name":"Test Bean","createdAt":"2025-01-01T00:00:00Z"}`},
		{"social.arabica.alpha.roaster", "roaster1", `{"$type":"social.arabica.alpha.roaster","name":"Test Roaster","createdAt":"2025-01-01T00:00:00Z"}`},
		{"social.arabica.alpha.grinder", "grinder1", `{"$type":"social.arabica.alpha.grinder","name":"Test Grinder","createdAt":"2025-01-01T00:00:00Z"}`},
		{"social.arabica.alpha.brewer", "brewer1", `{"$type":"social.arabica.alpha.brewer","name":"Test Brewer","createdAt":"2025-01-01T00:00:00Z"}`},
		{"social.arabica.alpha.brew", "brew1", `{"$type":"social.arabica.alpha.brew","beanRef":"at://did:plc:testuser/social.arabica.alpha.bean/bean1","createdAt":"2025-01-01T00:00:00Z"}`},
		{"social.arabica.alpha.recipe", "recipe1", `{"$type":"social.arabica.alpha.recipe","name":"Test Recipe","createdAt":"2025-01-01T00:00:00Z"}`},
	}

	for _, tt := range types {
		err := idx.UpsertRecord(context.Background(), did, tt.collection, tt.rkey, "cid-"+tt.rkey, []byte(tt.record), now)
		assert.NoError(t, err, "failed to upsert %s", tt.collection)
	}

	assert.Equal(t, 6, idx.RecordCount())

	ctx := context.Background()

	// Delete each record and verify it's gone
	for _, tt := range types {
		err := idx.DeleteRecord(ctx, did, tt.collection, tt.rkey)
		assert.NoError(t, err, "failed to delete %s/%s", tt.collection, tt.rkey)

		uri := "at://" + did + "/" + tt.collection + "/" + tt.rkey
		rec, err := idx.GetRecord(ctx, uri)
		assert.NoError(t, err)
		assert.Nil(t, rec, "%s should not exist after delete", tt.collection)
	}

	assert.Equal(t, 0, idx.RecordCount(), "all records should be deleted")
}
