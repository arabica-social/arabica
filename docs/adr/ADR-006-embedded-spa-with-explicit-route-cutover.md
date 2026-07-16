# ADR-006: Embed the SPA with Explicit Route Cutover

**Status:** Accepted
**Recorded:** 2026-07-16
**Decision period:** Retrospective

## Context

Arabica is migrating from server-rendered Templ/HTMX pages and Svelte islands to
full SvelteKit page routes. The application should remain a Go-hosted deployment
without requiring a Node runtime, while existing routes must continue working
during an incremental migration. Arabica and Oolong also need to migrate at
different rates despite sharing frontend code.

## Decision Drivers

- Preserve a single Go-served deployment artifact.
- Add testable frontend page state without requiring SvelteKit SSR.
- Migrate one route at a time with working legacy fallbacks.
- Keep OAuth, mutations, security headers, and Open Graph behavior in Go.
- Let Arabica and Oolong own independent route-cutover inventories.

## Decision

Build SvelteKit as a client-rendered static SPA with an `index.html` fallback
and embed the generated assets in the Go binary.

Go serves the SPA shell and injects server-controlled head, session, security,
trace, and Open Graph data. It continues to own OAuth, JSON and mutation
endpoints, static serving, and OG image generation.

Each app explicitly lists page patterns owned by the SPA. A route enters
`SPAOwnedRoutes` only after its direct-load SvelteKit path, JSON dependencies,
authentication/error behavior, and relevant tests exist. Unlisted routes retain
legacy handling during migration.

## Alternatives Considered

### Rewrite every page before switching to SvelteKit

A single cutover avoids coexistence code, but creates a large high-risk rewrite
with no route-level rollback and delays frontend testing benefits.

### Run SvelteKit with a Node SSR server

SSR provides server rendering and native SvelteKit request handling, but adds a
second production runtime and duplicates responsibilities already owned by Go.

### Keep Templ pages with isolated Svelte components indefinitely

This avoids migration work but preserves split rendering ownership and the
implicit JSON/HTML contracts that motivated the transition.

### Serve every page through the SPA shell immediately

This is simple at the router but makes unimplemented direct loads fail and
forces Arabica and Oolong to migrate in lockstep.

## Consequences

### Positive

- Production remains Go-hosted with embedded frontend assets.
- Routes can migrate, test, and roll back independently.
- Backend and frontend contracts become more explicit.
- Arabica and Oolong can share components without sharing route-completion state.

### Negative

- Legacy and SPA paths coexist temporarily.
- Some pages retain duplicate handlers or compatibility endpoints during
  transition.
- Client rendering requires deliberate loading, session, and error states.
- Frontend asset changes require rebuilding the embedded artifact.

## Current Constraints

- Direct-load behavior is required before route ownership changes.
- JSON contracts and integration tests remain authoritative for backend data
  behavior.
- Do not expand legacy page architecture without a deliberate reason.
- Do not remove shared legacy code while Oolong or an unported route still uses
  it.
- Go remains responsible for OAuth, mutations, security/head injection, assets,
  and OG surfaces unless superseded by a new decision.

## Related

- Architecture inventory: [Interfaces and route ownership](../architecture/interfaces-and-route-ownership.md), [Frontend transition](../architecture/frontend-transition.md)
- Plans: [SvelteKit migration](../plans/2026-07-08-sveltekit-migration.md)
