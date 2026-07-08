import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [svelte()],
  resolve: {
    conditions: ["browser"],
  },
  test: {
    environment: "jsdom",
    setupFiles: ["internal/web/assets/svelte/src/testSetup.ts"],
    // The web/ SvelteKit workspace has its own vitest config; exclude it
    // so the root runner only covers the legacy Svelte islands.
    exclude: ["node_modules/**", "web/**"],
  },
  build: {
    emptyOutDir: false,
    lib: {
      entry: "internal/web/assets/svelte/src/main.ts",
      formats: ["es"],
      fileName: () => "svelte-islands.js",
    },
    outDir: "internal/web/assets/js",
    rollupOptions: {
      output: {
        chunkFileNames: "svelte-islands-[name]-[hash].js",
      },
    },
  },
});
