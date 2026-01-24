package feed

import (
	"context"
	"fmt"
	"sync"
	"time"

	"arabica/internal/atproto"
	"arabica/internal/models"

	"github.com/rs/zerolog/log"
)

// PublicFeedCacheTTL is the duration for which the public feed cache is valid.
// This value can be adjusted based on desired freshness vs. performance tradeoff.
// Consider values between 5-10 minutes for a good balance.
const PublicFeedCacheTTL = 5 * time.Minute

// PublicFeedLimit is the number of items to show for unauthenticated users
const PublicFeedLimit = 5

// FeedItem represents an activity in the social feed with author info
type FeedItem struct {
	// Record type and data (only one will be non-nil)
	RecordType string // "brew", "bean", "roaster", "grinder", "brewer"
	Action     string // "added a new brew", "added a new bean", etc.

	Brew    *models.Brew
	Bean    *models.Bean
	Roaster *models.Roaster
	Grinder *models.Grinder
	Brewer  *models.Brewer

	Author    *atproto.Profile
	Timestamp time.Time
	TimeAgo   string // "2 hours ago", "yesterday", etc.
}

// publicFeedCache holds cached feed items for unauthenticated users
type publicFeedCache struct {
	items     []*FeedItem
	expiresAt time.Time
	mu        sync.RWMutex
}

// FirehoseIndex is the interface for the firehose feed index
// This allows the feed service to use firehose data when available
type FirehoseIndex interface {
	IsReady() bool
	GetRecentFeed(ctx context.Context, limit int) ([]*FirehoseFeedItem, error)
}

// FirehoseFeedItem matches the FeedItem structure from firehose package
// This avoids import cycles
type FirehoseFeedItem struct {
	RecordType string
	Action     string
	Brew       *models.Brew
	Bean       *models.Bean
	Roaster    *models.Roaster
	Grinder    *models.Grinder
	Brewer     *models.Brewer
	Author     *atproto.Profile
	Timestamp  time.Time
	TimeAgo    string
}

// Service fetches and aggregates brews from registered users
type Service struct {
	registry      *Registry
	cache         *publicFeedCache
	firehoseIndex FirehoseIndex
}

// NewService creates a new feed service
func NewService(registry *Registry) *Service {
	return &Service{
		registry: registry,
		cache:    &publicFeedCache{},
	}
}

// SetFirehoseIndex configures the service to use firehose-based feed
func (s *Service) SetFirehoseIndex(index FirehoseIndex) {
	s.firehoseIndex = index
	log.Info().Msg("feed: firehose index configured")
}

// GetCachedPublicFeed returns cached feed items for unauthenticated users.
// It returns up to PublicFeedLimit items from the cache, refreshing if expired.
func (s *Service) GetCachedPublicFeed(ctx context.Context) ([]*FeedItem, error) {
	s.cache.mu.RLock()
	cacheValid := time.Now().Before(s.cache.expiresAt) && len(s.cache.items) > 0
	items := s.cache.items
	s.cache.mu.RUnlock()

	if cacheValid {
		log.Debug().Int("item_count", len(items)).Msg("feed: returning cached public feed")
		return items, nil
	}

	// Cache is expired or empty, refresh it
	return s.refreshPublicFeedCache(ctx)
}

// refreshPublicFeedCache fetches fresh feed items and updates the cache
func (s *Service) refreshPublicFeedCache(ctx context.Context) ([]*FeedItem, error) {
	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()

	// Double-check if another goroutine already refreshed the cache
	if time.Now().Before(s.cache.expiresAt) && len(s.cache.items) > 0 {
		return s.cache.items, nil
	}

	log.Debug().Msg("feed: refreshing public feed cache")

	// Fetch fresh feed items (limited to PublicFeedLimit)
	items, err := s.GetRecentRecords(ctx, PublicFeedLimit)
	if err != nil {
		// If we have stale data, return it rather than failing
		if len(s.cache.items) > 0 {
			log.Warn().Err(err).Msg("feed: failed to refresh cache, returning stale data")
			return s.cache.items, nil
		}
		return nil, err
	}

	// Update cache
	s.cache.items = items
	s.cache.expiresAt = time.Now().Add(PublicFeedCacheTTL)

	log.Debug().
		Int("item_count", len(items)).
		Time("expires_at", s.cache.expiresAt).
		Msg("feed: updated public feed cache")

	return items, nil
}

// GetRecentRecords fetches recent activity (brews and other records) from firehose index
// Returns up to `limit` items sorted by most recent first
func (s *Service) GetRecentRecords(ctx context.Context, limit int) ([]*FeedItem, error) {
	if s.firehoseIndex == nil || !s.firehoseIndex.IsReady() {
		log.Warn().Msg("feed: firehose index not ready")
		return nil, fmt.Errorf("firehose index not ready")
	}

	log.Debug().Msg("feed: using firehose index")
	return s.getRecentRecordsFromFirehose(ctx, limit)
}

// getRecentRecordsFromFirehose fetches feed items from the firehose index
func (s *Service) getRecentRecordsFromFirehose(ctx context.Context, limit int) ([]*FeedItem, error) {
	firehoseItems, err := s.firehoseIndex.GetRecentFeed(ctx, limit)
	if err != nil {
		log.Warn().Err(err).Msg("feed: firehose index error")
		return nil, err
	}

	// Convert FirehoseFeedItem to FeedItem
	items := make([]*FeedItem, len(firehoseItems))
	for i, fi := range firehoseItems {
		items[i] = &FeedItem{
			RecordType: fi.RecordType,
			Action:     fi.Action,
			Brew:       fi.Brew,
			Bean:       fi.Bean,
			Roaster:    fi.Roaster,
			Grinder:    fi.Grinder,
			Brewer:     fi.Brewer,
			Author:     fi.Author,
			Timestamp:  fi.Timestamp,
			TimeAgo:    fi.TimeAgo,
		}
	}

	log.Debug().Int("count", len(items)).Msg("feed: returning items from firehose index")
	return items, nil
}

// FormatTimeAgo returns a human-readable relative time string
func FormatTimeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return formatPlural(mins, "minute")
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return formatPlural(hours, "hour")
	case diff < 48*time.Hour:
		return "yesterday"
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return formatPlural(days, "day")
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return formatPlural(weeks, "week")
	case diff < 365*24*time.Hour:
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return formatPlural(months, "month")
	default:
		years := int(diff.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}
		return formatPlural(years, "year")
	}
}

func formatPlural(n int, unit string) string {
	if n == 1 {
		return "1 " + unit + " ago"
	}
	return fmt.Sprintf("%d %ss ago", n, unit)
}
