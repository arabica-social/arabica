<script>
  export let onSave;
  export let onCancel;
  export let isOpen = false;
  export let title = "Modal";
</script>

{#if isOpen}
  <div class="fixed inset-0 bg-black/40 flex items-center justify-center z-50">
    <div
      class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl border-2 border-brown-300 p-8 max-w-md w-full mx-4 shadow-2xl"
    >
      <h3 class="text-xl font-semibold mb-4 text-brown-900">{title}</h3>
      <div class="space-y-4">
        <slot />

        <div class="flex gap-2">
          <button
            type="button"
            on:click={onSave}
            class="flex-1 bg-gradient-to-r from-brown-700 to-brown-800 text-white px-4 py-2 rounded-lg hover:from-brown-800 hover:to-brown-900 font-medium transition-all shadow-md"
          >
            Save
          </button>
          <button
            type="button"
            on:click={onCancel}
            class="flex-1 bg-brown-300 text-brown-900 px-4 py-2 rounded-lg hover:bg-brown-400 font-medium transition-colors"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}
