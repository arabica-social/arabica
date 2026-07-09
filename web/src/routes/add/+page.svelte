<script lang="ts">
	import GetStartedCard from "$lib/components/GetStartedCard.svelte";
	import type { PageData } from "./$types";

	let { data }: { data: PageData } = $props();
</script>

<svelte:head>
	<title>Add Records - Arabica</title>
</svelte:head>

<div class="onboarding-wrap">
	<header class="onboarding-hero">
		<span class="onboarding-eyebrow">your coffee bar</span>
		<h1 class="onboarding-title">Add records.</h1>
		<p class="onboarding-lede">
			Add a <strong>brewer</strong>, <strong>roaster</strong>, <strong>bean</strong>, or <strong>grinder</strong> — anything new on the shelf shows up across your brews.
		</p>
	</header>

	{#if data.error}
		<div class="card card-inner text-center py-8">
			<p class="text-secondary mb-4">{data.error}</p>
			<a href="/login" class="btn-primary">Log In</a>
		</div>
	{:else}
		<GetStartedCard onboarding={data.onboarding} mode="library" />
	{/if}
</div>
