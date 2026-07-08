{
  "id": "dcbaca47",
  "title": "P1.1 — Feed JSON endpoint (GET /api/feed)",
  "tags": [
    "stream-a",
    "p1.1",
    "feed"
  ],
  "status": "done",
  "created_at": "2026-07-08T15:50:57.090Z"
}

Add JSON response path to feed handler. Drop RequireHTMXMiddleware gate via content negotiation (Accept: application/json) or a dedicated JSON path. Response shape: {items, next_cursor, is_authenticated, query}. Reuse feedService.GetFeedWithQuery / GetCachedPublicFeed. Update docs/api/feed.md. Add integration test.
