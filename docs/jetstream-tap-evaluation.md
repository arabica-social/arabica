# Jetstream and Tap Evaluation for Arabica

## Executive Summary

This document evaluates two AT Protocol synchronization tools - **Jetstream** and **Tap** - for potential integration with Arabica. These tools could help reduce API requests for the community feed feature and simplify real-time data synchronization.

**Recommendation:** Consider Jetstream for community feed improvements in the near term; Tap is overkill for Arabica's current scale but valuable for future growth.

---

## Background: Current Arabica Architecture

Arabica currently interacts with AT Protocol in two ways:

1. **Authenticated User Operations** (`internal/atproto/store.go`)
   - Direct XRPC calls to user's PDS for CRUD operations
   - Per-session in-memory cache (5-minute TTL)
   - Each user's data stored in their own PDS

2. **Community Feed** (`internal/feed/service.go`)
   - Polls registered users' PDSes to aggregate recent activity
   - Fetches profiles, brews, beans, roasters, grinders, brewers from each user
   - Public feed cached for 5 minutes
   - **Problem:** N+1 query pattern - each registered user requires multiple API calls

### Current Feed Inefficiency

For N registered users, the feed service makes approximately:
- N profile fetches
- N x 5 collection fetches (brew, bean, roaster, grinder, brewer) for recent items
- N x 4 collection fetches for reference resolution
- **Total: ~10N API calls per feed refresh**

---

## Tool 1: Jetstream

### What It Is

Jetstream is a streaming service that consumes the AT Protocol firehose (`com.atproto.sync.subscribeRepos`) and converts it into lightweight JSON events. It's operated by Bluesky at public endpoints.

**Public Instances:**
- `jetstream1.us-east.bsky.network`
- `jetstream2.us-east.bsky.network`
- `jetstream1.us-west.bsky.network`
- `jetstream2.us-west.bsky.network`

### Key Features

| Feature | Description |
|---------|-------------|
| JSON Output | Simple JSON instead of CBOR/CAR binary encoding |
| Filtering | Filter by collection (NSID) or repo (DID) |
| Compression | ~56% smaller messages with zstd compression |
| Low Latency | Real-time event delivery |
| Easy to Use | Standard WebSocket connection |

### Jetstream Event Example

```json
{
  "did": "did:plc:eygmaihciaxprqvxpfvl6flk",
  "time_us": 1725911162329308,
  "kind": "commit",
  "commit": {
    "rev": "3l3qo2vutsw2b",
    "operation": "create",
    "collection": "social.arabica.alpha.brew",
    "rkey": "3l3qo2vuowo2b",
    "record": {
      "$type": "social.arabica.alpha.brew",
      "method": "pourover",
      "rating": 4,
      "createdAt": "2024-09-09T19:46:02.102Z"
    },
    "cid": "bafyreidwaivazkwu67xztlmuobx35hs2lnfh3kolmgfmucldvhd3sgzcqi"
  }
}
```

### How Arabica Could Use Jetstream

**Use Case: Real-time Community Feed**

Instead of polling each user's PDS every 5 minutes, Arabica could:

1. Subscribe to Jetstream filtered by:
   - `wantedCollections`: `social.arabica.alpha.*`
   - `wantedDids`: List of registered feed users

2. Maintain a local feed index updated in real-time

3. Serve feed directly from local index (instant response, no API calls)

**Implementation Sketch:**

```go
// Subscribe to Jetstream for Arabica collections
ws, _ := websocket.Dial("wss://jetstream1.us-east.bsky.network/subscribe?" +
    "wantedCollections=social.arabica.alpha.brew&" +
    "wantedCollections=social.arabica.alpha.bean&" +
    "wantedDids=" + strings.Join(registeredDids, "&wantedDids="))

// Process events in background goroutine
for {
    var event JetstreamEvent
    ws.ReadJSON(&event)
    
    switch event.Commit.Collection {
    case "social.arabica.alpha.brew":
        feedIndex.AddBrew(event.DID, event.Commit.Record)
    case "social.arabica.alpha.bean":
        feedIndex.AddBean(event.DID, event.Commit.Record)
    }
}
```

### Jetstream Tradeoffs

| Pros | Cons |
|------|------|
| Dramatically reduces API calls | No cryptographic verification of data |
| Real-time updates (sub-second latency) | Requires persistent WebSocket connection |
| Simple JSON format | Trust relationship with Jetstream operator |
| Can filter by collection/DID | Not part of formal AT Protocol spec |
| Free public instances available | No built-in backfill mechanism |

### Jetstream Verdict for Arabica

**Recommended for:** Community feed real-time updates

**Not suitable for:** Authenticated user operations (those need direct PDS calls)

**Effort estimate:** Medium (1-2 weeks)
- Add WebSocket client for Jetstream
- Build local feed index (could use BoltDB or in-memory)
- Handle reconnection/cursor management
- Still need initial backfill via direct API

---

## Tool 2: Tap

### What It Is

