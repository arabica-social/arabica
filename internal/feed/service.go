package feed

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"arabica/internal/atproto"
	"arabica/internal/models"

	"github.com/rs/zerolog/log"
)

// FeedItem represents a brew in the social feed with author info
type FeedItem struct {
	Brew      *models.Brew
	Author    *atproto.Profile
	Timestamp time.Time
	TimeAgo   string // "2 hours ago", "yesterday", etc.
}

// Service fetches and aggregates brews from registered users
type Service struct {
	registry     *Registry
	publicClient *atproto.PublicClient
}

// NewService creates a new feed service
func NewService(registry *Registry) *Service {
	return &Service{
		registry:     registry,
		publicClient: atproto.NewPublicClient(),
	}
}

// GetRecentBrews fetches recent brews from all registered users
// Returns up to `limit` items sorted by most recent first
func (s *Service) GetRecentBrews(ctx context.Context, limit int) ([]*FeedItem, error) {
	dids := s.registry.List()
	if len(dids) == 0 {
		log.Debug().Msg("feed: no registered users")
		return nil, nil
	}

	log.Debug().Int("user_count", len(dids)).Msg("feed: fetching brews from registered users")

	// Fetch brews from all users in parallel
	type userBrews struct {
		did     string
		profile *atproto.Profile
		brews   []*models.Brew
		err     error
	}

	results := make(chan userBrews, len(dids))
	var wg sync.WaitGroup

	for _, did := range dids {
		wg.Add(1)
		go func(did string) {
			defer wg.Done()

			result := userBrews{did: did}

			// Fetch profile
			profile, err := s.publicClient.GetProfile(ctx, did)
			if err != nil {
				log.Warn().Err(err).Str("did", did).Msg("failed to fetch profile for feed")
				result.err = err
				results <- result
				return
			}
			result.profile = profile

			// Fetch recent brews (limit per user to avoid fetching too many)
			brewsOutput, err := s.publicClient.ListRecords(ctx, did, atproto.NSIDBrew, 10)
			if err != nil {
				log.Warn().Err(err).Str("did", did).Msg("failed to fetch brews for feed")
				result.err = err
				results <- result
				return
			}

			// Convert records to Brew models
			brews := make([]*models.Brew, 0, len(brewsOutput.Records))
			for _, record := range brewsOutput.Records {
				brew, err := atproto.RecordToBrew(record.Value, record.URI)
				if err != nil {
					log.Warn().Err(err).Str("uri", record.URI).Msg("failed to parse brew record")
					continue
				}
				brews = append(brews, brew)
			}
			result.brews = brews

			results <- result
		}(did)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect all feed items
	var items []*FeedItem
	for result := range results {
		if result.err != nil {
			continue
		}

		log.Debug().
			Str("did", result.did).
			Str("handle", result.profile.Handle).
			Int("brew_count", len(result.brews)).
			Msg("feed: collected brews from user")

		for _, brew := range result.brews {
			items = append(items, &FeedItem{
				Brew:      brew,
				Author:    result.profile,
				Timestamp: brew.CreatedAt,
				TimeAgo:   FormatTimeAgo(brew.CreatedAt),
			})
		}
	}

	// Sort by timestamp descending (most recent first)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Timestamp.After(items[j].Timestamp)
	})

	// Limit results
	if len(items) > limit {
		items = items[:limit]
	}

	log.Debug().Int("total_items", len(items)).Msg("feed: returning items")

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
