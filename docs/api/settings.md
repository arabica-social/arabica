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
    "temperature_unit": "fahrenheit",
    "smart_autofill": "on",
    "brew_form_mode": "quick"
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

Partially updates DID-scoped operational preferences. Omitted fields retain their
stored values in both transports. Missing or unreadable stored preferences use
defaults; this does not change PDS records. Updates are serialized within the
running process to preserve unrelated fields across concurrent requests.

### Content Negotiation

- **`Accept: application/json`** — returns `{"saved": true}`.
- **No JSON Accept** — returns HTML `<span>Saved</span>` (existing HTMX behavior).

### Request (form-urlencoded)

| Field | Type | Description |
|-------|------|-------------|
| `temperature_unit` | string | `recorded`, `fahrenheit` or `celsius`. Invalid submitted values use `recorded`. |
| `smart_autofill` | string | `on` or `off`. Invalid submitted values use `on`. |
| `brew_form_mode` | string | `quick` or `detailed`. Missing/invalid stored values and invalid submitted values use `quick`. |

`brew_form_mode` selects the initial new-brew layout only. Clients should save it
on explicit mode selection, not edit-page initialization. Preference persistence
failure must not block brew submission; clients may fall back to Quick.

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
