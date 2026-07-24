# Social API

## `POST /api/likes/toggle`

Toggles a like on a record. Returns the new like state and count as JSON.

### Request (form-urlencoded)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `subject_uri` | string | yes | AT-URI of the record to like/unlike. |
| `subject_cid` | string | yes | CID of the record. |

### Response (JSON)

```json
{
  "is_liked": true,
  "like_count": 5,
  "subject_uri": "at://did:plc:.../social.arabica.alpha.brew/xyz"
}
```

## `GET /api/comments`

Returns the comment thread for a subject record as JSON.

### Request

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `subject_uri` | string | yes | AT-URI of the record. |
| `subject_cid` | string | no | CID of the record (for the form). |

### Response (JSON)

```json
{
  "comments": [Comment],
  "subject_uri": "at://...",
  "is_authenticated": true
}
```

## `POST /api/comments`

Creates a comment on a record. Returns the created comment and the updated
comment thread as JSON.

### Request (form-urlencoded)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `subject_uri` | string | yes | AT-URI of the record. |
| `subject_cid` | string | yes | CID of the record. |
| `text` | string | yes | Comment text (max 1000 chars). |
| `parent_uri` | string | no | AT-URI of the parent comment (for replies). |
| `parent_cid` | string | no | CID of the parent comment. |

### Response (JSON)

```json
{
  "comment": Comment,
  "comments": [Comment]
}
```

## `DELETE /api/comments/{id}`

Deletes a comment by rkey.

### Content Negotiation

- **`Accept: application/json`** — returns JSON (below).
- **No JSON Accept** — returns empty body with `HX-Trigger: entityDeleted`.

### Response (JSON)

```json
{"deleted": true}
```

## `POST /api/report`

Submits a content report. Returns the report ID on success.

### Content Negotiation

- **`Accept: application/json`** — returns JSON (below).
- **No JSON Accept** — returns HTML success partial with `HX-Trigger` toast.

### Request (form-urlencoded)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `subject_uri` | string | yes | AT-URI of the reported record. |
| `reason` | string | no | Report reason (max 500 chars). |

### Response (JSON, success)

```json
{
  "report_id": "3mq5...",
  "submitted": true
}
```

### Response (JSON, error)

```json
{
  "error": "You have already reported this content",
  "code": "conflict"
}
```

### Notes

- Rate limited: 10 reports per hour per user.
- Cannot report own content.
- Automod may auto-hide the record if thresholds are met (3 reports on a
  single record, or 5 total reports against a user's content).

### Errors

| Status | Cause |
|--------|-------|
| 401 | Not authenticated. |
| 400 | Missing subject_uri, invalid URI, or self-report. |
| 409 | The viewer already reported this content. |
| 429 | Rate limit exceeded. |
| 500 | Report storage failed. |
| 503 | Reporting is not configured. |
