// App-scoped storage key derivation.
//
// The Go shell injects data-app on <body>. All localStorage / sessionStorage
// keys and custom event names are scoped by the active app name so caches,
// themes, and mutation events never collide across apps that might share an
// origin.

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
