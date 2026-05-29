package firehose

import (
	"fmt"
	"strings"
	"time"

	"tangled.org/arabica.social/arabica/internal/arabica/entities"

	"github.com/rs/zerolog/log"
)

// CreateNotification stores a notification for the target user.
// Deduplicates by (type + actorDID + subjectURI) via unique index.
// Self-notifications (actorDID == targetDID) are silently skipped.
func (idx *FeedIndex) CreateNotification(targetDID string, notif arabica.Notification) error {
	if targetDID == "" || targetDID == notif.ActorDID {
		return nil // skip self-notifications
	}

	// Generate ID from timestamp
	if notif.ID == "" {
		notif.ID = fmt.Sprintf("%d", notif.CreatedAt.UnixNano())
	}

	return idx.notifications.create(targetDID, storedNotification{
		ID:         notif.ID,
		Type:       string(notif.Type),
		ActorDID:   notif.ActorDID,
		SubjectURI: notif.SubjectURI,
		CreatedAt:  notif.CreatedAt,
	})
}

// GetNotifications returns notifications for a user, newest first.
// Uses cursor-based pagination. Returns notifications, next cursor, and error.
func (idx *FeedIndex) GetNotifications(targetDID string, limit int, cursor string) ([]arabica.Notification, string, error) {
	stored, nextCursor, err := idx.notifications.list(targetDID, limit, cursor)
	if err != nil {
		return nil, "", err
	}

	notifications := make([]arabica.Notification, 0, len(stored))
	for _, storedNotif := range stored {
		notifications = append(notifications, arabica.Notification{
			ID:         storedNotif.ID,
			Type:       arabica.NotificationType(storedNotif.Type),
			ActorDID:   storedNotif.ActorDID,
			SubjectURI: storedNotif.SubjectURI,
			CreatedAt:  storedNotif.CreatedAt,
			Read:       storedNotif.Read,
		})
	}

	return notifications, nextCursor, nil
}

// GetUnreadCount returns the number of unread notifications for a user.
func (idx *FeedIndex) GetUnreadCount(targetDID string) int {
	return idx.notifications.unreadCount(targetDID)
}

// MarkAllRead updates the last_read timestamp to now for the user.
func (idx *FeedIndex) MarkAllRead(targetDID string) error {
	return idx.notifications.markAllRead(targetDID)
}

// parseTargetDID extracts the DID from an AT-URI (at://did:plc:xxx/collection/rkey)
func parseTargetDID(atURI string) string {
	if !strings.HasPrefix(atURI, "at://") {
		return ""
	}
	rest := atURI[5:] // strip "at://"
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) == 0 {
		return ""
	}
	did := parts[0]
	if !strings.HasPrefix(did, "did:") {
		return ""
	}
	return did
}

// DeleteNotification removes a notification matching (type + actorDID + subjectURI)
// from the target user's notification list. No-op if not found.
func (idx *FeedIndex) DeleteNotification(targetDID string, notifType arabica.NotificationType, actorDID, subjectURI string) {
	if targetDID == "" {
		return
	}

	if err := idx.notifications.delete(targetDID, string(notifType), actorDID, subjectURI); err != nil {
		log.Warn().Err(err).Str("target", targetDID).Str("actor", actorDID).Msg("failed to delete notification")
	}
}

// DeleteLikeNotification removes the notification for a like that was undone
func (idx *FeedIndex) DeleteLikeNotification(actorDID, subjectURI string) {
	targetDID := parseTargetDID(subjectURI)
	idx.DeleteNotification(targetDID, arabica.NotificationLike, actorDID, subjectURI)
}

// DeleteCommentNotification removes notifications for a deleted comment
func (idx *FeedIndex) DeleteCommentNotification(actorDID, subjectURI, parentURI string) {
	// Remove the comment notification sent to the brew owner
	targetDID := parseTargetDID(subjectURI)
	idx.DeleteNotification(targetDID, arabica.NotificationComment, actorDID, subjectURI)

	// Remove the reply notification sent to the parent comment's author
	if parentURI != "" {
		parentAuthorDID := parseTargetDID(parentURI)
		if parentAuthorDID != targetDID {
			idx.DeleteNotification(parentAuthorDID, arabica.NotificationCommentReply, actorDID, subjectURI)
		}
	}
}

// GetCommentSubjectURI returns the subject URI for a comment by actor+rkey.
// Returns empty string if not found.
func (idx *FeedIndex) GetCommentSubjectURI(actorDID, rkey string) string {
	var subjectURI string
	err := idx.db.QueryRow(`SELECT subject_uri FROM comments WHERE actor_did = ? AND rkey = ?`,
		actorDID, rkey).Scan(&subjectURI)
	if err != nil {
		return ""
	}
	return subjectURI
}

// CreateLikeNotification creates a notification for a like event
func (idx *FeedIndex) CreateLikeNotification(actorDID, subjectURI string) {
	targetDID := parseTargetDID(subjectURI)
	if targetDID == "" || targetDID == actorDID {
		return
	}

	notif := arabica.Notification{
		Type:       arabica.NotificationLike,
		ActorDID:   actorDID,
		SubjectURI: subjectURI,
		CreatedAt:  time.Now(),
	}

	if err := idx.CreateNotification(targetDID, notif); err != nil {
		log.Warn().Err(err).Str("actor", actorDID).Str("subject", subjectURI).Msg("failed to create like notification")
	}
}

// CreateCommentNotification creates notifications for a comment event.
// Notifies the brew owner (comment) and the parent comment author (reply).
func (idx *FeedIndex) CreateCommentNotification(actorDID, subjectURI, parentURI string) {
	now := time.Now()

	// Notify the brew owner
	targetDID := parseTargetDID(subjectURI)
	if targetDID != "" && targetDID != actorDID {
		notif := arabica.Notification{
			Type:       arabica.NotificationComment,
			ActorDID:   actorDID,
			SubjectURI: subjectURI,
			CreatedAt:  now,
		}
		if err := idx.CreateNotification(targetDID, notif); err != nil {
			log.Warn().Err(err).Str("actor", actorDID).Str("subject", subjectURI).Msg("failed to create comment notification")
		}
	}

	// If this is a reply, also notify the parent comment's author.
	if parentURI != "" {
		parentAuthorDID := parseTargetDID(parentURI)
		if parentAuthorDID != "" && parentAuthorDID != actorDID && parentAuthorDID != targetDID {
			replyNotif := arabica.Notification{
				Type:       arabica.NotificationCommentReply,
				ActorDID:   actorDID,
				SubjectURI: subjectURI, // brew URI, not parent comment URI
				CreatedAt:  now,
			}
			if err := idx.CreateNotification(parentAuthorDID, replyNotif); err != nil {
				log.Warn().Err(err).Str("actor", actorDID).Str("parent", parentURI).Msg("failed to create reply notification")
			}
		}
	}
}
