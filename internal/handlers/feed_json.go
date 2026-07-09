package handlers

import (
	"time"

	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/firehose"
)

// FeedResponseJSON is the JSON envelope returned by GET /api/feed for the
// SvelteKit SPA. See docs/api/feed.md for the contract.
type FeedResponseJSON struct {
	Items           []FeedItemJSON `json:"items"`
	NextCursor      string         `json:"next_cursor"`
	IsAuthenticated bool           `json:"is_authenticated"`
	Query           FeedQueryJSON  `json:"query"`
}

// FeedQueryJSON echoes the active feed query back to the client so the SPA
// can reflect filter/sort state in its UI without re-deriving it.
type FeedQueryJSON struct {
	Type string `json:"type"`
	Sort string `json:"sort"`
}

// FeedItemJSON is the JSON-serializable view of a feed.FeedItem. The typed
// record (Brew, Bean, etc.) is carried in the Record field as-is; the
// per-entity model structs already carry json tags.
type FeedItemJSON struct {
	RecordType      string         `json:"record_type"`
	Action          string         `json:"action"`
	Record          any            `json:"record"`
	Author          *AuthorSummary `json:"author,omitempty"`
	Timestamp       time.Time      `json:"timestamp"`
	TimeAgo         string         `json:"time_ago"`
	LikeCount       int            `json:"like_count"`
	CommentCount    int            `json:"comment_count"`
	SubjectURI      string         `json:"subject_uri"`
	SubjectCID      string         `json:"subject_cid"`
	IsLikedByViewer bool           `json:"is_liked_by_viewer"`
	IsOwner         bool           `json:"is_owner"`
}

// NewFeedItemJSON converts a feed.FeedItem to its JSON-serializable form.
func NewFeedItemJSON(item *feed.FeedItem) FeedItemJSON {
	out := FeedItemJSON{
		RecordType:      string(item.RecordType),
		Action:          item.Action,
		Record:          item.Record,
		Timestamp:       item.Timestamp,
		TimeAgo:         item.TimeAgo,
		LikeCount:       item.LikeCount,
		CommentCount:    item.CommentCount,
		SubjectURI:      item.SubjectURI,
		SubjectCID:      item.SubjectCID,
		IsLikedByViewer: item.IsLikedByViewer,
		IsOwner:         item.IsOwner,
	}
	if item.Author != nil {
		out.Author = &AuthorSummary{
			DID:    item.Author.DID,
			Handle: item.Author.Handle,
		}
		if item.Author.DisplayName != nil {
			out.Author.DisplayName = *item.Author.DisplayName
		}
		if item.Author.Avatar != nil {
			out.Author.Avatar = *item.Author.Avatar
		}
	}
	return out
}

// CommentJSON is the JSON-serializable view of a firehose.IndexedComment.
// The IndexedComment struct marks computed profile fields (Handle, DisplayName,
// Avatar, Depth, Replies) with json:"-" because the templ layer reads them
// via accessor functions. The JSON API flattens these into the comment object
// so the SPA has everything it needs in one payload.
type CommentJSON struct {
	RKey        string    `json:"rkey"`
	SubjectURI  string    `json:"subject_uri"`
	Text        string    `json:"text"`
	ActorDID    string    `json:"actor_did"`
	CreatedAt   time.Time `json:"created_at"`
	ParentURI   string    `json:"parent_uri,omitempty"`
	ParentRKey  string    `json:"parent_rkey,omitempty"`
	CID         string    `json:"cid,omitempty"`
	Depth       int       `json:"depth"`
	Handle      string    `json:"handle,omitempty"`
	DisplayName string    `json:"display_name,omitempty"`
	Avatar      string    `json:"avatar,omitempty"`
	LikeCount   int       `json:"like_count"`
	IsLiked     bool      `json:"is_liked"`
}

// NewCommentJSON converts a firehose.IndexedComment to its JSON-serializable form.
func NewCommentJSON(c firehose.IndexedComment) CommentJSON {
	out := CommentJSON{
		RKey:       c.RKey,
		SubjectURI: c.SubjectURI,
		Text:       c.Text,
		ActorDID:   c.ActorDID,
		CreatedAt:  c.CreatedAt,
		ParentURI:  c.ParentURI,
		ParentRKey: c.ParentRKey,
		CID:        c.CID,
		Depth:      c.Depth,
		Handle:     c.Handle,
		LikeCount:  c.LikeCount,
		IsLiked:    c.IsLiked,
	}
	if c.DisplayName != nil {
		out.DisplayName = *c.DisplayName
	}
	if c.Avatar != nil {
		out.Avatar = *c.Avatar
	}
	return out
}

// NewCommentsJSON converts a slice of IndexedComment to JSON form.
func NewCommentsJSON(comments []firehose.IndexedComment) []CommentJSON {
	if len(comments) == 0 {
		return nil
	}
	out := make([]CommentJSON, 0, len(comments))
	for _, c := range comments {
		out = append(out, NewCommentJSON(c))
	}
	return out
}
