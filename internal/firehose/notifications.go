package firehose

import (
	"fmt"
	"strings"
	"time"

	"arabica/internal/models"

	"github.com/rs/zerolog/log"
)

// CreateNotification stores a notification for the target user.
// Deduplicates by (type + actorDID + subjectURI) via unique index.
// Self-notifications (actorDID == targetDID) are silently skipped.
func (idx *FeedIndex) CreateNotification(targetDID string, notif models.Notification) error {
	if targetDID == "" || targetDID == notif.ActorDID {
		return nil // skip self-notifications
	}

	// Generate ID from timestamp
	if notif.ID == "" {
		notif.ID = fmt.Sprintf("%d", notif.CreatedAt.UnixNano())
	}

	// INSERT OR IGNORE deduplicates via the unique index on (target_did, type, actor_did, subject_uri)
	_, err := idx.db.Exec(`
		INSERT OR IGNORE INTO notifications (id, target_did, type, actor_did, subject_uri, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, notif.ID, targetDID, string(notif.Type), notif.ActorDID, notif.SubjectURI,
		notif.CreatedAt.Format(time.RFC3339Nano))
	return err
}

// GetNotifications returns notifications for a user, newest first.
// Uses cursor-based pagination. Returns notifications, next cursor, and error.
func (idx *FeedIndex) GetNotifications(targetDID string, limit int, cursor string) ([]models.Notification, string, error) {
	if limit <= 0 {
		limit = 20
	}

	lastRead := idx.getLastRead(targetDID)

	var args []any
	query := `SELECT id, type, actor_did, subject_uri, created_at
		FROM notifications WHERE target_did = ?`
	args = append(args, targetDID)

	if cursor != "" {
		query += ` AND created_at < ?`
		args = append(args, cursor)
	}

	query += ` ORDER BY created_at DESC LIMIT ?`
	// Fetch one extra to determine if there's a next page
	args = append(args, limit+1)

	rows, err := idx.db.Query(query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var notif models.Notification
		var typeStr, createdAtStr string
		if err := rows.Scan(&notif.ID, &typeStr, &notif.ActorDID, &notif.SubjectURI, &createdAtStr); err != nil {
			continue
		}
		notif.Type = models.NotificationType(typeStr)
		notif.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)

		if !lastRead.IsZero() && !notif.CreatedAt.After(lastRead) {
			notif.Read = true
		}

		notifications = append(notifications, notif)
	}

	var nextCursor string
	if len(notifications) > limit {
		// There are more results
		last := notifications[limit-1]
		nextCursor = last.CreatedAt.Format(time.RFC3339Nano)
		notifications = notifications[:limit]
	}

	return notifications, nextCursor, rows.Err()
}

// GetUnreadCount returns the number of unread notifications for a user.
func (idx *FeedIndex) GetUnreadCount(targetDID string) int {
	if targetDID == "" {
		return 0
	}

	lastRead := idx.getLastRead(targetDID)

	var count int
	if lastRead.IsZero() {
		_ = idx.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE target_did = ?`, targetDID).Scan(&count)
	} else {
		_ = idx.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE target_did = ? AND created_at > ?`,
			targetDID, lastRead.Format(time.RFC3339Nano)).Scan(&count)
	}

	return count
}

// MarkAllRead updates the last_read timestamp to now for the user.
func (idx *FeedIndex) MarkAllRead(targetDID string) error {
	_, err := idx.db.Exec(`INSERT OR REPLACE INTO notifications_meta (target_did, last_read) VALUES (?, ?)`,
		targetDID, time.Now().Format(time.RFC3339Nano))
	return err
}

// getLastRead returns the last_read timestamp for a user.
func (idx *FeedIndex) getLastRead(targetDID string) time.Time {
	var lastReadStr string
	err := idx.db.QueryRow(`SELECT last_read FROM notifications_meta WHERE target_did = ?`, targetDID).Scan(&lastReadStr)
	if err != nil {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339Nano, lastReadStr)
	return t
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
func (idx *FeedIndex) DeleteNotification(targetDID string, notifType models.NotificationType, actorDID, subjectURI string) {
	if targetDID == "" {
		return
	}

	_, err := idx.db.Exec(`
		DELETE FROM notifications
		WHERE target_did = ? AND type = ? AND actor_did = ? AND subject_uri = ?
	`, targetDID, string(notifType), actorDID, subjectURI)
	if err != nil {
		log.Warn().Err(err).Str("target", targetDID).Str("actor", actorDID).Msg("failed to delete notification")
	}
}

// DeleteLikeNotification removes the notification for a like that was undone
func (idx *FeedIndex) DeleteLikeNotification(actorDID, subjectURI string) {
	targetDID := parseTargetDID(subjectURI)
	idx.DeleteNotification(targetDID, models.NotificationLike, actorDID, subjectURI)
}

// DeleteCommentNotification removes notifications for a deleted comment
func (idx *FeedIndex) DeleteCommentNotification(actorDID, subjectURI, parentURI string) {
	// Remove the comment notification sent to the brew owner
	targetDID := parseTargetDID(subjectURI)
	idx.DeleteNotification(targetDID, models.NotificationComment, actorDID, subjectURI)

	// Remove the reply notification sent to the parent comment's author
	if parentURI != "" {
		parentAuthorDID := parseTargetDID(parentURI)
		if parentAuthorDID != targetDID {
			idx.DeleteNotification(parentAuthorDID, models.NotificationCommentReply, actorDID, subjectURI)
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

	notif := models.Notification{
		Type:       models.NotificationLike,
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
		notif := models.Notification{
			Type:       models.NotificationComment,
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
			replyNotif := models.Notification{
				Type:       models.NotificationCommentReply,
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
