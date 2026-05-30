package firehose

import (
	"database/sql"
	"time"
)

type notificationIndexStorage struct {
	db *sql.DB
}

func newNotificationIndexStorage(db *sql.DB) *notificationIndexStorage {
	return &notificationIndexStorage{db: db}
}

type storedNotification struct {
	ID         string
	Type       string
	ActorDID   string
	SubjectURI string
	CreatedAt  time.Time
	Read       bool
}

func (s *notificationIndexStorage) create(targetDID string, notif storedNotification) error {
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO notifications (id, target_did, type, actor_did, subject_uri, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, notif.ID, targetDID, notif.Type, notif.ActorDID, notif.SubjectURI,
		notif.CreatedAt.Format(time.RFC3339Nano))
	return err
}

func (s *notificationIndexStorage) list(targetDID string, limit int, cursor string) ([]storedNotification, string, error) {
	if limit <= 0 {
		limit = 20
	}

	lastRead := s.lastRead(targetDID)

	var args []any
	query := `SELECT id, type, actor_did, subject_uri, created_at
		FROM notifications WHERE target_did = ?`
	args = append(args, targetDID)

	if cursor != "" {
		query += ` AND created_at < ?`
		args = append(args, cursor)
	}

	query += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit+1)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	notifications := make([]storedNotification, 0, limit)
	for rows.Next() {
		var notif storedNotification
		var createdAtStr string
		if err := rows.Scan(&notif.ID, &notif.Type, &notif.ActorDID, &notif.SubjectURI, &createdAtStr); err != nil {
			continue
		}
		notif.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)

		if !lastRead.IsZero() && !notif.CreatedAt.After(lastRead) {
			notif.Read = true
		}

		notifications = append(notifications, notif)
	}

	var nextCursor string
	if len(notifications) > limit {
		last := notifications[limit-1]
		nextCursor = last.CreatedAt.Format(time.RFC3339Nano)
		notifications = notifications[:limit]
	}

	return notifications, nextCursor, rows.Err()
}

func (s *notificationIndexStorage) unreadCount(targetDID string) int {
	if targetDID == "" {
		return 0
	}

	lastRead := s.lastRead(targetDID)

	var count int
	if lastRead.IsZero() {
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE target_did = ?`, targetDID).Scan(&count)
	} else {
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE target_did = ? AND created_at > ?`,
			targetDID, lastRead.Format(time.RFC3339Nano)).Scan(&count)
	}

	return count
}

func (s *notificationIndexStorage) markAllRead(targetDID string) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO notifications_meta (target_did, last_read) VALUES (?, ?)`,
		targetDID, time.Now().Format(time.RFC3339Nano))
	return err
}

func (s *notificationIndexStorage) lastRead(targetDID string) time.Time {
	var lastReadStr string
	err := s.db.QueryRow(`SELECT last_read FROM notifications_meta WHERE target_did = ?`, targetDID).Scan(&lastReadStr)
	if err != nil {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339Nano, lastReadStr)
	return t
}

func (s *notificationIndexStorage) delete(targetDID, notifType, actorDID, subjectURI string) error {
	_, err := s.db.Exec(`
		DELETE FROM notifications
		WHERE target_did = ? AND type = ? AND actor_did = ? AND subject_uri = ?
	`, targetDID, notifType, actorDID, subjectURI)
	return err
}
