<script lang="ts">
	import BrewForm from "$lib/components/BrewForm.svelte";
	import type { PageData } from "./$types";

	let { data }: { data: PageData } = $props();
</script>

<svelte:head>
	<title>New Brew - Arabica</title>
</svelte:head>

{#if data.error}
	<div class="page-container-sm">
		<div class="card card-inner text-center py-8">
			<p class="text-secondary mb-4">{data.error}</p>
			<a href="/login" class="btn-primary">Log In</a>
		</div>
	</div>
{:else}
	<BrewForm
		brew={null}
		recipeRKey={data.recipeRKey}
		recipeOwnerDID={data.recipeOwnerDID}
		isEdit={false}
	/>
{/if}
