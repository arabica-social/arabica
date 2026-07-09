# Backlinks API

## `GET /api/{entity}/{actor}/{id}/backlinks`

Returns community backlinks data for an entity: records that reference this
entity (library entries, usage, and ratings).

### Registered Routes

| Route | Entity |
|-------|--------|
| `GET /api/beans/{actor}/{id}/backlinks` | Bean |
| `GET /api/roasters/{actor}/{id}/backlinks` | Roaster |
| `GET /api/grinders/{actor}/{id}/backlinks` | Grinder |
| `GET /api/brewers/{actor}/{id}/backlinks` | Brewer |
| `GET /api/recipes/{actor}/{id}/backlinks` | Recipe |

These are registered via the `JSONBacklinks` slot on `EntityRouteBundle`.

### Request

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `usage` | string | (none) | Filter to a specific usage group key. |
| `page` | int | 1 | Usage pagination page. |

### Response

```json
{
  "entity_noun": "roaster",
  "entity_name": "Onyx Coffee Lab",
  "back_url": "/roasters/did:plc:.../xyz",
  "detail_url": "/roasters/did:plc:.../xyz/backlinks",
  "result": {
    "LibraryEntries": [...],
    "LibraryCount": 3,
    "Usage": [...],
    "UsageCount": 5,
    "RatingAverage": 4.2,
    "RatingCount": 8
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `entity_noun` | `string` | Lowercase entity noun (e.g. `bean`, `roaster`). |
| `entity_name` | `string` | Display name of the entity. |
| `back_url` | `string` | URL back to the entity view page. |
| `detail_url` | `string` | URL for the backlinks detail page (HTML). |
| `result` | `backlinks.Result` | Backlinks lookup result (library, usage, ratings). |

### Notes

- The backlinks result comes from the witness cache (firehose index).
- Usage is paginated (25 per page).

### Errors

| Status | Cause |
|--------|-------|
| 400 | Missing or invalid rkey. |
| 401 | Not authenticated and no owner param. |
| 404 | Record not found. |
