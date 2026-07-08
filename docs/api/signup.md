# Signup API

Endpoints for the account creation flow.

## `GET /api/signup/categories`

Returns the PDS provider catalog shown on the `/join/create` page. Dev-only
categories are included when the server has developer mode enabled
(`<APP>_DEV=1`).

### Response

```json
{
  "categories": [
    {
      "title": "Recommended",
      "description": "A reliable, open community provider.",
      "providers": [
        {
          "url": "https://selfhosted.social",
          "name": "selfhosted.social",
          "domain": "selfhosted.social",
          "description": "Community provider",
          "location": "United States",
          "badge": "Open",
          "badge_color": "green",
          "operator_name": "@baileytownsend.dev",
          "operator_url": "https://bsky.app/profile/baileytownsend.dev",
          "signup_url": ""
        }
      ],
      "dev_only": false
    }
  ]
}
```

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `categories` | `Category[]` | Provider groups, ordered by display priority |
| `Category.title` | `string` | Section heading (e.g. "Recommended", "App Providers") |
| `Category.description` | `string` | Section subtitle |
| `Category.providers` | `Provider[]` | PDS options in this category |
| `Category.dev_only` | `boolean` | True if only shown when dev mode is enabled |
| `Provider.url` | `string` | PDS URL (used as `pds_url` in the POST form) |
| `Provider.name` | `string` | Display name |
| `Provider.domain` | `string` | Display domain |
| `Provider.description` | `string` | Short description |
| `Provider.location` | `string` | Geographic location |
| `Provider.badge` | `string` | Status badge text (e.g. "Open", "Invite Only") |
| `Provider.badge_color` | `string` | Tailwind color prefix for the badge (e.g. "green", "amber") |
| `Provider.operator_name` | `string` | Optional operator handle |
| `Provider.operator_url` | `string` | Optional link to operator profile |
| `Provider.signup_url` | `string` | If set, link directly to this URL; otherwise render a POST form to `/join/create` with `pds_url` |

### Notes

- The `POST /join/create` mutation (OAuth `prompt=create` flow) is not a JSON
  endpoint — it redirects to the PDS authorization URL. The SvelteKit
  `/join/create` route renders a regular HTML form for providers without a
  `signup_url`.
