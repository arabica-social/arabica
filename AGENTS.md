# Arabica Agent Guidance

Read this file first. It contains repository-wide rules and routes deeper
context to the document or skill that owns it.

## Project

Arabica is a coffee brew tracking application built on AT Protocol. User
records live in each user's Personal Data Server (PDS). The application
authenticates with OAuth and performs record CRUD through XRPC calls to the
user's PDS. Platform packages are app-agnostic so the same runtime can host a
sister app via per-app configuration, registries, and interfaces.

## Where Context Lives

- [`docs/GLOSSARY.md`](docs/GLOSSARY.md) — canonical product, AT Protocol,
  architecture, and frontend-transition terminology.
- [`docs/architecture/INDEX.md`](docs/architecture/INDEX.md) — current runtime
  structure, data authority, record contracts, interfaces, and migration state.
- [`docs/adr/INDEX.md`](docs/adr/INDEX.md) — cross-cutting technical decisions
  and their consequences.
- [`docs/fdr/INDEX.md`](docs/fdr/INDEX.md) — current user-visible feature
  behavior and feature-specific rationale.
- [`docs/api/README.md`](docs/api/README.md) — current HTTP and JSON contracts.
- [`docs/plans/`](docs/plans/) — proposed work, migrations, and unresolved
  decisions. Plans are not authoritative descriptions of current behavior.
- [`docs/road-to-v1.md`](docs/road-to-v1.md) — unresolved work targeted before
  the first stable release.
- [`PRODUCT.md`](PRODUCT.md) — audience, emotional goals, and product principles.
- [`DESIGN.md`](DESIGN.md) — visual tokens and reusable interface patterns.
- [`.agents/skills/arabica-record-evolution/SKILL.md`](.agents/skills/arabica-record-evolution/SKILL.md)
  — compatibility workflow for lexicon and persisted-record changes.
- [`.agents/skills/arabica-entity-change/SKILL.md`](.agents/skills/arabica-entity-change/SKILL.md)
  — discovery-based workflow for adding or changing entity capabilities.

## Current Status

Last reviewed: 2026-07-16.

- Arabica is transitioning page routes from Templ/HTMX to an embedded
  SvelteKit SPA. `internal/arabica/handlers/routes.go` is the executable
  route-cutover inventory.
- Breaking brew and recipe lexicon changes remain unresolved before v1. Treat
  existing PDS records as durable compatibility inputs, not centrally
  migratable rows.
- Arabica cafe and drink records remain deferred. Do not infer support from
  historical plans.

## Prime Directives

- **The PDS is authoritative.** Local record indexes and in-memory record caches
  are read optimizations and rebuildable projections, not owners of user
  records. Local operational stores can own sessions and other deployment state.
- **Preserve existing-record compatibility.** Discuss breaking NSID, field,
  reference, type, or unit changes before implementation and use the
  `arabica-record-evolution` skill.
- **Protect user boundaries.** Cross-user reads must respect visibility and
  moderation; mutations must remain scoped to the authenticated user's store.
- **Respect package seams.** Shared packages must not import
  `internal/arabica`. App behavior is supplied through app
  configuration, registries, and interfaces.
- **Keep entity identity lightweight.** Descriptors identify record types;
  codecs, reference hydration, routing, and presentation live in their owning
  layers.
- **Cut SPA routes over explicitly.** Add a route to `SPAOwnedRoutes` only after
  its direct-load path, JSON dependencies, session/error handling, and relevant
  tests exist.
- Prefer simple, clear changes over clever abstractions.
- Prefer standard-library solutions. Add a dependency only when it materially
  improves the result and standard library code is not a reasonable fit.
- Use `jj` for version control. Never push unless the user explicitly asks.

## Change Routing

- Lexicon, record shape, reference, NSID, or numeric-unit change → use
  `arabica-record-evolution` and update compatibility fixtures/tests.
- New entity or changed entity capabilities → use `arabica-entity-change`.
- HTTP/JSON behavior change → update the relevant file in `docs/api/` and its
  integration contract tests.
- Current runtime boundary or data-flow change → update
  `docs/architecture/`; write or supersede an ADR when the rationale is durable
  and cross-cutting.
- Current user-visible feature behavior change → update the relevant FDR.
  Undecided future behavior remains in `docs/plans/`.
- New or renamed canonical concept → update `docs/GLOSSARY.md`. The glossary
  governs human-facing terminology; stable NSIDs, API paths, and persisted
  identifiers remain compatibility contracts.
- Product or visual-principle change → update `PRODUCT.md` or `DESIGN.md`.

## Verification

Run the narrowest checks that can catch regressions in the changed area, then
broaden when the risk crosses layers.

| Change | Useful checks |
|---|---|
| Go/domain logic | `just test` or a targeted `go test` |
| Integration/API contract | `just integration-test` |
| Svelte components/routes | `pnpm run check:svelte`, `pnpm run test:svelte` |
| Browser and cross-route behavior | `just e2e` or a targeted Playwright spec |
| Broad local CI checkpoint | `just ci-check` |
| Formatting | `just format` |

Never claim full verification when only a partial signal was run.

## Conditional Rules

- `.templ` files use tabs. After editing them, run `templ generate` (or
  `just templ-generate`) and include generated Go changes.
- Do not expand legacy Templ/HTMX surfaces during the SPA migration unless the
  user explicitly chooses that direction or a working fallback requires it.
- Frontend work should reuse established components and the patterns in
  `PRODUCT.md` and `DESIGN.md`; accessibility constraints are not "squishy"
  visual preferences.
