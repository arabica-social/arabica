import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vite";

// Keep browser requests same-origin in development: Vite owns SPA pages and
// HMR, while Go continues to own APIs, OAuth, mutations, and shared assets.
// Keep changeOrigin false so Go sees the browser-facing Vite host.
const backendTarget = process.env.VITE_BACKEND_URL ?? "http://127.0.0.1:18910";
const devPort = Number(process.env.VITE_DEV_PORT ?? "5173");
const backendRoute =
  "^/(?:api(?:/|$)|auth(?:/|$)|oauth(?:/|$)|\\.well-known(?:/|$)|static(?:/|$)|_mod(?:/|$)|at(?:/|$)|login$|logout$|reauth$|join/create$|settings/bluesky-profile/upgrade-scopes$|og-image$|favicon\\.ico$|robots\\.txt$|healthz$|.+/og-image$)";

export default defineConfig({
  plugins: [sveltekit()],
  server: {
    host: "127.0.0.1",
    port: devPort,
    strictPort: true,
    proxy: {
      [backendRoute]: {
        target: backendTarget,
        changeOrigin: false,
      },
    },
  },
  resolve: {
    conditions: ["browser"],
  },
  test: {
    environment: "jsdom",
    setupFiles: ["src/testSetup.ts"],
    exclude: ["tests/e2e/**", "node_modules/**", "dist/**"],
  },
});
