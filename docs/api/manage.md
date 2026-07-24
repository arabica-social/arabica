# Manage API

## `GET /api/manage`

Returns the authenticated user's records plus witness-cache-derived usage
stats in a single response. This is the heavier counterpart to `GET /api/data`
(which returns raw records without stats for the lightweight appCache use case).

### Response

The endpoint always returns JSON (the SvelteKit SPA is the sole frontend).

### Response (JSON)

```json
{
  "did": "did:plc:...",
  "beans": [Bean],
  "roasters": [Roaster],
  "grinders": [Grinder],
  "brewers": [Brewer],
  "recipes": [Recipe],
  "stats": {
    "bean_brew_counts": {"at://.../bean/xyz": 3},
    "grinder_brew_counts": {"at://.../grinder/abc": 5},
    "brewer_brew_counts": {"at://.../brewer/def": 2},
    "roaster_bean_counts": {"at://.../roaster/ghi": 4},
    "bean_avg_brew_ratings": {"at://.../bean/xyz": 4.2},
    "roaster_avg_brew_ratings": {"at://.../roaster/ghi": 4.5}
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `did` | `string` | The authenticated user's DID. |
| `beans` | `Bean[]` | User's beans, linked to roasters. |
| `roasters` | `Roaster[]` | User's roasters. |
| `grinders` | `Grinder[]` | User's grinders. |
| `brewers` | `Brewer[]` | User's brewers. |
| `recipes` | `Recipe[]` | User's recipes, linked to brewers. |
| `stats` | `object` | Usage counts and average ratings, keyed by AT-URI. |

### Stats Object

All stat maps are keyed by the referenced entity's AT-URI. A missing key means
zero usage. Average ratings are only present for entities that have at least one
brew with a rating > 0.

| Field | Type | Description |
|-------|------|-------------|
| `bean_brew_counts` | `{uri: int}` | Number of brews referencing each bean. |
| `grinder_brew_counts` | `{uri: int}` | Number of brews referencing each grinder. |
| `brewer_brew_counts` | `{uri: int}` | Number of brews referencing each brewer. |
| `roaster_bean_counts` | `{uri: int}` | Number of beans referencing each roaster. |
| `bean_avg_brew_ratings` | `{uri: float}` | Average brew rating per bean. |
| `roaster_avg_brew_ratings` | `{uri: float}` | Average brew rating per roaster. |

### Notes

- The session cache is invalidated before fetching so reads go through the
  witness cache (or PDS) for fresh data.
- Beans are linked to roasters (`bean.roaster`) and recipes to brewers
  (`recipe.brewer_obj`) before serialization.

### Errors

| Status | Cause |
|--------|-------|
| 401 | Not authenticated. |
| 500 | Store error (PDS fetch failure). |
