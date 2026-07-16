# Interfaces and Route Ownership

Authoritative sources:

- [`internal/routing/`](../../internal/routing/)
- [`internal/handlers/`](../../internal/handlers/)
- [`internal/arabica/handlers/routes.go`](../../internal/arabica/handlers/routes.go)
- [`internal/oolong/handlers/routes.go`](../../internal/oolong/handlers/routes.go)
- [`docs/api/`](../api/README.md)

Related decisions: [ADR-004](../adr/ADR-004-shared-platform-and-app-owned-package-boundary.md)
and [ADR-006](../adr/ADR-006-embedded-spa-with-explicit-route-cutover.md).

## External Boundaries

| Surface | Responsibility | Authority/contract |
|---|---|---|
| AT Protocol OAuth | Authenticate users and establish scoped PDS access | OAuth client/session configuration |
| XRPC to a user's PDS | Authoritative record CRUD | App lexicons and PDS behavior |
| Public AT Protocol reads | Resolve public records and profiles across users | Public PDS/XRPC responses plus visibility rules |
| Go HTTP pages | Legacy Templ pages or the embedded SPA shell | Route registration and page ownership |
| Go JSON APIs | Data consumed by SvelteKit and legacy clients | [`docs/api/`](../api/README.md) plus integration tests |
| Mutation endpoints | CSRF-protected creates, updates, deletes, and actions | Handler validation, authenticated store, and JSON response contracts |
| OG image/head surfaces | Link previews, page titles, and social metadata | Go handlers and SPA shell injection |

## Route Ownership

Shared routing infrastructure owns common mechanics, middleware, and the choice
between SPA and legacy page handlers. Each app package owns its app-specific
route registration and page-cutover inventory.

Arabica's `Routes.SPAOwnedRoutes()` is the executable list of page patterns
served by the SvelteKit shell. A listed route must have a working direct-load
SvelteKit path and all required JSON/session/error behavior. Unlisted routes
retain legacy handling.

Oolong has a separate inventory and can migrate independently. Shared frontend
code does not imply shared route ownership or identical enabled entities.

JSON, mutation, OG image, modal, and compatibility endpoints remain independent
of which frontend owns the corresponding page. A page cutover should not
silently remove a backend contract still used by another surface.

## API Documentation Boundary

`docs/api/` owns current endpoint, parameter, and response-envelope details.
This inventory records only the transport and ownership boundaries. API changes
should update focused integration tests so the contract is executable rather
than relying on prose alone.

## Current Constraints

- Mutations use the authenticated user's store and retain CSRF protection where
  applicable.
- Route cutover is explicit and reversible during migration.
- A shared package must not import app handlers to register app-specific
  behavior.
- Direct loads, in-app navigation, unauthenticated behavior, and error states
  are part of page-route correctness.
- Public/cross-user reads must preserve moderation, visibility, and ownership
  semantics regardless of whether data came from a PDS or local witness index.
