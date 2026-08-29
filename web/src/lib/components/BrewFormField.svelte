<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    label,
    helper = "",
    badge,
    children,
  }: { label: string; helper?: string; badge?: string; children: Snippet } = $props();
</script>

<div>
  <label>
    <span class="form-label">{label}</span>
    {@render children()}
  </label>
  {#if badge !== undefined}
    <div class="autofill-status" aria-live="polite">
      {#if badge}<span class="autofill-badge">{badge}</span>{/if}
    </div>
  {/if}
  {#if helper}<p class="text-helper">{helper}</p>{/if}
</div>

<style>
  .form-label { display: block; }
  .autofill-status {
    display: flex;
    align-items: center;
    min-height: 1.25rem;
    margin-top: 0.25rem;
  }
  .autofill-badge {
    display: inline-flex;
    align-items: center;
    min-height: 1.1rem;
    padding: 0.15rem 0.5rem;
    border: 1px dashed var(--autofill-text);
    border-radius: 999px;
    background: color-mix(in oklch, var(--surface-bg) 74%, transparent);
    color: var(--autofill-text);
    font-size: 0.68rem;
    font-weight: 600;
    letter-spacing: 0.02em;
    line-height: 1.1;
  }
</style>
