# Profile API

## `GET /api/profile/{actor}`

Returns a user's full profile data bundle: profile metadata, paginated brews,
entity lists, brew social stats, and entity usage counts — all in one response.

### Content Negotiation

- **`Accept: application/json`** — returns JSON (below). Used by the SvelteKit SPA.
- **`HX-Request: true`** — returns the existing HTML profile partial for HTMX clients.

### Path Parameters

| Param | Description |
|-------|-------------|
| `actor` | The profile owner's DID (`did:plc:...`) or handle. |

### Request

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `brews_offset` | int | 0 | Pagination offset for the brews tab. |
| `brews_limit` | int | 25 | Page size for brews (max 100). |

### Response (JSON)

```json
{
  "profile": {"handle": "alice.test", "display_name": "Alice", "avatar": "..."},
  "did": "did:plc:...",
  "is_own_profile": true,
  "is_authenticated": true,
  "is_arabica_user": true,
  "brews": [Brew],
  "total_brews": 42,
  "brews_has_more": true,
  "brews_next_offset": 25,
  "beans": [Bean],
  "roasters": [Roaster],
  "grinders": [Grinder],
  "brewers": [Brewer],
  "brew_like_counts": {"<rkey>": 3},
  "brew_comment_counts": {"<rkey>": 1},
  "brew_liked_by_user": {"<rkey>": false},
  "brew_cids": {"<rkey>": "bafy..."},
  "bean_brew_counts": {"<uri>": 5},
  "grinder_brew_counts": {"<uri>": 8},
  "brewer_brew_counts": {"<uri>": 3},
  "roaster_bean_counts": {"<uri>": 2},
  "bean_avg_brew_ratings": {"<uri>": 4.2},
  "roaster_avg_brew_ratings": {"<uri>": 4.5}
}
```

| Field | Type | Description |
|-------|------|-------------|
| `profile` | `object` | Profile summary (handle, display_name, avatar). |
| `did` | `string` | The profile owner's DID. |
| `is_own_profile` | `bool` | Whether the viewer owns this profile. |
| `is_authenticated` | `bool` | Whether the viewer is authenticated. |
| `is_arabica_user` | `bool` | Whether the user has arabica records. |
| `brews` | `Brew[]` | Paginated brews (newest first). |
| `total_brews` | `int` | Total brew count (may exceed `len(brews)`). |
| `brews_has_more` | `bool` | Whether there are more brews. |
| `brews_next_offset` | `int` | Next page offset for brews. |
| `beans` | `Bean[]` | User's beans (with roaster links). |
| `roasters` | `Roaster[]` | User's roasters. |
| `grinders` | `Grinder[]` | User's grinders. |
| `brewers` | `Brewer[]` | User's brewers. |
| `brew_like_counts` | `{rkey: int}` | Like count per brew (keyed by rkey). |
| `brew_comment_counts` | `{rkey: int}` | Comment count per brew. |
| `brew_liked_by_user` | `{rkey: bool}` | Whether the viewer liked each brew. |
| `brew_cids` | `{rkey: string}` | CID per brew (for like/comment forms). |
| `bean_brew_counts` | `{uri: int}` | Brew count per bean URI. |
| `grinder_brew_counts` | `{uri: int}` | Brew count per grinder URI. |
| `brewer_brew_counts` | `{uri: int}` | Brew count per brewer URI. |
| `roaster_bean_counts` | `{uri: int}` | Bean count per roaster URI. |
| `bean_avg_brew_ratings` | `{uri: float}` | Avg brew rating per bean. |
| `roaster_avg_brew_ratings` | `{uri: float}` | Avg brew rating per roaster. |

### Profile Visibility

Average brew ratings respect the profile owner's visibility preferences
(`profileprefs.ProfileStatsVisibility`). When the viewer is not the profile
owner and the owner has set ratings to private, the corresponding avg rating
maps are empty. The owner always sees their own ratings.

### Data Source

Profile data is loaded via `fetchUserProfileData`, which tries:
1. **Witness cache** (firehose index) — all collections for this user.
2. **PDS fallback** — direct public XRPC calls to the owner's PDS.

Brews are sorted in reverse chronological order (newest first).

### Errors

| Status | Cause |
|--------|-------|
| 400 | Missing `actor` parameter. |
| 404 | User not found (unresolvable handle, blacklisted, no records, or PDS error). |
| 500 | Internal error loading data. |
