package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/entities/arabica"
	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/firehose"
	"tangled.org/arabica.social/arabica/internal/lexicons"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const firehoseWait = 5 * time.Second

// TestFirehose_RecordIndexing verifies that records created via the HTTP API
// are picked up by the testpds firehose and indexed into the FeedIndex.
func TestFirehose_RecordIndexing(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})
	ctx := context.Background()

	// Create a roaster via the HTTP API.
	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Firehose Roaster", "location", "Portland, OR")), "roaster")
	uri := atproto.BuildATURI(h.PrimaryAccount.DID, arabica.NSIDRoaster, rkey)

	// Wait for the firehose to index the record.
	h.WaitForRecord(uri, firehoseWait)

	rec, err := h.FeedIndex.GetRecord(ctx, uri)
	require.NoError(t, err)
	require.NotNil(t, rec)
	assert.Equal(t, h.PrimaryAccount.DID, rec.DID)
	assert.Equal(t, arabica.NSIDRoaster, rec.Collection)
	assert.Equal(t, rkey, rec.RKey)
	assert.Contains(t, string(rec.Record), "Firehose Roaster")
}

// TestFirehose_MultipleEntityTypes verifies that different entity types are all
// indexed through the firehose pipeline.
func TestFirehose_MultipleEntityTypes(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Multi Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form(
		"name", "Multi Bean",
		"origin", "Colombia",
		"roaster_rkey", roasterRKey,
	)), "bean")
	grinderRKey := mustRKey(t, h.PostForm("/api/grinders", form("name", "Multi Grinder")), "grinder")
	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "Multi Brewer")), "brewer")

	uris := map[string]string{
		"roaster": atproto.BuildATURI(h.PrimaryAccount.DID, arabica.NSIDRoaster, roasterRKey),
		"bean":    atproto.BuildATURI(h.PrimaryAccount.DID, arabica.NSIDBean, beanRKey),
		"grinder": atproto.BuildATURI(h.PrimaryAccount.DID, arabica.NSIDGrinder, grinderRKey),
		"brewer":  atproto.BuildATURI(h.PrimaryAccount.DID, arabica.NSIDBrewer, brewerRKey),
	}

	for label, uri := range uris {
		h.WaitForRecord(uri, firehoseWait)
		rec, err := h.FeedIndex.GetRecord(context.Background(), uri)
		require.NoError(t, err, "%s should be indexed", label)
		require.NotNil(t, rec, "%s should be indexed", label)
	}
}

// TestFirehose_DeleteRemovesFromIndex verifies that deleting a record via the
// HTTP API triggers a firehose delete event that removes it from the index.
func TestFirehose_DeleteRemovesFromIndex(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Delete Me")), "roaster")
	uri := atproto.BuildATURI(h.PrimaryAccount.DID, arabica.NSIDRoaster, rkey)

	// Wait for firehose to index it first.
	h.WaitForRecord(uri, firehoseWait)

	// Delete via HTTP API.
	resp := h.Delete("/api/roasters/" + rkey)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, ReadBody(t, resp)))

	// Wait for firehose to remove it.
	h.WaitForRecordAbsent(uri, firehoseWait)

	rec, err := h.FeedIndex.GetRecord(context.Background(), uri)
	assert.NoError(t, err)
	assert.Nil(t, rec, "record should be removed from index after delete")
}

