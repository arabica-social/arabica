<script lang="ts">
  type BeanPayload = {
    name: string;
    origin: string;
    variety: string;
    roast_level: string;
    process: string;
    description: string;
    roaster_rkey: string;
    rating: number | null;
    closed: boolean;
  };

  let {
    beanRKey,
    baseBean,
    initialRating = 5,
    hasRating = false,
    closed = false
  }: {
    beanRKey: string;
    baseBean: Record<string, unknown>;
    initialRating?: number;
    hasRating?: boolean;
    closed?: boolean;
  } = $props();

  // svelte-ignore state_referenced_locally
  let rating = $state(initialRating);
  let savingAction = $state<'close' | 'save' | 'remove' | ''>('');
  let closeDialog = $state<HTMLDialogElement | undefined>();
  let rateDialog = $state<HTMLDialogElement | undefined>();

  function notify(message: string) {
    window.dispatchEvent(new CustomEvent('notify', { detail: { message } }));
  }

  async function patchBean(overrides: Partial<BeanPayload>, errorMessage: string, action: 'close' | 'save' | 'remove') {
    if (savingAction) {
      return;
    }
    savingAction = action;
    try {
      const response = await fetch(`/api/beans/${beanRKey}`, {
        method: 'PUT',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...baseBean, ...overrides })
      });
      if (response.status === 401) {
        window.__showSessionExpiredModal?.();
        return;
      }
      if (!response.ok) {
        throw new Error('Failed to update bean');
      }
      localStorage.removeItem('htmx-history-cache');
      window.location.reload();
    } catch {
      notify(errorMessage);
      savingAction = '';
    }
  }
</script>

{#if !closed}
  <button
    type="button"
    onclick={() => closeDialog?.showModal()}
    class="btn-secondary text-sm text-center"
  >
    Close Bag
  </button>
  <dialog
    id={`close-bag-confirm-${beanRKey}`}
    class="modal-dialog"
    bind:this={closeDialog}
    aria-labelledby={`close-bag-title-${beanRKey}`}
  >
    <div class="modal-content">
      <h3 id={`close-bag-title-${beanRKey}`} class="modal-title">Close Bag</h3>
      <p class="text-emphasis text-sm mb-4">
        Mark this bag as finished? You can reopen it later from the edit menu.
      </p>
      <div class="flex gap-2">
        <button
          type="button"
          onclick={() => patchBean({ closed: true }, 'Failed to close bag', 'close')}
          class="flex-1 btn-primary"
          disabled={!!savingAction}
        >
          {savingAction === 'close' ? 'Closing...' : 'Close Bag'}
        </button>
        <button type="button" onclick={() => closeDialog?.close()} class="flex-1 btn-secondary">
          Cancel
        </button>
      </div>
    </div>
  </dialog>
{/if}

<button
  type="button"
  onclick={() => rateDialog?.showModal()}
  class="btn-secondary text-sm text-center"
>
  {hasRating ? 'Edit Rating' : 'Rate Bag'}
</button>
<dialog
  id={`rate-bag-modal-${beanRKey}`}
  class="modal-dialog"
  bind:this={rateDialog}
  aria-labelledby={`rate-bag-title-${beanRKey}`}
>
  <div class="modal-content">
    <h3 id={`rate-bag-title-${beanRKey}`} class="modal-title">
      {hasRating ? 'Edit Rating' : 'Rate Bag'}
    </h3>
    <div class="space-y-4">
      <input
        type="range"
        min="1"
        max="10"
        bind:value={rating}
        class="w-full accent-brown-700"
        aria-label="Bag rating, 1 to 10"
      />
      <div class="text-center text-3xl font-bold text-secondary" aria-hidden="true">
        {rating}/10
      </div>
    </div>
    <div class="flex gap-2 mt-4">
      <button
        type="button"
        onclick={() => patchBean({ rating }, 'Failed to save rating', 'save')}
        class="flex-1 btn-primary"
        disabled={!!savingAction}
      >
        {savingAction === 'save' ? 'Saving...' : 'Save'}
      </button>
      {#if hasRating}
        <button
          type="button"
          onclick={() => patchBean({ rating: null }, 'Failed to remove rating', 'remove')}
          class="flex-1 btn-secondary text-danger"
          disabled={!!savingAction}
        >
          {savingAction === 'remove' ? 'Removing...' : 'Remove'}
        </button>
      {/if}
      <button type="button" onclick={() => rateDialog?.close()} class="flex-1 btn-secondary">
        Cancel
      </button>
    </div>
  </div>
</dialog>
