# Brew List API

## `GET /api/brews`

Returns a paginated list of the authenticated user's brews.

### Content Negotiation

- **`Accept: application/json`** — returns JSON (below). Used by the SvelteKit SPA.
- **`HX-Request: true`** — returns the existing HTML brew table partial for HTMX clients.

### Request

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `offset` | int | 0 | Pagination offset (number of brews to skip). |
| `limit` | int | 25 | Page size (max 100). |

### Response (JSON)

```json
{
  "brews": [Brew],
  "has_more": true,
  "next_offset": 25
}
```

| Field | Type | Description |
|-------|------|-------------|
| `brews` | `Brew[]` | Brew records for the current page, newest first. |
| `has_more` | `bool` | Whether there are more brews beyond this page. |
| `next_offset` | `int` | Offset to use for the next page (`offset + limit`). |

### Notes

- The handler requests `limit + 1` records to detect whether there are more
  results, then trims to the requested limit.
- Brews are ordered by creation time, newest first.
- Each brew includes resolved references (bean, grinder, brewer) when available
  from the session/witness cache.

### Errors

| Status | Cause |
|--------|-------|
| 401 | Not authenticated. |
| 500 | Store error (PDS fetch failure). |
