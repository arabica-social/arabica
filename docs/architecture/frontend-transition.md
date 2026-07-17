# Frontend Transition

Authoritative sources:

- [`web/svelte.config.js`](../../web/svelte.config.js)
- [`web/src/routes/`](../../web/src/routes/)
- [`internal/web/spa/`](../../internal/web/spa/)
- [`internal/arabica/handlers/routes.go`](../../internal/arabica/handlers/routes.go)

Related decision: [ADR-006](../adr/ADR-006-embedded-spa-with-explicit-route-cutover.md).

## Target Runtime

SvelteKit builds a client-rendered static application with an `index.html`
fallback. The generated assets are embedded in the Go binary. Go serves the
shell and injects server-owned head, session, CSP, trace, and Open Graph data
before the Svelte application starts.

OAuth flows, record mutations, JSON APIs, OG image generation, static asset
serving, security headers, and well-known endpoints remain Go responsibilities.

## Current Ownership

| Area | State | Notes |
|---|---|---|
| Arabica page routes | Transitional | Routes move individually through `SPAOwnedRoutes`; legacy fallbacks coexist |
| JSON APIs | Sturdy and expanding | Shared by the SPA and remaining legacy surfaces; documented under `docs/api/` |
| Templ pages/components | Transitional | Retained for unported pages and fallbacks; avoid expanding without a deliberate reason |
| Svelte islands in legacy pages | Transitional | Reusable components may survive the migration, but island mounting is not the target page architecture |
| Go-owned OAuth/mutations/head/OG | Sturdy | These remain server responsibilities after page migration |
| Visual composition and copy | Squishy within constraints | Must still follow accessibility, `PRODUCT.md`, and `DESIGN.md` |

## Route Cutover Contract

Before adding a page pattern to `SPAOwnedRoutes`:

1. Its SvelteKit route works on direct load, not only client navigation.
2. Required JSON contracts exist and are documented.
3. Authentication-required, unauthorized, not-found, and server-error states
   have deliberate behavior.
4. Head and Open Graph behavior remains correct for shareable pages.
5. Focused backend and frontend tests cover the route's failure boundary.
6. Any retained legacy handler is clearly a fallback rather than a second
   independently evolving implementation.

Use an explicit cutover inventory instead of a catch-all assumption so
individual routes can be reviewed or rolled back independently.

## Removal Conditions

A legacy page or island path can be removed when:

- every route that depended on it is SPA-owned;
- direct-load and browser behavior are covered at the appropriate level;
- no remaining legacy surface still imports it;
- its HTML/HTMX endpoint is not an intentional public or compatibility contract;
- generated Templ code and stale assets are removed in the same change.

## Current Constraints

- Do not treat the current coexistence layer as a permanent requirement.
- Do not cut over a route by serving the shell before its data and error paths
  are ready.
- Keep app vocabulary and entity availability app-scoped when reusing shared
  Svelte components.
- Frontend tests verify presentation and interaction; Go/integration tests
  remain responsible for data, authorization, and JSON contracts.
