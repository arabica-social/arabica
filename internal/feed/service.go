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
	items        []*FeedItem
	expiresAt    time.Time
	fromFirehose bool // tracks if cache was populated from firehose
	mu           sync.RWMutex
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
	registry       *Registry
	publicClient   *atproto.PublicClient
	cache          *publicFeedCache
	firehoseIndex  FirehoseIndex
	useFirehose    bool
}

// NewService creates a new feed service
func NewService(registry *Registry) *Service {
	return &Service{
		registry:     registry,
		publicClient: atproto.NewPublicClient(),
		cache:        &publicFeedCache{},
	}
}

// SetFirehoseIndex configures the service to use firehose-based feed when available
func (s *Service) SetFirehoseIndex(index FirehoseIndex) {
	s.firehoseIndex = index
	s.useFirehose = true
	log.Info().Msg("feed: firehose index configured")
}

// GetCachedPublicFeed returns cached feed items for unauthenticated users.
// It returns up to PublicFeedLimit items from the cache, refreshing if expired.
func (s *Service) GetCachedPublicFeed(ctx context.Context) ([]*FeedItem, error) {
	s.cache.mu.RLock()
	cacheValid := time.Now().Before(s.cache.expiresAt) && len(s.cache.items) > 0
	cacheFromFirehose := s.cache.fromFirehose
	items := s.cache.items
	s.cache.mu.RUnlock()

	// Check if we need to refresh: cache expired, empty, or firehose is now ready but cache was from polling
	firehoseReady := s.useFirehose && s.firehoseIndex != nil && s.firehoseIndex.IsReady()
	needsRefresh := !cacheValid || (firehoseReady && !cacheFromFirehose)

	if !needsRefresh {
		log.Debug().Int("item_count", len(items)).Bool("from_firehose", cacheFromFirehose).Msg("feed: returning cached public feed")
		return items, nil
	}

	// Cache is expired, empty, or we need to switch to firehose data
	return s.refreshPublicFeedCache(ctx)
}

// refreshPublicFeedCache fetches fresh feed items and updates the cache
func (s *Service) refreshPublicFeedCache(ctx context.Context) ([]*FeedItem, error) {
	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()

	// Check if firehose is ready (for tracking cache source)
	firehoseReady := s.useFirehose && s.firehoseIndex != nil && s.firehoseIndex.IsReady()

	// Double-check if another goroutine already refreshed the cache
	// But still refresh if firehose is ready and cache was from polling
	if time.Now().Before(s.cache.expiresAt) && len(s.cache.items) > 0 {
		if !firehoseReady || s.cache.fromFirehose {
			return s.cache.items, nil
		}
		// Firehose is ready but cache was from polling, continue to refresh
	}

	log.Debug().Bool("firehose_ready", firehoseReady).Msg("feed: refreshing public feed cache")

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
	s.cache.fromFirehose = firehoseReady

	log.Debug().
		Int("item_count", len(items)).
		Time("expires_at", s.cache.expiresAt).
		Bool("from_firehose", firehoseReady).
		Msg("feed: updated public feed cache")

	return items, nil
}

// GetRecentRecords fetches recent activity (brews and other records) from all registered users
// Returns up to `limit` items sorted by most recent first
func (s *Service) GetRecentRecords(ctx context.Context, limit int) ([]*FeedItem, error) {
	// Try firehose index first if available and ready
	if s.useFirehose && s.firehoseIndex != nil && s.firehoseIndex.IsReady() {
		log.Debug().Msg("feed: using firehose index")
		return s.getRecentRecordsFromFirehose(ctx, limit)
	}

	// Fallback to polling
	return s.getRecentRecordsViaPolling(ctx, limit)
}

