# ADR-004: Separate Shared Platform and App-Owned Packages

**Status:** Accepted
**Recorded:** 2026-07-16
**Decision period:** Retrospective

## Context

Arabica and Oolong share AT Protocol, OAuth, caching, firehose, feed, routing,
and server infrastructure while differing in entity sets, vocabulary, routes,
presentation, and some record behavior.

Without an explicit dependency direction, shared packages can gradually import
one app to answer app-specific questions, making the other app a special case
and turning shared infrastructure into coffee-specific code with tea branches.

## Decision Drivers

- Reuse platform infrastructure without merging product domains.
- Prevent app-specific imports and registration side effects from leaking across
  app boundaries.
- Allow Arabica and Oolong to evolve and migrate independently.
- Keep dependency direction mechanically enforceable.
- Avoid duplicating entire server stacks for each app.

## Decision

Shared platform packages must not import `internal/arabica` or
`internal/oolong`. App-owned packages may import shared platform packages.

Shared code learns which records, routes, branding, and store behavior apply
through `domain.App`, descriptors, interfaces, and app-supplied registration.
App-specific handlers, presentation, record codecs, and route policy remain in
the owning app package.

Architecture tests enforce the dependency direction and should be changed only
as part of a deliberate boundary decision.

## Alternatives Considered

### Let shared packages import both apps

This is initially convenient but makes shared behavior depend on package-global
app knowledge and creates repeated Arabica/Oolong conditionals.

### Duplicate all backend infrastructure per app

Duplication preserves isolation but would multiply OAuth, firehose, routing,
feed, cache, testing, and operational work for nearly identical mechanics.

### Put every app concern in one universal descriptor registry

A sufficiently broad registry can eliminate imports, but it becomes a service
locator that owns routes, rendering, codecs, and product policy. That weakens
package boundaries instead of clarifying them.

## Consequences

### Positive

- Shared infrastructure remains genuinely reusable.
- App-specific vocabulary and behavior stay locally discoverable.
- Oolong does not accidentally inherit Arabica entities, routes, or scopes.
- Architecture tests catch dependency drift early.

### Negative

- Some behavior requires explicit interfaces, callbacks, or app configuration.
- Similar app handlers can remain duplicated until a stable shared seam emerges.
- Moving code across the boundary requires judgment rather than maximizing
  deduplication.

## Current Constraints

- Shared `internal/` packages do not add imports of app-owned packages.
- Per-app enabled entities come from app-scoped descriptors, not the global
  registry alone.
- App route registration remains app-owned.
- Prefer a small concrete seam over a broad registry of presentation callbacks.

## Related

- Architecture inventory: [Entities and record contracts](../architecture/entities-and-record-contracts.md), [Interfaces and route ownership](../architecture/interfaces-and-route-ownership.md)
- ADRs: [ADR-005](ADR-005-separate-entity-descriptors-from-record-behavior.md)
- Plans: [Backend consolidation](../plans/2026-05-23-unify-app-backend-plan.md)
