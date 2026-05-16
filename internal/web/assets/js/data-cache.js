/**
 * Client-side data cache for Arabica
 * Caches beans, roasters, grinders, and brewers in localStorage
 * to reduce PDS round-trips on page loads.
 *
 * Data is fetched on page load (if stale) and after mutations.
 * No background polling — callers invalidate explicitly after CRUD ops.
 */

const CACHE_KEY = "arabica_data_cache";
const CACHE_VERSION = 1;
const CACHE_TTL_MS = 5 * 60 * 1000; // 5 minutes

// Module state
let isRefreshing = false;
let listeners = [];

/**
 * Get the current cache from localStorage
 */
function getCache() {
  try {
    const raw = localStorage.getItem(CACHE_KEY);
    if (!raw) return null;

    const cache = JSON.parse(raw);

    // Check version
    if (cache.version !== CACHE_VERSION) {
      localStorage.removeItem(CACHE_KEY);
      return null;
    }

    return cache;
  } catch (e) {
    console.warn("Failed to read cache:", e);
    localStorage.removeItem(CACHE_KEY);
    return null;
  }
}

/**
 * Save data to the cache
 * Stores the user DID alongside the data for validation
 */
function setCache(data) {
  try {
    const cache = {
      version: CACHE_VERSION,
      timestamp: Date.now(),
      did: data.did || null, // Store user DID for cache validation
      app: getCurrentApp(), // Store running app so cross-app loads invalidate
      data: data,
    };
    localStorage.setItem(CACHE_KEY, JSON.stringify(cache));
  } catch (e) {
    console.warn("Failed to write cache:", e);
  }
}

/**
 * Get the DID stored in the cache
 */
function getCachedDID() {
  const cache = getCache();
  return cache?.did || null;
}

/**
 * Check if cache is valid (exists and not expired)
 */
function isCacheValid() {
  const cache = getCache();
  if (!cache) return false;

  // A cache from a different app or user is treated as invalid so callers
  // re-fetch via the running app's /api/data handler.
  const currentDID = getCurrentUserDID();
  if (currentDID && cache.did && currentDID !== cache.did) return false;
  const currentApp = getCurrentApp();
  if (currentApp && cache.app && currentApp !== cache.app) return false;

  const age = Date.now() - cache.timestamp;
  return age < CACHE_TTL_MS;
}

/**
 * Get the current user's DID from the page
 */
function getCurrentUserDID() {
  return document.body?.dataset?.userDid || null;
}

/**
 * Get the running app's identifier from the page (e.g. "arabica", "oolong").
 * The /api/data endpoint returns different entity shapes per app, so the
 * cache must invalidate when the user navigates across apps.
 */
function getCurrentApp() {
  return document.body?.dataset?.app || null;
}

/**
 * Get cached data if available and valid for the current user
 */
function getCachedData() {
  const cache = getCache();
  if (!cache) return null;

  // Validate that cached data belongs to the current user
  const currentDID = getCurrentUserDID();
  const cachedDID = cache.did;

  // If we have both DIDs and they don't match, cache is invalid
  if (currentDID && cachedDID && currentDID !== cachedDID) {
    console.log("Cache belongs to different user, invalidating");
    invalidateCache();
    return null;
  }

  // Each app (arabica, oolong) returns a different entity shape from /api/data.
  // If the cache was populated by a different app, the entity keys won't match
  // what callers expect, so drop it and force a refresh.
  const currentApp = getCurrentApp();
  const cachedApp = cache.app;
  if (currentApp && cachedApp && currentApp !== cachedApp) {
    invalidateCache();
    return null;
  }

  // Return data even if expired - caller can decide to refresh
  return cache.data;
}

/**
 * Fetch fresh data from the API
 */
