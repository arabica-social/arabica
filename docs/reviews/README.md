# Backend code quality reviews

These documents are thermo-nuclear code-quality review passes over the backend
packages. They focus on implementation quality, maintainability, abstraction
boundaries, and structural simplification opportunities rather than small style
nits.

Review shards:

- [`domain-model-packages.md`](domain-model-packages.md) — entity models,
  lexicon/domain metadata, suggestions, matching, moderation, and OG-card domain
  rendering.
- [`infrastructure-data-packages.md`](infrastructure-data-packages.md) —
  ATProto store, database boundaries, firehose/feed infrastructure, backup,
  logging, metrics, tracing, and server runtime setup.
- [`api-orchestration-packages.md`](api-orchestration-packages.md) — command
  entrypoints, routing, shared handlers, app handlers, middleware, signup,
  onboarding, BFF, and backend tests.
- [`package-seams-followup.md`](package-seams-followup.md) — additional worker
  passes focused specifically on package seams, code organization, package
  ownership lines, app/shared leakage, and abstraction boundaries.

