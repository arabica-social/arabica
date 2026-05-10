package integration

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"tangled.org/arabica.social/arabica/internal/entities/arabica"
	"tangled.org/arabica.social/arabica/internal/firehose"
	"tangled.org/pdewey.com/atp"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// subjectRefFor looks up the AT-URI and CID for a record from the witness
// cache. Likes and comments need a (subject_uri, subject_cid) pair, but the
// entity create handlers don't return CID directly — it's persisted into the
// witness cache by the write-through, which is what view handlers also use to
// build social-feature props.
func subjectRefFor(t *testing.T, h *Harness, acct TestAccount, collection, rkey string) (uri, cid string) {
	t.Helper()
	uri = atp.BuildATURI(acct.DID, collection, rkey)
	wr, err := h.FeedIndex.GetWitnessRecord(context.Background(), uri)
	require.NoError(t, err)
	require.NotNil(t, wr, "witness record missing for %s", uri)
	require.NotEmpty(t, wr.CID, "witness record has empty CID")
	return uri, wr.CID
}

// TestHTTP_LikeToggleFlow exercises the like toggle endpoint end-to-end:
// like → verify count → unlike → verify count. The handler returns a rendered
// LikeButton fragment, but the source of truth is the firehose index, so we
// assert against GetLikeCount rather than parse HTML.
func TestHTTP_LikeToggleFlow(t *testing.T) {
	h := StartHarness(t, nil)

	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Likeable Roaster")), "roaster")
	subjectURI, subjectCID := subjectRefFor(t, h, h.PrimaryAccount, arabica.NSIDRoaster, rkey)

	// Initial state: no likes.
	assert.Equal(t, 0, h.FeedIndex.GetLikeCount(context.Background(), subjectURI))

	// Like.
	likeResp := h.PostForm("/api/likes/toggle", form(
		"subject_uri", subjectURI,
		"subject_cid", subjectCID,
	))
	likeBody := ReadBody(t, likeResp)
	require.Equal(t, 200, likeResp.StatusCode, statusErr(likeResp, likeBody))
	assert.Equal(t, 1, h.FeedIndex.GetLikeCount(context.Background(), subjectURI),
		"like count should be 1 after liking")

	// Toggle off.
	unlikeResp := h.PostForm("/api/likes/toggle", form(
		"subject_uri", subjectURI,
		"subject_cid", subjectCID,
	))
	unlikeBody := ReadBody(t, unlikeResp)
	require.Equal(t, 200, unlikeResp.StatusCode, statusErr(unlikeResp, unlikeBody))
	assert.Equal(t, 0, h.FeedIndex.GetLikeCount(context.Background(), subjectURI),
		"like count should be 0 after unliking")
}

// TestHTTP_LikeCrossUser verifies that when Bob likes Alice's record, the
// count reflects both users' likes independently. Each like is stored in the
// liker's PDS but indexed against the subject URI, so this exercises the
// "many likers, one subject" path.
func TestHTTP_LikeCrossUser(t *testing.T) {
	h := StartHarness(t, nil)

	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Popular Roaster")), "roaster")
	subjectURI, subjectCID := subjectRefFor(t, h, h.PrimaryAccount, arabica.NSIDRoaster, rkey)

	// Alice likes her own record.
	resp := h.PostForm("/api/likes/toggle", form("subject_uri", subjectURI, "subject_cid", subjectCID))
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, ReadBody(t, resp)))
	require.Equal(t, 1, h.FeedIndex.GetLikeCount(context.Background(), subjectURI))

	// Bob signs in and likes the same record.
	bob := h.CreateAccount("bob@test.com", "bob.test", "hunter2")
	bobClient := h.NewClientForAccount(bob)
	func() {
		restore := withClient(h, bobClient)
		defer restore()
		resp := h.PostForm("/api/likes/toggle", form("subject_uri", subjectURI, "subject_cid", subjectCID))
		require.Equal(t, 200, resp.StatusCode, statusErr(resp, ReadBody(t, resp)))
	}()

	assert.Equal(t, 2, h.FeedIndex.GetLikeCount(context.Background(), subjectURI),
		"count should reflect likes from both Alice and Bob")
}

// TestHTTP_LikeValidation covers the input rejection paths in the like
// toggle handler: missing subject_uri or subject_cid must return 400.
func TestHTTP_LikeValidation(t *testing.T) {
	h := StartHarness(t, nil)

	cases := []struct {
		name string
		form url.Values
	}{
		{"missing_uri", form("subject_cid", "bafyfake")},
		{"missing_cid", form("subject_uri", "at://did:plc:test/social.arabica.alpha.roaster/abc")},
		{"both_missing", url.Values{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := h.PostForm("/api/likes/toggle", tc.form)
			body := ReadBody(t, resp)
			assert.Equal(t, 400, resp.StatusCode, statusErr(resp, body))
		})
	}
}

