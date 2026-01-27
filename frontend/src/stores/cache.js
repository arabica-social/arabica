import { writable } from "svelte/store";
import { api } from "../lib/api.js";

/**
 * Cache store - stale-while-revalidate pattern for user data
 * Replaces the old data-cache.js with reactive Svelte store
 */
function createCacheStore() {
  const { subscribe, set, update } = writable({
    beans: [],
    roasters: [],
    grinders: [],
    brewers: [],
    brews: [],
    lastFetch: null,
    loading: false,
  });

  const CACHE_KEY = "arabica_data_cache";
  const STALE_TIME = 5 * 60 * 1000; // 5 minutes

  return {
    subscribe,

    /**
     * Load data from cache or API
     * Uses stale-while-revalidate pattern
     */
    async load(force = false) {
      // Try to load from localStorage first
      if (!force) {
        const cached = localStorage.getItem(CACHE_KEY);
        if (cached) {
          try {
            const data = JSON.parse(cached);
            const age = Date.now() - data.timestamp;

            if (age < STALE_TIME) {
              // Fresh cache, use it
              set({
                ...data,
                lastFetch: data.timestamp,
                loading: false,
              });
              return;
            }

            // Stale cache, show it but refetch in background
            set({
              ...data,
              lastFetch: data.timestamp,
              loading: true,
            });
          } catch (e) {
            console.error("Failed to parse cache:", e);
          }
        }
      }

      // Fetch fresh data
      try {
        update((state) => ({ ...state, loading: true }));

        const data = await api.get("/api/data");
        const newState = {
          beans: data.beans || [],
          roasters: data.roasters || [],
          grinders: data.grinders || [],
          brewers: data.brewers || [],
          brews: data.brews || [],
          lastFetch: Date.now(),
          loading: false,
        };

        set(newState);

        // Save to localStorage
        localStorage.setItem(
          CACHE_KEY,
          JSON.stringify({
            ...newState,
            timestamp: newState.lastFetch,
          }),
        );
      } catch (error) {
        console.error("Failed to fetch data:", error);
        update((state) => ({ ...state, loading: false }));
      }
    },

    /**
     * Invalidate cache and refetch
     */
    async invalidate() {
      localStorage.removeItem(CACHE_KEY);
      await this.load(true);
    },

    /**
     * Clear cache completely
     */
    clear() {
      localStorage.removeItem(CACHE_KEY);
      set({
        beans: [],
        roasters: [],
        grinders: [],
        brewers: [],
        brews: [],
        lastFetch: null,
        loading: false,
      });
    },
  };
}

export const cacheStore = createCacheStore();
