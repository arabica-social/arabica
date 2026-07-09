# Settings API

## `GET /api/settings`

Returns the authenticated user's settings: profile stats visibility,
preferences, and Bluesky profile sync state.

### Response

```json
{
  "profile_stats_visibility": {
    "bean_avg_rating": "public",
    "roaster_avg_rating": "public"
  },
  "user_preferences": {
    "temperature_unit": "fahrenheit"
  },
  "bluesky_profile": {
    "has_scopes": false,
    "display_name": "Alice",
    "avatar_url": "https://...",
    "load_error": "",
    "needs_auth_again": false
  }
}
```

## `POST /api/settings/preferences`

Saves user preferences (temperature unit).

### Content Negotiation

- **`Accept: application/json`** — returns `{"saved": true}`.
- **No JSON Accept** — returns HTML `<span>Saved</span>` (existing HTMX behavior).

### Request (form-urlencoded)

| Field | Type | Description |
|-------|------|-------------|
| `temperature_unit` | string | `fahrenheit` or `celsius`. |

### Response (JSON)

```json
{"saved": true}
```

## `POST /api/settings/profile-visibility`

Saves profile stats visibility settings.

### Content Negotiation

- **`Accept: application/json`** — returns `{"saved": true}`.
- **No JSON Accept** — returns HTML `<span>Saved</span>`.

### Request (form-urlencoded)

| Field | Type | Description |
|-------|------|-------------|
| `bean_avg_rating` | string | `public` or `private`. |
| `roaster_avg_rating` | string | `public` or `private`. |

### Response (JSON)

```json
{"saved": true}
```

### Errors

| Status | Cause |
|--------|-------|
| 401 | Not authenticated. |
| 500 | Failed to save settings. |
