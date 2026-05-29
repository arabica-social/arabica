package social

import (
	"errors"
	"time"
)

const MaxCommentLength = 1000

var (
	ErrCommentRequired = errors.New("comment text is required")
	ErrCommentTooLong  = errors.New("comment text is too long")
)

// Like represents a like on a record.
type Like struct {
	RKey       string    `json:"rkey"`
	SubjectURI string    `json:"subject_uri"`
	SubjectCID string    `json:"subject_cid"`
	CreatedAt  time.Time `json:"created_at"`
	ActorDID   string    `json:"actor_did,omitempty"`
}

// CreateLikeRequest contains the data needed to create a like.
type CreateLikeRequest struct {
	SubjectURI string `json:"subject_uri"`
	SubjectCID string `json:"subject_cid"`
}

// Comment represents a comment on a record.
type Comment struct {
	RKey       string    `json:"rkey"`
	CID        string    `json:"cid,omitempty"`
	SubjectURI string    `json:"subject_uri"`
	SubjectCID string    `json:"subject_cid"`
	Text       string    `json:"text"`
	CreatedAt  time.Time `json:"created_at"`
	ActorDID   string    `json:"actor_did,omitempty"`
	ParentURI  string    `json:"parent_uri,omitempty"`
	ParentCID  string    `json:"parent_cid,omitempty"`
}

// CreateCommentRequest contains the data needed to create a comment.
type CreateCommentRequest struct {
	SubjectURI string `json:"subject_uri"`
	SubjectCID string `json:"subject_cid"`
	Text       string `json:"text"`
	ParentURI  string `json:"parent_uri,omitempty"`
	ParentCID  string `json:"parent_cid,omitempty"`
}

// Validate checks that all fields are within acceptable limits.
func (r *CreateCommentRequest) Validate() error {
	if r.Text == "" {
		return ErrCommentRequired
	}
	if len(r.Text) > MaxCommentLength {
		return ErrCommentTooLong
	}
	if r.SubjectURI == "" {
		return errors.New("subject_uri is required")
	}
	if r.SubjectCID == "" {
		return errors.New("subject_cid is required")
	}
	if (r.ParentURI != "" && r.ParentCID == "") || (r.ParentURI == "" && r.ParentCID != "") {
		return errors.New("both parent_uri and parent_cid must be provided together")
	}
	return nil
}
