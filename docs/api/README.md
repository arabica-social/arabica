# Arabica JSON API Contract

This directory is the **interface contract** between three parallel work
streams in the SvelteKit SPA migration:

- **Stream A (Go JSON API)** implements these endpoints.
- **Stream B (SvelteKit frontend)** consumes them.
- **Stream C (testing)** generates contract tests from these shapes.

Each file documents the request/response shapes for a domain area. When
adding or changing an endpoint, update the spec here first, then implement.

## Conventions

- All JSON endpoints return `Content-Type: application/json; charset=utf-8`.
- Mutations (POST/PUT/DELETE) are CSRF-protected via the `CrossOriginProtection`
  middleware. The SvelteKit frontend sends the same cookies/forms it does today.
- Error responses use a stable envelope with an appropriate HTTP status:

  ```json
  {
    "error": "Human-readable safe message",
    "code": "stable_machine_code"
  }
  ```

  Validation responses may also include `fields`, a map from field name to a
  safe validation message. Machine codes are low-cardinality values such as
  `authentication_required`, `session_expired`, `permission_denied`,
  `not_found`, `validation_failed`, `invalid_request`, `conflict`,
  `rate_limited`, `service_unavailable`, and `internal_error`.
- Error messages must not expose dependency errors, stack traces, tokens,
  session identifiers, or other internal details.
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

## Phase 1 Endpoints (implemented)

These endpoints were added in Phase 1 of the SvelteKit migration:

| Endpoint | Method | Response shape | Spec |
|----------|--------|----------------|------|
| `/api/feed` | GET | `{items[], next_cursor, is_authenticated, query}` | [feed.md](feed.md) |
| `/api/beans/{actor}/{id}` | GET | `{record, subject_uri, social, backlinks, ...}` | [entities.md](entities.md) |
| `/api/roasters/{actor}/{id}` | GET | `{record, subject_uri, social, backlinks, ...}` | [entities.md](entities.md) |
| `/api/grinders/{actor}/{id}` | GET | `{record, subject_uri, social, backlinks, ...}` | [entities.md](entities.md) |
| `/api/brewers/{actor}/{id}` | GET | `{record, subject_uri, social, backlinks, ...}` | [entities.md](entities.md) |
| `/api/brews/{actor}/{id}` | GET | `{record, subject_uri, social, backlinks, ...}` | [entities.md](entities.md) |
| `/api/recipes/{actor}/{id}` | GET | `{record, subject_uri, social, backlinks, ...}` | [entities.md](entities.md) |
| `/api/brews` | POST | `{brew, incomplete_nudge?}` (JSON) or HX-Redirect | [brews.md](brews.md) |
| `/brews/{id}` | PUT | `{brew}` (JSON) or HX-Redirect | [brews.md](brews.md) |
| `/api/manage` | GET | `{did, beans[], roasters[], ..., stats}` | [manage.md](manage.md) |
| `/api/brews` | GET | `{brews[], has_more, next_offset}` | [brew_list.md](brew_list.md) |
| `/api/profile/{actor}` | GET | `{profile, brews[], beans[], ..., stats}` | [profile.md](profile.md) |
| `/api/notifications` | GET | `{notifications[], next_cursor}` | [notifications.md](notifications.md) |
| `/api/notifications/read` | POST | `{read: true}` (JSON) or redirect | [notifications.md](notifications.md) |
| `/api/explore` | GET | `{items[], documents, facet_counts, next_cursor, health}` | [explore.md](explore.md) |
| `/api/likes/toggle` | POST | `{is_liked, like_count, subject_uri}` (JSON) or HTML | [social.md](social.md) |
| `/api/comments` | GET | `{comments[], subject_uri, is_authenticated}` (JSON) or HTML | [social.md](social.md) |
| `/api/comments` | POST | `{comment, comments[]}` (JSON) or HTML | [social.md](social.md) |
| `/api/comments/{id}` | DELETE | `{deleted: true}` (JSON) or HX-Trigger | [social.md](social.md) |
| `/api/report` | POST | `{report_id, submitted}` (JSON) or HTML | [social.md](social.md) |
| `/api/settings` | GET | `{profile_stats_visibility, user_preferences, bluesky_profile}` | [settings.md](settings.md) |
| `/api/settings/preferences` | POST | `{saved: true}` (JSON) or HTML | [settings.md](settings.md) |
| `/api/settings/profile-visibility` | POST | `{saved: true}` (JSON) or HTML | [settings.md](settings.md) |
| `/api/onboarding` | GET | `{readiness, beans[], brewers[], ...}` | [onboarding.md](onboarding.md) |
| `/api/incomplete-records` | GET | `{records[]}` (JSON) or HTML | [onboarding.md](onboarding.md) |
| `/api/popular-recipes` | GET | `[Recipe]` (JSON) or HTML | [onboarding.md](onboarding.md) |
| `/api/{entity}/{actor}/{id}/backlinks` | GET | `{entity_noun, entity_name, result}` | [backlinks.md](backlinks.md) |
| `/api/_mod` | GET | `{hidden_records, reports, audit_log, ...}` | [admin.md](admin.md) |
| `/api/_mod/stats` | GET | `{stats, backups}` | [admin.md](admin.md) |
| `/_mod/*` | POST | `{ok, action, message}` (JSON) or HX-Trigger | [admin.md](admin.md) |
| `/api/signup/categories` | GET | `{categories: [{title, description, providers: [...], dev_only}]}` | P1.14 |

## Endpoints to Add (Phase 1)

See the migration plan (`docs/plans/2026-07-08-sveltekit-migration.md`) for
the full list and priority order. Each endpoint's shape is documented in
its own file as it is implemented:

- `feed.md` — `GET /api/feed` (JSON feed items) **[done]**
- `entities.md` — `GET /api/{entity}/{actor}/{id}` (entity view data) **[done]**
- `brews.md` — `POST /brews`, `PUT /brews/{id}` (brew mutations) **[done]**
- `manage.md` — `GET /api/manage` (records + stats) **[done]**
- `brew_list.md` — `GET /api/brews` (paginated brew list) **[done]**
- `profile.md` — `GET /api/profile/{actor}` (profile data bundle) **[done]**
- `notifications.md` — `GET /api/notifications` **[done]**
- `explore.md` — `GET /api/explore` **[done]**
- `social.md` — likes, comments, reports (JSON mutations) **[done]**
- `settings.md` — `GET /api/settings`, settings mutations **[done]**
- `onboarding.md` — `GET /api/onboarding`, incomplete records **[done]**
- `backlinks.md` — `GET /api/{entity}/{actor}/{id}/backlinks` **[done]**
- `admin.md` — moderation admin endpoints **[done]**
- `signup.md` — `GET /api/signup/categories` (PDS provider catalog) **[done]**
