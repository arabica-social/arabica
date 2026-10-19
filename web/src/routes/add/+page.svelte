<script lang="ts">
  import GetStartedCard from "$lib/components/GetStartedCard.svelte";
  import { openLoginModal } from "$lib/stores/session";
  import type { PageData } from "./$types";

  let { data }: { data: PageData } = $props();
</script>

<svelte:head>
  <title>Add - Arabica</title>
</svelte:head>

<div class="onboarding-wrap">
  <header class="onboarding-hero">
    <h1 class="onboarding-title">What would you like to add?</h1>
  </header>

  {#if data.error}
    <div class="card card-inner text-center py-8">
      <p class="text-secondary mb-4">{data.error}</p>
      <button type="button" onclick={openLoginModal} class="btn-primary"
        >Log in</button
      >
    </div>
  {:else}
    <GetStartedCard onboarding={data.onboarding} mode="library" />
  {/if}
</div>
