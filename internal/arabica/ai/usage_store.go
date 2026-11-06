package ai

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type DailyUsageStore struct{ db *sql.DB }

func NewDailyUsageStore(db *sql.DB) (*DailyUsageStore, error) {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS ai_daily_chat_usage (
		did TEXT NOT NULL,
		day TEXT NOT NULL,
		chars INTEGER NOT NULL,
		PRIMARY KEY (did, day)
	)`); err != nil {
		return nil, err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS ai_daily_chat_usage_day ON ai_daily_chat_usage (day)`); err != nil {
		return nil, err
	}
	return &DailyUsageStore{db: db}, nil
}

func UsageDay(at time.Time) string { return at.UTC().Format(time.DateOnly) }

// Consume records chars against one account's day budget in a single statement,
// so concurrent or repeated requests can never exceed limit. A false result
// means the request was not charged.
func (s *DailyUsageStore) Consume(ctx context.Context, did, day string, chars, limit int64) (int64, bool, error) {
	if did == "" || day == "" || chars < 0 || limit < 0 {
		return 0, false, errors.New("aiusage: invalid consumption")
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO ai_daily_chat_usage (did, day, chars) VALUES (?, ?, ?)
ON CONFLICT(did, day) DO UPDATE SET chars = ai_daily_chat_usage.chars + excluded.chars
WHERE ai_daily_chat_usage.chars + excluded.chars <= ?`, did, day, chars, limit)
	if err != nil {
		return 0, false, err
	}
	charged, err := result.RowsAffected()
	if err != nil {
		return 0, false, err
	}
	used, err := s.Used(ctx, did, day)
	if err != nil {
		return 0, false, err
	}
	return used, charged > 0, nil
}

func (s *DailyUsageStore) Used(ctx context.Context, did, day string) (int64, error) {
	var chars int64
	err := s.db.QueryRowContext(ctx, `SELECT chars FROM ai_daily_chat_usage WHERE did = ? AND day = ?`, did, day).Scan(&chars)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return chars, err
}

func (s *DailyUsageStore) PurgeBefore(ctx context.Context, day string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM ai_daily_chat_usage WHERE day < ?`, day)
	return err
}
