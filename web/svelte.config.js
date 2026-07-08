import adapter from "@sveltejs/adapter-static";
import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

/** @type {import('@sveltejs/kit').Config} */
const config = {
  preprocess: vitePreprocess(),
  kit: {
    // SPA mode: pre-render nothing, fall back to index.html for all routes.
    // The Go server serves this shell with <head> populated server-side for
    // OG tags; SvelteKit hydrates and takes over routing client-side.
    adapter: adapter({
      fallback: "index.html",
    }),
    // Files under web/static/ are copied as-is into the build output and
    // served at the root. We keep this minimal — the Go server owns
    // /static/ (CSS, fonts, favicons) via its own embed FS.
    files: {
      assets: "static",
    },
  },
};

export default config;
