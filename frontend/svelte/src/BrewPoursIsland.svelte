<script lang="ts">
  import { onMount } from 'svelte';

  type Pour = {
    water: number | string;
    time: number | string;
  };

  let { target }: { target: HTMLElement } = $props();

  let visible = $state(false);
  let pours = $state<Pour[]>([]);
  let title = $state('Pours');
  let description = $state('Track individual pours for bloom and subsequent additions');

  function normalizePour(value: unknown): Pour {
    if (!value || typeof value !== 'object') {
      return { water: '', time: '' };
    }
    const pour = value as Partial<Pour> & {
      water_amount?: number | string;
      time_seconds?: number | string;
    };
    return {
      water: pour.water ?? pour.water_amount ?? '',
      time: pour.time ?? pour.time_seconds ?? ''
    };
  }

  function setPours(nextPours: unknown) {
    pours = Array.isArray(nextPours) ? nextPours.map(normalizePour) : [];
    visible = pours.length > 0;
    dispatchVisibility();
  }

  function showEditor() {
    visible = true;
    if (pours.length === 0) {
      addPour();
      return;
    }
    dispatchVisibility();
  }

  function addPour() {
    pours = [...pours, { water: '', time: '' }];
    visible = true;
    dispatchVisibility();
  }

  function removePour(index: number) {
    pours = pours.filter((_, i) => i !== index);
    visible = pours.length > 0;
    dispatchVisibility();
  }

  function dispatchVisibility() {
    target.dispatchEvent(
      new CustomEvent('brew-pours-visibility-change', {
        detail: { visible },
        bubbles: true
      })
    );
  }

  onMount(() => {
    title = target.dataset.title || 'Pours';
    description =
      target.dataset.description ?? 'Track individual pours for bloom and subsequent additions';

    const initialPours = target.dataset.currentPours || target.dataset.initialPours;
    if (initialPours) {
      try {
        setPours(JSON.parse(initialPours));
      } catch {
        setPours([]);
      }
    }
    if (target.dataset.startVisible === 'true') {
      visible = true;
    }

    const handleSetPours = (event: Event) => {
      const detail = (event as CustomEvent<{ pours: unknown }>).detail;
      setPours(detail?.pours);
    };
    const handleShowPours = () => {
      showEditor();
    };

    target.addEventListener('brew-pours:set', handleSetPours);
    target.addEventListener('brew-pours:show', handleShowPours);
    dispatchVisibility();

    return () => {
      target.removeEventListener('brew-pours:set', handleSetPours);
      target.removeEventListener('brew-pours:show', handleShowPours);
    };
  });
</script>

{#if !visible}
  <button
    type="button"
    class="text-sm text-muted hover:text-secondary font-medium"
    onclick={showEditor}
  >
    + Add pours
  </button>
{:else}
  <div>
    <div class="flex items-center justify-between mb-2">
      <span class="block text-sm font-medium text-primary">{title}</span>
      <button type="button" onclick={addPour} class="text-sm btn-secondary">
        + Add Pour
      </button>
    </div>
    {#if description}
      <p class="text-sm text-emphasis mb-3">{description}</p>
    {/if}
    <div class="space-y-3">
      {#each pours as pour, index}
        <div
          class="flex gap-2 items-center p-3 rounded-lg"
          style="background: var(--surface-bg); border: 1px solid var(--surface-border);"
        >
          <div class="flex-1">
            <label class="text-xs text-emphasis font-medium" for={`pour-water-${index}`}>
              Pour {index + 1}
            </label>
            <input
              id={`pour-water-${index}`}
              type="number"
              name={`pour_water_${index}`}
              bind:value={pour.water}
              placeholder="Water (g)"
              class="w-full form-input text-sm py-2 px-3 mt-1"
            />
          </div>
          <div class="flex-1">
            <label class="text-xs text-emphasis font-medium" for={`pour-time-${index}`}>Time (sec)</label>
            <input
              id={`pour-time-${index}`}
              type="number"
              name={`pour_time_${index}`}
              bind:value={pour.time}
              placeholder="e.g. 45"
              class="w-full form-input text-sm py-2 px-3 mt-1"
            />
          </div>
          <button
            type="button"
            onclick={() => removePour(index)}
            class="text-emphasis hover:text-primary mt-5 font-bold"
            aria-label={`Remove pour ${index + 1}`}
          >
            <svg aria-hidden="true" viewBox="0 0 20 20" class="w-4 h-4" fill="currentColor">
              <path
                fill-rule="evenodd"
                d="M4.293 4.293a1 1 0 0 1 1.414 0L10 8.586l4.293-4.293a1 1 0 1 1 1.414 1.414L11.414 10l4.293 4.293a1 1 0 0 1-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 0 1-1.414-1.414L8.586 10 4.293 5.707a1 1 0 0 1 0-1.414Z"
                clip-rule="evenodd"
              />
            </svg>
          </button>
        </div>
      {/each}
    </div>
  </div>
{/if}