// TestHTTP_CommentCreateAndList covers the basic comment lifecycle on a
// brew (the most common comment subject): post a comment, list it via the
// HTMX-only GET endpoint, then delete it.
func TestHTTP_CommentCreateAndList(t *testing.T) {
	h := StartHarness(t, nil)

	// Set up something to comment on.
	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "C Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form(
		"name", "C Bean", "roaster_rkey", roasterRKey, "roast_level", "Medium",
	)), "bean")

	createBrew := url.Values{}
	createBrew.Set("bean_rkey", beanRKey)
	createBrew.Set("water_amount", "300")
	createBrew.Set("coffee_amount", "18")
	brewResp := h.PostForm("/brews", createBrew)
	require.Equal(t, 200, brewResp.StatusCode, statusErr(brewResp, ReadBody(t, brewResp)))

	data := fetchData(t, h)
	require.Len(t, data.Brews, 1)
	brewRKey := data.Brews[0].RKey
	subjectURI, subjectCID := subjectRefFor(t, h, h.PrimaryAccount, arabica.NSIDBrew, brewRKey)

	// Post a comment.
	commentResp := h.PostForm("/api/comments", form(
		"subject_uri", subjectURI,
		"subject_cid", subjectCID,
		"text", "great extraction",
	))
	commentBody := ReadBody(t, commentResp)
	require.Equal(t, 200, commentResp.StatusCode, statusErr(commentResp, commentBody))
	assert.Contains(t, commentBody, "great extraction",
		"create response should re-render the comment section including the new comment")

	// Verify via the threaded-comments source of truth.
	indexed := h.FeedIndex.GetThreadedCommentsForSubject(context.Background(), subjectURI, 100, h.PrimaryAccount.DID)
	require.Len(t, indexed, 1)
	assert.Equal(t, "great extraction", indexed[0].Text)
	assert.Equal(t, h.PrimaryAccount.DID, indexed[0].ActorDID)
	commentRKey := indexed[0].RKey
	require.NotEmpty(t, commentRKey)

	// Verify the HTMX list endpoint also returns it.
	listResp := h.GetHTMX("/api/comments?subject_uri=" + url.QueryEscape(subjectURI) + "&subject_cid=" + url.QueryEscape(subjectCID))
	listBody := ReadBody(t, listResp)
	require.Equal(t, 200, listResp.StatusCode, statusErr(listResp, listBody))
	assert.Contains(t, listBody, "great extraction")

	// Delete and verify gone from the index.
	delResp := h.Delete("/api/comments/" + commentRKey)
	require.Equal(t, 200, delResp.StatusCode, statusErr(delResp, ReadBody(t, delResp)))

	indexed = h.FeedIndex.GetThreadedCommentsForSubject(context.Background(), subjectURI, 100, h.PrimaryAccount.DID)
	assert.Empty(t, indexed, "comment should be gone after delete")
}

// TestHTTP_CommentReplyThreading exercises the parent_uri/parent_cid
// strongRef path used for reply threading. After posting a top-level
// comment and a reply, the threaded list should return both with the reply
// nested under its parent.
func TestHTTP_CommentReplyThreading(t *testing.T) {
	h := StartHarness(t, nil)

	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Threaded Roaster")), "roaster")
	subjectURI, subjectCID := subjectRefFor(t, h, h.PrimaryAccount, arabica.NSIDRoaster, rkey)

	// Top-level comment.
	parentResp := h.PostForm("/api/comments", form(
		"subject_uri", subjectURI,
		"subject_cid", subjectCID,
		"text", "parent comment",
	))
	require.Equal(t, 200, parentResp.StatusCode, statusErr(parentResp, ReadBody(t, parentResp)))

	indexed := h.FeedIndex.GetThreadedCommentsForSubject(context.Background(), subjectURI, 100, h.PrimaryAccount.DID)
	require.Len(t, indexed, 1)
	parent := indexed[0]
	require.NotEmpty(t, parent.CID, "parent comment should have a CID we can strongRef")

	parentURI := atp.BuildATURI(h.PrimaryAccount.DID, arabica.NSIDComment, parent.RKey)

	// Reply referencing the parent.
	replyResp := h.PostForm("/api/comments", form(
		"subject_uri", subjectURI,
		"subject_cid", subjectCID,
		"text", "child reply",
		"parent_uri", parentURI,
		"parent_cid", parent.CID,
	))
	require.Equal(t, 200, replyResp.StatusCode, statusErr(replyResp, ReadBody(t, replyResp)))

	// Both comments should appear, with the reply at depth 1 and naming the parent.
	indexed = h.FeedIndex.GetThreadedCommentsForSubject(context.Background(), subjectURI, 100, h.PrimaryAccount.DID)
	require.Len(t, indexed, 2)

	var top, reply *firehose.IndexedComment
	for i := range indexed {
		switch indexed[i].Text {
		case "parent comment":
			top = &indexed[i]
		case "child reply":
			reply = &indexed[i]
		}
	}
	require.NotNil(t, top, "parent comment missing from threaded list")
	require.NotNil(t, reply, "child reply missing from threaded list")
	assert.Equal(t, 0, top.Depth, "parent should be at depth 0")
	assert.Equal(t, 1, reply.Depth, "reply should be at depth 1")
	assert.Equal(t, parentURI, reply.ParentURI,
		"reply should reference the parent URI")
}

