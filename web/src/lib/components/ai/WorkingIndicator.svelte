<script lang="ts">
  let { label = "Working on your draft…" }: { label?: string } = $props();
</script>

<div
  class="working-indicator"
  role="status"
  aria-live="polite"
  aria-atomic="true"
>
  <span class="spinner" aria-hidden="true"></span>
  <span>{label}</span>
</div>

<style>
  .working-indicator {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    color: var(--text-secondary);
    font-size: 0.875rem;
    animation: appear 160ms ease-out;
  }
  .spinner {
    width: 1rem;
    height: 1rem;
    flex-shrink: 0;
    border: 1.5px solid currentColor;
    border-right-color: transparent;
    border-radius: 50%;
    animation: spin 900ms linear infinite;
  }
  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
  @keyframes appear {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .working-indicator,
    .spinner {
      animation: none;
    }
    .spinner {
      border-right-color: currentColor;
    }
  }
</style>
