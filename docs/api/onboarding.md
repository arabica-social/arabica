# Onboarding API

## `GET /api/onboarding`

Returns the user's brew readiness status and entity lists for the onboarding flow.

### Response

```json
{
  "readiness": {
    "HasBean": true,
    "HasBrewer": true,
    "HasRoaster": true
  },
  "beans": [Bean],
  "brewers": [Brewer],
  "grinders": [Grinder],
  "roasters": [Roaster]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `readiness` | `ReadinessStatus` | Whether the user has the minimum entities to log a brew. |
| `beans` | `Bean[]` | User's beans. |
| `brewers` | `Brewer[]` | User's brewers. |
| `grinders` | `Grinder[]` | User's grinders. |
| `roasters` | `Roaster[]` | User's roasters. |

### ReadinessStatus

`Ready()` returns true when the user owns at least one brewer, one roaster,
and one bean — the minimum required to log a brew.

## `GET /api/incomplete-records`

Returns the user's incomplete records (beans/grinders/brewers with missing
fields) as JSON.

### Response

```json
{
  "records": [
    {
      "EntityType": "bean",
      "RKey": "xyz",
      "Name": "My Bean",
      "MissingFields": ["origin", "roast_level"]
    }
  ]
}
```

### Notes

- Limited to 5 records.
- A bean is incomplete if it's missing origin, roast_level, or other key fields.

## `GET /api/popular-recipes`

Returns popular recipes sorted by brew_count + fork_count (descending).

### Content Negotiation

### Response

```json
[Recipe, Recipe, Recipe]
```

Returns up to 3 recipes, sorted by popularity. May be empty if no recipes exist.

### Errors

| Status | Cause |
|--------|-------|
| 401 | Not authenticated. |
| 500 | Failed to fetch data. |
