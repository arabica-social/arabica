<script lang="ts">
	let { data } = $props();
</script>

<svelte:head>
	<title>Arabica — SPA Shell</title>
</svelte:head>

<main class="page-container" style="padding: 2rem;">
	<h1>SvelteKit SPA is alive</h1>
	<p>
		This page is served by the SvelteKit SPA shell. Once the Go JSON API and
		page migrations are complete, this route will be replaced by the real
		home page.
	</p>
	<p>
		App: <strong>{data?.app ?? "arabica"}</strong>
	</p>
</main>
