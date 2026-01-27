<script>
  import { isOnline } from "../stores/pwa.js";
</script>

{#if !$isOnline}
  <div
    class="fixed top-0 left-0 right-0 bg-red-500 text-white px-4 py-2 text-center text-sm font-medium z-50 animate-pulse"
  >
    <div class="flex items-center justify-center gap-2">
      <svg class="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
        <path
          fill-rule="evenodd"
          d="M5.293 9.707a1 1 0 010-1.414l4-4a1 1 0 011.414 0l4 4a1 1 0 01-1.414 1.414L11 7.414V15a1 1 0 11-2 0V7.414L6.707 9.707a1 1 0 01-1.414 0z"
          clip-rule="evenodd"
        />
      </svg>
      <span>You're offline. Some features may be limited.</span>
    </div>
  </div>
{/if}

<style>
  @keyframes animate-pulse {
    0%,
    100% {
      opacity: 1;
    }
    50% {
      opacity: 0.5;
    }
  }

  :global(.animate-pulse) {
    animation: animate-pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
  }
</style>
