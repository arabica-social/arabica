# Notifications API

## `GET /api/notifications`

Returns the authenticated user's notifications as JSON.

### Request

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `cursor` | string | (none) | Pagination cursor from a previous response's `next_cursor`. |

### Response

```json
{
  "notifications": [
    {
      "id": "3mq5...",
      "type": "like",
      "actor_did": "did:plc:...",
      "subject_uri": "at://did:plc:.../social.arabica.alpha.brew/xyz",
      "created_at": "2026-07-08T12:00:00Z",
      "read": false,
      "actor_handle": "bob.test",
      "actor_display_name": "Bob",
      "actor_avatar": "https://...",
      "link": "/brews/did:plc:.../xyz",
      "action_text": "liked your brew"
    }
  ],
  "next_cursor": "..."
}
```

| Field | Type | Description |
|-------|------|-------------|
| `notifications` | `NotificationItem[]` | Notifications, newest first. |
| `next_cursor` | `string` | Cursor for the next page; empty if no more. |

### NotificationItem

Embeds `notifications.Notification` (id, type, actor_did, subject_uri, created_at, read) plus:

| Field | Type | Description |
|-------|------|-------------|
| `actor_handle` | `string` | Actor's handle (or DID if unresolvable). |
| `actor_display_name` | `string` | Actor's display name (omitempty). |
| `actor_avatar` | `string` | Actor's avatar URL (omitempty). |
| `link` | `string` | Local page URL for the subject record. |
| `action_text` | `string` | Human-readable action (e.g. "liked your brew"). |

### Notes

- Viewing notifications marks all as read.
- Returns 30 notifications per page.

## `POST /api/notifications/read`

Marks all notifications as read.

### Content Negotiation

- **`Accept: application/json`** — returns `{"read": true}`.
- **No JSON Accept** — redirects to `/notifications` (existing HTMX behavior).

### Response (JSON)

```json
{"read": true}
```

### Errors

| Status | Cause |
|--------|-------|
| 401 | Not authenticated. |
| 500 | Failed to mark notifications as read. |

Errors use the shared `{error, code, fields?}` envelope. Authentication
failures use `authentication_required`; storage failures use `internal_error`.