// TestFirehose_LikeCreatesNotification verifies that a like event from the
// firehose creates a notification for the record owner.
func TestFirehose_LikeCreatesNotification(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})
	ctx := context.Background()

	// Alice creates a roaster.
	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Likeable Roaster")), "roaster")
	subjectURI, subjectCID := subjectRefFor(t, h, h.PrimaryAccount, arabica.NSIDRoaster, rkey)

	// Bob likes Alice's roaster.
	bob := h.CreateAccount("bob@test.com", "bob.test", "hunter2")
	bobClient := h.NewClientForAccount(bob)

	likeResp, err := bobClient.PostForm(h.URL("/api/likes/toggle"), url.Values{
		"subject_uri": {subjectURI},
		"subject_cid": {subjectCID},
	})
	require.NoError(t, err)
	require.Equal(t, 200, likeResp.StatusCode, statusErr(likeResp, ReadBody(t, likeResp)))

	// Poll for the like notification. The like count may appear first (via
	// HTTP handler write-through), but the notification is only created by
	// the firehose consumer, so poll until it shows up.
	deadline := time.Now().Add(firehoseWait)
	var foundLikeNotif bool
	for time.Now().Before(deadline) {
		notifs, _, err := h.FeedIndex.GetNotifications(h.PrimaryAccount.DID, 10, "")
		require.NoError(t, err)
		for _, n := range notifs {
			if n.Type == "like" && n.ActorDID == bob.DID && n.SubjectURI == subjectURI {
				foundLikeNotif = true
				break
			}
		}
		if foundLikeNotif {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	assert.True(t, foundLikeNotif, "Alice should have a like notification from Bob")
	assert.Equal(t, 1, h.FeedIndex.GetLikeCount(ctx, subjectURI), "firehose should index the like")
}

// TestFirehose_CommentCreatesNotification verifies that a comment event from
// the firehose creates a notification for the record owner.
func TestFirehose_CommentCreatesNotification(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})
	ctx := context.Background()

	// Alice creates a roaster.
	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Commentable Roaster")), "roaster")
	subjectURI, subjectCID := subjectRefFor(t, h, h.PrimaryAccount, arabica.NSIDRoaster, rkey)

	// Bob comments on Alice's roaster.
	bob := h.CreateAccount("bob@test.com", "bob.test", "hunter2")
	bobClient := h.NewClientForAccount(bob)

	commentResp, err := bobClient.PostForm(h.URL("/api/comments"), url.Values{
		"subject_uri": {subjectURI},
		"subject_cid": {subjectCID},
		"text":        {"Great roaster!"},
	})
	require.NoError(t, err)
	require.Equal(t, 200, commentResp.StatusCode, statusErr(commentResp, ReadBody(t, commentResp)))

	// Wait for the comment to be indexed.
	deadline := time.Now().Add(firehoseWait)
	for time.Now().Before(deadline) {
		if h.FeedIndex.GetCommentCount(ctx, subjectURI) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	assert.Equal(t, 1, h.FeedIndex.GetCommentCount(ctx, subjectURI), "firehose should index the comment")

	// Alice should have a notification.
	notifs, _, err := h.FeedIndex.GetNotifications(h.PrimaryAccount.DID, 10, "")
	require.NoError(t, err)

	var foundCommentNotif bool
	for _, n := range notifs {
		if n.Type == "comment" && n.ActorDID == bob.DID && n.SubjectURI == subjectURI {
			foundCommentNotif = true
			break
		}
	}
	assert.True(t, foundCommentNotif, "Alice should have a comment notification from Bob")
}

// --- Feed query end-to-end ---

// TestFirehose_FeedQueryReturnsIndexedRecords creates several entity types via
// HTTP, waits for firehose indexing, and verifies they appear in feed queries.
func TestFirehose_FeedQueryReturnsIndexedRecords(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})
	ctx := context.Background()

	// Create entities that should appear in the feed.
	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Feed Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form(
		"name", "Feed Bean",
		"roaster_rkey", roasterRKey,
	)), "bean")
	grinderRKey := mustRKey(t, h.PostForm("/api/grinders", form("name", "Feed Grinder")), "grinder")

	// Wait for all records to be indexed.
	roasterURI := atproto.BuildATURI(h.PrimaryAccount.DID, arabica.NSIDRoaster, roasterRKey)
	beanURI := atproto.BuildATURI(h.PrimaryAccount.DID, arabica.NSIDBean, beanRKey)
	grinderURI := atproto.BuildATURI(h.PrimaryAccount.DID, arabica.NSIDGrinder, grinderRKey)

	h.WaitForRecord(roasterURI, firehoseWait)
	h.WaitForRecord(beanURI, firehoseWait)
	h.WaitForRecord(grinderURI, firehoseWait)

	// Query the feed — all three should appear.
	result, err := h.FeedIndex.GetFeedWithQuery(ctx, firehose.FeedQuery{Limit: 10})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.GreaterOrEqual(t, len(result.Items), 3, "feed should contain at least 3 items")

	// Verify specific items are present by checking record types.
	types := map[lexicons.RecordType]bool{}
	for _, item := range result.Items {
		types[item.RecordType] = true
	}
	assert.True(t, types[lexicons.RecordTypeRoaster], "feed should contain a roaster")
	assert.True(t, types[lexicons.RecordTypeBean], "feed should contain a bean")
	assert.True(t, types[lexicons.RecordTypeGrinder], "feed should contain a grinder")
}

