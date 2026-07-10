<script lang="ts">
	import RecipeForm from "$lib/components/RecipeForm.svelte";
	import type { PageData } from "./$types";

	let { data }: { data: PageData } = $props();
</script>

<svelte:head>
	<title>Edit Recipe - Arabica</title>
</svelte:head>

{#if data.error}
	<div class="page-container-sm">
		<div class="card card-inner text-center py-8">
			<p class="text-secondary mb-4">{data.error}</p>
			<a href="/my-coffee" class="btn-primary">Back to My Coffee</a>
		</div>
	</div>
{:else if data.recipe}
	<RecipeForm recipe={data.recipe} isEdit={true} />
{:else}
	<div class="page-container-sm">
		<div class="card card-inner text-center py-8">
			<p class="text-secondary">Recipe not found.</p>
		</div>
	</div>
{/if}