// getRecentRecordsFromFirehose fetches feed items from the firehose index
func (s *Service) getRecentRecordsFromFirehose(ctx context.Context, limit int) ([]*FeedItem, error) {
	firehoseItems, err := s.firehoseIndex.GetRecentFeed(ctx, limit)
	if err != nil {
		log.Warn().Err(err).Msg("feed: firehose index error, falling back to polling")
		return s.getRecentRecordsViaPolling(ctx, limit)
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

// getRecentRecordsViaPolling fetches feed items by polling each user's PDS
func (s *Service) getRecentRecordsViaPolling(ctx context.Context, limit int) ([]*FeedItem, error) {
	dids := s.registry.List()
	if len(dids) == 0 {
		log.Debug().Msg("feed: no registered users")
		return nil, nil
	}

	log.Debug().Int("user_count", len(dids)).Msg("feed: fetching activity from registered users (polling)")

	// Fetch all records from all users in parallel
	type userActivity struct {
		did      string
		profile  *atproto.Profile
		brews    []*models.Brew
		beans    []*models.Bean
		roasters []*models.Roaster
		grinders []*models.Grinder
		brewers  []*models.Brewer
		err      error
	}

	results := make(chan userActivity, len(dids))
	var wg sync.WaitGroup

	for _, did := range dids {
		wg.Add(1)
		go func(did string) {
			defer wg.Done()

			result := userActivity{did: did}

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

			// Fetch recent beans
			beansOutput, err := s.publicClient.ListRecords(ctx, did, atproto.NSIDBean, 10)
			if err != nil {
				log.Warn().Err(err).Str("did", did).Msg("failed to fetch beans for feed")
			}

			// Fetch recent roasters
			roastersOutput, err := s.publicClient.ListRecords(ctx, did, atproto.NSIDRoaster, 10)
			if err != nil {
				log.Warn().Err(err).Str("did", did).Msg("failed to fetch roasters for feed")
			}

			// Fetch recent grinders
			grindersOutput, err := s.publicClient.ListRecords(ctx, did, atproto.NSIDGrinder, 10)
			if err != nil {
				log.Warn().Err(err).Str("did", did).Msg("failed to fetch grinders for feed")
			}

			// Fetch recent brewers
			brewersOutput, err := s.publicClient.ListRecords(ctx, did, atproto.NSIDBrewer, 10)
			if err != nil {
				log.Warn().Err(err).Str("did", did).Msg("failed to fetch brewers for feed")
			}

			// Fetch all beans, roasters, brewers, and grinders for this user to resolve references
			allBeansOutput, _ := s.publicClient.ListRecords(ctx, did, atproto.NSIDBean, 100)
			allRoastersOutput, _ := s.publicClient.ListRecords(ctx, did, atproto.NSIDRoaster, 100)
			allBrewersOutput, _ := s.publicClient.ListRecords(ctx, did, atproto.NSIDBrewer, 100)
			allGrindersOutput, _ := s.publicClient.ListRecords(ctx, did, atproto.NSIDGrinder, 100)

			// Build lookup maps (keyed by AT-URI)
			beanMap := make(map[string]*models.Bean)
			beanRoasterRefMap := make(map[string]string) // bean URI -> roaster URI
			roasterMap := make(map[string]*models.Roaster)
			brewerMap := make(map[string]*models.Brewer)
			grinderMap := make(map[string]*models.Grinder)

			// Populate bean map
			if allBeansOutput != nil {
				for _, beanRecord := range allBeansOutput.Records {
					bean, err := atproto.RecordToBean(beanRecord.Value, beanRecord.URI)
					if err == nil {
						beanMap[beanRecord.URI] = bean
						// Store roaster reference if present
						if roasterRef, ok := beanRecord.Value["roasterRef"].(string); ok && roasterRef != "" {
							beanRoasterRefMap[beanRecord.URI] = roasterRef
						}
					}
				}
			}

			// Populate roaster map
			if allRoastersOutput != nil {
				for _, roasterRecord := range allRoastersOutput.Records {
					roaster, err := atproto.RecordToRoaster(roasterRecord.Value, roasterRecord.URI)
					if err == nil {
						roasterMap[roasterRecord.URI] = roaster
					}
				}
			}

			// Populate brewer map
			if allBrewersOutput != nil {
				for _, brewerRecord := range allBrewersOutput.Records {
					brewer, err := atproto.RecordToBrewer(brewerRecord.Value, brewerRecord.URI)
					if err == nil {
						brewerMap[brewerRecord.URI] = brewer
					}
				}
			}

			// Populate grinder map
			if allGrindersOutput != nil {
				for _, grinderRecord := range allGrindersOutput.Records {
					grinder, err := atproto.RecordToGrinder(grinderRecord.Value, grinderRecord.URI)
					if err == nil {
						grinderMap[grinderRecord.URI] = grinder
					}
				}
			}

			// Convert records to Brew models and resolve references
			brews := make([]*models.Brew, 0, len(brewsOutput.Records))
			for _, record := range brewsOutput.Records {
				brew, err := atproto.RecordToBrew(record.Value, record.URI)
				if err != nil {
					log.Warn().Err(err).Str("uri", record.URI).Msg("failed to parse brew record")
					continue
				}

				// Resolve bean reference
				if beanRef, ok := record.Value["beanRef"].(string); ok && beanRef != "" {
					if bean, found := beanMap[beanRef]; found {
						brew.Bean = bean

						// Resolve roaster reference for this bean
						if roasterRef, found := beanRoasterRefMap[beanRef]; found {
							if roaster, found := roasterMap[roasterRef]; found {
								brew.Bean.Roaster = roaster
							}
						}
					}
				}

				// Resolve brewer reference
				if brewerRef, ok := record.Value["brewerRef"].(string); ok && brewerRef != "" {
					if brewer, found := brewerMap[brewerRef]; found {
						brew.BrewerObj = brewer
					}
				}

				// Resolve grinder reference
				if grinderRef, ok := record.Value["grinderRef"].(string); ok && grinderRef != "" {
					if grinder, found := grinderMap[grinderRef]; found {
						brew.GrinderObj = grinder
					}
				}

				brews = append(brews, brew)
			}
			result.brews = brews

			// Convert beans to models and resolve roaster references
			beans := make([]*models.Bean, 0)
			if beansOutput != nil {
				for _, record := range beansOutput.Records {
					bean, err := atproto.RecordToBean(record.Value, record.URI)
					if err != nil {
						log.Warn().Err(err).Str("uri", record.URI).Msg("failed to parse bean record")
						continue
					}

					// Resolve roaster reference
					if roasterRef, found := beanRoasterRefMap[record.URI]; found {
						if roaster, found := roasterMap[roasterRef]; found {
							bean.Roaster = roaster
						}
					}

					beans = append(beans, bean)
				}
			}
			result.beans = beans

			// Convert roasters to models
			roasters := make([]*models.Roaster, 0)
			if roastersOutput != nil {
				for _, record := range roastersOutput.Records {
					roaster, err := atproto.RecordToRoaster(record.Value, record.URI)
					if err != nil {
						log.Warn().Err(err).Str("uri", record.URI).Msg("failed to parse roaster record")
						continue
					}
					roasters = append(roasters, roaster)
				}
			}
			result.roasters = roasters

			// Convert grinders to models
			grinders := make([]*models.Grinder, 0)
			if grindersOutput != nil {
				for _, record := range grindersOutput.Records {
					grinder, err := atproto.RecordToGrinder(record.Value, record.URI)
					if err != nil {
						log.Warn().Err(err).Str("uri", record.URI).Msg("failed to parse grinder record")
						continue
					}
					grinders = append(grinders, grinder)
				}
			}
			result.grinders = grinders

			// Convert brewers to models
			brewers := make([]*models.Brewer, 0)
			if brewersOutput != nil {
				for _, record := range brewersOutput.Records {
					brewer, err := atproto.RecordToBrewer(record.Value, record.URI)
					if err != nil {
						log.Warn().Err(err).Str("uri", record.URI).Msg("failed to parse brewer record")
						continue
					}
					brewers = append(brewers, brewer)
				}
			}
			result.brewers = brewers

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

		totalRecords := len(result.brews) + len(result.beans) + len(result.roasters) + len(result.grinders) + len(result.brewers)

		log.Debug().
			Str("did", result.did).
			Str("handle", result.profile.Handle).
			Int("brew_count", len(result.brews)).
			Int("bean_count", len(result.beans)).
			Int("roaster_count", len(result.roasters)).
			Int("grinder_count", len(result.grinders)).
			Int("brewer_count", len(result.brewers)).
			Int("total_records", totalRecords).
			Msg("feed: collected records from user")

		// Add brews to feed
		for _, brew := range result.brews {
			items = append(items, &FeedItem{
				RecordType: "brew",
				Action:     "☕ added a new brew",
				Brew:       brew,
				Author:     result.profile,
				Timestamp:  brew.CreatedAt,
				TimeAgo:    FormatTimeAgo(brew.CreatedAt),
			})
		}

		// Add beans to feed
		for _, bean := range result.beans {
			items = append(items, &FeedItem{
				RecordType: "bean",
				Action:     "🫘 added a new bean",
				Bean:       bean,
				Author:     result.profile,
				Timestamp:  bean.CreatedAt,
				TimeAgo:    FormatTimeAgo(bean.CreatedAt),
			})
		}

		// Add roasters to feed
		for _, roaster := range result.roasters {
			items = append(items, &FeedItem{
				RecordType: "roaster",
				Action:     "🏪 added a new roaster",
				Roaster:    roaster,
				Author:     result.profile,
				Timestamp:  roaster.CreatedAt,
				TimeAgo:    FormatTimeAgo(roaster.CreatedAt),
			})
		}

		// Add grinders to feed
		for _, grinder := range result.grinders {
			items = append(items, &FeedItem{
				RecordType: "grinder",
				Action:     "⚙️ added a new grinder",
				Grinder:    grinder,
				Author:     result.profile,
				Timestamp:  grinder.CreatedAt,
				TimeAgo:    FormatTimeAgo(grinder.CreatedAt),
			})
		}

		// Add brewers to feed
		for _, brewer := range result.brewers {
			items = append(items, &FeedItem{
				RecordType: "brewer",
				Action:     "☕ added a new brewer",
				Brewer:     brewer,
				Author:     result.profile,
				Timestamp:  brewer.CreatedAt,
				TimeAgo:    FormatTimeAgo(brewer.CreatedAt),
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
