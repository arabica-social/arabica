// Shared type for the appCache store. Mirrors the API of the legacy
// `internal/web/assets/svelte/src/appCache.ts` so existing islands can read
// `window.AppCache` without changes during the SvelteKit migration.
export type AppCacheAPI = {
  getData: (forceRefresh?: boolean) => Promise<Record<string, unknown> | null>;
  getCachedData: () => Record<string, unknown> | null;
  refreshCache: (force?: boolean) => Promise<Record<string, unknown> | null>;
  invalidateCache: () => void;
  invalidateAndRefresh: () => Promise<Record<string, unknown> | null>;
  addListener: (callback: (data: Record<string, unknown>) => void) => void;
  removeListener: (callback: (data: Record<string, unknown>) => void) => void;
  init: () => Promise<void>;
  preload: () => Promise<Record<string, unknown> | null>;
  isCacheValid: () => boolean;
};
