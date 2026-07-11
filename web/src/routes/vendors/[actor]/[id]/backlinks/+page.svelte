<script lang="ts">
	import BacklinksView from "$lib/components/BacklinksView.svelte";
	import { openLoginModal } from "$lib/stores/session";
	import type { PageData } from "./$types";

	let { data }: { data: PageData } = $props();
	let view = $derived(data.data);
</script>

<svelte:head>
	<title>Community{view ? ` · ${view.entity_name}` : ""} - Oolong</title>
	<meta name="description" content={view ? `Community around ${view.entity_name}` : "Community"} />
</svelte:head>

{#if data.error}
	<div class="page-container-sm">
		<div class="card card-inner text-center py-8">
			<p class="text-secondary mb-4">{data.error}</p>
			{#if data.error === "Authentication required"}
				<button type="button" onclick={openLoginModal} class="btn-primary">Log In</button>
			{/if}
		</div>
	</div>
{:else if view}
	<BacklinksView
		result={view.result}
		entityNoun={view.entity_noun}
		entityName={view.entity_name}
		backURL={view.back_url}
	/>
{/if}
