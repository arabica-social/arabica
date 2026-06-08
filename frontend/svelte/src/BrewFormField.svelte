<script lang="ts">
  import type { Snippet } from 'svelte';

  let { label, helper = '', children }: { label: string; helper?: string; children: Snippet } = $props();
</script>

<div>
  <label class="form-label">{label}</label>
  {@render children()}
  {#if helper}<p class="text-helper">{helper}</p>{/if}
</div>
