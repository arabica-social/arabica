# Architecture Decision Records

Architecture Decision Records capture durable, cross-cutting technical choices,
their alternatives, and their consequences. They preserve decision history;
current runtime facts belong in [`docs/architecture/`](../architecture/INDEX.md),
and user-visible feature behavior belongs in
[`docs/fdr/`](../fdr/INDEX.md).

Accepted ADRs are not rewritten to record a materially different choice. Create
a new ADR and mark the earlier one superseded. Retrospective ADRs explicitly say
that they document a decision already present in the codebase.

Use [`TEMPLATE.md`](TEMPLATE.md) for new records.

## Decisions

| # | Decision | Status | Recorded |
|---|---|---|---|
| [ADR-001](ADR-001-pds-records-are-authoritative.md) | PDS Records Are Authoritative | Accepted | 2026-07-16 |
| [ADR-002](ADR-002-layered-local-reads-with-dirty-bypass.md) | Layer Local Reads with Dirty-Collection Bypass | Accepted | 2026-07-16 |
| [ADR-003](ADR-003-local-indexes-are-rebuildable-projections.md) | Local Indexes Are Rebuildable Projections | Accepted | 2026-07-16 |
| [ADR-004](ADR-004-shared-platform-and-app-owned-package-boundary.md) | Separate Shared Platform and App-Owned Packages | Accepted | 2026-07-16 |
| [ADR-005](ADR-005-separate-entity-descriptors-from-record-behavior.md) | Separate Entity Descriptors from Record Behavior | Accepted | 2026-07-16 |
| [ADR-006](ADR-006-embedded-spa-with-explicit-route-cutover.md) | Embed the SPA with Explicit Route Cutover | Accepted | 2026-07-16 |
