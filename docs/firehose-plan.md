# Firehose Integration Plan for Arabica

## Executive Summary

This document proposes refactoring Arabica's home page feed to consume events from the AT Protocol firehose via Jetstream, replacing the current polling-based approach. This will provide real-time updates, dramatically reduce API calls, and improve scalability.

**Recommendation:** Implement Jetstream consumer with local BoltDB index as Phase 1, with optional Slingshot/Constellation integration in Phase 2.

---

## Problem Statement

### Current Architecture

The feed service (`internal/feed/service.go`) polls each registered user's PDS directly:

```
For N registered users:
- N profile fetches
- N × 5 collection fetches (brew, bean, roaster, grinder, brewer)
- N × 4 reference resolution fetches
- Total: ~10N API calls per refresh
```

### Issues

| Problem                  | Impact                              |
| ------------------------ | ----------------------------------- |
| High API call volume     | Risk of rate limiting as users grow |
| 5-minute cache staleness | Users don't see recent activity     |
| N+1 query pattern        | Linear scaling, O(N) per refresh    |
| PDS dependency           | Feed fails if any PDS is slow/down  |
| No real-time updates     | Requires manual refresh             |

---

## Proposed Solution: Jetstream Consumer

### Architecture Overview

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   AT Protocol   │     │    Jetstream     │     │    Arabica      │
│    Firehose     │────▶│  (Public/Self)   │────▶│    Consumer     │
│  (all records)  │     │  JSON over WS    │     │  (background)   │
└─────────────────┘     └──────────────────┘     └────────┬────────┘
                                                          │
                                                          ▼
                                                 ┌─────────────────┐
                                                 │   Feed Index    │
                                                 │    (BoltDB)     │
                                                 └────────┬────────┘
                                                          │
                                                          ▼
                                                 ┌─────────────────┐
                                                 │   HTTP Handler  │
                                                 │   (instant)     │
                                                 └─────────────────┘
```

### How It Works

1. **Background Consumer** connects to Jetstream WebSocket
2. **Filters** for `social.arabica.alpha.*` collections
3. **Indexes** incoming events into local BoltDB
4. **Serves** feed requests instantly from local index
5. **Fallback** to direct polling if consumer disconnects

### Benefits

| Metric                | Current          | With Jetstream    |
| --------------------- | ---------------- | ----------------- |
| API calls per refresh | ~10N             | 0                 |
| Feed latency          | 5 min cache      | Real-time (<1s)   |
| PDS dependency        | High             | None (after sync) |
| User discovery        | Manual registry  | Automatic         |
| Scalability           | O(N) per request | O(1) per request  |

---

## Technical Design

### 1. Jetstream Client Configuration

```go
// internal/firehose/config.go

type JetstreamConfig struct {
    // Public endpoints (fallback rotation)
    Endpoints []string

    // Filter to Arabica collections only
    WantedCollections []string

    // Optional: filter to registered DIDs only
    // Leave empty to discover all Arabica users
    WantedDids []string

    // Enable zstd compression (~56% bandwidth reduction)
    Compress bool

    // Cursor file path for restart recovery
    CursorFile string
}

func DefaultConfig() *JetstreamConfig {
    return &JetstreamConfig{
        Endpoints: []string{
            "wss://jetstream1.us-east.bsky.network/subscribe",
            "wss://jetstream2.us-east.bsky.network/subscribe",
            "wss://jetstream1.us-west.bsky.network/subscribe",
            "wss://jetstream2.us-west.bsky.network/subscribe",
        },
        WantedCollections: []string{
            "social.arabica.alpha.brew",
            "social.arabica.alpha.bean",
            "social.arabica.alpha.roaster",
            "social.arabica.alpha.grinder",
            "social.arabica.alpha.brewer",
        },
        Compress: true,
        CursorFile: "jetstream-cursor.txt",
    }
}
```

### 2. Event Processing

```go
// internal/firehose/consumer.go

type Consumer struct {
    config    *JetstreamConfig
    index     *FeedIndex
    client    *jetstream.Client
    cursor    atomic.Int64
    connected atomic.Bool
}

