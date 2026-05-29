package social

import (
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// LikeToRecord converts a Like to an atproto record map.
func LikeToRecord(collection string, like *Like) (map[string]any, error) {
	if like.SubjectURI == "" {
		return nil, fmt.Errorf("subject URI is required")
	}
	if like.SubjectCID == "" {
		return nil, fmt.Errorf("subject CID is required")
	}

	return map[string]any{
		"$type": collection,
		"subject": map[string]any{
			"uri": like.SubjectURI,
			"cid": like.SubjectCID,
		},
		"createdAt": like.CreatedAt.Format(time.RFC3339),
	}, nil
}

// RecordToLike converts an atproto record map to a Like.
func RecordToLike(record map[string]any, atURI string) (*Like, error) {
	like := &Like{}
	if atURI != "" {
		parsedURI, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		like.RKey = parsedURI.RecordKey().String()
	}

	subject, ok := record["subject"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("subject is required")
	}
	subjectURI, ok := subject["uri"].(string)
	if !ok || subjectURI == "" {
		return nil, fmt.Errorf("subject.uri is required")
	}
	like.SubjectURI = subjectURI
	subjectCID, ok := subject["cid"].(string)
	if !ok || subjectCID == "" {
		return nil, fmt.Errorf("subject.cid is required")
	}
	like.SubjectCID = subjectCID

	createdAtStr, ok := record["createdAt"].(string)
	if !ok {
		return nil, fmt.Errorf("createdAt is required")
	}
	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("invalid createdAt format: %w", err)
	}
	like.CreatedAt = createdAt
	return like, nil
}

// CommentToRecord converts a Comment to an atproto record map.
func CommentToRecord(collection string, comment *Comment) (map[string]any, error) {
	if comment.SubjectURI == "" {
		return nil, fmt.Errorf("subject URI is required")
	}
	if comment.SubjectCID == "" {
		return nil, fmt.Errorf("subject CID is required")
	}
	if comment.Text == "" {
		return nil, fmt.Errorf("text is required")
	}

	record := map[string]any{
		"$type": collection,
		"subject": map[string]any{
			"uri": comment.SubjectURI,
			"cid": comment.SubjectCID,
		},
		"text":      comment.Text,
		"createdAt": comment.CreatedAt.Format(time.RFC3339),
	}
	if comment.ParentURI != "" && comment.ParentCID != "" {
		record["parent"] = map[string]any{
			"uri": comment.ParentURI,
			"cid": comment.ParentCID,
		}
	}
	return record, nil
}

// RecordToComment converts an atproto record map to a Comment.
func RecordToComment(record map[string]any, atURI string) (*Comment, error) {
	comment := &Comment{}
	if atURI != "" {
		parsedURI, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		comment.RKey = parsedURI.RecordKey().String()
	}

	subject, ok := record["subject"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("subject is required")
	}
	subjectURI, ok := subject["uri"].(string)
	if !ok || subjectURI == "" {
		return nil, fmt.Errorf("subject.uri is required")
	}
	comment.SubjectURI = subjectURI
	subjectCID, ok := subject["cid"].(string)
	if !ok || subjectCID == "" {
		return nil, fmt.Errorf("subject.cid is required")
	}
	comment.SubjectCID = subjectCID

	text, ok := record["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("text is required")
	}
	comment.Text = text

	createdAtStr, ok := record["createdAt"].(string)
	if !ok {
		return nil, fmt.Errorf("createdAt is required")
	}
	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("invalid createdAt format: %w", err)
	}
	comment.CreatedAt = createdAt

	if parent, ok := record["parent"].(map[string]any); ok {
		if parentURI, ok := parent["uri"].(string); ok && parentURI != "" {
			comment.ParentURI = parentURI
		}
		if parentCID, ok := parent["cid"].(string); ok && parentCID != "" {
			comment.ParentCID = parentCID
		}
	}
	return comment, nil
}
