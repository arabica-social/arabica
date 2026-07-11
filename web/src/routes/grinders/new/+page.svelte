<script lang="ts">
	import GrinderForm from "$lib/components/GrinderForm.svelte";
	import { openLoginModal } from "$lib/stores/session";
	import type { PageData } from "./$types";

	let { data }: { data: PageData } = $props();
</script>

<svelte:head>
	<title>Add a Grinder - Arabica</title>
	<meta name="description" content="Add a coffee grinder to your Arabica journal." />
</svelte:head>

{#if data.error}
	<div class="page-container-sm">
		<div class="card card-inner text-center py-8">
			<p class="text-secondary mb-4">{data.error}</p>
			<button type="button" onclick={openLoginModal} class="btn-primary">Log In</button>
		</div>
	</div>
{:else}
	<GrinderForm grinder={null} isEdit={false} />
{/if}
