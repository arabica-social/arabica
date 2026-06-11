<script lang="ts">
  export type Pour = {
    water: number | string;
    time: number | string;
    water_amount?: number;
    time_seconds?: number;
  };

  type Props = {
    pours: Pour[];
    title?: string;
    description?: string;
    emptyLabel?: string;
    expectedWater?: number | string;
  };

  let {
    pours = $bindable([]),
    title = "Pours",
    description = "Track individual pours for bloom and subsequent additions",
    emptyLabel = "+ Add pours",
    expectedWater = "",
  }: Props = $props();

  function addPour() {
    pours = [...pours, { water: "", time: "" }];
  }

  function numericValue(value: number | string | undefined) {
    if (value === undefined || value === null || value === "") return null;
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : null;
  }

  let pourTotal = $derived(
    pours.reduce((total, pour) => total + (numericValue(pour.water) ?? 0), 0),
  );
  let lastPourTime = $derived(
    pours.reduce((latest: number | null, pour) => {
      const seconds = numericValue(pour.time);
      if (seconds === null) return latest;
      return latest === null ? seconds : Math.max(latest, seconds);
    }, null),
  );
  let expectedWaterValue = $derived(numericValue(expectedWater));
  let waterMismatch = $derived(
    expectedWaterValue !== null &&
      pourTotal > 0 &&
      Math.abs(pourTotal - expectedWaterValue) > 0.01,
  );

  function removePour(index: number) {
    pours = pours.filter((_, i) => i !== index);
  }
</script>

{#if pours.length === 0}
  <button
    type="button"
    class="text-sm text-muted hover:text-secondary font-medium"
    onclick={addPour}>{emptyLabel}</button
  >
{:else}
  <div>
    <div class="flex items-center justify-between mb-2">
      <span class="block text-sm font-medium text-primary">{title}</span>
      <button type="button" onclick={addPour} class="text-sm btn-secondary"
        >+ Add Pour</button
      >
    </div>
    {#if description}
      <p class="text-sm text-emphasis mb-3">{description}</p>
    {/if}
    <div class="text-xs text-emphasis mb-3" data-testid="pour-summary">
      {pours.length}
      {pours.length === 1 ? "pour" : "pours"} · {pourTotal}g total
      {#if lastPourTime !== null}
        · last at {lastPourTime}s
      {/if}
    </div>
    {#if waterMismatch}
      <p class="alert-warning px-3 py-2 mb-3 text-xs" role="status">
        Pour water totals {pourTotal}g, which does not match total water {expectedWaterValue}g.
      </p>
    {/if}
    <div class="space-y-3">
      {#each pours as pour, index}
        <div
          class="flex gap-2 items-center p-3 rounded-lg"
          style="background: var(--surface-bg); border: 1px solid var(--surface-border);"
        >
          <div class="flex-1">
            <label
              class="text-xs text-emphasis font-medium"
              for={`pour-water-${index}`}>Pour {index + 1}</label
            >
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
            <label
              class="text-xs text-emphasis font-medium"
              for={`pour-time-${index}`}>Time (sec)</label
            >
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
            aria-label={`Remove pour ${index + 1}`}>×</button
          >
        </div>
      {/each}
    </div>
  </div>
{/if}
