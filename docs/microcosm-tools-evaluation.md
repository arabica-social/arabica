# Microcosm Tools Evaluation for Arabica

## Executive Summary

This document evaluates three community-built AT Protocol infrastructure tools from [microcosm.blue](https://microcosm.blue/) - **Constellation**, **Spacedust**, and **Slingshot** - for potential integration with Arabica's community feed feature.

**Recommendation:** Adopt Constellation immediately for future social features (likes/comments). Consider Slingshot as an optional optimization for feed performance. Spacedust is ideal for real-time notifications when social features are implemented.

---

## Background: Current Arabica Architecture

### The Problem

Arabica's community feed (`internal/feed/service.go`) currently polls each registered user's PDS directly. For N registered users:

| API Call Type | Count per Refresh |
|---------------|-------------------|
| Profile fetches | N |
| Brew collections | N |
| Bean collections | N |
| Roaster collections | N |
| Grinder collections | N |
| Brewer collections | N |
| Reference resolution | ~4N |
| **Total** | **~10N API calls** |

This approach has several issues:
- **Latency**: Feed refresh is slow with many users
- **Rate limits**: Risk of PDS rate limiting
- **Reliability**: Feed fails if any PDS is slow/down
- **Scalability**: Linear growth in API calls per user

### Future Social Features

Arabica plans to add likes, comments, and follows (see `AGENTS.md`). These interactions require **backlink queries** - given a brew, find all likes pointing at it. This is impossible with current polling approach.

---

## Tool 1: Constellation (Backlink Index)

### What It Is

Constellation is a **global backlink index** that crawls every record in the AT Protocol firehose and indexes all links (AT-URIs, DIDs, URLs). It answers "who/what points at this target?" queries.

**Public Instance:** `https://constellation.microcosm.blue`

### Key Capabilities

| Feature | Description |
|---------|-------------|
| Backlink queries | Find all records linking to a target |
| Like/follow counts | Get interaction counts instantly |
| Any lexicon support | Works with `social.arabica.alpha.*` |
| DID filtering | Filter links by specific users |
| Distinct DID counts | Count unique users, not just records |

### API Examples

**Get like count for a brew:**
```bash
curl "https://constellation.microcosm.blue/links/count/distinct-dids" \
  -G --data-urlencode "target=at://did:plc:xxx/social.arabica.alpha.brew/abc123" \
  --data-urlencode "collection=social.arabica.alpha.like" \
  --data-urlencode "path=.subject.uri"
```

**Get all users who liked a brew:**
```bash
curl "https://constellation.microcosm.blue/links/distinct-dids" \
  -G --data-urlencode "target=at://did:plc:xxx/social.arabica.alpha.brew/abc123" \
  --data-urlencode "collection=social.arabica.alpha.like" \
  --data-urlencode "path=.subject.uri"
```

**Get all comments on a brew:**
```bash
curl "https://constellation.microcosm.blue/links" \
  -G --data-urlencode "target=at://did:plc:xxx/social.arabica.alpha.brew/abc123" \
  --data-urlencode "collection=social.arabica.alpha.comment" \
  --data-urlencode "path=.subject.uri"
```

### How Arabica Could Use Constellation

**Use Case 1: Social Interaction Counts**

When displaying a brew in the feed, fetch interaction counts:

```go
// Get like count for a brew
func (c *ConstellationClient) GetLikeCount(ctx context.Context, brewURI string) (int, error) {
    url := fmt.Sprintf("%s/links/count/distinct-dids?target=%s&collection=%s&path=%s",
        c.baseURL,
        url.QueryEscape(brewURI),
        "social.arabica.alpha.like",
        url.QueryEscape(".subject.uri"))
    
    // Returns {"total": 42}
    var result struct { Total int `json:"total"` }
    // ... fetch and decode
    return result.Total, nil
}
```

**Use Case 2: Comment Threads**

Fetch all comments for a brew detail page:

```go
func (c *ConstellationClient) GetComments(ctx context.Context, brewURI string) ([]Comment, error) {
    // Constellation returns the AT-URIs of comment records
    // Then fetch each comment from Slingshot or user's PDS
}
```

**Use Case 3: "Who liked this" List**

```go
func (c *ConstellationClient) GetLikers(ctx context.Context, brewURI string) ([]string, error) {
    // Returns list of DIDs who liked this brew
    // Can hydrate with profile info from Slingshot
}
```

### Constellation Tradeoffs

| Pros | Cons |
|------|------|
| Instant interaction counts (no polling) | Third-party dependency |
| Works with any lexicon including Arabica's | Not self-hosted (yet) |
| Handles likes from any PDS globally | Slight index delay (~seconds) |
| 11B+ links indexed, production-ready | Trusts Constellation operator |
| Free public instance | Query limits may apply |

### Constellation Verdict

**Essential for:** Social features (likes, comments, follows)

**Not needed for:** Current feed polling (Constellation indexes interactions, not record listings)

**Effort estimate:** Low (1 week)
- Add HTTP client for Constellation API
- Integrate counts into brew display
- Cache counts locally (5-minute TTL)

---

## Tool 2: Spacedust (Interactions Firehose)

### What It Is

Spacedust extracts **links** from every record in the AT Protocol firehose and re-emits them over WebSocket. Unlike Jetstream (which emits full records), Spacedust emits just the link relationships.

**Public Instance:** `wss://spacedust.microcosm.blue`

### Key Capabilities

| Feature | Description |
|---------|-------------|
| Real-time link events | Instantly know when someone likes/follows |
| Filter by source/target | Subscribe to specific collections or targets |
| Any lexicon support | Works with `social.arabica.alpha.*` |
| Lightweight | Just links, not full records |

### Example: Subscribe to Likes on Your Brews

```javascript
// WebSocket connection to Spacedust
const ws = new WebSocket(
  "wss://spacedust.microcosm.blue/subscribe" +
  "?wantedSources=social.arabica.alpha.like:subject.uri" +
  "&wantedSubjects=did:plc:your-did"
);

ws.onmessage = (event) => {
  const link = JSON.parse(event.data);
  // { source: "at://...", target: "at://...", ... }
  console.log("Someone liked your brew!");
};
```

### How Arabica Could Use Spacedust

**Use Case: Real-time Notifications**

When social features are added, Spacedust enables instant notifications:

```go
// Background goroutine subscribes to Spacedust
func (s *NotificationService) subscribeToInteractions(userDID string) {
    ws := dial("wss://spacedust.microcosm.blue/subscribe" +
        "?wantedSources=social.arabica.alpha.like:subject.uri" +
        "&wantedSubjects=" + userDID)
    
    for {
        link := readLink(ws)
        // Someone liked a brew by userDID
        s.notify(userDID, "Someone liked your brew!")
    }
}
```

**Use Case: Live Feed Updates**

Push new brews to connected clients without polling:

```go
// Subscribe to all Arabica brew creations
ws := dial("wss://spacedust.microcosm.blue/subscribe" +
    "?wantedSources=social.arabica.alpha.brew:beanRef")

// When a link event arrives, a new brew was created
// Fetch full record from Slingshot and push to feed
```

### Spacedust Tradeoffs

| Pros | Cons |
|------|------|
| Real-time, sub-second latency | Requires persistent WebSocket |
| Lightweight link-only events | Still in v0 (missing some features) |
| Filter by collection/target | No cursor replay yet |
| Perfect for notifications | Need to hydrate records separately |

### Spacedust Verdict

**Ideal for:** Real-time notifications, live feed updates

**Not suitable for:** Current feed needs (need full records, not just links)

**Effort estimate:** Medium (2-3 weeks)
- WebSocket client with reconnection
- Notification service for social interactions
- Integration with frontend for live updates
- Depends on social features being implemented first

---

## Tool 3: Slingshot (Records & Identities Cache)

### What It Is

Slingshot is an **edge cache** for AT Protocol records and identities. It pre-caches records from the firehose and provides fast, authenticated access. Also resolves handles to DIDs with bi-directional verification.

**Public Instance:** `https://slingshot.microcosm.blue`

### Key Capabilities

| Feature | Description |
|---------|-------------|
| Fast record fetching | Pre-cached from firehose |
| Identity resolution | `resolveMiniDoc` for handle/DID |
| Bi-directional verification | Only returns verified handles |
| Works with slow PDS | Cache serves even if PDS is down |
| Standard XRPC API | Drop-in replacement for PDS calls |

### API Examples

**Resolve identity:**
```bash
curl "https://slingshot.microcosm.blue/xrpc/com.bad-example.identity.resolveMiniDoc?identifier=bad-example.com"
# Returns: { "did": "did:plc:...", "handle": "bad-example.com", "pds": "https://..." }
```

**Get record (standard XRPC):**
```bash
curl "https://slingshot.microcosm.blue/xrpc/com.atproto.repo.getRecord?repo=did:plc:xxx&collection=social.arabica.alpha.brew&rkey=abc123"
```

**List records:**
```bash
curl "https://slingshot.microcosm.blue/xrpc/com.atproto.repo.listRecords?repo=did:plc:xxx&collection=social.arabica.alpha.brew&limit=10"
```

### How Arabica Could Use Slingshot

**Use Case 1: Faster Feed Fetching**

Replace direct PDS calls with Slingshot for public data:

```go
// Before: Each user's PDS
pdsEndpoint, _ := c.GetPDSEndpoint(ctx, did)
url := fmt.Sprintf("%s/xrpc/com.atproto.repo.listRecords...", pdsEndpoint)

// After: Single Slingshot endpoint
url := fmt.Sprintf("https://slingshot.microcosm.blue/xrpc/com.atproto.repo.listRecords...")
```

**Benefits:**
- Eliminates N DNS lookups for N user PDS endpoints
- Single, fast endpoint for all public record fetches
- Continues working even if individual PDS is slow/down
- Pre-cached records = faster response times

**Use Case 2: Identity Resolution**

Replace multiple API calls with single `resolveMiniDoc`:

```go
// Before: Two calls
handle := resolveHandle(did)      // Call 1
pds := resolvePDSEndpoint(did)    // Call 2

// After: One call
mini := resolveMiniDoc(did)       
// { handle: "user.bsky.social", pds: "https://...", did: "did:plc:..." }
```

**Use Case 3: Hydrate Records from Constellation**

When Constellation returns AT-URIs (e.g., comments on a brew), fetch the actual records from Slingshot:

```go
// Constellation returns: ["at://did:plc:a/social.arabica.alpha.comment/123", ...]
commentURIs := constellation.GetComments(ctx, brewURI)

// Fetch each comment record from Slingshot
for _, uri := range commentURIs {
    record := slingshot.GetRecord(ctx, uri)
    // ...
}
```

### Implementation: Slingshot-Backed PublicClient

```go
// internal/atproto/slingshot_client.go

const SlingshotBaseURL = "https://slingshot.microcosm.blue"

type SlingshotClient struct {
    baseURL    string
    httpClient *http.Client
}

func NewSlingshotClient() *SlingshotClient {
    return &SlingshotClient{
        baseURL: SlingshotBaseURL,
        httpClient: &http.Client{Timeout: 10 * time.Second},
    }
}

// ListRecords uses Slingshot instead of user's PDS
func (c *SlingshotClient) ListRecords(ctx context.Context, did, collection string, limit int) (*PublicListRecordsOutput, error) {
    // Same XRPC API, different endpoint
    url := fmt.Sprintf("%s/xrpc/com.atproto.repo.listRecords?repo=%s&collection=%s&limit=%d&reverse=true",
        c.baseURL, url.QueryEscape(did), url.QueryEscape(collection), limit)
    // ... standard HTTP request
}

// ResolveMiniDoc gets handle + PDS in one call
func (c *SlingshotClient) ResolveMiniDoc(ctx context.Context, identifier string) (*MiniDoc, error) {
    url := fmt.Sprintf("%s/xrpc/com.bad-example.identity.resolveMiniDoc?identifier=%s",
        c.baseURL, url.QueryEscape(identifier))
    // ... returns { did, handle, pds }
}
```

### Slingshot Tradeoffs

| Pros | Cons |
|------|------|
| Faster than direct PDS calls | Third-party dependency |
| Single endpoint for all users | May not have custom lexicons cached |
| Identity verification built-in | Not all XRPC APIs implemented |
| Resilient to slow/down PDS | Trusts Slingshot operator |
| Pre-cached from firehose | Still in v0, some features missing |

### Slingshot Verdict

**Recommended for:** Feed performance optimization, identity resolution

**Not suitable for:** Authenticated user operations (still need direct PDS)

**Effort estimate:** Low (3-5 days)
- Add SlingshotClient as optional PublicClient backend
- Feature flag to toggle between direct PDS and Slingshot
- Test with Arabica collections to ensure they're indexed

---

## Comparison: Current vs. Microcosm Tools

| Aspect | Current Polling | + Slingshot | + Constellation | + Spacedust |
|--------|-----------------|-------------|-----------------|-------------|
| Feed refresh latency | Slow (N PDS calls) | Fast (1 endpoint) | N/A | Real-time |
| Like/comment counts | Impossible | Impossible | Instant | N/A |
| Rate limit risk | High | Low | Low | None |
| PDS failure resilience | Poor | Good | N/A | N/A |
| Real-time updates | No (5min cache) | No | No | Yes |
| Effort to integrate | N/A | Low | Low | Medium |

---

## Recommendation

### Immediate (Social Features Prerequisite)

**1. Integrate Constellation when adding likes/comments**

Constellation is essential for social features. When a brew is displayed, use Constellation to:
- Show like count
- Show comment count
- Power "who liked this" lists
- Power comment threads

**Implementation priority:** Do this alongside `social.arabica.alpha.like` and `social.arabica.alpha.comment` lexicon implementation.

### Short Term (Performance Optimization)

**2. Evaluate Slingshot for feed performance**

If feed latency becomes an issue:
- Add SlingshotClient as alternative to direct PDS calls
- A/B test performance improvement
- Use for public record fetches only (keep direct PDS for authenticated writes)

**Trigger:** When registered users exceed ~20-30, or feed refresh exceeds 5 seconds

### Medium Term (Real-time Features)

**3. Add Spacedust for notifications**

When social features are live and users want notifications:
- Subscribe to Spacedust for likes/comments on user's content
- Push notifications via WebSocket to connected clients
- Optional: background job for email notifications

**Trigger:** After social features launch, when users request notifications

---

## Comparison with Official Tools (Jetstream/Tap)

See `jetstream-tap-evaluation.md` for official Bluesky tools. Key differences:

| Aspect | Microcosm Tools | Official Tools |
|--------|-----------------|----------------|
| Focus | Links/interactions | Full records |
| Backlink queries | Constellation (yes) | Not available |
| Record caching | Slingshot | Not available |
| Real-time | Spacedust (links) | Jetstream (records) |
| Self-hosting | Not yet documented | Available |
| Community | Community-supported | Bluesky-supported |

**Recommendation:** Use Microcosm tools for social features (likes/comments/follows) where backlink queries are essential. Consider Jetstream for full feed real-time if needed later.

---

## Implementation Plan

### Phase 1: Constellation Integration (with social features)

```
1. Create internal/atproto/constellation.go
   - ConstellationClient with HTTP client
   - GetBacklinks(), GetLinkCount(), GetDistinctDIDs()
   
2. Create internal/social/interactions.go
   - GetBrewLikeCount(brewURI)
   - GetBrewComments(brewURI)
   - GetBrewLikers(brewURI)

3. Update templates to show interaction counts
   - Modify feed item display
   - Add like button (when like lexicon ready)
```

### Phase 2: Slingshot Optimization (optional)

```
1. Create internal/atproto/slingshot.go
   - SlingshotClient implementing same interface as PublicClient
   
2. Add feature flag: ARABICA_USE_SLINGSHOT=true
   
3. Modify feed/service.go to use SlingshotClient
   - Keep PublicClient as fallback
```

### Phase 3: Spacedust Notifications (future)

```
1. Create internal/notifications/spacedust.go
   - WebSocket client with reconnection
   - Subscribe to user's content interactions
   
2. Create notification storage (BoltDB)
   
3. Add /api/notifications endpoint for frontend polling
   
4. Optional: WebSocket to frontend for real-time
```

---

## Related Documentation

- Microcosm Main: https://microcosm.blue/
- Constellation API: https://constellation.microcosm.blue/
- Source Code: https://github.com/at-microcosm/microcosm-rs
- Discord: https://discord.gg/tcDfe4PGVB
- See also: `jetstream-tap-evaluation.md` for official Bluesky tools
