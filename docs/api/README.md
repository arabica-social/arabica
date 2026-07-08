# Arabica JSON API Contract

This directory is the **interface contract** between three parallel work
streams in the SvelteKit SPA migration:

- **Stream A (Go JSON API)** implements these endpoints.
- **Stream B (SvelteKit frontend)** consumes them.
- **Stream C (testing)** generates contract tests from these shapes.

Each file documents the request/response shapes for a domain area. When
adding or changing an endpoint, update the spec here first, then implement.

## Conventions

- All endpoints return JSON with `Content-Type: application/json`.
- Mutations (POST/PUT/DELETE) are CSRF-protected via the `CrossOriginProtection`
  middleware. The SvelteKit frontend sends the same cookies/forms it does today.
- Error responses use `{"error": "message"}` with appropriate HTTP status codes.
- Timestamps are RFC 3339 strings.
- AT-URIs are full `at://did:plc:.../collection/rkey` strings.
- Pagination uses cursor strings; an empty `next_cursor` means no more pages.

## Existing JSON Endpoints (already working)

These endpoints already return JSON and need no changes:

| Endpoint | Method | Response shape |
|----------|--------|----------------|
| `/.well-known/oauth-client-metadata.json` | GET | OAuth client metadata |
| `/healthz` | GET | `{status, firehose, feed_index}` |
| `/api/resolve-handle` | GET | `{did, handle, displayName?, avatar?}` |
| `/api/search-actors` | GET | `{actors: [...]}` |
| `/api/suggestions/{entity}` | GET | `[]EntitySuggestion` |
| `/api/data` | GET | `{did, beans[], roasters[], ...}` |
| `/api/recipes` | GET | `[]Recipe` |
| `/api/recipes/suggestions` | GET | `[]Recipe` |
| `/api/recipes/{id}` | GET | `Recipe` |
| `/api/recipes` | POST | `Recipe` |
| `/api/recipes/from-brew/{id}` | POST | `Recipe` |
| `/api/recipes/fork/{id}` | POST | `Recipe` |
| `/api/{entity}` | POST | `<model>` (simple entities) |
| `/api/{entity}/{id}` | PUT | `<model>` or HX-Redirect |
| `/brews/export` | GET | `[]Brew` |

## Endpoints to Add (Phase 1)

See the migration plan (`docs/plans/2026-07-08-sveltekit-migration.md`) for
the full list and priority order. Each endpoint's shape will be documented
in its own file as it's implemented:

- `feed.md` — `GET /api/feed` (JSON feed items)
- `entities.md` — `GET /api/{entity}/{actor}/{id}` (entity view data)
- `profile.md` — `GET /api/profile/{actor}` (profile data bundle)
- `social.md` — likes, comments, reports (JSON mutations)
- `manage.md` — `GET /api/manage` (records + stats)
- `notifications.md` — `GET /api/notifications`
- `settings.md` — `GET /api/settings`, settings mutations
- `explore.md` — `GET /api/explore`
- `onboarding.md` — `GET /api/onboarding`, incomplete records
- `admin.md` — moderation admin endpoints
