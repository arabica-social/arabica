// Disable SSR — this is a pure SPA. The Go server serves the HTML shell;
// SvelteKit only runs client-side.
export const ssr = false;
export const prerender = false;
