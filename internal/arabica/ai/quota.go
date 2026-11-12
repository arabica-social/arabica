package ai

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// DefaultDailyChars bounds the assistant context one account may send per UTC
// day. ARABICA_AI_DAILY_CHARS overrides it; zero disables the limit.
const DefaultDailyChars = 50000

// imageChars charges an attached image roughly as the text it costs upstream.
const imageChars = 1000

const maxUnlimitedDIDs = 1000

// UsageStore records daily account usage. It owns no user records.
type UsageStore interface {
	Consume(ctx context.Context, did, day string, chars, limit int64) (int64, bool, error)
	PurgeBefore(ctx context.Context, day string) error
}

// quota is configured once at bootstrap; the store and admin check arrive after
// the shared database exists.
type quota struct {
	store     UsageStore
	chars     int64
	unlimited map[string]bool
	isAdmin   func(string) bool
	mu        sync.Mutex
	purged    string
}

func (q *quota) exempt(did string) bool {
	if q.unlimited[did] {
		return true
	}
	return q.isAdmin != nil && q.isAdmin(did)
}

// SetUsageStore starts enforcing a daily allowance of chars per account. Zero
// chars or a nil store leaves the assistant unlimited.
func (s *Service) SetUsageStore(store UsageStore, chars int64, isAdmin func(string) bool) {
	if s == nil {
		return
	}
	if s.quota == nil {
		s.quota = &quota{unlimited: map[string]bool{}}
	}
	s.quota.store = store
	s.quota.chars = chars
	s.quota.isAdmin = isAdmin
}

// DailyChars reports the configured allowance, zero when unlimited.
func (s *Service) DailyChars() int64 {
	if s == nil || s.quota == nil {
		return 0
	}
	return s.quota.chars
}

// dailyQuota requires explicit configuration: a malformed value fails start-up
// rather than silently disabling the allowance.
func dailyQuota() (*quota, error) {
	q := &quota{chars: DefaultDailyChars, unlimited: map[string]bool{}}
	if raw := strings.TrimSpace(os.Getenv("ARABICA_AI_DAILY_CHARS")); raw != "" {
		chars, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || chars < 0 || chars > 1_000_000_000 {
			return nil, errors.New("ai: invalid ARABICA_AI_DAILY_CHARS")
		}
		q.chars = chars
	}
	for _, did := range strings.Split(os.Getenv("ARABICA_AI_UNLIMITED_DIDS"), ",") {
		did = strings.TrimSpace(did)
		if did == "" {
			continue
		}
		if _, err := syntax.ParseDID(did); err != nil || len(q.unlimited) >= maxUnlimitedDIDs {
			return nil, errors.New("ai: invalid ARABICA_AI_UNLIMITED_DIDS")
		}
		q.unlimited[did] = true
	}
	return q, nil
}

// requestChars charges what one request sends upstream: every message body plus
// a fixed rate per image. Re-sent history counts again because it is billed again.
func requestChars(req Request) int64 {
	var chars int64
	for _, m := range req.Messages {
		chars += int64(utf8.RuneCountInString(m.Content)) + imageChars*int64(len(m.Images))
	}
	return chars
}

func (s *Service) chargeDailyQuota(ctx context.Context, did string, req Request) error {
	q := s.quota
	if q == nil || q.store == nil || q.chars <= 0 || q.exempt(did) {
		return nil
	}
	now := time.Now().UTC()
	day := UsageDay(now)
	q.mu.Lock()
	purge := q.purged != day
	q.purged = day
	q.mu.Unlock()
	if purge {
		// Counts are only ever read for today, so yesterday's rows are dead weight.
		_ = q.store.PurgeBefore(ctx, UsageDay(now.AddDate(0, 0, -1)))
	}
	_, allowed, err := q.store.Consume(ctx, did, day, requestChars(req), q.chars)
	if err != nil {
		return ErrQuota
	}
	if !allowed {
		return ErrDailyLimit
	}
	return nil
}
