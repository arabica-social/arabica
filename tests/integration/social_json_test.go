//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"tangled.org/pdewey.com/atp"
)

// --- Notifications ---

// notificationsJSONResponse mirrors the GET /api/notifications JSON envelope.
type notificationsJSONResponse struct {
	Notifications []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		ActorDID   string `json:"actor_did"`
		SubjectURI string `json:"subject_uri"`
		Link       string `json:"link"`
		ActionText string `json:"action_text"`
	} `json:"notifications"`
	NextCursor string `json:"next_cursor"`
}

// TestHTTP_NotificationsJSON verifies that GET /api/notifications returns JSON.
// Requires a like/comment notification to exist, so we create one first.
func TestHTTP_NotificationsJSON(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	// Create a roaster and like it to generate a notification.
	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Notif JSON Roaster")), "roaster")
	subjectURI := atp.BuildATURI(h.PrimaryAccount.DID, "social.arabica.alpha.roaster", rkey)
	subjectCID := "bafyfake"
	h.PostForm("/api/likes/toggle", form("subject_uri", subjectURI, "subject_cid", subjectCID))

	resp := getJSON(t, h, "/api/notifications")
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var notifs notificationsJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &notifs))
	// The like notification may or may not be indexed yet, but the envelope
	// should be valid with a non-nil notifications array.
	assert.NotNil(t, notifs.Notifications)
}

// TestHTTP_NotificationsMarkReadJSON verifies POST /api/notifications/read
// returns JSON when Accept: application/json is sent.
func TestHTTP_NotificationsMarkReadJSON(t *testing.T) {
	h := StartHarness(t, nil)

	req, err := http.NewRequest("POST", h.URL("/api/notifications/read"), strings.NewReader(""))
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")
	resp, err := h.Client.Do(req)
	require.NoError(t, err)
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))

	var result map[string]bool
	require.NoError(t, json.Unmarshal([]byte(body), &result))
	assert.True(t, result["read"])
}

// --- Explore ---

// exploreJSONResponse mirrors the GET /api/explore JSON envelope.
type exploreJSONResponse struct {
	Items       []json.RawMessage `json:"items"`
	Documents   map[string]json.RawMessage `json:"documents"`
	FacetCounts []json.RawMessage `json:"facet_counts"`
	NextCursor  string            `json:"next_cursor"`
}

// TestHTTP_ExploreJSON verifies that GET /api/explore returns JSON.
func TestHTTP_ExploreJSON(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	// Create some records so explore has data.
	mustRKey(t, h.PostForm("/api/roasters", form("name", "Explore JSON Roaster")), "roaster")
	mustRKey(t, h.PostForm("/api/beans", form("name", "Explore JSON Bean", "origin", "Ethiopia")), "bean")
	beanURI := atp.BuildATURI(h.PrimaryAccount.DID, "social.arabica.alpha.bean", "")
	_ = beanURI

	resp := getJSON(t, h, "/api/explore")
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var explore exploreJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &explore))
	// Documents map should be non-nil even if empty.
	assert.NotNil(t, explore.Documents)
}

// TestHTTP_ExploreJSONUnauth verifies unauthenticated requests get 401.
func TestHTTP_ExploreJSONUnauth(t *testing.T) {
	h := StartHarness(t, nil)

	req, err := http.NewRequest("GET", h.URL("/api/explore"), nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 401, resp.StatusCode)
}

// --- Social: Likes ---

// likeToggleJSONResponse mirrors the POST /api/likes/toggle JSON envelope.
type likeToggleJSONResponse struct {
	IsLiked    bool   `json:"is_liked"`
	LikeCount  int    `json:"like_count"`
	SubjectURI string `json:"subject_uri"`
}

// TestHTTP_LikeToggleJSON verifies that POST /api/likes/toggle with Accept:
// application/json returns JSON (not an HTML LikeButton fragment).
func TestHTTP_LikeToggleJSON(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Like JSON Roaster")), "roaster")
	subjectURI := atp.BuildATURI(h.PrimaryAccount.DID, "social.arabica.alpha.roaster", rkey)
	subjectCID := "bafyfake"

	// Like via JSON path.
	formData := url.Values{}
	formData.Set("subject_uri", subjectURI)
	formData.Set("subject_cid", subjectCID)
	req, err := http.NewRequest("POST", h.URL("/api/likes/toggle"), strings.NewReader(formData.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := h.Client.Do(req)
	require.NoError(t, err)
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var result likeToggleJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &result))
	assert.True(t, result.IsLiked)
	assert.Equal(t, subjectURI, result.SubjectURI)

	// Unlike via JSON path.
	resp2, err := h.Client.Do(req.Clone(req.Context()))
	require.NoError(t, err)
	body2 := ReadBody(t, resp2)
	require.Equal(t, 200, resp2.StatusCode, statusErr(resp2, body2))

	var result2 likeToggleJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body2), &result2))
	assert.False(t, result2.IsLiked)
}

// --- Social: Comments ---

