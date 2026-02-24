package firehose

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"arabica/internal/models"

	"github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"
)

// Bucket names for notifications
var (
	// BucketNotifications stores notifications: {target_did}:{inverted_timestamp}:{id} -> {Notification JSON}
	BucketNotifications = []byte("notifications")

	// BucketNotificationsMeta stores per-user metadata: {target_did}:last_read -> {timestamp RFC3339}
	BucketNotificationsMeta = []byte("notifications_meta")
)

// CreateNotification stores a notification for the target user.
// Deduplicates by (type + actorDID + subjectURI) to prevent duplicates from backfills.
// Self-notifications (actorDID == targetDID) are silently skipped.
func (idx *FeedIndex) CreateNotification(targetDID string, notif models.Notification) error {
	if targetDID == "" || targetDID == notif.ActorDID {
		return nil // skip self-notifications
	}

	return idx.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketNotifications)

		// Deduplication: scan for existing notification with same type+actor+subject
		prefix := []byte(targetDID + ":")
		c := b.Cursor()
		for k, v := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, v = c.Next() {
			var existing models.Notification
			if err := json.Unmarshal(v, &existing); err != nil {
				continue
			}
			if existing.Type == notif.Type && existing.ActorDID == notif.ActorDID && existing.SubjectURI == notif.SubjectURI {
				return nil // duplicate, skip
			}
		}

		// Generate ID from timestamp
		if notif.ID == "" {
			notif.ID = fmt.Sprintf("%d", notif.CreatedAt.UnixNano())
		}

		data, err := json.Marshal(notif)
		if err != nil {
			return fmt.Errorf("failed to marshal notification: %w", err)
		}

		// Key: {target_did}:{inverted_timestamp}:{id} for reverse chronological order
		inverted := ^uint64(notif.CreatedAt.UnixNano())
		key := fmt.Sprintf("%s:%016x:%s", targetDID, inverted, notif.ID)
		return b.Put([]byte(key), data)
	})
}

// GetNotifications returns notifications for a user, newest first.
// Uses cursor-based pagination. Returns notifications, next cursor, and error.
func (idx *FeedIndex) GetNotifications(targetDID string, limit int, cursor string) ([]models.Notification, string, error) {
	var notifications []models.Notification
	var nextCursor string

	if limit <= 0 {
		limit = 20
	}

	// Get last_read timestamp for marking read status
	lastRead := idx.getLastRead(targetDID)

	err := idx.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketNotifications)
		c := b.Cursor()

		prefix := []byte(targetDID + ":")
		var k, v []byte

		if cursor != "" {
			// Seek to cursor position, then advance past it
			k, v = c.Seek([]byte(cursor))
			if k != nil && string(k) == cursor {
				k, v = c.Next()
			}
		} else {
			k, v = c.Seek(prefix)
		}

		var lastKey []byte
		count := 0
		for ; k != nil && strings.HasPrefix(string(k), string(prefix)); k, v = c.Next() {
			if count >= limit {
				// There are more items beyond our limit
				nextCursor = string(lastKey)
				break
			}

			var notif models.Notification
			if err := json.Unmarshal(v, &notif); err != nil {
				continue
			}

			// Determine read status based on last_read timestamp
			if !lastRead.IsZero() && !notif.CreatedAt.After(lastRead) {
				notif.Read = true
			}

			notifications = append(notifications, notif)
			lastKey = make([]byte, len(k))
			copy(lastKey, k)
			count++
		}

		return nil
	})

	return notifications, nextCursor, err
}

// GetUnreadCount returns the number of unread notifications for a user.
func (idx *FeedIndex) GetUnreadCount(targetDID string) int {
	if targetDID == "" {
		return 0
	}

	lastRead := idx.getLastRead(targetDID)

	var count int
	_ = idx.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketNotifications)
		c := b.Cursor()

		prefix := []byte(targetDID + ":")
		for k, v := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, v = c.Next() {
			var notif models.Notification
			if err := json.Unmarshal(v, &notif); err != nil {
				continue
			}
			// If no last_read set, all are unread
			if lastRead.IsZero() || notif.CreatedAt.After(lastRead) {
				count++
			} else {
				// Since keys are in reverse chronological order,
				// once we hit a read notification, all remaining are also read
				break
			}
		}
		return nil
	})

	return count
}

// MarkAllRead updates the last_read timestamp to now for the user.
func (idx *FeedIndex) MarkAllRead(targetDID string) error {
	return idx.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketNotificationsMeta)
		key := []byte(targetDID + ":last_read")
		return b.Put(key, []byte(time.Now().Format(time.RFC3339Nano)))
	})
}

// getLastRead returns the last_read timestamp for a user.
func (idx *FeedIndex) getLastRead(targetDID string) time.Time {
	var lastRead time.Time
	_ = idx.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketNotificationsMeta)
		v := b.Get([]byte(targetDID + ":last_read"))
		if v != nil {
			if t, err := time.Parse(time.RFC3339Nano, string(v)); err == nil {
				lastRead = t
			}
		}
		return nil
	})
	return lastRead
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
	// We store the brew's subjectURI (not the parent comment URI) so the
	// notification links directly to the brew page with comments.
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
