import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [sveltekit()],
  resolve: {
    conditions: ["browser"],
  },
  test: {
    environment: "jsdom",
    setupFiles: ["src/testSetup.ts"],
    exclude: ["tests/e2e/**", "node_modules/**", "dist/**"],
  },
});
