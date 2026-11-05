<script lang="ts">
  import { page } from "$app/state";
  import NotFound from "#lib/components/NotFound.svelte";
</script>

<svelte:head>
  <title>{page.status === 404 ? "Page Not Found" : page.status} - Arabica</title
  >
</svelte:head>

{#if page.status === 404}
  <NotFound />
{:else}
  <h1>{page.status}</h1>
  <p>{page.error?.message ?? "An unexpected error occurred."}</p>
{/if}
