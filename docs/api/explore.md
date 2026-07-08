# Explore API

## `GET /api/explore`

Returns explore search results as JSON: feed items, explore documents (with
ratings/counts), facet counts for filtering, and pagination cursor.

### Request

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `type` | string | (all) | Entity type filter (e.g. `bean`, `brew`). |
| `q` | string | (none) | Full-text search query. |
| `sort` | string | (default) | Sort order. |
| `cursor` | string | (none) | Pagination cursor. |
| `origin` | string | | Facet filter: bean origin. |
| `variety` | string | | Facet filter: bean variety. |
| `process` | string | | Facet filter: bean process. |
| `roast_level` | string | | Facet filter: roast level. |
| `roaster` | string | | Facet filter: roaster name. |
| `min_rating` | string | | Facet filter: minimum rating. |
| `closed` | string | | Facet filter: bean closed status. |
| `location` | string | | Facet filter: roaster location. |
| `grinder_type` | string | | Facet filter: grinder type. |
| `burr_type` | string | | Facet filter: burr type. |
| `brewer_type` | string | | Facet filter: brewer type. |
| `ratio_min` | string | | Facet filter: minimum brew ratio. |
| `ratio_max` | string | | Facet filter: maximum brew ratio. |

### Response

```json
{
  "items": [FeedItem],
  "documents": {"at://...": ExploreDocument},
  "facet_counts": [ExploreFacetCount],
  "next_cursor": "...",
  "health": ExploreHealth
}
```

| Field | Type | Description |
|-------|------|-------------|
| `items` | `FeedItem[]` | Feed items matching the query. |
| `documents` | `{uri: ExploreDocument}` | Per-record explore metadata (ratings, counts). |
| `facet_counts` | `ExploreFacetCount[]` | Facet values for filter UI. |
| `next_cursor` | `string` | Cursor for the next page; empty if no more. |
| `health` | `ExploreHealth` | Explore index readiness status. |

### ExploreDocument

```json
{
  "URI": "at://...",
  "DID": "did:plc:...",
  "RecordType": "bean",
  "ClusterKey": "...",
  "Title": "Ethiopian Yirgacheffe",
  "Summary": "...",
  "OwnRating": 4.5,
  "CommunityRating": 4.2,
  "RatingCount": 8,
  "LikeCount": 3,
  "CommentCount": 1,
  "SourceRefCount": 2,
  "PopularScore": 0.85,
  "CreatedAt": "2026-07-08T12:00:00Z"
}
```

### Notes

- Moderation filtering is applied: hidden records are excluded.
- `IsLikedByViewer` and `IsOwner` are populated per-item for authenticated viewers.
- The handler fetches up to 3x the requested limit to account for moderation filtering.

### Errors

| Status | Cause |
|--------|-------|
| 401 | Not authenticated. |
| 503 | Explore index unavailable. |
| 500 | Internal error. |
