# Arabica Social Features Plan: Likes, Follows, and Comments

**Version:** 1.0  
**Date:** January 25, 2026  
**Status:** Planning

---

TODO:

- This is not going to be the current state, I don't love the plan claude made
- Likes will probably be their own lexicon (maybe with a lens to bsky likes? -- probably not)
- Comments tbd, I would like to avoid forcing users onto bsky for social features though
- Follows, allow importing social graph from bsky (might be able to use a sort of statndardized lexicon here?)
  - Likely creating a custom lexicon that is structurally similar/the same as bsky (maybe standard.site sub/pub lex if that would work?)

---

## Executive Summary

This document outlines the implementation plan for adding social features to Arabica: likes, follows, and comments. The plan leverages AT Protocol's decentralized architecture while evaluating strategic reuse of Bluesky's existing social lexicons versus creating Arabica-specific ones.

## Table of Contents

1. [Goals & Non-Goals](#goals--non-goals)
2. [Architecture Overview](#architecture-overview)
3. [Lexicon Design Decisions](#lexicon-design-decisions)
4. [Implementation Phases](#implementation-phases)
5. [Technical Details](#technical-details)
6. [Bluesky Integration Strategies](#bluesky-integration-strategies)
7. [Data Flow & Storage](#data-flow--storage)
8. [UI/UX Considerations](#uiux-considerations)
9. [Migration & Rollout](#migration--rollout)
10. [Future Enhancements](#future-enhancements)

---

## Goals & Non-Goals

### Goals

- **Enable likes** on brews, beans, roasters, grinders, and brewers
- **Support follows** to create personalized feeds of coffee enthusiasts
- **Add comments** to enable discussions around brews and equipment
- **Maintain decentralization**: Social interactions stored in users' own PDS
- **Leverage existing infrastructure**: Use Bluesky's lexicons where beneficial
- **Preserve portability**: Users can take their data anywhere
- **Enable discoverability**: Surface popular content and active users

### Non-Goals

- Building a full social network (messaging, DMs, notifications beyond basic)
- Implementing moderation tools (initial phase)
- Creating a mobile app (web-first approach)
- Supporting multimedia beyond existing image support

---

## Architecture Overview

### Current State

```
User's PDS
├── social.arabica.alpha.bean       (coffee beans)
├── social.arabica.alpha.roaster    (roasters)
├── social.arabica.alpha.grinder    (grinders)
├── social.arabica.alpha.brewer     (brewing devices)
└── social.arabica.alpha.brew       (brew sessions)

Arabica Server
├── Firehose Listener (crawls network for brew data)
├── Feed Index (BoltDB - aggregated feed)
├── Session Store (BoltDB - sessions/registry)
└── Profile Cache (in-memory, 1hr TTL)
```

### Proposed State

```
User's PDS
├── Arabica Records
│   ├── social.arabica.alpha.bean
│   ├── social.arabica.alpha.roaster
│   ├── social.arabica.alpha.grinder
│   ├── social.arabica.alpha.brewer
│   └── social.arabica.alpha.brew
│
├── Social Interactions (Option A: Arabica-specific)
│   ├── social.arabica.alpha.like
│   ├── social.arabica.alpha.follow
│   └── social.arabica.alpha.comment
│
└── Social Interactions (Option B: Bluesky lexicons)
    ├── app.bsky.feed.like          (reuse for likes)
    ├── app.bsky.graph.follow       (reuse for follows)
    └── social.arabica.alpha.comment (custom for comments)

Arabica Server
├── Firehose Listener (+ like/follow/comment indexing)
├── Social Index (BoltDB - likes, follows, comments)
├── Feed Index (enhanced with social signals)
├── Session Store
└── Profile Cache
```

---

## Lexicon Design Decisions

### Decision Matrix

| Feature      | Custom Lexicon                 | Bluesky Lexicon                | Recommendation    |
| ------------ | ------------------------------ | ------------------------------ | ----------------- |
| **Likes**    | `social.arabica.alpha.like`    | `app.bsky.feed.like`           | **Use Bluesky**   |
| **Follows**  | `social.arabica.alpha.follow`  | `app.bsky.graph.follow`        | **Use Bluesky**   |
| **Comments** | `social.arabica.alpha.comment` | `app.bsky.feed.post` (replies) | **Create Custom** |

### Rationale

#### ✅ Use `app.bsky.feed.like` for Likes

**Pros:**

- Simple, well-tested schema (just subject + timestamp)
- Enables cross-app discoverability (Bluesky users can see popular coffee content)
- No need to maintain our own lexicon
- Future compatibility with Bluesky social graph features
- Users' existing Bluesky likes are already in their PDS

**Cons:**

- Couples us to Bluesky's schema evolution
- Mixing Arabica and Bluesky content in like feeds

**Schema:**

```json
{
  "lexicon": 1,
  "id": "app.bsky.feed.like",
  "defs": {
    "main": {
      "type": "record",
      "key": "tid",
      "record": {
        "type": "object",
        "required": ["subject", "createdAt"],
        "properties": {
          "subject": {
            "type": "ref",
            "ref": "com.atproto.repo.strongRef",
            "description": "AT-URI + CID of the liked record"
          },
          "createdAt": {
            "type": "string",
            "format": "datetime"
          }
        }
      }
    }
  }
}
```

**Example Record:**

```json
{
  "$type": "app.bsky.feed.like",
  "subject": {
    "uri": "at://did:plc:user123/social.arabica.alpha.brew/abc123",
    "cid": "bafyreibjifzpqj6o6wcq3hejh7y4z4z2vmiklkvykc57tw3pcbx3kxifpm"
  },
  "createdAt": "2026-01-25T12:30:00.000Z"
}
```

#### ✅ Use `app.bsky.graph.follow` for Follows

**Pros:**

- Standard social graph representation
- Interoperability: Arabica follows visible in Bluesky social graph
- Enables "import follows from Bluesky" (see below)
- Could power recommendations ("Users who brew X also follow Y")
- Simplifies social graph queries

**Cons:**

- Follow graph will mix Arabica and Bluesky users
- Need to filter by context in queries

**Schema:**

```json
{
  "lexicon": 1,
  "id": "app.bsky.graph.follow",
  "defs": {
    "main": {
      "type": "record",
      "key": "tid",
      "record": {
        "type": "object",
        "required": ["subject", "createdAt"],
        "properties": {
          "subject": {
            "type": "string",
            "format": "did",
            "description": "DID of the user being followed"
          },
          "createdAt": {
            "type": "string",
            "format": "datetime"
          }
        }
      }
    }
  }
}
```

**Example Record:**

```json
{
  "$type": "app.bsky.graph.follow",
  "subject": "did:plc:coffee-enthusiast-456",
  "createdAt": "2026-01-25T12:30:00.000Z"
}
```

#### ✅ Create `social.arabica.alpha.comment` for Comments

**Pros:**

- Coffee-specific comment features (e.g., ratings, tasting notes)
- Can extend with Arabica-specific fields
- Cleaner separation from Bluesky post threads
- No confusion between "replies" and "comments"

**Cons:**

- Maintains another lexicon
- Comments won't appear in Bluesky's thread views
- Need to build our own comment threading

**Proposed Schema:**

```json
{
  "lexicon": 1,
  "id": "social.arabica.alpha.comment",
  "defs": {
    "main": {
      "type": "record",
      "key": "tid",
      "description": "A comment on a brew or equipment",
      "record": {
        "type": "object",
        "required": ["subject", "text", "createdAt"],
        "properties": {
          "subject": {
            "type": "ref",
            "ref": "com.atproto.repo.strongRef",
            "description": "The brew/bean/roaster/etc being commented on"
          },
          "text": {
            "type": "string",
            "maxLength": 2000,
            "maxGraphemes": 500,
            "description": "Comment text"
          },
          "parent": {
            "type": "ref",
            "ref": "com.atproto.repo.strongRef",
            "description": "Parent comment for threading (optional)"
          },
          "facets": {
            "type": "array",
            "description": "Mentions, links, hashtags",
            "items": {
              "type": "ref",
              "ref": "app.bsky.richtext.facet"
            }
          },
          "rating": {
            "type": "integer",
            "minimum": 1,
            "maximum": 10,
            "description": "Optional rating (1-10)"
          },
          "createdAt": {
            "type": "string",
            "format": "datetime"
          }
        }
      }
    }
  }
}
```

**Example Record:**

```json
{
  "$type": "social.arabica.alpha.comment",
  "subject": {
    "uri": "at://did:plc:user123/social.arabica.alpha.brew/xyz789",
    "cid": "bafyreig2fjxi3rptqdgylg7e5hmjl6mcke7rn2b6cugzlqq3i4zu6rq52q"
  },
  "text": "Lovely floral notes! What was your water temp?",
  "rating": 8,
  "createdAt": "2026-01-25T14:00:00.000Z"
}
```

---

## Implementation Phases

### Phase 1: Likes (2-3 weeks)

**Deliverables:**

1. ✅ Lexicon decision: Use `app.bsky.feed.like`
2. Backend: Index likes from firehose
3. Backend: Aggregate like counts per record
4. Backend: API endpoints for liking/unliking
5. Frontend: Like button UI on brew cards
6. Frontend: Display like counts
7. Testing: Snapshot tests for like endpoints

**Technical Tasks:**

- Update firehose listener to capture `app.bsky.feed.like` records
- Add `LikesIndex` to BoltDB (keyed by subject AT-URI)
- Implement `GetLikeCount(uri string)` function
- Implement `UserHasLiked(userDID, uri string)` function
- Create/delete like via PDS client
- Frontend: Like button component with optimistic updates

**Database Schema (BoltDB):**

```
Bucket: Likes
Key: <subject-at-uri>
Value: {
  "count": 42,
  "recent": ["did:plc:user1", "did:plc:user2", ...] // last 10 likers
}

Bucket: UserLikes
Key: <user-did>/<subject-at-uri>
Value: <like-record-uri>  // for quick "has user liked this?" checks
```

### Phase 2: Follows (3-4 weeks)

**Deliverables:**

1. ✅ Lexicon decision: Use `app.bsky.graph.follow`
2. Backend: Index follows from firehose
3. Backend: Build follower/following graph
4. Backend: Personalized feed based on follows
5. Frontend: Follow button on user profiles
6. Frontend: Followers/Following pages
7. Feature: Import follows from Bluesky (see below)

**Technical Tasks:**

- Update firehose listener to capture `app.bsky.graph.follow` records
- Add `FollowsIndex` to BoltDB
- Implement `GetFollowers(did string)` function
- Implement `GetFollowing(did string)` function
- Implement `UserFollows(followerDID, followedDID string)` function
- Create/delete follow via PDS client
- Frontend: Follow button component
- Frontend: "Following" feed filter

**Database Schema (BoltDB):**

```
Bucket: Follows
Key: follower:<did>
Value: ["did:plc:followed1", "did:plc:followed2", ...]

Bucket: Followers
Key: followed:<did>
Value: ["did:plc:follower1", "did:plc:follower2", ...]

Bucket: FollowCounts
Key: <did>
Value: {
  "followers": 120,
  "following": 87
}
```

### Phase 3: Comments (4-5 weeks)

**Deliverables:**

1. ✅ Lexicon: Create `social.arabica.alpha.comment`
2. Backend: Index comments from firehose
3. Backend: Comment threading logic
4. Backend: Comment counts per record
5. Frontend: Comment display UI
6. Frontend: Comment creation form
7. Frontend: Comment threading/replies

**Technical Tasks:**

- Define and publish `social.arabica.alpha.comment` lexicon
- Update firehose listener to capture comment records
- Add `CommentsIndex` to BoltDB
- Implement `GetComments(uri string, limit, offset int)` function
- Implement comment threading/tree building
- Create comment via PDS client
- Frontend: Comment list component
- Frontend: Comment form with mentions/facets support

**Database Schema (BoltDB):**

```
Bucket: Comments
Key: <subject-at-uri>/<timestamp>
Value: {
  "author": "did:plc:user1",
  "text": "Great brew!",
  "parent": "at://...",  // null for top-level
  "rating": 9,
  "createdAt": "2026-01-25T12:00:00Z",
  "uri": "at://did:plc:user1/social.arabica.alpha.comment/abc123"
}

Bucket: CommentCounts
Key: <subject-at-uri>
Value: 15
```

### Phase 4: Social Feed Enhancements (2-3 weeks)

**Deliverables:**

1. Following-only feed view
2. Popular brews (by like count)
3. Trending equipment
4. Active users widget
5. Social notifications (basic)

---

## Technical Details

### 1. Firehose Integration

**Current:**

- Listens for `social.arabica.alpha.*` records
- Indexes brews, beans, roasters, grinders, brewers

**Enhanced:**

```go
// internal/firehose/listener.go

func (l *Listener) handleFirehoseEvent(evt *events.RepoCommit) {
    for _, op := range evt.Ops {
        switch op.Collection {
        // Existing collections
        case atproto.NSIDBrew, atproto.NSIDBean, atproto.NSIDRoaster,
             atproto.NSIDGrinder, atproto.NSIDBrewer:
            l.handleArabicaRecord(op)

        // Social interactions
        case "app.bsky.feed.like":
            l.handleLike(op)
        case "app.bsky.graph.follow":
            l.handleFollow(op)
        case atproto.NSIDComment: // social.arabica.alpha.comment
            l.handleComment(op)
        }
    }
}

func (l *Listener) handleLike(op *events.RepoOp) error {
    // Parse like record
    var like atproto.Like
    if err := json.Unmarshal(op.Record, &like); err != nil {
        return err
    }

    // Filter: only index likes on Arabica content
    if !strings.HasPrefix(like.Subject.URI, "at://") {
        return nil
    }
    components := atproto.ParseATURI(like.Subject.URI)
    if !strings.HasPrefix(components.Collection, "social.arabica.alpha.") {
        return nil // Skip non-Arabica likes
    }

    // Index the like
    return l.socialIndex.IndexLike(op.Author, &like)
}
```

### 2. API Endpoints

**New endpoints:**

```
POST   /api/likes                    # Create a like
DELETE /api/likes                    # Unlike
GET    /api/likes?uri=<record-uri>   # Get like count & likers

POST   /api/follows                  # Follow a user
DELETE /api/follows                  # Unfollow
GET    /api/followers?did=<did>      # Get followers
GET    /api/following?did=<did>      # Get following list
POST   /api/import-follows           # Import from Bluesky

POST   /api/comments                 # Create a comment
GET    /api/comments?uri=<uri>       # Get comments for a record
```

**Example: Like endpoint**

```go
// internal/handlers/likes.go

type LikeRequest struct {
    SubjectURI string `json:"uri"`
    SubjectCID string `json:"cid"`
}

func (h *Handlers) CreateLike(w http.ResponseWriter, r *http.Request) {
    store, authenticated := h.getAtprotoStore(r)
    if !authenticated {
        http.Error(w, "Authentication required", http.StatusUnauthorized)
        return
    }

    var req LikeRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    // Create like record in user's PDS
    like := &atproto.Like{
        Type: "app.bsky.feed.like",
        Subject: &atproto.StrongRef{
            URI: req.SubjectURI,
            CID: req.SubjectCID,
        },
        CreatedAt: time.Now().Format(time.RFC3339),
    }

    uri, err := store.CreateLike(r.Context(), like)
    if err != nil {
        http.Error(w, "Failed to create like", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(map[string]string{
        "uri": uri,
    })
}
```

### 3. Store Interface Extensions

```go
// internal/database/store.go

type Store interface {
    // Existing methods...

    // Likes
    CreateLike(ctx context.Context, like *atproto.Like) (string, error)
    DeleteLike(ctx context.Context, likeURI string) error
    GetLikeCount(ctx context.Context, subjectURI string) (int, error)
    GetLikers(ctx context.Context, subjectURI string, limit int) ([]*Profile, error)
    UserHasLiked(ctx context.Context, userDID, subjectURI string) (bool, error)

    // Follows
    CreateFollow(ctx context.Context, follow *atproto.Follow) (string, error)
    DeleteFollow(ctx context.Context, followURI string) error
    GetFollowers(ctx context.Context, did string, limit, offset int) ([]*Profile, error)
    GetFollowing(ctx context.Context, did string, limit, offset int) ([]*Profile, error)
    UserFollows(ctx context.Context, followerDID, followedDID string) (bool, error)

    // Comments
    CreateComment(ctx context.Context, comment *atproto.Comment) (string, error)
    GetComments(ctx context.Context, subjectURI string, limit, offset int) ([]*Comment, error)
    GetCommentCount(ctx context.Context, subjectURI string) (int, error)
}
```

---

## Bluesky Integration Strategies

### Import Follows from Bluesky

**User Story:**  
"As a coffee enthusiast on Bluesky, I want to import my Bluesky follows into Arabica so I can follow coffee friends without re-discovering them."

**Implementation:**

1. **Fetch Bluesky Follows**
   - Use `app.bsky.graph.getFollows` API
   - Query user's PDS: `GET /xrpc/app.bsky.graph.getFollows?actor={userDID}`
   - Paginate through results (cursor-based)

2. **Filter for Arabica Users**
   - Check if followed user has Arabica records
   - Query: `listRecords` for `social.arabica.alpha.brew` in their PDS
   - Cache results to avoid repeated lookups

3. **Create Follow Records**
   - For each Arabica user in Bluesky follows, create `app.bsky.graph.follow` in user's PDS
   - Skip if already following

**API Endpoint:**

```go
POST /api/import-follows

Request:
{
  "source": "bluesky",
  "filter": "arabica-users-only"  // or "all"
}

Response:
{
  "imported": 42,
  "skipped": 8,
  "failed": 1,
  "details": [
    {"did": "did:plc:user1", "handle": "@coffee-nerd.bsky.social", "status": "imported"},
    {"did": "did:plc:user2", "handle": "@bean-expert.bsky.social", "status": "already-following"}
  ]
}
```

**Implementation:**

```go
func (h *Handlers) ImportFollows(w http.ResponseWriter, r *http.Request) {
    store, authenticated := h.getAtprotoStore(r)
    if !authenticated {
        http.Error(w, "Authentication required", http.StatusUnauthorized)
        return
    }

    userDID := h.getUserDID(r)

    // 1. Fetch Bluesky follows
    follows, err := h.fetchBlueskyFollows(r.Context(), userDID)
    if err != nil {
        http.Error(w, "Failed to fetch follows", http.StatusInternalServerError)
        return
    }

    // 2. Filter for Arabica users
    arabicaUsers := []string{}
    for _, follow := range follows {
        hasArabicaContent, err := h.hasArabicaRecords(r.Context(), follow.DID)
        if err != nil {
            log.Warn().Err(err).Str("did", follow.DID).Msg("Failed to check Arabica records")
            continue
        }
        if hasArabicaContent {
            arabicaUsers = append(arabicaUsers, follow.DID)
        }
    }

    // 3. Create follow records
    imported := 0
    for _, targetDID := range arabicaUsers {
        // Check if already following
        alreadyFollows, _ := store.UserFollows(r.Context(), userDID, targetDID)
        if alreadyFollows {
            continue
        }

        follow := &atproto.Follow{
            Type: "app.bsky.graph.follow",
            Subject: targetDID,
            CreatedAt: time.Now().Format(time.RFC3339),
        }
        _, err := store.CreateFollow(r.Context(), follow)
        if err == nil {
            imported++
        }
    }

    json.NewEncoder(w).Encode(map[string]interface{}{
        "imported": imported,
        "total_follows": len(follows),
        "arabica_users": len(arabicaUsers),
    })
}

func (h *Handlers) hasArabicaRecords(ctx context.Context, did string) (bool, error) {
    // Use public client to check for any Arabica records
    client := atproto.NewPublicClient()
    records, err := client.ListRecords(ctx, did, atproto.NSIDBrew, 1)
    if err != nil {
        return false, err
    }
    return len(records) > 0, nil
}
```

**Challenges:**

- **Rate limiting**: Bluesky API has rate limits; may need to batch/queue imports
- **Stale data**: Follows may be out of sync if user unfollows on Bluesky
- **Performance**: Checking each DID for Arabica content is slow
  - **Solution**: Maintain a "known Arabica users" index from firehose

**Enhancement:** Two-way sync

- Export Arabica follows → Bluesky follows (optional)
- Periodic sync job to keep in sync

---

## Data Flow & Storage

### Like Flow

```
User clicks "Like" on a brew
    ↓
Frontend sends POST /api/likes
    ↓
Backend creates app.bsky.feed.like record in user's PDS
    ↓
PDS broadcasts record to Relay via firehose
    ↓
Arabica firehose listener receives event
    ↓
SocialIndex updates like count for subject URI
    ↓
Cache invalidated (optional)
    ↓
Feed refreshes with new like count
```

### Follow Flow

```
User clicks "Follow" on profile
    ↓
Frontend sends POST /api/follows
    ↓
Backend creates app.bsky.graph.follow record in user's PDS
    ↓
PDS broadcasts to firehose
    ↓
Arabica listener updates FollowsIndex
    ↓
User's feed now includes followed user's brews
```

### Comment Flow

```
User submits comment on brew
    ↓
Frontend sends POST /api/comments
    ↓
Backend creates social.arabica.alpha.comment in user's PDS
    ↓
Firehose broadcasts event
    ↓
Arabica listener indexes comment
    ↓
Comment appears on brew detail page
```

---

## UI/UX Considerations

### Like Button

**States:**

- Not liked: Gray heart outline
- Liked: Red filled heart
- Loading: Gray heart with spinner

**Display:**

- Show like count next to heart
- On hover: Show "X people liked this"
- Click: Optimistic update (instant UI change, API call in background)

**Location:**

- Brew cards in feed
- Brew detail page
- Bean/Roaster/Grinder/Brewer detail pages

### Follow Button

**States:**

- Not following: "Follow" button (blue)
- Following: "Following" button (gray, checkmark)
- Hover over "Following": "Unfollow" (red)

**Location:**

- User profile header
- Brew author byline (small follow button)
- Followers/Following lists

### Comments Section

**Layout:**

- Threaded comments (indented replies)
- Show comment count
- "Load more" pagination (20 per page)
- Sort by: Newest, Oldest, Most Liked

**Comment Form:**

- Textarea with mention support (@username autocomplete)
- Optional rating (1-10 stars)
- Cancel/Submit buttons
- Character count (500 max)

---

## Migration & Rollout

### Step 1: Backend Deployment (Week 1)

1. Deploy firehose listener with like/follow indexing
2. Backfill existing likes/follows from firehose history
3. Test API endpoints in staging
4. Monitor BoltDB storage growth

### Step 2: Frontend Soft Launch (Week 2)

1. Deploy like button (feature flag: enabled for beta users)
2. Collect feedback
3. Fix bugs

### Step 3: Public Launch (Week 3)

1. Enable likes for all users
2. Announce on Bluesky: "You can now like brews on Arabica!"
3. Monitor server load

### Step 4: Follow Feature (Week 4-5)

1. Deploy follow indexing
2. Add follow button to profiles
3. Add "Following" feed filter
4. Launch import-from-Bluesky tool

### Step 5: Comments (Week 6-8)

1. Define and publish comment lexicon
2. Deploy comment indexing
3. Add comment UI
4. Test threading

---

## Future Enhancements

### Phase 5+: Advanced Social Features

1. **Notifications**
   - "X liked your brew"
   - "Y commented on your brew"
   - WebSocket-based real-time updates

2. **Social Discovery**
   - "Trending brews this week"
   - "Popular roasters"
   - "Top coffee influencers"

3. **Activity Feed**
   - "Your friend Alice brewed a new espresso"
   - "Bob rated a bean you liked"

4. **Lists & Collections**
   - "My favorite light roasts" (curated bean list)
   - "Seattle coffee shops" (location-based)

5. **Collaborative Brewing**
   - Share brew recipes
   - Clone someone's brew with credit
   - Brew challenges ("30 days of pour-over")

6. **Cross-App Features**
   - Share brew to Bluesky as a post (with photo)
   - Embed brew cards in Bluesky posts
   - "Post to Bluesky" button on brew creation

---

## Appendix A: Lexicon Files

### Proposed: `social.arabica.alpha.like.json` (NOT RECOMMENDED)

If we decide NOT to use `app.bsky.feed.like`, here's our custom lexicon:

```json
{
  "lexicon": 1,
  "id": "social.arabica.alpha.like",
  "defs": {
    "main": {
      "type": "record",
      "key": "tid",
      "description": "A like on a brew, bean, roaster, grinder, or brewer",
      "record": {
        "type": "object",
        "required": ["subject", "createdAt"],
        "properties": {
          "subject": {
            "type": "ref",
            "ref": "com.atproto.repo.strongRef",
            "description": "The record being liked"
          },
          "createdAt": {
            "type": "string",
            "format": "datetime"
          }
        }
      }
    }
  }
}
```

### Proposed: `social.arabica.alpha.follow.json` (NOT RECOMMENDED)

```json
{
  "lexicon": 1,
  "id": "social.arabica.alpha.follow",
  "defs": {
    "main": {
      "type": "record",
      "key": "tid",
      "description": "Following another coffee enthusiast",
      "record": {
        "type": "object",
        "required": ["subject", "createdAt"],
        "properties": {
          "subject": {
            "type": "string",
            "format": "did",
            "description": "DID of the user being followed"
          },
          "createdAt": {
            "type": "string",
            "format": "datetime"
          }
        }
      }
    }
  }
}
```

---

## Appendix B: Estimated Effort

| Phase     | Feature           | Backend | Frontend | Testing | Total                    |
| --------- | ----------------- | ------- | -------- | ------- | ------------------------ |
| 1         | Likes             | 5 days  | 3 days   | 2 days  | **10 days**              |
| 2         | Follows           | 7 days  | 5 days   | 3 days  | **15 days**              |
| 3         | Comments          | 8 days  | 6 days   | 4 days  | **18 days**              |
| 4         | Feed Enhancements | 4 days  | 4 days   | 2 days  | **10 days**              |
| **Total** |                   |         |          |         | **53 days (10.6 weeks)** |

---

## Appendix C: Open Questions

1. **Moderation**: How do we handle spam comments or abusive likes?
   - Use AT Protocol's label system?
   - Admin moderation tools?

2. **Privacy**: Should follows be private?
   - Current plan: Public (like Bluesky)
   - Could add private follows later

3. **Notifications**: What delivery mechanism?
   - WebSocket for real-time?
   - Polling API?
   - Email digests?

4. **Analytics**: Track engagement metrics?
   - Like/comment rates
   - User retention
   - Popular content

5. **Mobile**: When to build native apps?
   - After web is stable
   - Consider PWA first

---

## Conclusion

This plan provides a **phased, pragmatic approach** to adding social features to Arabica. By **reusing Bluesky's `like` and `follow` lexicons**, we gain:

- ✅ Cross-app discoverability
- ✅ Simpler implementation
- ✅ Follow import from Bluesky
- ✅ Future-proof social graph

While **custom comments** allow:

- ✅ Coffee-specific features (ratings, tasting notes)
- ✅ Cleaner separation from Bluesky threads
- ✅ Control over threading UX

**Next Steps:**

1. Review and approve this plan
2. Begin Phase 1 (Likes) implementation
3. Iterate based on user feedback
4. Expand to follows and comments

**Timeline:** ~11 weeks for all phases (with 1 developer)

---

**Document Status:** Draft for Review  
**Last Updated:** January 25, 2026  
**Author:** AI Assistant (with human review pending)
