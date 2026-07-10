<script lang="ts">
	import GrinderForm from "$lib/components/GrinderForm.svelte";
	import type { PageData } from "./$types";

	let { data }: { data: PageData } = $props();
</script>

<svelte:head>
	<title>Edit Grinder - Arabica</title>
</svelte:head>

{#if data.error}
	<div class="page-container-sm">
		<div class="card card-inner text-center py-8">
			<p class="text-secondary mb-4">{data.error}</p>
			<a href="/my-coffee" class="btn-primary">Back to My Coffee</a>
		</div>
	</div>
{:else if data.grinder}
	<GrinderForm grinder={data.grinder} isEdit={true} />
{:else}
	<div class="page-container-sm">
		<div class="card card-inner text-center py-8">
			<p class="text-secondary">Grinder not found.</p>
		</div>
	</div>
{/if}
