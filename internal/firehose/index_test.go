package firehose

import (
	"context"
	"testing"
	"time"

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

	testDID := "did:plc:test123abc"

	// Initially should not be backfilled
	if idx.IsBackfilled(testDID) {
		t.Error("DID should not be backfilled initially")
	}

	// Mark as backfilled
	if err := idx.MarkBackfilled(testDID); err != nil {
		t.Fatalf("Failed to mark DID as backfilled: %v", err)
	}

	// Now should be backfilled
	if !idx.IsBackfilled(testDID) {
		t.Error("DID should be marked as backfilled")
	}

	// Different DID should not be backfilled
	otherDID := "did:plc:other456def"
	if idx.IsBackfilled(otherDID) {
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

		if err := idx.MarkBackfilled(testDID); err != nil {
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

		if !idx.IsBackfilled(testDID) {
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

	// Mark all as backfilled
	for _, did := range dids {
		if err := idx.MarkBackfilled(did); err != nil {
			t.Fatalf("Failed to mark DID %s as backfilled: %v", did, err)
		}
	}

	// Verify all are marked
	for _, did := range dids {
		if !idx.IsBackfilled(did) {
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
	err = idx.UpsertComment(actorDID, "comment1", subjectURI, "", "cid1", "Top level comment", now)
	assert.NoError(t, err)

	// Create a reply to the top-level comment
	parentURI := "at://did:plc:commenter1/social.arabica.alpha.comment/comment1"
	err = idx.UpsertComment("did:plc:commenter2", "comment2", subjectURI, parentURI, "cid2", "Reply to comment", now.Add(time.Second))
	assert.NoError(t, err)

	// Create a nested reply (depth 2)
	parentURI2 := "at://did:plc:commenter2/social.arabica.alpha.comment/comment2"
	err = idx.UpsertComment("did:plc:commenter3", "comment3", subjectURI, parentURI2, "cid3", "Nested reply", now.Add(2*time.Second))
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
	count := idx.GetCommentCount(subjectURI)
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
		err = idx.UpsertComment("did:plc:user", rkey, subjectURI, parentURI, "cid"+rkey, "Comment", now.Add(time.Duration(i)*time.Second))
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
	err = idx.UpsertComment("did:plc:user1", "topA", subjectURI, "", "cidA", "First top comment", now)
	assert.NoError(t, err)
	err = idx.UpsertComment("did:plc:user2", "topB", subjectURI, "", "cidB", "Second top comment", now.Add(5*time.Second))
	assert.NoError(t, err)

	// Reply to first top-level comment
	err = idx.UpsertComment("did:plc:user3", "replyA1", subjectURI, "at://did:plc:user1/social.arabica.alpha.comment/topA", "cidA1", "Reply to first", now.Add(2*time.Second))
	assert.NoError(t, err)

	// Reply to second top-level comment
	err = idx.UpsertComment("did:plc:user4", "replyB1", subjectURI, "at://did:plc:user2/social.arabica.alpha.comment/topB", "cidB1", "Reply to second", now.Add(6*time.Second))
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

	// Verify it exists
	uri := "at://" + did + "/" + collection + "/" + rkey
	rec, err := idx.GetRecord(uri)
	assert.NoError(t, err)
	assert.NotNil(t, rec, "record should exist after upsert")

	// Verify it appears in collection listing
	records, err := idx.ListRecordsByCollection(collection)
	assert.NoError(t, err)
	assert.Len(t, records, 1)

	// Delete the record
	err = idx.DeleteRecord(did, collection, rkey)
	assert.NoError(t, err)

	// Verify it no longer exists via GetRecord
	rec, err = idx.GetRecord(uri)
	assert.NoError(t, err)
	assert.Nil(t, rec, "record should not exist after delete")

	// Verify it no longer appears in collection listing
	records, err = idx.ListRecordsByCollection(collection)
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

	// Delete only the first record
	err = idx.DeleteRecord(did, collection, "bean1")
	assert.NoError(t, err)

	// Second record should still exist
	uri2 := "at://" + did + "/" + collection + "/bean2"
	rec, err := idx.GetRecord(uri2)
	assert.NoError(t, err)
	assert.NotNil(t, rec, "second record should still exist after deleting first")

	// Only one record should remain
	assert.Equal(t, 1, idx.RecordCount())

	records, err := idx.ListRecordsByCollection(collection)
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
	err = idx.DeleteRecord("did:plc:nobody", "social.arabica.alpha.bean", "nonexistent")
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

	// Delete each record and verify it's gone
	for _, tt := range types {
		err := idx.DeleteRecord(did, tt.collection, tt.rkey)
		assert.NoError(t, err, "failed to delete %s/%s", tt.collection, tt.rkey)

		uri := "at://" + did + "/" + tt.collection + "/" + tt.rkey
		rec, err := idx.GetRecord(uri)
		assert.NoError(t, err)
		assert.Nil(t, rec, "%s should not exist after delete", tt.collection)
	}

	assert.Equal(t, 0, idx.RecordCount(), "all records should be deleted")
}