async function fetchFreshData() {
  const headers = {
    "X-Page-Context": window.location.pathname,
  };

  const response = await fetch("/api/data", {
    credentials: "same-origin",
    headers,
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch data: ${response.status}`);
  }

  return await response.json();
}

/**
 * Refresh the cache from the API
 * Returns the fresh data
 * @param {boolean} force - If true, always fetch fresh data even if a refresh is in progress
 */
async function refreshCache(force = false) {
  if (isRefreshing) {
    // Wait for existing refresh to complete
    await new Promise((resolve) => {
      const checkInterval = setInterval(() => {
        if (!isRefreshing) {
          clearInterval(checkInterval);
          resolve();
        }
      }, 100);
    });

    // If not forcing, return the cached data from the completed refresh
    if (!force) {
      return getCachedData();
    }
    // Otherwise, continue to do a new refresh with fresh data
  }

  isRefreshing = true;
  try {
    const data = await fetchFreshData();

    // Check if user changed (different DID)
    const cachedDID = getCachedDID();
    if (cachedDID && data.did && cachedDID !== data.did) {
      console.log("User changed, clearing stale cache");
      invalidateCache();
    }

    setCache(data);
    notifyListeners(data);
    return data;
  } finally {
    isRefreshing = false;
  }
}

/**
 * Get data - returns cached if valid, otherwise fetches fresh
 * @param {boolean} forceRefresh - Force a refresh even if cache is valid
 */
async function getData(forceRefresh = false) {
  if (!forceRefresh && isCacheValid()) {
    return getCachedData();
  }

  // Try to get cached data while refreshing
  const cached = getCachedData();

  try {
    return await refreshCache();
  } catch (e) {
    console.warn("Failed to refresh cache:", e);
    // Return stale data if available
    if (cached) {
      return cached;
    }
    throw e;
  }
}

/**
 * Invalidate the cache (call after CRUD operations)
 */
function invalidateCache() {
  localStorage.removeItem(CACHE_KEY);
}

/**
 * Invalidate and immediately refresh the cache
 * Forces a fresh fetch even if a background refresh is in progress
 */
async function invalidateAndRefresh() {
  invalidateCache();
  return await refreshCache(true);
}

/**
 * Register a listener for cache updates
 * @param {function} callback - Called with new data when cache is refreshed
 */
function addListener(callback) {
  listeners.push(callback);
}

/**
 * Remove a listener
 */
function removeListener(callback) {
  listeners = listeners.filter((l) => l !== callback);
}

/**
 * Notify all listeners of new data
 */
function notifyListeners(data) {
  listeners.forEach((callback) => {
    try {
      callback(data);
    } catch (e) {
      console.warn("Cache listener error:", e);
    }
  });
}

/**
 * Initialize the cache - call on page load for pages that need entity data.
 * Fetches once if cache is stale. No background polling.
 */
async function init() {
  if (!isCacheValid()) {
    try {
      await refreshCache();
    } catch (e) {
      console.warn("Initial cache load failed:", e);
    }
  }
}

/**
 * Preload cache - useful to call after login
 */
async function preload() {
  return await refreshCache();
}

// Invalidate caches when entities are created/updated/deleted
document.addEventListener("DOMContentLoaded", () => {
  document.body.addEventListener("refreshManage", () => {
    invalidateCache();
    // Clear HTMX history cache so navigating to other pages fetches fresh data
    try {
      localStorage.removeItem("htmx-history-cache");
    } catch (e) {
      // ignore
    }
  });

  // Listen for entityDeleted events triggered by HX-Trigger response headers.
  // This covers all HTMX delete flows (brew list, entity tables, action bar,
  // comments) regardless of which page they originate from.
  document.body.addEventListener("entityDeleted", () => {
    invalidateCache();
  });
});

// Export as global for use in other scripts
window.ArabicaCache = {
  getData,
  getCachedData,
  refreshCache,
  invalidateCache,
  invalidateAndRefresh,
  addListener,
  removeListener,
  init,
  preload,
  isCacheValid,
};