func (c *Consumer) handleEvent(ctx context.Context, event *models.Event) error {
    if event.Kind != "commit" || event.Commit == nil {
        return nil
    }

    commit := event.Commit

    // Only process Arabica collections
    if !strings.HasPrefix(commit.Collection, "social.arabica.alpha.") {
        return nil
    }

    switch commit.Operation {
    case "create", "update":
        return c.index.UpsertRecord(ctx, event.Did, commit)
    case "delete":
        return c.index.DeleteRecord(ctx, event.Did, commit.Collection, commit.RKey)
    }

    // Update cursor for recovery
    c.cursor.Store(event.TimeUS)

    return nil
}
```

### 3. Feed Index Schema (BoltDB)

```go
// internal/firehose/index.go

// BoltDB Buckets:
// - "records"     : {at-uri} -> {record JSON + metadata}
// - "by_time"     : {timestamp:at-uri} -> {} (for chronological queries)
// - "by_did"      : {did:at-uri} -> {} (for user-specific queries)
// - "by_type"     : {collection:timestamp:at-uri} -> {} (for type filtering)
// - "profiles"    : {did} -> {profile JSON} (cached profiles)
// - "cursor"      : "jetstream" -> {cursor value}

type FeedIndex struct {
    db *bbolt.DB
}

type IndexedRecord struct {
    URI        string          `json:"uri"`
    DID        string          `json:"did"`
    Collection string          `json:"collection"`
    RKey       string          `json:"rkey"`
    Record     json.RawMessage `json:"record"`
    CID        string          `json:"cid"`
    IndexedAt  time.Time       `json:"indexed_at"`
}

func (idx *FeedIndex) GetRecentFeed(ctx context.Context, limit int) ([]*FeedItem, error) {
    // Query by_time bucket in reverse order
    // Hydrate with profile data from profiles bucket
    // Return feed items instantly from local data
}
```

### 4. Profile Resolution

Profiles are not part of Arabica's lexicons, so we need a strategy:

**Option A: Lazy Loading (Recommended for Phase 1)**

```go
func (idx *FeedIndex) resolveProfile(ctx context.Context, did string) (*Profile, error) {
    // Check local cache first
    if profile := idx.getCachedProfile(did); profile != nil {
        return profile, nil
    }

    // Fetch from public API and cache
    profile, err := publicClient.GetProfile(ctx, did)
    if err != nil {
        return nil, err
    }

    idx.cacheProfile(did, profile, 1*time.Hour)
    return profile, nil
}
```

**Option B: Slingshot Integration (Phase 2)**

```go
// Use Slingshot's resolveMiniDoc for faster profile resolution
func (idx *FeedIndex) resolveProfileViaSlingshot(ctx context.Context, did string) (*Profile, error) {
    url := fmt.Sprintf("https://slingshot.microcosm.blue/xrpc/com.bad-example.identity.resolveMiniDoc?identifier=%s", did)
    // Returns {did, handle, pds} in one call
}
```

### 5. Reference Resolution

Brews reference beans, grinders, and brewers. The index already has these records:

```go
func (idx *FeedIndex) resolveBrew(ctx context.Context, brew *IndexedRecord) (*FeedItem, error) {
    var record map[string]interface{}
    json.Unmarshal(brew.Record, &record)

    item := &FeedItem{RecordType: "brew"}

    // Resolve bean reference from local index
    if beanRef, ok := record["beanRef"].(string); ok {
        if bean := idx.getRecord(beanRef); bean != nil {
            item.Bean = recordToBean(bean)
        }
    }

    // Similar for grinder, brewer references
    // All from local index - no API calls

    return item, nil
}
```

### 6. Fallback and Resilience

```go
// internal/firehose/consumer.go

func (c *Consumer) Run(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            if err := c.connectAndConsume(ctx); err != nil {
                log.Warn().Err(err).Msg("jetstream connection lost, reconnecting...")

                // Exponential backoff
                time.Sleep(c.backoff.NextBackOff())

                // Rotate to next endpoint
                c.rotateEndpoint()
                continue
            }
        }
    }
}

func (c *Consumer) connectAndConsume(ctx context.Context) error {
    cursor := c.loadCursor()

    // Rewind cursor slightly to handle duplicates safely
    if cursor > 0 {
        cursor -= 5 * time.Second.Microseconds()
    }

    return c.client.ConnectAndRead(ctx, &cursor)
}
```

### 7. Feed Service Integration

```go
// internal/feed/service.go (modified)

type Service struct {
    registry     *Registry
    publicClient *atproto.PublicClient
    cache        *publicFeedCache

    // New: firehose index
    firehoseIndex *firehose.FeedIndex
    useFirehose   bool
}