// TestFirehose_FeedQueryTypeFilter verifies that feed queries can be filtered
// to a specific record type.
func TestFirehose_FeedQueryTypeFilter(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})
	ctx := context.Background()

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Filter Roaster")), "roaster")
	mustRKey(t, h.PostForm("/api/grinders", form("name", "Filter Grinder")), "grinder")

	roasterURI := atproto.BuildATURI(h.PrimaryAccount.DID, arabica.NSIDRoaster, roasterRKey)
	h.WaitForRecord(roasterURI, firehoseWait)

	// Filter to roasters only.
	result, err := h.FeedIndex.GetFeedWithQuery(ctx, firehose.FeedQuery{
		Limit:      10,
		TypeFilter: lexicons.RecordTypeRoaster,
	})
	require.NoError(t, err)
	for _, item := range result.Items {
		assert.Equal(t, lexicons.RecordTypeRoaster, item.RecordType,
			"all items should be roasters when filtered")
	}
}

// TestFirehose_FeedQueryWithLikeAndCommentCounts verifies that feed items
// include accurate like and comment counts populated by the firehose.
func TestFirehose_FeedQueryWithLikeAndCommentCounts(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})
	ctx := context.Background()

	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Popular Roaster")), "roaster")
	subjectURI, subjectCID := subjectRefFor(t, h, h.PrimaryAccount, arabica.NSIDRoaster, rkey)

	// Bob likes the roaster.
	bob := h.CreateAccount("bob@test.com", "bob.test", "hunter2")
	bobClient := h.NewClientForAccount(bob)

	likeResp, err := bobClient.PostForm(h.URL("/api/likes/toggle"), url.Values{
		"subject_uri": {subjectURI},
		"subject_cid": {subjectCID},
	})
	require.NoError(t, err)
	require.Equal(t, 200, likeResp.StatusCode)

	// Bob comments on the roaster.
	commentResp, err := bobClient.PostForm(h.URL("/api/comments"), url.Values{
		"subject_uri": {subjectURI},
		"subject_cid": {subjectCID},
		"text":        {"Nice one!"},
	})
	require.NoError(t, err)
	require.Equal(t, 200, commentResp.StatusCode)

	// Wait for firehose to process the like and comment.
	waitFor(t, firehoseWait, func() bool {
		return h.FeedIndex.GetLikeCount(ctx, subjectURI) >= 1 &&
			h.FeedIndex.GetCommentCount(ctx, subjectURI) >= 1
	})

	// Query the feed and find the roaster item.
	result, err := h.FeedIndex.GetFeedWithQuery(ctx, firehose.FeedQuery{
		Limit:      10,
		TypeFilter: lexicons.RecordTypeRoaster,
	})
	require.NoError(t, err)

	var found *feed.FeedItem
	for _, item := range result.Items {
		if item.SubjectURI == subjectURI {
			found = item
			break
		}
	}
	require.NotNil(t, found, "roaster should appear in feed")
	assert.Equal(t, 1, found.LikeCount, "feed item should show 1 like")
	assert.Equal(t, 1, found.CommentCount, "feed item should show 1 comment")
}

// --- Record update via firehose ---

// TestFirehose_RecordUpdateReflected verifies that updating a record via HTTP
// results in the firehose re-indexing it with the new data.
func TestFirehose_RecordUpdateReflected(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})
	ctx := context.Background()

	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Original Name", "location", "NYC")), "roaster")
	uri := atproto.BuildATURI(h.PrimaryAccount.DID, arabica.NSIDRoaster, rkey)

	h.WaitForRecord(uri, firehoseWait)

	// Verify original data is indexed.
	rec, err := h.FeedIndex.GetRecord(ctx, uri)
	require.NoError(t, err)
	assert.Contains(t, string(rec.Record), "Original Name")

	// Update the roaster.
	updateResp := h.PutForm("/api/roasters/"+rkey, form("name", "Updated Name", "location", "SF"))
	require.Equal(t, 200, updateResp.StatusCode, statusErr(updateResp, ReadBody(t, updateResp)))

	// Wait for the firehose to re-index with the updated data.
	waitFor(t, firehoseWait, func() bool {
		rec, err := h.FeedIndex.GetRecord(ctx, uri)
		if err != nil || rec == nil {
			return false
		}
		return string(rec.Record) != "" &&
			assert.ObjectsAreEqual("Updated Name", extractField(rec.Record, "name"))
	})

	rec, err = h.FeedIndex.GetRecord(ctx, uri)
	require.NoError(t, err)
	assert.Contains(t, string(rec.Record), "Updated Name", "index should reflect updated name")
	assert.Contains(t, string(rec.Record), "SF", "index should reflect updated location")
	assert.NotContains(t, string(rec.Record), "Original Name", "index should not contain old name")
}