Tap is a synchronization tool for AT Protocol that handles the complexity of repo synchronization. It subscribes to a Relay and outputs filtered, verified events. Tap is more comprehensive than Jetstream but requires running your own instance.

**Repository:** `github.com/bluesky-social/indigo/cmd/tap`

### Key Features

| Feature | Description |
|---------|-------------|
| Automatic Backfill | Fetches complete history when tracking new repos |
| Verification | MST integrity checks, signature validation |
| Recovery | Auto-resyncs if repo becomes desynchronized |
| Flexible Delivery | WebSocket, fire-and-forget, or webhooks |
| Filtered Output | DID and collection filtering |

### Tap Operating Modes

1. **Dynamic (default):** Add DIDs via API as needed
2. **Collection Signal:** Auto-track repos with records in specified collection
3. **Full Network:** Mirror entire AT Protocol network (resource-intensive)

### How Arabica Could Use Tap

**Use Case: Complete Feed Infrastructure**

Tap could replace the entire feed polling mechanism:

1. Run Tap instance with `TAP_SIGNAL_COLLECTION=social.arabica.alpha.brew`
2. Tap automatically discovers and tracks users who create brew records
3. Feed service consumes events from local Tap instance
4. No manual user registration needed - Tap discovers users automatically

**Collection Signal Mode:**

```bash
# Start Tap to auto-track repos with Arabica records
TAP_SIGNAL_COLLECTION=social.arabica.alpha.brew \
  go run ./cmd/tap --disable-acks=true
```

**Webhook Delivery (Serverless-friendly):**

Tap can POST events to an HTTP endpoint, making it compatible with serverless architectures:

```bash
# Tap sends events to Arabica webhook
TAP_WEBHOOK_URL=https://arabica.example/api/feed-webhook \
  go run ./cmd/tap
```

### Tap Tradeoffs

| Pros | Cons |
|------|------|
| Automatic backfill when adding repos | Requires running your own service |
| Full cryptographic verification | More operational complexity |
| Handles cursor management | Resource requirements (DB, network) |
| Auto-discovers users via collection signal | Overkill for small user bases |
| Webhook support for serverless | Still in beta |

### Tap Verdict for Arabica

**Recommended for:** Future growth when feed has many users

**Not suitable for:** Current scale (< 100 registered users)

**Effort estimate:** High (2-4 weeks)
- Deploy and operate Tap service
- Integrate webhook or WebSocket consumer
- Migrate feed service to consume from Tap
- Handle Tap service reliability/monitoring

---

## Comparison Matrix

| Aspect | Current Polling | Jetstream | Tap |
|--------|----------------|-----------|-----|
| API Calls per Refresh | ~10N | 0 (after connection) | 0 (after backfill) |
| Latency | 5 min cache | Real-time | Real-time |
| Backfill | Full fetch each time | Manual | Automatic |
| Verification | Trusts PDS | Trusts Jetstream | Full verification |
| Operational Cost | None | None (public) | Run own service |
| Complexity | Low | Medium | High |
| User Discovery | Manual registry | Manual | Auto via collection |
| Recommended Scale | < 50 users | 50-1000 users | 1000+ users |

---

## Recommendation

### Short Term (Now - 6 months)

**Stick with current polling + caching approach**

Rationale:
- Current implementation works
- User base is small
- Polling N users with caching is acceptable

**Consider adding Jetstream for feed** if:
- Feed latency becomes user-visible issue
- Registered users exceed ~50
- API rate limiting becomes a problem

### Medium Term (6-12 months)

**Implement Jetstream integration**

1. Add background Jetstream consumer
2. Build local feed index (BoltDB or SQLite)
3. Serve feed from local index
4. Keep polling as fallback for backfill

### Long Term (12+ months)

**Evaluate Tap when:**
- User base exceeds 500+ registered users
- Want automatic user discovery
- Need cryptographic verification for social features (likes, comments)
- Building moderation/anti-abuse features

---

## Implementation Notes

### Jetstream Client Library

Bluesky provides a Go client library:

```go
import "github.com/bluesky-social/jetstream/pkg/client"
```

### Tap TypeScript Library

For frontend integration:

```typescript
import { TapClient } from '@atproto/tap';
```

### Connection Resilience

Both tools require handling:
- WebSocket reconnection
- Cursor persistence across restarts
- Backpressure when events arrive faster than processing

### Caching Integration

Can coexist with current `SessionCache`:
- Jetstream/Tap updates the local index
- Local index serves feed requests
- SessionCache continues for authenticated user operations

---

## Related Documentation

- Jetstream GitHub: https://github.com/bluesky-social/jetstream
- Tap README: https://github.com/bluesky-social/indigo/blob/main/cmd/tap/README.md
- Jetstream Blog Post: https://docs.bsky.app/blog/jetstream
- Tap Blog Post: https://docs.bsky.app/blog/introducing-tap

---

## Note on "Constellation" and "Slingshot"

These terms don't appear to correspond to official AT Protocol tools as of this evaluation. If these refer to specific community projects or internal codenames, please provide additional context for evaluation.
