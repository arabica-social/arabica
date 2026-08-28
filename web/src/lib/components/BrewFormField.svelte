<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    label,
    helper = "",
    badge = "",
    children,
  }: { label: string; helper?: string; badge?: string; children: Snippet } = $props();
</script>

<div>
  <label>
    <span class="form-label">{label}</span>
    {#if badge}
      <span class="autofill-badge">{badge}</span>
    {/if}
    {@render children()}
  </label>
  {#if helper}<p class="text-helper">{helper}</p>{/if}
</div>

<style>
  .form-label { display: inline; }
  .autofill-badge {
    display: inline-block;
    margin-left: 0.5rem;
    padding: 0.15rem 0.5rem;
    border: 1px dashed var(--card-border);
    border-radius: 999px;
    background: color-mix(in oklch, var(--surface-bg) 74%, transparent);
    color: var(--text-muted);
    font-size: 0.68rem;
    font-weight: 600;
    letter-spacing: 0.02em;
    vertical-align: middle;
  }
</style>
