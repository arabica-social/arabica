{
  "id": "a2ee9adf",
  "title": "P2.2: Entity view JSON types",
  "tags": [],
  "status": "completed",
  "created_at": "2026-07-08T16:51:16.276Z"
}

Create hand-maintained TS types for the entity view JSON envelope (`EntityViewJSONResponse`), `AuthorSummary`, `SocialDataJSON`, `IndexedComment`, and `backlinks.Result`. These mirror the Go structs in entity_view_json.go and the generated types. Add to `web/src/lib/types/entity_view.ts`.
