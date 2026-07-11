<script lang="ts">
	import BacklinksView from "$lib/components/BacklinksView.svelte";
	import type { PageData } from "./$types";

	let { data }: { data: PageData } = $props();

	let d = $derived(data.data);
</script>

<svelte:head>
	<title>Community{d ? ` · ${d.entity_name}` : ""} - Arabica</title>
	<meta
		name="description"
		content={d ? `Community around ${d.entity_name}` : "Community"}
	/>
</svelte:head>

{#if data.error}
	<div class="page-container-sm">
		<div class="card card-inner text-center py-8">
			<p class="text-secondary mb-4">{data.error}</p>
			{#if data.error === "Authentication required"}
				<a href="/login" class="btn-primary">Log In</a>
			{/if}
		</div>
	</div>
{:else if d}
	<BacklinksView
		result={d.result}
		entityNoun={d.entity_noun}
		entityName={d.entity_name}
		backURL={d.back_url}
	/>
{/if}
