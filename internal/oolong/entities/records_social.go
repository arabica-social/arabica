package oolong

import (
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// ========== Like ==========

func LikeToRecord(l *Like) (map[string]any, error) {
	if l.SubjectURI == "" {
		return nil, ErrSubjectRequired
	}
	return map[string]any{
		"$type": NSIDLike,
		"subject": map[string]any{
			"uri": l.SubjectURI,
			"cid": l.SubjectCID,
		},
		"createdAt": l.CreatedAt.Format(time.RFC3339),
	}, nil
}

func RecordToLike(record map[string]any, atURI string) (*Like, error) {
	l := &Like{}
	if atURI != "" {
		parsed, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		l.RKey = parsed.RecordKey().String()
	}
	subj, ok := record["subject"].(map[string]any)
	if !ok {
		return nil, ErrSubjectRequired
	}
	if uri, ok := subj["uri"].(string); ok {
		l.SubjectURI = uri
	}
	if cid, ok := subj["cid"].(string); ok {
		l.SubjectCID = cid
	}
	if createdStr, ok := record["createdAt"].(string); ok {
		t, err := time.Parse(time.RFC3339, createdStr)
		if err != nil {
			return nil, fmt.Errorf("invalid createdAt: %w", err)
		}
		l.CreatedAt = t
	}
	return l, nil
}

// ========== Comment ==========

func CommentToRecord(c *Comment) (map[string]any, error) {
	if c.SubjectURI == "" {
		return nil, ErrSubjectRequired
	}
	if c.Text == "" {
		return nil, ErrTextRequired
	}
	rec := map[string]any{
		"$type": NSIDComment,
		"subject": map[string]any{
			"uri": c.SubjectURI,
			"cid": c.SubjectCID,
		},
		"text":      c.Text,
		"createdAt": c.CreatedAt.Format(time.RFC3339),
	}
	if c.ParentURI != "" && c.ParentCID != "" {
		rec["parent"] = map[string]any{
			"uri": c.ParentURI,
			"cid": c.ParentCID,
		}
	}
	return rec, nil
}

func RecordToComment(record map[string]any, atURI string) (*Comment, error) {
	c := &Comment{}
	if atURI != "" {
		parsed, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		c.RKey = parsed.RecordKey().String()
	}
	subj, ok := record["subject"].(map[string]any)
	if !ok {
		return nil, ErrSubjectRequired
	}
	if uri, ok := subj["uri"].(string); ok {
		c.SubjectURI = uri
	}
	if cid, ok := subj["cid"].(string); ok {
		c.SubjectCID = cid
	}
	text, ok := record["text"].(string)
	if !ok {
		return nil, ErrTextRequired
	}
	c.Text = text

	if createdStr, ok := record["createdAt"].(string); ok {
		t, err := time.Parse(time.RFC3339, createdStr)
		if err != nil {
			return nil, fmt.Errorf("invalid createdAt: %w", err)
		}
		c.CreatedAt = t
	}
	if parent, ok := record["parent"].(map[string]any); ok {
		if uri, ok := parent["uri"].(string); ok {
			c.ParentURI = uri
		}
		if cid, ok := parent["cid"].(string); ok {
			c.ParentCID = cid
		}
	}
	return c, nil
}