func (s *Service) GetRecentRecords(ctx context.Context, limit int) ([]*FeedItem, error) {
    // Prefer firehose index if available and populated
    if s.useFirehose && s.firehoseIndex.IsReady() {
        return s.firehoseIndex.GetRecentFeed(ctx, limit)
    }

    // Fallback to polling (existing code)
    return s.getRecentRecordsViaPolling(ctx, limit)
}
```

---

## Implementation Phases

### Phase 1: Core Jetstream Consumer (2 weeks)

**Goal:** Replace polling with firehose consumption for the feed.

**Tasks:**

1. Create `internal/firehose/` package
   - `config.go` - Jetstream configuration
   - `consumer.go` - WebSocket consumer with reconnection
   - `index.go` - BoltDB-backed feed index
   - `scheduler.go` - Event processing scheduler

2. Integrate with existing feed service
   - Add feature flag: `ARABICA_USE_FIREHOSE=true` (just use a cli flag)
   - Keep polling as fallback

3. Handle profile resolution
   - Cache profiles locally with 1-hour TTL
   - Lazy fetch on first access
   - Background refresh for active users

4. Cursor management
   - Persist cursor to survive restarts
   - Rewind on reconnection for safety

**Deliverables:**

- Real-time feed updates
- Reduced API calls to near-zero
- Automatic user discovery (anyone using Arabica lexicons)

### Phase 2: Slingshot Optimization (1 week)

**Goal:** Faster profile and record hydration.

**Tasks:**

1. Add Slingshot client (`internal/atproto/slingshot.go`)
2. Use `resolveMiniDoc` for profile resolution
3. Use Slingshot as fallback for missing records

**Deliverables:**

- Faster profile loading
- Resilience to slow PDS endpoints

### Phase 3: Constellation for Social (1 week)

**Goal:** Enable like/comment counts when social features are added.

**Tasks:**

1. Add Constellation client (`internal/atproto/constellation.go`)
2. Query backlinks for interaction counts
3. Display counts on feed items

**Deliverables:**

- Like count on brews
- Comment count on brews
- Foundation for social features

### Phase 4: Spacedust for Real-time Notifications (Future)

**Goal:** Push notifications for interactions.

**Tasks:**

1. Subscribe to Spacedust for user's content interactions
2. Build notification storage and API
3. WebSocket to frontend for live updates

---

## Data Flow Comparison

### Before (Polling)

```
User Request → Check Cache → [Cache Miss] → Poll N PDSes → Build Feed → Return
                                  ↓
                           ~10N API calls
                           5-10 second latency
```

### After (Jetstream)

```
Jetstream → Consumer → Index (BoltDB)
                            ↓
User Request → Query Index → Return
                    ↓
              0 API calls
              <10ms latency
```

---

## Automatic User Discovery

A major benefit of firehose consumption is automatic user discovery:

**Current:** Users must explicitly register via `/api/feed/register`

**With Jetstream:** Any user who creates an Arabica record is automatically indexed

```go
// When we see a new DID creating Arabica records
func (c *Consumer) handleNewUser(did string) {
    // Auto-register for feed
    c.registry.Register(did)

    // Fetch and cache their profile
    go c.index.fetchAndCacheProfile(did)

    // Backfill their existing records
    go c.backfillUser(did)
}
```

This could replace the manual registry entirely, or supplement it for "featured" users.

---

## Backfill Strategy

When starting fresh or discovering a new user, we need historical data:

**Option A: Direct PDS Fetch (Simple)**

```go
func (c *Consumer) backfillUser(ctx context.Context, did string) error {
    for _, collection := range arabicaCollections {
        records, _ := publicClient.ListRecords(ctx, did, collection, 100)
        for _, record := range records {
            c.index.UpsertFromPDS(record)
        }
    }
    return nil
}
```

**Option B: Slingshot Fetch (Faster)**

```go
func (c *Consumer) backfillUserViaSlingshot(ctx context.Context, did string) error {
    // Single endpoint, pre-cached records
    // Same API as PDS but faster
}
```

**Option C: Jetstream Cursor Rewind (Events Only)**

- Rewind cursor to desired point in time
- Replay events (no records available before cursor)
- Limited to ~24h of history typically

**Recommendation:** Use Option A for Phase 1, add Option B in Phase 2.

---

## Configuration

```bash
# Environment variables

# Enable firehose-based feed (default: false during rollout)
ARABICA_USE_FIREHOSE=true

# Jetstream endpoint (default: public Bluesky instances)
JETSTREAM_URL=wss://jetstream1.us-east.bsky.network/subscribe

