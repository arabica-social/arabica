# Brew Mutation API

## `POST /brews` (create)

Creates a new brew. Supports content negotiation: form POST with
`Accept: application/json` returns the created brew as JSON; otherwise
returns an `HX-Redirect` to `/my-coffee` (existing HTMX behavior).

### Request (form-urlencoded)

| Field                  | Type   | Required | Description                                    |
|------------------------|--------|----------|------------------------------------------------|
| `bean_rkey`            | string | yes      | RKey of the referenced bean.                   |
| `grinder_rkey`         | string | no       | RKey of the referenced grinder.                |
| `brewer_rkey`          | string | yes      | RKey of the referenced brewer (create only).   |
| `recipe_rkey`          | string | no       | RKey of the referenced recipe.                  |
| `recipe_owner_did`     | string | no       | DID of the recipe owner (for cross-user recipes). |
| `method`               | string | no       | Brewing method (e.g. "Pour Over", "Espresso"). |
| `temperature`          | float  | no       | Water temperature.                             |
| `water_amount`         | int    | no       | Total water in ml.                             |
| `coffee_amount`        | int    | no       | Coffee dose in grams.                           |
| `time_seconds`         | int    | no       | Nonnegative integer seconds; no upper time bound.                    |
| `grind_size`           | string | no       | Grind size description.                        |
| `tasting_notes`        | string | no       | Free-form tasting notes.                       |
| `rating`               | int    | no       | Rating 1-10; omitted, empty, or 0 means unrated. |
| `pour_water_{n}`       | int    | no       | Water amount for pour *n* (0-indexed).         |
| `pour_time_{n}`        | int    | no       | Time for pour *n*.                             |
| `espresso_yield_weight`| float  | no       | Espresso yield in grams.                       |
| `espresso_pressure`   | float  | no       | Espresso pressure in bar.                      |
| `espresso_pre_infusion_seconds` | int | no | Pre-infusion time in seconds.                  |
| `pourover_bloom_water`| int    | no       | Bloom water in ml.                             |
| `pourover_bloom_seconds` | int  | no       | Bloom time in seconds.                         |
| `pourover_drawdown_seconds` | int | no   | Drawdown time in seconds.                      |
| `pourover_bypass_water`| int   | no       | Bypass water in ml.                            |
| `pourover_filter`      | string | no       | Filter type.                                   |

### Content Negotiation

- **`Accept: application/json`** — returns the JSON envelope below. Used by the SvelteKit SPA.
- **No JSON Accept** — returns `HX-Redirect: /my-coffee` (existing HTMX behavior).

### Response (JSON)

```json
{
  "brew": {
    "rkey": "xyz",
    "method": "Pour Over",
    "bean_rkey": "abc",
    "...all Brew fields..."
  },
  "incomplete_nudge": {
    "entity_type": "bean",
    "rkey": "abc",
    "name": "My Bean",
    "missing": "origin, roast_level"
  }
}
```

| Field              | Type    | Description                                                  |
|--------------------|---------|--------------------------------------------------------------|
| `brew`             | `Brew`  | The created brew model with all fields.                      |
| `incomplete_nudge` | `object`| Present only when the referenced bean is missing fields. `null` otherwise. |

The returned `brew` includes `recipe_owner_did` when a recipe reference is
present. This value is derived from the recipe reference authority and is not
stored as a separate lexicon field.

### Incomplete Nudge

When the referenced bean `IsIncomplete()`, the response includes a nudge
object prompting the user to fill in missing bean fields. The HTMX path
sets the same data via the `X-Incomplete-Nudge` response header.

## `POST /api/brews` (create)

Accepts a JSON body using the same snake_case fields, or a form-urlencoded
body. Returns the same JSON envelope as `POST /brews`.
Both create endpoints require a non-empty, valid `bean_rkey` and `brewer_rkey`.
A missing brewer returns 400 with `Brewer selection is required`; the JSON
endpoint uses error code `validation_failed`. Method, grinder, measurements,
and rating remain optional. JSON method-specific blocks use `espresso_params`
and `pourover_params`; pours use an array of `{water_amount, time_seconds}`.

This is a create-handler policy, not a new required lexicon field. Existing PDS
brews without `brewerRef` remain readable and editable.

## `PUT /brews/{id}` and `PUT /api/brews/{id}` (update)

Updates an existing brew. `/brews/{id}` accepts the same form fields and content
negotiation as create; `/api/brews/{id}` accepts JSON or form data and returns
JSON. Unlike create, `brewer_rkey` is optional so legacy brewer-less brews can
still be updated.

Updates replace the editable fields rather than patching them. Omitting a rating
(or sending an empty form rating or numeric 0) removes the persisted rating;
responses represent an unrated brew as `rating: 0`. Legacy free-text `method`
and descriptive or numeric-string `grind_size` remain supported. Resubmitted
pours and both method-specific parameter blocks are retained independently of
the selected method; switching methods does not implicitly discard them.

### Response (JSON)

Returns the same envelope as create (`{brew, incomplete_nudge?}`), with
`incomplete_nudge` omitted (update does not currently compute it).

### Errors

| Status | Cause                                        |
|--------|----------------------------------------------|
| 400    | Validation error (invalid field, missing bean_rkey, or missing brewer_rkey on create). |
| 401    | Not authenticated.                           |
| 500    | Store error (PDS write failure).             |
