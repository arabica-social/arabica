<script lang="ts">
	import GetStartedCard from "$lib/components/GetStartedCard.svelte";
	import { openLoginModal } from "$lib/stores/session";
	import type { PageData } from "./$types";

	let { data }: { data: PageData } = $props();
</script>

<svelte:head>
	<title>Get Started - Arabica</title>
</svelte:head>

<div class="onboarding-wrap">
	<header class="onboarding-hero">
		<span class="onboarding-eyebrow">Welcome to Arabica</span>
		<h1 class="onboarding-title">Set up your first brew.</h1>
		<p class="onboarding-lede">
			Add a <strong>brewer</strong>, a <strong>roaster</strong>, and a bag of <strong>beans</strong> before logging your first brew.
		</p>
	</header>

	{#if data.error}
		<div class="card card-inner text-center py-8">
			<p class="text-secondary mb-4">{data.error}</p>
			<button type="button" onclick={openLoginModal} class="btn-primary">Log in</button>
		</div>
	{:else}
		<GetStartedCard onboarding={data.onboarding} mode="onboarding" />
	{/if}
</div>