# Optional: self-hosted Jetstream
# JETSTREAM_URL=ws://localhost:6008/subscribe

# Feed index database path
ARABICA_FEED_INDEX_PATH=~/.local/share/arabica/feed-index.db

# Profile cache TTL (default: 1h)
ARABICA_PROFILE_CACHE_TTL=1h

# Optional: Slingshot endpoint for Phase 2
# SLINGSHOT_URL=https://slingshot.microcosm.blue

# Optional: Constellation endpoint for Phase 3
# CONSTELLATION_URL=https://constellation.microcosm.blue
```

---

## Monitoring and Metrics

```go
// Prometheus metrics to track firehose health

var (
    eventsReceived = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "arabica_firehose_events_total",
            Help: "Total events received from Jetstream",
        },
        []string{"collection", "operation"},
    )

    indexSize = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "arabica_feed_index_records",
            Help: "Number of records in feed index",
        },
    )

    consumerLag = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "arabica_firehose_lag_seconds",
            Help: "Lag between event time and processing time",
        },
    )

    connectionState = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "arabica_firehose_connected",
            Help: "1 if connected to Jetstream, 0 otherwise",
        },
    )
)
```

---

## Risk Assessment

| Risk                    | Mitigation                                    |
| ----------------------- | --------------------------------------------- |
| Jetstream unavailable   | Fallback to polling, rotate endpoints         |
| Index corruption        | Rebuild from backfill, periodic snapshots     |
| Duplicate events        | Idempotent upserts using AT-URI as key        |
| Missing historical data | Backfill on startup and new user discovery    |
| High event volume       | Filter to Arabica collections only (~0 noise) |
| Profile resolution lag  | Local cache with background refresh           |

---

## Open Questions

1. **Should we remove the registry entirely?**
   - Pro: Simpler, automatic discovery
   - Con: Lose ability to curate "featured" users
   - Recommendation: Keep registry for admin features, but don't require it for feed inclusion

2. **Self-host Jetstream or use public?**
   - Public is free and reliable
   - Self-host gives control and removes dependency
   - Recommendation: Start with public, evaluate self-hosting if issues arise

3. **How long to keep historical data?**
   - Option: Rolling 30-day window
   - Option: Keep everything (disk is cheap)
   - Recommendation: Keep 90 days, prune older records

4. **Real-time feed updates to frontend?**
   - Could push new items via WebSocket/SSE
   - Or just reduce cache TTL to ~30 seconds
   - Recommendation: Phase 1 just reduces staleness; real-time push is future enhancement

---

## Alternatives Considered

### 1. Tap (Bluesky's Full Sync Tool)

**Pros:** Full verification, automatic backfill, collection signal mode
**Cons:** Heavy operational overhead, overkill for current scale
**Verdict:** Revisit when user base exceeds 500+

### 2. Direct Firehose Consumption

**Pros:** No Jetstream dependency
**Cons:** Complex CBOR/CAR parsing, high bandwidth
**Verdict:** Jetstream provides the simplicity we need

### 3. Slingshot as Primary Data Source

**Pros:** Pre-cached records, single endpoint
**Cons:** Still polling-based, no real-time
**Verdict:** Use as optimization layer, not primary

### 4. Spacedust Instead of Jetstream

**Pros:** Link-focused, lightweight
**Cons:** Only links, no full records
**Verdict:** Use for notifications, not feed content

---

## Success Criteria

| Metric                     | Target                  |
| -------------------------- | ----------------------- |
| Feed latency               | <100ms (from >5s)       |
| API calls per feed request | 0 (from ~10N)           |
| Time to see new content    | <5s (from 5min)         |
| Feed availability          | 99.9% (with fallback)   |
| New user discovery         | Automatic (from manual) |

---

## References

- [Jetstream GitHub](https://github.com/bluesky-social/jetstream)
- [Jetstream Blog Post](https://docs.bsky.app/blog/jetstream)
- [Jetstream Go Client](https://pkg.go.dev/github.com/bluesky-social/jetstream/pkg/client)
- [Microcosm.blue Services](https://microcosm.blue/)
- [Constellation API](https://constellation.microcosm.blue/)
- [Slingshot API](https://slingshot.microcosm.blue/)
- [Existing Evaluation: Jetstream/Tap](./jetstream-tap-evaluation.md)
- [Existing Evaluation: Microcosm Tools](./microcosm-tools-evaluation.md)
