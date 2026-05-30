package firehose

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

type socialIndexStorage struct {
	db *sql.DB
}

func newSocialIndexStorage(db *sql.DB) *socialIndexStorage {
	return &socialIndexStorage{db: db}
}

func (s *socialIndexStorage) upsertLike(ctx context.Context, actorDID, rkey, subjectURI string) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO likes (subject_uri, actor_did, rkey) VALUES (?, ?, ?)`,
		subjectURI, actorDID, rkey)
	return err
}

func (s *socialIndexStorage) deleteLike(ctx context.Context, actorDID, subjectURI string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM likes WHERE subject_uri = ? AND actor_did = ?`,
		subjectURI, actorDID)
	return err
}

func (s *socialIndexStorage) likeCount(ctx context.Context, subjectURI string) int {
	var count int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM likes WHERE subject_uri = ?`, subjectURI).Scan(&count)
	return count
}

func (s *socialIndexStorage) hasUserLiked(ctx context.Context, actorDID, subjectURI string) bool {
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM likes WHERE actor_did = ? AND subject_uri = ? LIMIT 1`,
		actorDID, subjectURI).Scan(&exists)
	return err == nil
}

func (s *socialIndexStorage) userLikeRKey(ctx context.Context, actorDID, subjectURI string) string {
	var rkey string
	err := s.db.QueryRowContext(ctx, `SELECT rkey FROM likes WHERE actor_did = ? AND subject_uri = ?`,
		actorDID, subjectURI).Scan(&rkey)
	if err != nil {
		return ""
	}
	return rkey
}

func (s *socialIndexStorage) likeCountsBatch(ctx context.Context, uris []string) map[string]int {
	counts := make(map[string]int, len(uris))
	if len(uris) == 0 {
		return counts
	}
	ph, args := placeholders(uris)
	rows, err := s.db.QueryContext(ctx,
		`SELECT subject_uri, COUNT(*) FROM likes WHERE subject_uri IN (`+ph+`) GROUP BY subject_uri`, args...)
	if err != nil {
		return counts
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		var count int
		if err := rows.Scan(&uri, &count); err == nil {
			counts[uri] = count
		}
	}
	return counts
}

func (s *socialIndexStorage) hasUserLikedBatch(ctx context.Context, actorDID string, uris []string) map[string]bool {
	liked := make(map[string]bool, len(uris))
	if len(uris) == 0 || actorDID == "" {
		return liked
	}
	ph, args := placeholders(uris)
	allArgs := make([]any, 0, len(args)+1)
	allArgs = append(allArgs, actorDID)
	allArgs = append(allArgs, args...)
	rows, err := s.db.QueryContext(ctx,
		`SELECT subject_uri FROM likes WHERE actor_did = ? AND subject_uri IN (`+ph+`)`, allArgs...)
	if err != nil {
		return liked
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		if err := rows.Scan(&uri); err == nil {
			liked[uri] = true
		}
	}
	return liked
}

func (s *socialIndexStorage) upsertComment(ctx context.Context, actorDID, rkey, subjectURI, parentURI, cid, text string, createdAt time.Time) error {
	var parentRKey string
	if parentURI != "" {
		parts := strings.Split(parentURI, "/")
		if len(parts) > 0 {
			parentRKey = parts[len(parts)-1]
		}
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO comments (actor_did, rkey, subject_uri, parent_uri, parent_rkey, cid, text, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(actor_did, rkey) DO UPDATE SET
			subject_uri = excluded.subject_uri,
			parent_uri = excluded.parent_uri,
			parent_rkey = excluded.parent_rkey,
			cid = excluded.cid,
			text = excluded.text,
			created_at = excluded.created_at
	`, actorDID, rkey, subjectURI, parentURI, parentRKey, cid, text, createdAt.Format(time.RFC3339Nano))
	return err
}

func (s *socialIndexStorage) deleteComment(ctx context.Context, actorDID, rkey string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM comments WHERE actor_did = ? AND rkey = ?`, actorDID, rkey)
	return err
}

func (s *socialIndexStorage) commentCount(ctx context.Context, subjectURI string) int {
	var count int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM comments WHERE subject_uri = ?`, subjectURI).Scan(&count)
	return count
}

func (s *socialIndexStorage) commentCountsBatch(ctx context.Context, uris []string) map[string]int {
	counts := make(map[string]int, len(uris))
	if len(uris) == 0 {
		return counts
	}
	ph, args := placeholders(uris)
	rows, err := s.db.QueryContext(ctx,
		`SELECT subject_uri, COUNT(*) FROM comments WHERE subject_uri IN (`+ph+`) GROUP BY subject_uri`, args...)
	if err != nil {
		return counts
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		var count int
		if err := rows.Scan(&uri, &count); err == nil {
			counts[uri] = count
		}
	}
	return counts
}

func (s *socialIndexStorage) commentsForSubject(ctx context.Context, subjectURI string, limit int) []IndexedComment {
	query := `SELECT actor_did, rkey, subject_uri, parent_uri, parent_rkey, cid, text, created_at
		FROM comments WHERE subject_uri = ? ORDER BY created_at`
	var args []any
	args = append(args, subjectURI)
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var comments []IndexedComment
	for rows.Next() {
		var c IndexedComment
		var createdAtStr string
		if err := rows.Scan(&c.ActorDID, &c.RKey, &c.SubjectURI, &c.ParentURI, &c.ParentRKey,
			&c.CID, &c.Text, &createdAtStr); err != nil {
			continue
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
		comments = append(comments, c)
	}
	return comments
}

func (s *socialIndexStorage) totalLikeCount() int {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM likes`).Scan(&count)
	return count
}

func (s *socialIndexStorage) totalCommentCount() int {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM comments`).Scan(&count)
	return count
}

func (s *socialIndexStorage) deleteAllForDID(ctx context.Context, tx *sql.Tx, did, uriPrefix string) error {
	stmts := []struct {
		sql  string
		args []any
	}{
		{`DELETE FROM likes WHERE actor_did = ?`, []any{did}},
		{`DELETE FROM likes WHERE subject_uri LIKE ?`, []any{uriPrefix}},
		{`DELETE FROM comments WHERE actor_did = ?`, []any{did}},
		{`DELETE FROM comments WHERE subject_uri LIKE ?`, []any{uriPrefix}},
	}

	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt.sql, stmt.args...); err != nil {
			return err
		}
	}

	return nil
}
