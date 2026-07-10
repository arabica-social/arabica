<script lang="ts">
	import BeanForm from "$lib/components/BeanForm.svelte";
	import type { PageData } from "./$types";

	let { data }: { data: PageData } = $props();
</script>

<svelte:head>
	<title>Add a Bean - Arabica</title>
	<meta name="description" content="Add a coffee bean to your Arabica journal." />
</svelte:head>

{#if data.error}
	<div class="page-container-sm">
		<div class="card card-inner text-center py-8">
			<p class="text-secondary mb-4">{data.error}</p>
			<a href="/login" class="btn-primary">Log In</a>
		</div>
	</div>
{:else}
	<BeanForm bean={null} isEdit={false} />
{/if}
