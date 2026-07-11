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
		<span class="onboarding-eyebrow">welcome to arabica</span>
		<h1 class="onboarding-title">Let's get you started.</h1>
		<p class="onboarding-lede">
			Add a <strong>brewer</strong>, a <strong>roaster</strong>, and your first bag of <strong>beans</strong> — that's all you need to start logging brews. Takes about two minutes.
		</p>
	</header>

	{#if data.error}
		<div class="card card-inner text-center py-8">
			<p class="text-secondary mb-4">{data.error}</p>
			<button type="button" onclick={openLoginModal} class="btn-primary">Log In</button>
		</div>
	{:else}
		<GetStartedCard onboarding={data.onboarding} mode="onboarding" />
	{/if}
</div>
