// Client record cache for the SvelteKit SPA.
//
// This is the SvelteKit port of `internal/web/assets/svelte/src/appCache.ts`.
// It talks to the same `GET /api/data` endpoint and uses the same localStorage
// envelope (same key, version, and TTL) so the cache survives a page reload
// regardless of which shell rendered it.
//
// Two access patterns coexist during migration:
//
//   1. SvelteKit routes import `appCache` and call `getData()` / `invalidate()`.
//   2. Existing Svelte islands (EntityCombo, BrewFormIsland, …) read
//      `window.AppCache`. The layout installs the same object onto
//      `window.AppCache` so islands keep working unchanged until they are
//      ported to SvelteKit components.

import { writable, type Writable } from "svelte/store";
import type { AppCacheAPI } from "../types/appCache";

type CacheEnvelope = {
  version: number;
  timestamp: number;
  did: string | null;
  app: string | null;
  data: Record<string, unknown>;
};

const CACHE_KEY = "arabica_data_cache";
const CACHE_VERSION = 1;
const CACHE_TTL_MS = 5 * 60 * 1000;

function getCurrentUserDID() {
  return document.body?.dataset?.userDid || null;
}

function getCurrentApp() {
  return document.body?.dataset?.app || null;
}

function getCache(): CacheEnvelope | null {
  try {
    const raw = localStorage.getItem(CACHE_KEY);
    if (!raw) return null;

    const cache = JSON.parse(raw) as CacheEnvelope;
    if (cache.version !== CACHE_VERSION) {
      localStorage.removeItem(CACHE_KEY);
      return null;
    }

    return cache;
  } catch (error) {
    console.warn("Failed to read cache:", error);
    localStorage.removeItem(CACHE_KEY);
    return null;
  }
}

function setCache(data: Record<string, unknown>) {
  try {
    const cache: CacheEnvelope = {
      version: CACHE_VERSION,
      timestamp: Date.now(),
      did: (data.did as string) || null,
      app: getCurrentApp(),
      data,
    };
    localStorage.setItem(CACHE_KEY, JSON.stringify(cache));
  } catch (error) {
    console.warn("Failed to write cache:", error);
  }
}

function getCachedDID() {
  return getCache()?.did || null;
}

function invalidateCacheStorage() {
  localStorage.removeItem(CACHE_KEY);
}

function isCacheValid() {
  const cache = getCache();
  if (!cache) return false;

  const currentDID = getCurrentUserDID();
  if (currentDID && cache.did && currentDID !== cache.did) return false;
  const currentApp = getCurrentApp();
  if (currentApp && cache.app && currentApp !== cache.app) return false;

  return Date.now() - cache.timestamp < CACHE_TTL_MS;
}

function getCachedData() {
  const cache = getCache();
  if (!cache) return null;

  const currentDID = getCurrentUserDID();
  if (currentDID && cache.did && currentDID !== cache.did) {
    console.log("Cache belongs to different user, invalidating");
    invalidateCacheStorage();
    return null;
  }

  const currentApp = getCurrentApp();
  if (currentApp && cache.app && currentApp !== cache.app) {
    invalidateCacheStorage();
    return null;
  }

  return cache.data;
}

async function fetchFreshData() {
  const response = await fetch("/api/data", {
    credentials: "same-origin",
    headers: {
      "X-Page-Context": window.location.pathname,
    },
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch data: ${response.status}`);
  }

  return (await response.json()) as Record<string, unknown>;
}

// --- store-backed singleton ---

// Reactive state for components that want to track the cache. Updated on
// refresh and invalidation.
export const appCacheData: Writable<Record<string, unknown> | null> =
  writable<Record<string, unknown> | null>(getCachedData());
export const appCacheLoading = writable(false);
export const appCacheError = writable<string | null>(null);

class AppCacheStore implements AppCacheAPI {
  private isRefreshing = false;
  private listeners = new Set<(data: Record<string, unknown>) => void>();
  private waiters: Array<() => void> = [];

  getData = async (
    forceRefresh = false,
  ): Promise<Record<string, unknown> | null> => {
    if (!forceRefresh && isCacheValid()) {
      const cached = getCachedData();
      appCacheData.set(cached);
      return cached;
    }

    const cached = getCachedData();
    try {
      return await this.refreshCache();
    } catch (error) {
      console.warn("Failed to refresh cache:", error);
      if (cached) return cached;
      throw error;
    }
  };

  getCachedData = (): Record<string, unknown> | null => {
    return getCachedData();
  };

  refreshCache = async (
    force = false,
  ): Promise<Record<string, unknown> | null> => {
    if (this.isRefreshing) {
      await new Promise<void>((resolve) => {
        this.waiters.push(resolve);
        const checkInterval = window.setInterval(() => {
          if (!this.isRefreshing) {
            window.clearInterval(checkInterval);
            resolve();
          }
        }, 100);
      });

      if (!force) {
        return getCachedData();
      }
    }

    this.isRefreshing = true;
    appCacheLoading.set(true);
    try {
      const data = await fetchFreshData();
      const cachedDID = getCachedDID();
      if (cachedDID && data.did && cachedDID !== data.did) {
        console.log("User changed, clearing stale cache");
        invalidateCacheStorage();
      }

      setCache(data);
      appCacheData.set(data);
      appCacheError.set(null);
      this.notifyListeners(data);
      return data;
    } catch (error) {
      appCacheError.set(error instanceof Error ? error.message : String(error));
      throw error;
    } finally {
      this.isRefreshing = false;
      appCacheLoading.set(false);
      const waiters = this.waiters.splice(0);
      waiters.forEach((resolve) => resolve());
    }
  };

  invalidateCache = (): void => {
    invalidateCacheStorage();
    appCacheData.set(null);
  };

  invalidateAndRefresh = async (): Promise<Record<string, unknown> | null> => {
    this.invalidateCache();
    return await this.refreshCache(true);
  };

  addListener = (callback: (data: Record<string, unknown>) => void): void => {
    this.listeners.add(callback);
  };

  removeListener = (callback: (data: Record<string, unknown>) => void): void => {
    this.listeners.delete(callback);
  };

  private notifyListeners(data: Record<string, unknown>) {
    this.listeners.forEach((callback) => {
      try {
        callback(data);
      } catch (error) {
        console.warn("Cache listener error:", error);
      }
    });
  }

  init = async (): Promise<void> => {
    if (!isCacheValid()) {
      try {
        await this.refreshCache();
      } catch (error) {
        console.warn("Initial cache load failed:", error);
      }
    } else {
      appCacheData.set(getCachedData());
    }
  };

  preload = async (): Promise<Record<string, unknown> | null> => {
    return await this.refreshCache();
  };

  isCacheValid = (): boolean => {
    return isCacheValid();
  };
}

// Singleton — one cache for the whole app.
export const appCache = new AppCacheStore();

// Install the singleton onto `window.AppCache` so existing Svelte islands
// (EntityCombo, BrewFormIsland, …) that read `window.AppCache` keep working
// during the migration. Safe to call multiple times.
export function installAppCacheGlobal() {
  (window as unknown as { AppCache?: AppCacheAPI }).AppCache = appCache;
}