// commentListJSONResponse mirrors the GET /api/comments JSON envelope.
type commentListJSONResponse struct {
	Comments       []json.RawMessage `json:"comments"`
	SubjectURI     string            `json:"subject_uri"`
	IsAuthenticated bool             `json:"is_authenticated"`
}

// commentCreateJSONResponse mirrors the POST /api/comments JSON envelope.
type commentCreateJSONResponse struct {
	Comment  json.RawMessage   `json:"comment"`
	Comments []json.RawMessage `json:"comments"`
}

// TestHTTP_CommentListJSON verifies that GET /api/comments with Accept:
// application/json returns JSON (not an HTML CommentSection fragment).
func TestHTTP_CommentListJSON(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Comment JSON Roaster")), "roaster")
	subjectURI := atp.BuildATURI(h.PrimaryAccount.DID, "social.arabica.alpha.roaster", rkey)

	resp := getJSON(t, h, "/api/comments?subject_uri="+url.QueryEscape(subjectURI))
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var list commentListJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &list))
	assert.Equal(t, subjectURI, list.SubjectURI)
	assert.True(t, list.IsAuthenticated)
}

// TestHTTP_CommentCreateJSON verifies that POST /api/comments with Accept:
// application/json returns the created comment + updated comment list as JSON.
func TestHTTP_CommentCreateJSON(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Comment Create JSON Roaster")), "roaster")
	subjectURI := atp.BuildATURI(h.PrimaryAccount.DID, "social.arabica.alpha.roaster", rkey)
	subjectCID := "bafyfake"

	formData := url.Values{}
	formData.Set("subject_uri", subjectURI)
	formData.Set("subject_cid", subjectCID)
	formData.Set("text", "JSON comment test!")
	req, err := http.NewRequest("POST", h.URL("/api/comments"), strings.NewReader(formData.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := h.Client.Do(req)
	require.NoError(t, err)
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var result commentCreateJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &result))

	var comment map[string]any
	require.NoError(t, json.Unmarshal(result.Comment, &comment))
	assert.Equal(t, "JSON comment test!", comment["text"])
	assert.NotEmpty(t, comment["rkey"])
}

// TestHTTP_CommentDeleteJSON verifies that DELETE /api/comments/{id} with
// Accept: application/json returns {deleted: true}.
func TestHTTP_CommentDeleteJSON(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Comment Delete JSON Roaster")), "roaster")
	subjectURI := atp.BuildATURI(h.PrimaryAccount.DID, "social.arabica.alpha.roaster", rkey)
	subjectCID := "bafyfake"

	// Create a comment first via JSON path.
	formData := url.Values{}
	formData.Set("subject_uri", subjectURI)
	formData.Set("subject_cid", subjectCID)
	formData.Set("text", "To be deleted")
	createReq, err := http.NewRequest("POST", h.URL("/api/comments"), strings.NewReader(formData.Encode()))
	require.NoError(t, err)
	createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createReq.Header.Set("Accept", "application/json")
	createResp, err := h.Client.Do(createReq)
	require.NoError(t, err)
	createBody := ReadBody(t, createResp)
	require.Equal(t, 200, createResp.StatusCode, statusErr(createResp, createBody))

	var created commentCreateJSONResponse
	require.NoError(t, json.Unmarshal([]byte(createBody), &created))
	var comment map[string]any
	require.NoError(t, json.Unmarshal(created.Comment, &comment))
	commentRKey := comment["rkey"].(string)
	require.NotEmpty(t, commentRKey)

	// Delete via JSON path.
	delReq, err := http.NewRequest("DELETE", h.URL("/api/comments/"+commentRKey), nil)
	require.NoError(t, err)
	delReq.Header.Set("Accept", "application/json")
	delResp, err := h.Client.Do(delReq)
	require.NoError(t, err)
	delBody := ReadBody(t, delResp)
	require.Equal(t, 200, delResp.StatusCode, statusErr(delResp, delBody))

	var result map[string]bool
	require.NoError(t, json.Unmarshal([]byte(delBody), &result))
	assert.True(t, result["deleted"])
}

// --- Social: Report ---

// TestHTTP_ReportJSON verifies that POST /api/report with Accept:
// application/json returns {report_id, submitted: true} when moderation is
// configured. Since the test harness has no moderation store, this test just
// verifies the error shape.
func TestHTTP_ReportJSON(t *testing.T) {
	h := StartHarness(t, nil)

	formData := url.Values{}
	formData.Set("subject_uri", "at://did:plc:other/social.arabica.alpha.roaster/xyz")
	formData.Set("reason", "spam")
	req, err := http.NewRequest("POST", h.URL("/api/report"), strings.NewReader(formData.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := h.Client.Do(req)
	require.NoError(t, err)
	body := ReadBody(t, resp)

	// Without a moderation store, the handler returns an error JSON.
	var result map[string]string
	require.NoError(t, json.Unmarshal([]byte(body), &result))
	assert.Contains(t, result["error"], "not enabled")
}
