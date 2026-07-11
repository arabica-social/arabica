# Entity View API

## `GET /api/{entity}/{actor}/{id}`

Returns a single entity's detail data as JSON for the SvelteKit SPA.

### Path Parameters

| Param    | Description                                                      |
|----------|------------------------------------------------------------------|
| `entity` | Entity URL segment for the active app. |
| `actor`  | The record owner's DID (`did:plc:...`) or handle.                |
| `id`     | The record key (rkey, TID format).                              |

### Registered Routes

| Route                              | Entity  |
|------------------------------------|---------|
| `GET /api/beans/{actor}/{id}`      | Bean    |
| `GET /api/roasters/{actor}/{id}`   | Roaster |
| `GET /api/grinders/{actor}/{id}`   | Grinder |
| `GET /api/brewers/{actor}/{id}`    | Brewer  |
| `GET /api/brews/{actor}/{id}`      | Brew    |
| `GET /api/recipes/{actor}/{id}`    | Recipe  |
| `GET /api/teas/{actor}/{id}`       | Oolong tea |
| `GET /api/vendors/{actor}/{id}`    | Oolong vendor |
| `GET /api/vessels/{actor}/{id}`    | Oolong vessel |
| `GET /api/infusers/{actor}/{id}`   | Oolong infuser |
| `GET /api/brews/{actor}/{id}`      | Oolong steep (when served by Oolong) |

These are registered via the `JSONView` slot on `EntityRouteBundle` (simple
entities) and explicitly for Arabica brew/recipe (which have additional
endpoints). The shared `/api/brews/{actor}/{id}` path is app-scoped: Arabica
returns a coffee brew and Oolong returns a steep record.

### Response

```json
{
  "record": { "...typed model..." },
  "subject_uri": "at://did:plc:.../social.arabica.alpha.bean/xyz",
  "subject_cid": "bafy...",
  "author": {
    "did": "did:plc:...",
    "handle": "alice.test",
    "display_name": "Alice",
    "avatar": "https://..."
  },
  "social": {
    "is_liked": false,
    "like_count": 3,
    "comment_count": 1,
    "comments": [Comment],
    "is_moderator": false,
    "can_hide_record": false,
    "can_block_user": false,
    "is_record_hidden": false
  },
  "backlinks": { "...backlinks.Result..." },
  "is_own_profile": true,
  "is_authenticated": true,
  "share_url": "/beans/alice.test/xyz",
  "entity_type": "bean",
  "entity_count": 5
}
```

| Field              | Type      | Description                                                    |
|--------------------|-----------|----------------------------------------------------------------|
| `record`           | `object`  | The typed model (Bean, Roaster, Brew, etc.) with json tags.   |
| `subject_uri`      | `string`  | AT-URI of this record.                                         |
| `subject_cid`      | `string`  | CID of this record.                                            |
| `author`           | `object`  | Author profile (did, handle, display_name, avatar).            |
| `social`           | `object`  | Social data (likes, comments, moderation state).               |
| `backlinks`        | `object`  | Backlinks result (library entries, usage, ratings). `null` if none. |
| `is_own_profile`   | `bool`    | Whether the viewer owns this record.                           |
| `is_authenticated` | `bool`    | Whether the viewer is authenticated.                           |
| `share_url`        | `string`  | Canonical share URL (`/{entity}/{actor}/{id}`).                |
| `entity_type`     | `string`  | Lowercase entity noun (e.g. `bean`, `brew`).                  |
| `entity_count`    | `int`     | Usage count (e.g. brew count for a bean). `0` if N/A.         |

### Social Object

| Field              | Type        | Description                                          |
|--------------------|-------------|------------------------------------------------------|
| `is_liked`         | `bool`      | Whether the viewer liked this record.                |
| `like_count`       | `int`       | Total likes on this record.                          |
| `comment_count`    | `int`       | Total comments on this record.                       |
| `comments`         | `Comment[]` | Threaded comments.                                   |
| `is_moderator`     | `bool`      | Whether the viewer is a moderator.                   |
| `can_hide_record`  | `bool`      | Whether the viewer can hide this record.              |
| `can_block_user`   | `bool`      | Whether the viewer can block the author.             |
| `is_record_hidden` | `bool`      | Whether this record is hidden by moderation.        |

### Comment Object

Comments include computed profile fields (handle, display_name, avatar) that
the underlying `IndexedComment` struct marks `json:"-"` for the templ layer.

```json
{
  "rkey": "xyz",
  "subject_uri": "at://...",
  "text": "Great brew!",
  "actor_did": "did:plc:...",
  "created_at": "2026-07-08T12:00:00Z",
  "parent_uri": "at://...",
  "parent_rkey": "abc",
  "cid": "bafy...",
  "depth": 0,
  "handle": "bob.test",
  "display_name": "Bob",
  "avatar": "https://...",
  "like_count": 2,
  "is_liked": true
}
```

### Data Source

The handler reuses `EntityViewLoader.Load`, which tries three sources in order:

1. **Own-store** (authenticated owner) — reads through the session cache for freshness.
2. **Witness cache** — SQLite-backed local firehose index (fast, no PDS call).
3. **PDS fallback** — direct public XRPC call to the owner's PDS.

References (e.g. bean.roaster, brew.bean) are hydrated via `ResolveRefs` on all
three paths.

### Errors

| Status | Cause                                    |
|--------|------------------------------------------|
| 400    | Missing `owner` query param or invalid rkey. |
| 404    | Owner (handle) not found, or record not found on PDS. |
| 500    | Internal error loading or decoding record. |

Every error uses the shared `{error, code, fields?}` envelope. The expected
codes are `invalid_request`, `not_found`, and `internal_error`.
