# Feed API

## `GET /api/feed`

Returns the community feed as JSON for the SvelteKit SPA.

### Request

| Param   | Type   | Default   | Description                                      |
|---------|--------|-----------|--------------------------------------------------|
| `type`  | string | (all)     | Entity type filter (e.g. `brew`, `bean`, `roaster`). Maps through the app's entity route nouns. |
| `sort`  | string | `recent`  | Sort order: `recent` or `popular`.               |
| `cursor`| string | (none)    | Pagination cursor from a previous response's `next_cursor`. |

### Response

The endpoint always returns JSON (the SvelteKit SPA is the sole frontend).

### Response (JSON)

```json
{
  "items": [FeedItem],
  "next_cursor": "string",
  "is_authenticated": true,
  "query": {
    "type": "brew",
    "sort": "recent"
  }
}
```

| Field             | Type       | Description                                           |
|-------------------|------------|-------------------------------------------------------|
| `items`           | `FeedItem[]` | Feed items, most recent first.                      |
| `next_cursor`     | `string`   | Cursor for the next page; empty string if no more.    |
| `is_authenticated`| `bool`     | Whether the requesting user is authenticated.         |
| `query.type`      | `string`   | The active type filter (empty string = all types).    |
| `query.sort`      | `string`   | The active sort order.                                |

### FeedItem

```json
{
  "record_type": "brew",
  "action": "added a new brew",
  "record": { "...typed model..." },
  "author": {
    "did": "did:plc:...",
    "handle": "alice.test",
    "display_name": "Alice",
    "avatar": "https://..."
  },
  "timestamp": "2026-07-08T12:00:00Z",
  "time_ago": "2 hours ago",
  "like_count": 3,
  "comment_count": 1,
  "subject_uri": "at://did:plc:.../social.arabica.alpha.brew/xyz",
  "subject_cid": "bafy...",
  "is_liked_by_viewer": false,
  "is_owner": true
}
```

| Field               | Type     | Description                                              |
|---------------------|----------|----------------------------------------------------------|
| `record_type`       | `string` | Entity type (e.g. `brew`, `bean`, `roaster`).           |
| `action`            | `string` | Human-readable action text.                              |
| `record`            | `object` | The typed model (Brew, Bean, etc.) with json tags.       |
| `author`            | `object` | Author profile (did, handle, display_name, avatar).      |
| `timestamp`         | `string` | RFC 3339 timestamp.                                       |
| `time_ago`          | `string` | Human-readable relative time.                            |
| `like_count`        | `int`    | Number of likes on this record.                          |
| `comment_count`     | `int`    | Number of comments on this record.                       |
| `subject_uri`       | `string` | AT-URI of this record (for like button).                 |
| `subject_cid`       | `string` | CID of this record (for like button).                    |
| `is_liked_by_viewer`| `bool`   | Whether the authenticated user liked this.               |
| `is_owner`          | `bool`   | Whether the authenticated user owns this record.         |

### Notes

- Unauthenticated users receive a limited cached public feed (no filtering, no cursor).
- Authenticated users get filtering by type, sorting, and pagination.
- `is_liked_by_viewer` and `is_owner` are `false` when unauthenticated.
