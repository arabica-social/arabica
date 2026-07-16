# ADR-005: Separate Entity Descriptors from Record Behavior

**Status:** Accepted
**Recorded:** 2026-07-16
**Decision period:** Retrospective

## Context

Shared infrastructure needs lightweight identity information for each record
type, while feed conversion, forms, record decoding, and reference hydration
need richer app-specific behavior. Combining these concerns in one descriptor
encourages the domain registry to import typed app models, routes, and rendering
frameworks.

That makes a convenient registry progressively responsible for every layer of
the application and weakens the shared/app boundary.

## Decision Drivers

- Keep record identity usable from shared domain and routing code.
- Keep app-specific typed behavior in app-owned packages.
- Avoid Templ or frontend dependencies in the identity registry.
- Allow behavior to grow without turning descriptors into service locators.
- Make the boundary enforceable with architecture tests.

## Decision

Keep `entities.Descriptor` limited to record identity metadata: record type,
collection NSID, and display name.

Register `RecordBehavior` separately for record decoding, form-field extraction,
record key/title access, reference declarations, and reference hydration. Keep
encoding, validation, route policy, edit actions, and feed/UI rendering in their
owning app, routing, or presentation layers.

## Alternatives Considered

### Put all entity behavior on the descriptor

This makes generic loops easy to write, but the descriptor becomes a global
service locator containing codecs, routes, callbacks, and framework types.

### Use type switches throughout shared packages

Type switches avoid a registry but require shared packages to know every app
model and make adding a sister-app entity a cross-cutting edit.

### Create one registry per layer with no shared identity

Strong separation avoids an overloaded descriptor but duplicates record type,
NSID, and naming metadata with no canonical identity mapping.

## Consequences

### Positive

- Shared identity metadata remains small and framework-independent.
- App packages own typed codecs and hydration behavior.
- Architecture tests can prevent route and presentation concerns from leaking
  into descriptors.
- New behaviors can be introduced without expanding every descriptor consumer.

### Negative

- Adding an entity can require coordinated registration in more than one place.
- Callers must deliberately choose whether they need identity or behavior.
- Some generic features need an additional app-owned configuration layer rather
  than one descriptor callback.

## Current Constraints

- Descriptors do not own route nouns, URLs, form callbacks, feed actions, or
  rendering components.
- The entity registry does not import Templ.
- Record behavior remains app-owned when it depends on app record models.
- Presentation-specific behavior stays outside both identity and codec
  registries.

## Related

- Architecture inventory: [Entities and record contracts](../architecture/entities-and-record-contracts.md)
- ADRs: [ADR-004](ADR-004-shared-platform-and-app-owned-package-boundary.md)
