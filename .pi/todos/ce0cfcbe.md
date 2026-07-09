{
  "id": "ce0cfcbe",
  "title": "P3.1: Finish TypeScript type generation (tygo coverage + CI + envelope types)",
  "tags": [],
  "status": "completed",
  "created_at": "2026-07-09T01:49:16.003Z",
  "assigned_to_session": "019f448a-85a1-73ee-8c11-16c0e8e3ce91"
}

- Add `internal/notifications` and `internal/backlinks` packages to tygo.yml
- Verify all Go structs exposed in JSON API responses have generated TS types
- Create `web/src/lib/types/api.ts` with response envelope types (feed pagination, entity view wrapper, etc.) — partially done in entity_view.ts/manage.ts/feed.ts/signup.ts; verify and consolidate
- Wire `just types-check` into CI (create .github/workflows/ci.yml)
- Run `just types-generate` and commit if changed
