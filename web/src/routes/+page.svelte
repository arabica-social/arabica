<script lang="ts">
	// Placeholder home route. The real feed renders here once the feed JSON
	// endpoint (Stream A P1.1) and feed components are ported (P2.5).
	// For now this proves the SPA shell + layout render end-to-end.
</script>

<svelte:head>
	<title>Arabica</title>
	<meta name="description" content="Arabica is a coffee brew tracking app built on AT Protocol. Your brewing data is stored in your own Personal Data Server, giving you full ownership and portability." />
</svelte:head>

<main class="page-container" style="padding: 2rem;">
	<h1>☕ Arabica</h1>
	<p class="text-secondary mt-2">
		The SvelteKit SPA is running. The home feed arrives in P2.5.
	</p>
	<p class="text-sm text-muted mt-4">
		Try the ported pages: <a href="/about" class="link">About</a>,
		<a href="/terms" class="link">Terms</a>,
		<a href="/atproto" class="link">AT Protocol</a>,
		<a href="/join/create" class="link">Create Account</a>.
	</p>
</main>
