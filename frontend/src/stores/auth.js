import { writable } from "svelte/store";
import { api } from "../lib/api.js";

/**
 * Auth store - tracks current user authentication state
 */
function createAuthStore() {
  const { subscribe, set, update } = writable({
    isAuthenticated: false,
    user: null,
    loading: true,
  });

  return {
    subscribe,

    /**
     * Check current authentication status
     */
    async checkAuth() {
      try {
        const user = await api.get("/api/me");
        set({
          isAuthenticated: true,
          user,
          loading: false,
        });
      } catch (error) {
        set({
          isAuthenticated: false,
          user: null,
          loading: false,
        });
      }
    },

    /**
     * Log out current user
     */
    async logout() {
      try {
        await api.post("/logout", {});
        set({
          isAuthenticated: false,
          user: null,
          loading: false,
        });
        window.location.href = "/";
      } catch (error) {
        console.error("Logout failed:", error);
      }
    },

    /**
     * Clear auth state (used after logout)
     */
    clear() {
      set({
        isAuthenticated: false,
        user: null,
        loading: false,
      });
    },
  };
}

export const authStore = createAuthStore();
