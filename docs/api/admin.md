# Admin API

Admin endpoints serve the moderation dashboard. All require moderator or
admin permissions (enforced by `RequireModerator`/`RequireAdmin` middleware).

## `GET /api/_mod`

Returns the full admin dashboard data as JSON: hidden records, reports,
audit log, blocked users, labels, stats, and the viewer's permissions.

### Response

```json
{
  "hidden_records": [HiddenRecord],
  "audit_log": [AuditEntry],
  "reports": [EnrichedReport],
  "blocked_users": [BlacklistedUser],
  "labels": [Label],
  "stats": AdminStats,
  "backups": [SourceStatus],
  "can_hide": true,
  "can_unhide": true,
  "can_view_logs": true,
  "can_view_reports": true,
  "can_block": false,
  "can_unblock": false,
  "can_reset_auto_hide": false,
  "can_manage_labels": false,
  "is_admin": false
}
```

The `can_*` fields reflect the viewer's permissions, so the SPA can
show/hide UI accordingly. `stats` and `backups` are only populated for
admins.

## `GET /api/_mod/stats`

Returns admin stats + backup status as JSON.

### Response

```json
{
  "stats": {
    "known_users": 42,
    "registered_users": 38,
    "indexed_records": 1500,
    "total_likes": 320,
    "total_comments": 89,
    "firehose_connected": true,
    "records_by_collection": {"social.arabica.alpha.brew": 500, ...}
  },
  "backups": [SourceStatus]
}
```

## Mutation Endpoints

All admin mutation endpoints support content negotiation: `Accept:
application/json` returns `{ok, action, message}`; HTMX clients get the
existing `HX-Trigger` response.

### Response (JSON)

```json
{
  "ok": true,
  "action": "hide",
  "message": "Record hidden from feed"
}
```

### Endpoints

| Endpoint | Method | Action | Permission |
|----------|--------|--------|------------|
| `/_mod/hide` | POST | `hide` | `PermissionHideRecord` |
| `/_mod/unhide` | POST | `unhide` | `PermissionUnhideRecord` |
| `/_mod/dismiss-report` | POST | `dismiss-report` | `PermissionDismissReport` |
| `/_mod/block` | POST | `block` | `PermissionBlacklistUser` |
| `/_mod/unblock` | POST | `unblock` | `PermissionUnblacklistUser` |
| `/_mod/reset-autohide` | POST | `reset-autohide` | `PermissionResetAutoHide` |
| `/_mod/label/add` | POST | `add-label` | `PermissionManageLabels` |
| `/_mod/label/remove` | POST | `remove-label` | `PermissionManageLabels` |

### Request Parameters

- **hide/unhide**: `uri` (required), `reason`
- **dismiss-report**: `id` (report ID, required)
- **block/unblock**: `did` (required), `reason` (block only)
- **reset-autohide**: `did` (required)
- **label/add**: `entity_type` (`user`/`record`), `entity_id`, `label`, `value`, `expires` (TTL duration)
- **label/remove**: `entity_type`, `entity_id`, `label`

### Errors

| Status | Cause |
|--------|-------|
| 400 | Missing required fields. |
| 403 | Not a moderator or lacks permission. |
| 500 | Store error. |
