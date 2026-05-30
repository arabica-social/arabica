package notifications

import "time"

// Type represents the type of notification.
type Type string

const (
	Like         Type = "like"
	Comment      Type = "comment"
	CommentReply Type = "comment_reply"
)

// Notification represents a notification for a user.
type Notification struct {
	ID         string    `json:"id"`          // Unique key (timestamp-based)
	Type       Type      `json:"type"`        // like, comment, comment_reply
	ActorDID   string    `json:"actor_did"`   // Who performed the action
	SubjectURI string    `json:"subject_uri"` // The record that was acted on
	CreatedAt  time.Time `json:"created_at"`
	Read       bool      `json:"read"`
}
