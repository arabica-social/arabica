{
  "id": "d1ad57ea",
  "title": "P2.0: Port appCache.ts to Svelte store",
  "tags": [],
  "status": "completed",
  "created_at": "2026-07-08T15:50:52.071Z"
}

Port `internal/web/assets/svelte/src/appCache.ts` to `web/src/lib/stores/appCache.ts` as a Svelte store. Reuse the fetch/invalidation logic that talks to `/api/data`. Keep the `window.AppCache` global bridge so existing islands (EntityCombo, BrewFormIsland) keep working during migration.
