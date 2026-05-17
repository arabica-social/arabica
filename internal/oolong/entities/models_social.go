package oolong

import (
	"errors"
	"time"
	"unicode/utf8"
)

type Like struct {
	RKey       string    `json:"rkey"`
	SubjectURI string    `json:"subject_uri"`
	SubjectCID string    `json:"subject_cid"`
	CreatedAt  time.Time `json:"created_at"`
	ActorDID   string    `json:"actor_did,omitempty"`
}

type CreateLikeRequest struct {
	SubjectURI string `json:"subject_uri"`
	SubjectCID string `json:"subject_cid"`
}

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

type CreateCommentRequest struct {
	SubjectURI string `json:"subject_uri"`
	SubjectCID string `json:"subject_cid"`
	Text       string `json:"text"`
	ParentURI  string `json:"parent_uri,omitempty"`
	ParentCID  string `json:"parent_cid,omitempty"`
}

func (r *CreateLikeRequest) Validate() error {
	if r.SubjectURI == "" {
		return ErrSubjectRequired
	}
	if r.SubjectCID == "" {
		return errors.New("subject_cid is required")
	}
	return nil
}

func (r *CreateCommentRequest) Validate() error {
	if r.Text == "" {
		return ErrTextRequired
	}
	if len(r.Text) > MaxCommentText {
		return ErrTextTooLong
	}
	if utf8.RuneCountInString(r.Text) > MaxCommentGraphemes {
		return ErrTextTooLong
	}
	if r.SubjectURI == "" {
		return ErrSubjectRequired
	}
	if r.SubjectCID == "" {
		return errors.New("subject_cid is required")
	}
	if (r.ParentURI != "" && r.ParentCID == "") || (r.ParentURI == "" && r.ParentCID != "") {
		return ErrParentInvalid
	}
	return nil
}
