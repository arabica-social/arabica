# ADR-007: Single-App Backend Composition

**Status:** Accepted
**Recorded:** 2026-11-05
**Decision period:** 2026-11
**Supersedes:** ADR-004

## Context

The sister application has been removed. Arabica is the only production app in
this repository, but the backend still routes configuration and dependencies
through extension points created to host several products.

## Decision Drivers

- Keep HTTP behavior and its dependencies locally discoverable.
- Remove duplicate startup and product-selection paths.
- Preserve PDS authority, record compatibility, authentication and visibility.

## Decision

The HTTP handler package owns Arabica request behavior. The routing package
registers Arabica endpoints directly and owns the explicit SPA page allowlist.
Server composition supplies concrete dependencies, including the optional AI
service, rather than an app route plugin. Application configuration lives in
`internal/arabica/app`; `internal/server` owns startup. `cmd/arabica` is the
production entrypoint.

Explore constructs its coffee registry directly. Suggestions use a private
static coffee configuration instead of app registration at import time.
Identity and record behavior remain distinct from route and presentation policy.
This decision does not replace ADR-005 with a universal descriptor.

Protocol clients and record-store contracts remain independent of application
composition. Interfaces remain where they support real callers and test
adapters, not hypothetical products. Operational SQLite stores are unaffected.

## Alternatives Considered

### Keep the shared/app extension framework

It preserves a future sister-app option, but adds configuration, callbacks and
runtime assertions to the only deployed product without a present consumer.

### Combine all backend packages

Protocol, caching, moderation and feature behavior still have distinct
responsibilities. A single backend package would spread those responsibilities
rather than remove the obsolete product-selection machinery.

## Consequences

### Positive

- One production startup path and one HTTP composition path.
- Handler helpers no longer need exporting merely to cross a product seam.
- Feature configuration no longer depends on optional app registration.

### Negative

- A future sister app would need a new design based on actual shared needs.
- Moving handlers requires preserving test injection and avoiding router cycles.

## Current Constraints

- Preserve endpoint paths, JSON contracts, CSRF protection and explicit SPA 404s.
- Preserve existing cookies, OAuth scopes, environment names and data paths.
- Do not change persisted NSIDs, record fields, numeric units or references.
- Preserve session/witness/PDS fallback and dirty-collection bypass.
- Keep protocol and record-store contracts independent of application imports.

## Related

- [Architecture inventory](../architecture/INDEX.md)
- [ADR-001](ADR-001-pds-records-are-authoritative.md)
- [ADR-002](ADR-002-layered-local-reads-with-dirty-bypass.md)
- [ADR-005](ADR-005-separate-entity-descriptors-from-record-behavior.md)
- [ADR-006](ADR-006-embedded-spa-with-explicit-route-cutover.md)
