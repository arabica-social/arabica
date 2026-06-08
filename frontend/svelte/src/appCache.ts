type CacheEnvelope = {
  version: number;
  timestamp: number;
  did: string | null;
  app: string | null;
  data: Record<string, any>;
};

export type AppCacheAPI = {
  getData: (forceRefresh?: boolean) => Promise<Record<string, any> | null>;
  getCachedData: () => Record<string, any> | null;
  refreshCache: (force?: boolean) => Promise<Record<string, any> | null>;
  invalidateCache: () => void;
  invalidateAndRefresh: () => Promise<Record<string, any> | null>;
  addListener: (callback: (data: Record<string, any>) => void) => void;
  removeListener: (callback: (data: Record<string, any>) => void) => void;
  init: () => Promise<void>;
  preload: () => Promise<Record<string, any> | null>;
  isCacheValid: () => boolean;
};

const CACHE_KEY = 'arabica_data_cache';
const CACHE_VERSION = 1;
const CACHE_TTL_MS = 5 * 60 * 1000;

let isRefreshing = false;
let listeners: Array<(data: Record<string, any>) => void> = [];

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
    console.warn('Failed to read cache:', error);
    localStorage.removeItem(CACHE_KEY);
    return null;
  }
}

function setCache(data: Record<string, any>) {
  try {
    const cache: CacheEnvelope = {
      version: CACHE_VERSION,
      timestamp: Date.now(),
      did: data.did || null,
      app: getCurrentApp(),
      data
    };
    localStorage.setItem(CACHE_KEY, JSON.stringify(cache));
  } catch (error) {
    console.warn('Failed to write cache:', error);
  }
}

function getCachedDID() {
  return getCache()?.did || null;
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
    console.log('Cache belongs to different user, invalidating');
    invalidateCache();
    return null;
  }

  const currentApp = getCurrentApp();
  if (currentApp && cache.app && currentApp !== cache.app) {
    invalidateCache();
    return null;
  }

  return cache.data;
}

async function fetchFreshData() {
  const response = await fetch('/api/data', {
    credentials: 'same-origin',
    headers: {
      'X-Page-Context': window.location.pathname
    }
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch data: ${response.status}`);
  }

  return (await response.json()) as Record<string, any>;
}

async function refreshCache(force = false) {
  if (isRefreshing) {
    await new Promise<void>((resolve) => {
      const checkInterval = window.setInterval(() => {
        if (!isRefreshing) {
          window.clearInterval(checkInterval);
          resolve();
        }
      }, 100);
    });

    if (!force) {
      return getCachedData();
    }
  }

  isRefreshing = true;
  try {
    const data = await fetchFreshData();
    const cachedDID = getCachedDID();
    if (cachedDID && data.did && cachedDID !== data.did) {
      console.log('User changed, clearing stale cache');
      invalidateCache();
    }

    setCache(data);
    notifyListeners(data);
    return data;
  } finally {
    isRefreshing = false;
  }
}

async function getData(forceRefresh = false) {
  if (!forceRefresh && isCacheValid()) {
    return getCachedData();
  }

  const cached = getCachedData();
  try {
    return await refreshCache();
  } catch (error) {
    console.warn('Failed to refresh cache:', error);
    if (cached) return cached;
    throw error;
  }
}

function invalidateCache() {
  localStorage.removeItem(CACHE_KEY);
}

async function invalidateAndRefresh() {
  invalidateCache();
  return await refreshCache(true);
}

function addListener(callback: (data: Record<string, any>) => void) {
  listeners.push(callback);
}

function removeListener(callback: (data: Record<string, any>) => void) {
  listeners = listeners.filter((listener) => listener !== callback);
}

function notifyListeners(data: Record<string, any>) {
  listeners.forEach((callback) => {
    try {
      callback(data);
    } catch (error) {
      console.warn('Cache listener error:', error);
    }
  });
}

async function init() {
  if (!isCacheValid()) {
    try {
      await refreshCache();
    } catch (error) {
      console.warn('Initial cache load failed:', error);
    }
  }
}

async function preload() {
  return await refreshCache();
}

export const appCache: AppCacheAPI = {
  getData,
  getCachedData,
  refreshCache,
  invalidateCache,
  invalidateAndRefresh,
  addListener,
  removeListener,
  init,
  preload,
  isCacheValid
};
