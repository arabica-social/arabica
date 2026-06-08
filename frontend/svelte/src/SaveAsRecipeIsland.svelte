<script lang="ts">
  interface Props {
    brewRKey: string;
  }

  let { brewRKey }: Props = $props();
  let showForm = $state(false);
  let name = $state('');
  let saving = $state(false);
  let error = $state('');
  let success = $state(false);

  async function saveRecipe() {
    if (!name.trim()) {
      error = 'Name is required';
      return;
    }

    saving = true;
    error = '';

    try {
      const response = await fetch(`/api/recipes/from-brew/${brewRKey}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: new URLSearchParams({ name }),
        credentials: 'same-origin'
      });

      if (!response.ok) {
        if (response.status === 401) {
          window.__showSessionExpiredModal?.();
        }
        throw new Error('Failed to save recipe');
      }

      await response.json();
      success = true;
      showForm = false;
    } catch {
      error = 'Failed to save recipe';
    } finally {
      saving = false;
    }
  }

  function cancel() {
    showForm = false;
    error = '';
  }
</script>

{#if !showForm && !success}
  <button type="button" onclick={() => (showForm = true)} class="w-full btn-secondary text-sm">
    Save as Recipe
  </button>
{:else if showForm && !success}
  <div class="space-y-3">
    <h3 class="text-sm font-medium text-muted uppercase tracking-wider">Save as Recipe</h3>
    <label for={`save-recipe-name-${brewRKey}`} class="sr-only">Recipe name</label>
    <input
      id={`save-recipe-name-${brewRKey}`}
      type="text"
      bind:value={name}
      placeholder="Recipe name"
      required
      aria-required="true"
      class="w-full form-input"
    />
    {#if error}
      <div class="text-danger text-sm">{error}</div>
    {/if}
    <div class="flex gap-2">
      <button type="button" onclick={saveRecipe} disabled={saving} class="flex-1 btn-primary text-sm">
        {saving ? 'Saving...' : 'Save'}
      </button>
      <button type="button" onclick={cancel} class="flex-1 btn-secondary text-sm">Cancel</button>
    </div>
  </div>
{:else if success}
  <div class="text-center text-success text-sm font-medium py-2">Recipe saved!</div>
{/if}
