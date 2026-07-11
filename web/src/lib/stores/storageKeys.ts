// App-scoped storage key derivation.
//
// Arabica and Oolong share a single SvelteKit build and may run on the same
// origin. All localStorage / sessionStorage keys and custom event names must
// be scoped by the active app name so the two apps never clobber each
// other's caches, themes, or mutation events.

function activeApp(): string {
  return document.body?.dataset?.app || "arabica";
}

/** localStorage key for the app data cache (GET /api/data payload). */
export function dataCacheKey(): string {
  return `${activeApp()}_data_cache`;
}

/** sessionStorage prefix for feed JSON cache entries. */
export function feedCachePrefix(): string {
  return `${activeApp()}_feed_cache:`;
}

/** Custom event name for feed mutation notifications. */
export function feedMutationEvent(): string {
  return `${activeApp()}:feed-mutation`;
}

/** localStorage key for the user's theme preference. */
export function themeStorageKey(): string {
  return `${activeApp()}-theme`;
}
