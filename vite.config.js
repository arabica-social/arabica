import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [svelte()],
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