// --- Unlike / uncomment via firehose ---

// TestFirehose_UnlikeCleansUpIndex verifies that unliking a record (toggling
// the like off) results in the firehose removing the like from the index and
// cleaning up the notification.
func TestFirehose_UnlikeCleansUpIndex(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})
	ctx := context.Background()

	// Alice creates a roaster.
	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Unlike Roaster")), "roaster")
	subjectURI, subjectCID := subjectRefFor(t, h, h.PrimaryAccount, arabica.NSIDRoaster, rkey)

	// Bob likes it.
	bob := h.CreateAccount("bob@test.com", "bob.test", "hunter2")
	bobClient := h.NewClientForAccount(bob)

	likeResp, err := bobClient.PostForm(h.URL("/api/likes/toggle"), url.Values{
		"subject_uri": {subjectURI},
		"subject_cid": {subjectCID},
	})
	require.NoError(t, err)
	require.Equal(t, 200, likeResp.StatusCode)

	// Wait for the like to be fully indexed via firehose.
	waitFor(t, firehoseWait, func() bool {
		return h.FeedIndex.GetLikeCount(ctx, subjectURI) == 1
	})
	assert.True(t, h.FeedIndex.HasUserLiked(ctx, bob.DID, subjectURI))

	// Bob unlikes it (toggle again).
	unlikeResp, err := bobClient.PostForm(h.URL("/api/likes/toggle"), url.Values{
		"subject_uri": {subjectURI},
		"subject_cid": {subjectCID},
	})
	require.NoError(t, err)
	require.Equal(t, 200, unlikeResp.StatusCode)

	// Wait for the unlike to propagate through the firehose. The HTTP handler
	// does write-through DeleteLike, but the firehose consumer also processes
	// the delete event and removes the notification.
	waitFor(t, firehoseWait, func() bool {
		return h.FeedIndex.GetLikeCount(ctx, subjectURI) == 0
	})

	assert.Equal(t, 0, h.FeedIndex.GetLikeCount(ctx, subjectURI), "like count should be 0 after unlike")
	assert.False(t, h.FeedIndex.HasUserLiked(ctx, bob.DID, subjectURI), "Bob should no longer have liked")

	// The notification should have been cleaned up.
	notifs, _, err := h.FeedIndex.GetNotifications(h.PrimaryAccount.DID, 10, "")
	require.NoError(t, err)
	for _, n := range notifs {
		if n.Type == "like" && n.ActorDID == bob.DID && n.SubjectURI == subjectURI {
			t.Error("like notification should have been removed after unlike")
		}
	}
}

