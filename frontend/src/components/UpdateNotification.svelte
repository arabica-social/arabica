<script>
  import { updateAvailable, updateServiceWorker } from '../stores/pwa.js';

  let showNotification = false;

  $: if ($updateAvailable) {
    showNotification = true;
  }

  function handleUpdate() {
    showNotification = false;
    updateServiceWorker();
  }

  function handleDismiss() {
    showNotification = false;
  }
</script>

{#if showNotification}
  <div class="fixed bottom-4 left-4 right-4 md:left-auto md:right-4 md:max-w-md bg-amber-100 border border-amber-400 rounded-lg shadow-lg p-4">
    <div class="flex items-start gap-3">
      <div class="flex-shrink-0">
        <svg class="h-5 w-5 text-amber-600" fill="currentColor" viewBox="0 0 20 20">
          <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
        </svg>
      </div>
      <div class="flex-1">
        <h3 class="text-sm font-medium text-amber-900">Update Available</h3>
        <p class="mt-1 text-sm text-amber-800">
          A new version of Arabica is available.
        </p>
      </div>
      <div class="flex gap-2 flex-shrink-0">
        <button
          on:click={handleUpdate}
          class="inline-flex items-center px-2.5 py-1.5 rounded text-sm font-medium text-white bg-amber-600 hover:bg-amber-700 transition-colors"
        >
          Update
        </button>
        <button
          on:click={handleDismiss}
          class="inline-flex items-center px-2.5 py-1.5 rounded text-sm font-medium text-amber-900 bg-transparent hover:bg-amber-200 transition-colors"
        >
          Dismiss
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  /* Mobile-friendly positioning */
  @media (max-width: 640px) {
    div {
      margin: 0 1rem;
      max-width: calc(100vw - 2rem);
    }
  }
</style>
