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

// GetRecentRecords fetches recent activity (brews and other records) from all registered users
// Returns up to `limit` items sorted by most recent first
func (s *Service) GetRecentRecords(ctx context.Context, limit int) ([]*FeedItem, error) {
	dids := s.registry.List()
	if len(dids) == 0 {
		log.Debug().Msg("feed: no registered users")
		return nil, nil
	}

	log.Debug().Int("user_count", len(dids)).Msg("feed: fetching activity from registered users")

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