// TestFirehose_CommentDeleteCleansUpIndex verifies that deleting a comment
// results in the firehose removing the comment from the index and cleaning up
// the notification.
func TestFirehose_CommentDeleteCleansUpIndex(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})
	ctx := context.Background()

	// Alice creates a roaster.
	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Uncomment Roaster")), "roaster")
	subjectURI, subjectCID := subjectRefFor(t, h, h.PrimaryAccount, arabica.NSIDRoaster, rkey)

	// Bob comments on it.
	bob := h.CreateAccount("bob@test.com", "bob.test", "hunter2")
	bobClient := h.NewClientForAccount(bob)

	commentResp, err := bobClient.PostForm(h.URL("/api/comments"), url.Values{
		"subject_uri": {subjectURI},
		"subject_cid": {subjectCID},
		"text":        {"Delete me later"},
	})
	require.NoError(t, err)
	require.Equal(t, 200, commentResp.StatusCode)

	// Wait for the comment to be indexed.
	waitFor(t, firehoseWait, func() bool {
		return h.FeedIndex.GetCommentCount(ctx, subjectURI) == 1
	})

	// Find the comment rkey.
	comments := h.FeedIndex.GetThreadedCommentsForSubject(ctx, subjectURI, 10, "")
	require.Len(t, comments, 1)
	commentRKey := comments[0].RKey

	// Bob deletes the comment.
	deleteReq, err := http.NewRequest("DELETE", h.URL("/api/comments/"+commentRKey), nil)
	require.NoError(t, err)
	deleteResp, err := bobClient.Do(deleteReq)
	require.NoError(t, err)
	require.Equal(t, 200, deleteResp.StatusCode)

	// Wait for the comment to be removed from the index via firehose.
	waitFor(t, firehoseWait, func() bool {
		return h.FeedIndex.GetCommentCount(ctx, subjectURI) == 0
	})

	assert.Equal(t, 0, h.FeedIndex.GetCommentCount(ctx, subjectURI), "comment count should be 0 after delete")

	// The notification should have been cleaned up.
	notifs, _, err := h.FeedIndex.GetNotifications(h.PrimaryAccount.DID, 10, "")
	require.NoError(t, err)
	for _, n := range notifs {
		if n.Type == "comment" && n.ActorDID == bob.DID && n.SubjectURI == subjectURI {
			t.Error("comment notification should have been removed after delete")
		}
	}
}

// TestFirehose_AccountDeactivated_KeepsData verifies an end-to-end account
// status change: deactivating an account on the test PDS emits a real #account
// firehose event (status=deactivated), which our bridge forwards to the
// ProfileWatcher. Because deactivation is reversible, indexed records must NOT
// be purged.
func TestFirehose_AccountDeactivated_KeepsData(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})
	ctx := context.Background()

	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Survives Deactivation")), "roaster")
	uri := atproto.BuildATURI(h.PrimaryAccount.DID, arabica.NSIDRoaster, rkey)
	h.WaitForRecord(uri, firehoseWait)

	apiClient := h.accounts[h.PrimaryAccount.DID]
	require.NotNil(t, apiClient)

	err := apiClient.Post(ctx, "com.atproto.server.deactivateAccount", map[string]any{}, nil)
	require.NoError(t, err)

	// Give the firehose bridge time to dispatch the event.
	time.Sleep(500 * time.Millisecond)

	rec, err := h.FeedIndex.GetRecord(ctx, uri)
	assert.NoError(t, err)
	assert.NotNil(t, rec, "deactivated accounts are reversible — record should remain indexed")
}

// TestFirehose_AccountDeleted_PurgesData verifies the deletion path through
// the bridge → ProfileWatcher → DeleteAllByDID pipeline. testpds gates real
// account deletion behind a server-side token (not exposed to clients), so we
// synthesize the account event directly into the watcher to exercise the
// integration wiring without bypassing it.
func TestFirehose_AccountDeleted_PurgesData(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})
	ctx := context.Background()

	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "About to be Purged")), "roaster")
	uri := atproto.BuildATURI(h.PrimaryAccount.DID, arabica.NSIDRoaster, rkey)
	h.WaitForRecord(uri, firehoseWait)

	require.NotNil(t, h.ProfileWatcher)
	h.ProfileWatcher.ProcessEvent(firehose.JetstreamEvent{
		DID:    h.PrimaryAccount.DID,
		TimeUS: time.Now().UnixMicro(),
		Kind:   "account",
		Account: &firehose.JetstreamAccount{
			Active: false,
			DID:    h.PrimaryAccount.DID,
			Status: "deleted",
			Time:   time.Now().Format(time.RFC3339),
		},
	})

	rec, err := h.FeedIndex.GetRecord(ctx, uri)
	assert.NoError(t, err)
	assert.Nil(t, rec, "deleted account's records should be purged from index")
}

// --- helpers ---

// waitFor polls condition until it returns true or the timeout expires.
func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("timed out waiting for condition")
}

// extractField pulls a top-level string field from a JSON record.
func extractField(record []byte, key string) string {
	var m map[string]any
	if err := json.Unmarshal(record, &m); err != nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}
