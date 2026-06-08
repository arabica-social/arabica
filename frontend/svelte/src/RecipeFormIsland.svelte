<script lang="ts">
  import { onMount } from "svelte";
  import Field from "./BrewFormField.svelte";
  import PoursEditor, { type Pour } from "./PoursEditor.svelte";
  import type { AppCacheAPI } from "./appCache";

  type BrewerRecord = Record<string, any>;

  let { target }: { target: HTMLElement } = $props();

  let name = $state("");
  let brewerRKey = $state("");
  let brewerType = $state("");
  let coffeeAmount = $state("");
  let waterAmount = $state("");
  let notes = $state("");
  let sourceRef = $state("");
  let pours = $state<Pour[]>([]);
  let brewers = $state<BrewerRecord[]>([]);

  function appCache(): AppCacheAPI | undefined {
    return window.AppCache;
  }

  function rkey(entity: BrewerRecord) {
    return entity.rkey || entity.RKey || "";
  }

  function brewerName(entity: BrewerRecord) {
    return entity.name || entity.Name || "";
  }

  function normalizeBrewerType(raw: string) {
    return (raw || "").trim();
  }

  function selectedBrewerType() {
    const selected = brewers.find((brewer) => rkey(brewer) === brewerRKey);
    return normalizeBrewerType(selected?.brewer_type || selected?.BrewerType || "");
  }

  function handleBrewerChange() {
    const selectedType = selectedBrewerType();
    if (selectedType) brewerType = selectedType;
  }

  function parsePours(raw: string) {
    try {
      const parsed = JSON.parse(raw || "[]");
      if (!Array.isArray(parsed)) return [];
      return parsed.map((pour) => ({
        water: pour.water ?? pour.water_amount ?? "",
        time: pour.time ?? pour.time_seconds ?? "",
      }));
    } catch {
      return [];
    }
  }

  function parseBrewers(raw: string) {
    try {
      const parsed = JSON.parse(raw || "[]");
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return [];
    }
  }

  function initializeFromDataset() {
    const d = target.dataset;
    name = d.name || "";
    brewerRKey = d.brewerRkey || "";
    brewerType = d.brewerType || "";
    coffeeAmount = d.coffeeAmount || "";
    waterAmount = d.waterAmount || "";
    notes = d.notes || "";
    sourceRef = d.sourceRef || "";
    pours = parsePours(d.pours || "[]");
    brewers = parseBrewers(d.brewers || "[]");
  }

  onMount(() => {
    initializeFromDataset();
    const cached = appCache()?.getCachedData?.();
    if (cached?.brewers?.length) brewers = cached.brewers;
    const listener = (data: Record<string, any>) => {
      if (data.brewers?.length) brewers = data.brewers;
    };
    appCache()?.addListener?.(listener);
    void appCache()
      ?.getData?.()
      .then((data) => {
        if (data?.brewers?.length) brewers = data.brewers;
      });

    return () => appCache()?.removeListener?.(listener);
  });
</script>

<fieldset class="space-y-6 border-0 p-0 m-0 min-w-0">
  {#if sourceRef}
    <input type="hidden" name="source_ref" value={sourceRef} />
  {/if}

  <fieldset class="form-fieldset">
    <div class="form-fieldset-label">Essentials</div>
    <Field label="Recipe Name">
      <input
        type="text"
        name="name"
        bind:value={name}
        placeholder="Name"
        required
        class="w-full form-input"
      />
    </Field>
    <Field label="Brewer">
      <select
        name="brewer_rkey"
        bind:value={brewerRKey}
        onchange={handleBrewerChange}
        class="w-full form-input"
      >
        <option value="">Select Brewer</option>
        {#each brewers as brewer}
          <option value={rkey(brewer)}>{brewerName(brewer)}</option>
        {/each}
      </select>
    </Field>
    <Field label="Brewer Type">
      <input
        type="text"
        name="brewer_type"
        bind:value={brewerType}
        placeholder="e.g. Pour-Over, Immersion"
        class="w-full form-input"
      />
    </Field>
  </fieldset>

  <div class="form-divider"></div>

  <fieldset class="form-fieldset">
    <div class="form-fieldset-label">Amounts <span class="form-optional-hint">(optional)</span></div>
    <div class="grid grid-cols-2 gap-3">
      <Field label="Coffee (g)">
        <input
          type="number"
          name="coffee_amount"
          bind:value={coffeeAmount}
          placeholder="Coffee (g)"
          step="0.1"
          class="w-full form-input"
        />
      </Field>
      <Field label="Water (g)">
        <input
          type="number"
          name="water_amount"
          bind:value={waterAmount}
          placeholder="Water (g)"
          step="0.1"
          class="w-full form-input"
        />
      </Field>
    </div>
  </fieldset>

  <div class="form-divider"></div>

  <fieldset class="form-fieldset">
    <PoursEditor bind:pours description="" emptyLabel="+ Add Pour" />
  </fieldset>

  <div class="form-divider"></div>

  <fieldset class="form-fieldset">
    <div class="form-fieldset-label">Notes <span class="form-optional-hint">(optional)</span></div>
    <textarea
      name="notes"
      bind:value={notes}
      placeholder="Notes"
      rows="3"
      class="w-full form-textarea"
    ></textarea>
  </fieldset>
</fieldset>
