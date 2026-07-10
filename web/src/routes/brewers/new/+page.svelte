<script lang="ts">
	import BrewerForm from "$lib/components/BrewerForm.svelte";
	import type { PageData } from "./$types";

	let { data }: { data: PageData } = $props();
</script>

<svelte:head>
	<title>Add a Brewer - Arabica</title>
	<meta name="description" content="Add a brewing device to your Arabica journal." />
</svelte:head>

{#if data.error}
	<div class="page-container-sm">
		<div class="card card-inner text-center py-8">
			<p class="text-secondary mb-4">{data.error}</p>
			<a href="/login" class="btn-primary">Log In</a>
		</div>
	</div>
{:else}
	<BrewerForm brewer={null} isEdit={false} />
{/if}
