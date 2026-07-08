<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    label,
    helper = "",
    children,
  }: { label: string; helper?: string; children: Snippet } = $props();
</script>

<div>
  <label>
    <span class="form-label">{label}</span>
    {@render children()}
  </label>
  {#if helper}<p class="text-helper">{helper}</p>{/if}
</div>
