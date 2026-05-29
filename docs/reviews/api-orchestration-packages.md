# API/orchestration package review

## Scope

Reviewed current backend code in:

- `cmd/arabica`
- `cmd/oolong`
- `internal/arabica/handlers`
- `internal/oolong/handlers`
- `internal/handlers`
- `internal/routing`
- `internal/middleware`
- `internal/signup`
- `internal/onboarding`
- `internal/web/bff`
- backend-relevant tests under `tests/`

## Strongest structural findings

### 1. Routing is still the app-composition bottleneck

Evidence:

- `internal/routing/routing.go` imports both app handler packages at lines `8`
  and `15`.
- The router branches on `cfg.App.Name` around lines `76`, `94`, `122`, `166`,
  `180`, and `222`.
- The codebase already has a better local pattern:
  `internal/handlers/entity_routes.go:9-13` defines `EntityRouteBundle`, and
  `internal/arabica/handlers/routes.go:8-11` plus
  `internal/oolong/handlers/routes.go:8-12` use that route-bundle pattern.

Impact:

- Every new app-specific backend surface still requires central router edits.
- The router undercuts the `EntityRouteBundle` abstraction by retaining direct
  app knowledge.

Code-judo move:

- Move app-specific route registration behind a per-app registrar on or near
  `domain.App`.
- Keep `routing.SetupRouter` responsible only for shared middleware, static
  assets, OAuth/common routes, and invoking the selected app registrar.

Risk: medium-high architectural drift. This will become painful with a third app
or another large Oolong surface.

Approval bar: acceptable for two apps; not acceptable as the long-term app
composition boundary.

---

### 2. Shared handler state has hidden app defaults

Evidence:

- `internal/handlers/handlers.go:70-71` stores app config as optional mutable
  state.
- `internal/handlers/handlers.go:93` sets the app separately with `SetApp`.
- Comments around `internal/handlers/handlers.go:98-101` describe legacy
  fallback behavior to `"arabica"`.
- `internal/handlers/handlers.go:126-131` makes `appNSIDs` depend on `SetApp`.

Impact:

- Tests or alternate bootstrap paths can silently run with coffee defaults.
- App identity is a core invariant, but the API models it as optional mutable
  state.

Code-judo move:

- Require `*domain.App` in the base handler constructor.
- Delete implicit app fallback behavior.
- Treat app config as immutable after construction.

Risk: medium. This is a boundary cleanliness issue with real potential for
cross-app bugs.

Approval bar: should be tightened before adding more app-specific shared handler
logic.

---

### 3. Request decoding is too permissive for central API plumbing

Evidence:

- `internal/handlers/handlers.go:247` detects JSON by content type.
- `internal/handlers/handlers.go:256-258` decodes JSON with plain
  `json.NewDecoder(r.Body).Decode`.
- `internal/handlers/handlers.go:263-266` treats all non-JSON requests as forms.

Impact:

- Trailing JSON payloads, unbounded bodies, unknown fields, and accidental
  content-type mistakes are handled inconsistently across backend APIs.
- This loose boundary invites handler-specific validation patches instead of one
  robust request contract.

Code-judo move:

- Replace `DecodeRequest` with a small request codec that enforces explicit
  media types, max body size, a single JSON document, and optional unknown-field
  rejection.
- Keep form decoding explicit for handlers that genuinely accept forms.

Risk: medium. Not necessarily exploitable as-is, but the boundary is too vague
for central API plumbing.

Approval bar: should be hardened before broad API reuse.

---

### 4. CRUD abstractions are duplicated by app instead of descriptor-driven

Evidence:

- Arabica has generic CRUD helpers in
  `internal/arabica/handlers/crud_generic.go:27` and `:68`.
- Oolong has parallel helpers in `internal/oolong/handlers/crud_generic.go:33`
  and record-writing in `internal/oolong/handlers/crud.go:47`.

Impact:

- App packages can evolve different validation/write/error behavior.
- Shared mechanics are copied while the actual domain differences are smaller
  than the duplication suggests.

Code-judo move:

- Create one shared typed CRUD adapter parameterized by descriptor, request
  validator, record conversion, and store operations.
- Leave app packages as thin registrations of their entity-specific behavior.

Risk: medium. Duplication is manageable now, but it will compound with each new
entity.

Approval bar: acceptable temporarily; refactor before more CRUD surfaces are
added.

---

### 5. Test convention drift exists in backend tests

Evidence:

- `internal/middleware/logging_test.go:25`, `:41`, and `:91` use
  `t.Error`/`t.Errorf` despite the project convention requiring
  `testify/assert`.

Impact:

- Low runtime risk, but it weakens consistency in the test suite.
- This is not a structural blocker, but it should be cleaned while touching
  middleware tests.

Code-judo move:

- Convert the file to `assert`/`require` style when middleware tests are next
  edited.

Risk: low.

Approval bar: not a blocker by itself.

## Positive findings

- `internal/handlers/entity_routes.go:9-13` is a good route bundle abstraction.
- `internal/handlers/entity_views.go:76-98` centralizes entity view rendering
  behind `EntityViewConfig`.
- `cmd/arabica/app_test.go:12,32` and `cmd/oolong/app_test.go:10,29` guard app
  NSID/scope configuration.

## Overall approval bar

The API/orchestration layer has useful local abstractions, but app composition is
still too centralized in routing and shared handler state. The best
simplification is to make app identity and app route registration explicit,
immutable construction-time concerns rather than optional mutable state and
central `cfg.App.Name` branching.

