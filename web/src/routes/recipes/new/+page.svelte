<script lang="ts">
	import RecipeForm from "$lib/components/RecipeForm.svelte";
	import type { PageData } from "./$types";

	let { data }: { data: PageData } = $props();
</script>

<svelte:head>
	<title>Add a Recipe - Arabica</title>
	<meta name="description" content="Add a brewing recipe to your Arabica journal." />
</svelte:head>

{#if data.error}
	<div class="page-container-sm">
		<div class="card card-inner text-center py-8">
			<p class="text-secondary mb-4">{data.error}</p>
			<a href="/login" class="btn-primary">Log In</a>
		</div>
	</div>
{:else}
	<RecipeForm recipe={null} isEdit={false} />
{/if}
