{
  "id": "b3cd5f4b",
  "title": "Register SPA-owned entity form routes with the mux",
  "tags": [],
  "status": "completed",
  "created_at": "2026-07-10T03:22:08.625Z"
}

The entity new/edit routes (`/beans/new`, `/roasters/new`, `/grinders/new`, `/brewers/new`, `/recipes/new`, and their `/{id}/edit` variants) are listed in `SPAOwnedRoutes()` (internal/arabica/handlers/routes.go) but never registered with `pages.Register`. They fall through to the 404 handler instead of serving the SvelteKit SPA shell. E2E manage-entities test fails at `/roasters/new` because `body[data-frontend="sveltekit"]` is absent.

Fix: register each of these patterns via `ctx.Pages.Register(mux, pattern, legacyHandler)` in `RegisterAppRoutes`. Since these have no legacy full-page handlers (roaster/bean/grinder/brewer/recipe new+edit were modal-only in the templ stack), pass a nil/NotFound legacy handler — `pages.Register` routes SPA-owned patterns to the SPA handler regardless of the legacy arg. Verify the SPA shell is served for each.