// TestHTTP_CommentValidation covers comment input rejection: missing subject,
// missing text, oversized text, and orphan parent_uri without parent_cid.
func TestHTTP_CommentValidation(t *testing.T) {
	h := StartHarness(t, nil)

	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Val Roaster")), "roaster")
	subjectURI, subjectCID := subjectRefFor(t, h, h.PrimaryAccount, arabica.NSIDRoaster, rkey)

	cases := []struct {
		name string
		form url.Values
	}{
		{
			name: "missing_subject_uri",
			form: form("subject_cid", subjectCID, "text", "x"),
		},
		{
			name: "missing_subject_cid",
			form: form("subject_uri", subjectURI, "text", "x"),
		},
		{
			name: "empty_text",
			form: form("subject_uri", subjectURI, "subject_cid", subjectCID, "text", ""),
		},
		{
			name: "text_too_long",
			form: form("subject_uri", subjectURI, "subject_cid", subjectCID, "text", strings.Repeat("a", arabica.MaxCommentLength+1)),
		},
		{
			name: "parent_uri_without_parent_cid",
			form: form(
				"subject_uri", subjectURI,
				"subject_cid", subjectCID,
				"text", "x",
				"parent_uri", "at://did:plc:test/social.arabica.alpha.comment/abc",
			),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := h.PostForm("/api/comments", tc.form)
			body := ReadBody(t, resp)
			assert.Equal(t, 400, resp.StatusCode, statusErr(resp, body))
		})
	}

	// Sanity: nothing was indexed.
	indexed := h.FeedIndex.GetThreadedCommentsForSubject(context.Background(), subjectURI, 100, h.PrimaryAccount.DID)
	assert.Empty(t, indexed, "no comments should have been created by failing validation cases")
}

// TestHTTP_LikeAndCommentTogether is a smoke test that walks the full social
// loop: create record, like, comment, list, unlike, delete comment. Catches
// any cross-feature interaction bugs that the focused tests miss.
func TestHTTP_LikeAndCommentTogether(t *testing.T) {
	h := StartHarness(t, nil)

	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Combined Roaster")), "roaster")
	subjectURI, subjectCID := subjectRefFor(t, h, h.PrimaryAccount, arabica.NSIDRoaster, rkey)

	// Like.
	likeResp := h.PostForm("/api/likes/toggle", form("subject_uri", subjectURI, "subject_cid", subjectCID))
	require.Equal(t, 200, likeResp.StatusCode)

	// Comment.
	commentResp := h.PostForm("/api/comments", form(
		"subject_uri", subjectURI,
		"subject_cid", subjectCID,
		"text", "first impressions: solid",
	))
	require.Equal(t, 200, commentResp.StatusCode)

	// Both should be visible.
	assert.Equal(t, 1, h.FeedIndex.GetLikeCount(context.Background(), subjectURI))
	indexed := h.FeedIndex.GetThreadedCommentsForSubject(context.Background(), subjectURI, 100, h.PrimaryAccount.DID)
	require.Len(t, indexed, 1)
	commentRKey := indexed[0].RKey

	// Render the roaster view page — it should show like + comment data
	// pulled from the same feed index. This is the visible end-to-end check.
	viewResp := h.Get("/roasters/" + rkey)
	viewBody := ReadBody(t, viewResp)
	require.Equal(t, 200, viewResp.StatusCode, statusErr(viewResp, viewBody))
	assert.Contains(t, viewBody, "first impressions: solid",
		"comment text should be embedded in the view page")

	// Unlike + delete comment.
	unlikeResp := h.PostForm("/api/likes/toggle", form("subject_uri", subjectURI, "subject_cid", subjectCID))
	require.Equal(t, 200, unlikeResp.StatusCode)
	delResp := h.Delete("/api/comments/" + commentRKey)
	require.Equal(t, 200, delResp.StatusCode)

	assert.Equal(t, 0, h.FeedIndex.GetLikeCount(context.Background(), subjectURI))
	assert.Empty(t, h.FeedIndex.GetThreadedCommentsForSubject(context.Background(), subjectURI, 100, h.PrimaryAccount.DID))
}
