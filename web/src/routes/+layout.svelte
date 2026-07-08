<script lang="ts">
	let { children } = $props();

	// The Go shell route injects these as data attributes on <body> before
	// serving the SPA. During local `vite dev` they're absent — fall back to
	// defaults so the app renders without a Go server.
	const body = typeof document !== "undefined" ? document.body : null;
	const userDID = $derived(body?.dataset.userDid ?? "");
	const app = $derived(body?.dataset.app ?? "arabica");
</script>

{@render children()}
