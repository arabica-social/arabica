<script lang="ts">
  import { onMount } from "svelte";
  import Field from "./BrewFormField.svelte";
  import type { AppCacheAPI } from "./appCache";
  import {
    comboSelectEntities,
    type EntityRecord,
    type Suggestion,
  } from "./comboSelectRegistry";

  type ComboType = "recipe" | "bean" | "grinder" | "brewer";
  type Pour = {
    water: number | string;
    time: number | string;
    water_amount?: number;
    time_seconds?: number;
  };
  type ComboState = {
    rkey: string;
    label: string;
    query: string;
    results: EntityRecord[];
    closedResults: EntityRecord[];
    suggestions: Suggestion[];
    open: boolean;
    highlight: number;
    showCreate: boolean;
    createData: EntityRecord;
    creating: boolean;
  };
  type ComboItem =
    | { kind: "entity"; entity: EntityRecord }
    | { kind: "closed"; entity: EntityRecord }
    | { kind: "suggestion"; suggestion: Suggestion }
    | { kind: "create" };

  const comboMeta: Record<
    ComboType,
    {
      label: string;
      inputName: string;
      endpoint: string;
      suggestEndpoint: string;
      placeholder: string;
      sectionLabel: string;
      required?: boolean;
      passthrough?: boolean;
      allowCreate?: boolean;
    }
  > = {
    recipe: {
      label: "Recipe (Optional)",
      inputName: "recipe_rkey",
      endpoint: "/api/recipes",
      suggestEndpoint: "/api/suggestions/recipes",
      placeholder: "Search recipes...",
      sectionLabel: "Your recipes",
      passthrough: true,
      allowCreate: false,
    },
    bean: {
      label: "Coffee Bean",
      inputName: "bean_rkey",
      endpoint: "/api/beans",
      suggestEndpoint: "/api/suggestions/beans",
      placeholder: "Search beans...",
      sectionLabel: "Your beans",
      required: true,
      allowCreate: false,
    },
    grinder: {
      label: "Grinder",
      inputName: "grinder_rkey",
      endpoint: "/api/grinders",
      suggestEndpoint: "/api/suggestions/grinders",
      placeholder: "Search grinders...",
      sectionLabel: "Your grinders",
    },
    brewer: {
      label: "Brew Method",
      inputName: "brewer_rkey",
      endpoint: "/api/brewers",
      suggestEndpoint: "/api/suggestions/brewers",
      placeholder: "Search brew methods...",
      sectionLabel: "Your brewers",
    },
  };

  let { target }: { target: HTMLElement } = $props();

  let cachedData = $state<Record<string, any>>({});
  let recipeOwnerDID = $state("");
  let activeRecipe = $state<EntityRecord | null>(null);
  let recipeExpanded = $state(false);
  let brewerCategory = $state("");
  let focusedCombo = $state<ComboType | "">("");
  let suggestTimers: Partial<Record<ComboType, ReturnType<typeof setTimeout>>> =
    {};

  let coffeeAmount = $state("");
  let waterAmount = $state("");
  let grindSize = $state("");
  let temperature = $state("");
  let timeSeconds = $state("");
  let tastingNotes = $state("");
  let rating = $state("5");
  let pours = $state<Pour[]>([]);
  let method = $state("");
  let espressoYieldWeight = $state("");
  let espressoPressure = $state("");
  let espressoPreInfusionSeconds = $state("");
  let pouroverBloomWater = $state("");
  let pouroverBloomSeconds = $state("");
  let pouroverDrawdownSeconds = $state("");
  let pouroverBypassWater = $state("");
  let pouroverFilter = $state("");
  let submitLabel = $state("Save Brew");

  let combos = $state<Record<ComboType, ComboState>>({
    recipe: emptyCombo(),
    bean: emptyCombo(),
    grinder: emptyCombo(),
    brewer: emptyCombo(),
  });

  function emptyCombo(): ComboState {
    return {
      rkey: "",
      label: "",
      query: "",
      results: [],
      closedResults: [],
      suggestions: [],
      open: false,
      highlight: -1,
      showCreate: false,
      createData: {},
      creating: false,
    };
  }

  function appCache(): AppCacheAPI | undefined {
    return window.AppCache;
  }

  function config(type: ComboType) {
    return comboSelectEntities[type] || {};
  }

  function formatLabel(type: ComboType, entity: EntityRecord | Suggestion) {
    return (
      config(type).formatLabel?.(entity) ||
      entity.name ||
      (entity as EntityRecord).Name ||
      ""
    );
  }

  function rkey(entity: EntityRecord) {
    return entity.rkey || entity.RKey || "";
  }

  function cachedEntities(type: ComboType) {
    if (type === "bean") return cachedData.beans || [];
    if (type === "grinder") return cachedData.grinders || [];
    if (type === "brewer") return cachedData.brewers || [];
    if (type === "recipe") return cachedData.recipes || [];
    return [];
  }

  function normalizeBrewerCategory(raw: string) {
    const lower = (raw || "").toLowerCase().trim();
    if (
      [
        "pourover",
        "espresso",
        "immersion",
        "mokapot",
        "coldbrew",
        "cupping",
        "other",
      ].includes(lower)
    )
      return lower;
    if (["pour-over", "pour over", "dripper"].includes(lower))
      return "pourover";
    if (
      ["espresso machine", "lever espresso", "lever espresso machine"].includes(
        lower,
      )
    )
      return "espresso";
    if (
      [
        "french press",
        "aeropress",
        "siphon",
        "clever",
        "clever dripper",
      ].includes(lower)
    )
      return "immersion";
    return "";
  }

  function setCombo(type: ComboType, patch: Partial<ComboState>) {
    combos = { ...combos, [type]: { ...combos[type], ...patch } };
  }

  function search(type: ComboType, open = true) {
    const state = combos[type];
    const q = state.query.trim().toLowerCase();
    const matches = q
      ? cachedEntities(type).filter((entity: EntityRecord) =>
          formatLabel(type, entity).toLowerCase().includes(q),
        )
      : cachedEntities(type).slice(0, 10);

    if (type === "bean") {
      setCombo(type, {
        results: matches.filter(
          (bean: EntityRecord) => !bean.closed && !bean.Closed,
        ),
        closedResults: q
          ? matches.filter((bean: EntityRecord) => bean.closed || bean.Closed)
          : [],
        open,
        highlight: -1,
      });
    } else {
      setCombo(type, {
        results: matches,
        closedResults: [],
        open,
        highlight: -1,
      });
    }

    clearTimeout(suggestTimers[type]);
    if (q.length >= 2) {
      suggestTimers[type] = setTimeout(
        () => void fetchSuggestions(type, q),
        350,
      );
    } else {
      setCombo(type, { suggestions: [] });
    }
  }

  async function fetchSuggestions(type: ComboType, q: string) {
    try {
      const response = await fetch(
        `${comboMeta[type].suggestEndpoint}?q=${encodeURIComponent(q)}&limit=5`,
        { credentials: "same-origin" },
      );
      if (!response.ok) return;
      const data = await response.json();
      const ownNames = new Set(
        cachedEntities(type).map((entity: EntityRecord) =>
          (entity.name || entity.Name || "").toLowerCase(),
        ),
      );
      setCombo(type, {
        suggestions: (data || []).filter(
          (suggestion: Suggestion) =>
            !ownNames.has((suggestion.name || "").toLowerCase()),
        ),
      });
    } catch (error) {
      console.error("Suggestion fetch failed:", error);
    }
  }

  function exactMatch(type: ComboType) {
    const state = combos[type];
    const q = state.query.trim().toLowerCase();
    if (!q) return false;
    return (
      [...state.results, ...state.closedResults].some(
        (entity) => formatLabel(type, entity).toLowerCase() === q,
      ) ||
      state.suggestions.some(
        (suggestion) => (suggestion.name || "").toLowerCase() === q,
      )
    );
  }

  function selectEntity(type: ComboType, entity: EntityRecord) {
    const label = formatLabel(type, entity);
    setCombo(type, { rkey: rkey(entity), label, query: label, open: false });
    if (type === "brewer")
      brewerCategory = normalizeBrewerCategory(
        entity.brewer_type || entity.BrewerType || "",
      );
    if (type === "recipe")
      void applyRecipe(rkey(entity), entity.author_did || "");
  }

  async function selectSuggestion(type: ComboType, suggestion: Suggestion) {
    if (comboMeta[type].passthrough) {
      const parts = (suggestion.source_uri || "").split("/");
      const selectedRKey = parts.length >= 5 ? parts[4] : "";
      const selectedOwner = parts.length >= 3 ? parts[2] : "";
      const label = formatLabel(type, suggestion);
      setCombo(type, { rkey: selectedRKey, label, query: label, open: false });
      if (type === "recipe") await applyRecipe(selectedRKey, selectedOwner);
      return;
    }

    const data = config(type).formatCreateData?.(
      suggestion.name || "",
      suggestion,
    ) || { name: suggestion.name || "" };
    if (suggestion.source_uri) data.source_ref = suggestion.source_uri;
    const extras = config(type).extraFields || [];
    if (extras.length > 0) {
      for (const field of extras)
        if (!(field.name in data)) data[field.name] = "";
      setCombo(type, { createData: data, showCreate: true, open: false });
      return;
    }
    await createEntity(type, data);
  }

  function startCreate(type: ComboType) {
    if (comboMeta[type].allowCreate === false) return;
    const name = combos[type].query.trim();
    if (!name) return;
    const data: EntityRecord = { name };
    for (const field of config(type).extraFields || []) data[field.name] = "";
    setCombo(type, { createData: data, showCreate: true, open: false });
  }

  async function createEntity(type: ComboType, data: EntityRecord) {
    setCombo(type, { creating: true });
    try {
      const response = await fetch(comboMeta[type].endpoint, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "same-origin",
        body: JSON.stringify(data),
      });
      if (!response.ok) {
        if (response.status === 401) window.__showSessionExpiredModal?.();
        throw new Error(`Create failed: ${response.status}`);
      }
      const created = await response.json();
      const label = data.name || formatLabel(type, created);
      setCombo(type, {
        rkey: rkey(created),
        label,
        query: label,
        showCreate: false,
        createData: {},
        open: false,
      });
      const refreshed = await appCache()?.invalidateAndRefresh?.();
      if (refreshed) cachedData = refreshed;
      if (type === "brewer")
        brewerCategory = normalizeBrewerCategory(
          created.brewer_type || created.BrewerType || "",
        );
    } finally {
      setCombo(type, { creating: false });
    }
  }

  function clearCombo(type: ComboType) {
    setCombo(type, { ...emptyCombo() });
    if (type === "recipe") {
      activeRecipe = null;
      recipeOwnerDID = "";
      recipeExpanded = false;
    }
    if (type === "brewer") brewerCategory = "";
  }

  function comboItems(type: ComboType): ComboItem[] {
    const state = combos[type];
    return [
      ...state.results.map((entity): ComboItem => ({ kind: "entity", entity })),
      ...state.closedResults.map(
        (entity): ComboItem => ({ kind: "closed", entity }),
      ),
      ...state.suggestions.map((suggestion) => ({
        kind: "suggestion",
        suggestion,
      }) as ComboItem),
      ...(comboMeta[type].allowCreate !== false &&
      state.query.trim() &&
      !exactMatch(type)
        ? ([{ kind: "create" }] as ComboItem[])
        : []),
    ];
  }

  function selectHighlighted(type: ComboType) {
    const item = comboItems(type)[combos[type].highlight];
    if (!item) return;
    if (item.kind === "entity" || item.kind === "closed")
      selectEntity(type, item.entity);
    if (item.kind === "suggestion")
      void selectSuggestion(type, item.suggestion);
    if (item.kind === "create") startCreate(type);
  }

  function moveHighlight(type: ComboType, delta: number) {
    const items = comboItems(type);
    if (items.length === 0) return;
    const next = combos[type].highlight + delta;
    setCombo(type, {
      open: true,
      highlight: next < 0 ? items.length - 1 : next % items.length,
    });
  }

  function recipeSummary() {
    if (!activeRecipe) return "";
    const parts: string[] = [];
    if (activeRecipe.coffee_amount > 0)
      parts.push(`${Math.round(activeRecipe.coffee_amount)}g coffee`);
    if (activeRecipe.water_amount > 0)
      parts.push(`${Math.round(activeRecipe.water_amount)}g water`);
    if (activeRecipe.brewer_rkey) {
      const brewer = cachedEntities("brewer").find(
        (candidate: EntityRecord) =>
          rkey(candidate) === activeRecipe?.brewer_rkey,
      );
      if (brewer) parts.push(formatLabel("brewer", brewer));
    }
    if ((activeRecipe.pours || []).length > 0)
      parts.push(`${activeRecipe.pours.length} pours`);
    return parts.join(" · ");
  }

  async function applyRecipe(selectedRKey: string, selectedOwner = "") {
    if (!selectedRKey) {
      activeRecipe = null;
      recipeOwnerDID = "";
      recipeExpanded = false;
      return;
    }

    recipeOwnerDID = selectedOwner;
    const cachedRecipe = cachedEntities("recipe").find(
      (recipe: EntityRecord) => rkey(recipe) === selectedRKey,
    );
    if (!recipeOwnerDID && cachedRecipe?.author_did)
      recipeOwnerDID = cachedRecipe.author_did;

    const ownerQuery = recipeOwnerDID
      ? `?owner=${encodeURIComponent(recipeOwnerDID)}`
      : "";
    const response = await fetch(`/api/recipes/${selectedRKey}${ownerQuery}`, {
      credentials: "same-origin",
    });
    if (!response.ok) return;
    const recipe = await response.json();
    activeRecipe = recipe;
    recipeExpanded = false;
    if (recipe.author_did) recipeOwnerDID = recipe.author_did;
    coffeeAmount =
      recipe.coffee_amount > 0 ? String(Math.round(recipe.coffee_amount)) : "";
    waterAmount =
      recipe.water_amount > 0 ? String(Math.round(recipe.water_amount)) : "";
    pours = (recipe.pours || []).map((pour: Pour) => ({
      water: pour.water_amount || pour.water || "",
      time: pour.time_seconds || pour.time || "",
    }));

    const brewer = recipe.brewer_rkey
      ? cachedEntities("brewer").find(
          (candidate: EntityRecord) => rkey(candidate) === recipe.brewer_rkey,
        )
      : null;
    if (brewer) selectEntity("brewer", brewer);
    const recipeBrewerType =
      recipe.brewer_type || recipe.brewer_obj?.brewer_type || "";
    if (!brewer && recipeBrewerType)
      brewerCategory = normalizeBrewerCategory(recipeBrewerType);
  }

  function showRecipeOverrides() {
    return !activeRecipe || recipeExpanded;
  }

  function addPour() {
    pours = [...pours, { water: "", time: "" }];
  }

  function removePour(index: number) {
    pours = pours.filter((_, i) => i !== index);
  }

  function initializeFromDataset() {
    const d = target.dataset;
    submitLabel = d.submitLabel || "Save Brew";
    recipeOwnerDID = d.recipeOwner || "";
    coffeeAmount = d.coffeeAmount || "";
    waterAmount = d.waterAmount || "";
    grindSize = d.grindSize || "";
    temperature = d.temperature || "";
    timeSeconds = d.timeSeconds || "";
    tastingNotes = d.tastingNotes || "";
    rating = d.rating || "5";
    method = d.method || "";
    espressoYieldWeight = d.espressoYieldWeight || "";
    espressoPressure = d.espressoPressure || "";
    espressoPreInfusionSeconds = d.espressoPreInfusionSeconds || "";
    pouroverBloomWater = d.pouroverBloomWater || "";
    pouroverBloomSeconds = d.pouroverBloomSeconds || "";
    pouroverDrawdownSeconds = d.pouroverDrawdownSeconds || "";
    pouroverBypassWater = d.pouroverBypassWater || "";
    pouroverFilter = d.pouroverFilter || "";
    brewerCategory = d.brewerCategory || "";

    for (const type of ["recipe", "bean", "grinder", "brewer"] as ComboType[]) {
      const selectedRKey = d[`${type}Rkey`] || "";
      const label = d[`${type}Label`] || "";
      setCombo(type, { rkey: selectedRKey, label, query: label });
    }

    try {
      pours = JSON.parse(d.pours || "[]").map((pour: Pour) => ({
        water: pour.water ?? pour.water_amount ?? "",
        time: pour.time ?? pour.time_seconds ?? "",
      }));
    } catch {
      pours = [];
    }
  }

  onMount(() => {
    initializeFromDataset();
    const cached = appCache()?.getCachedData?.();
    if (cached) cachedData = cached;
    const listener = (data: Record<string, any>) => {
      cachedData = data;
      for (const type of ["recipe", "bean", "grinder", "brewer"] as ComboType[])
        search(type, false);
    };
    appCache()?.addListener?.(listener);
    void appCache()
      ?.getData?.()
      .then((data) => {
        if (data) cachedData = data;
        for (const type of [
          "recipe",
          "bean",
          "grinder",
          "brewer",
        ] as ComboType[])
          search(type, false);
        if (target.dataset.recipeRkey)
          void applyRecipe(
            target.dataset.recipeRkey,
            target.dataset.recipeOwner || "",
          );
      });

    return () => {
      appCache()?.removeListener?.(listener);
      Object.values(suggestTimers).forEach(clearTimeout);
    };
  });
</script>

<fieldset class="space-y-6 border-0 p-0 m-0 min-w-0">
  <input type="hidden" name="recipe_owner_did" value={recipeOwnerDID} />

  <div class="combo-select">
    <label class="form-label">Recipe (Optional)</label>
    <p class="text-sm text-muted mb-2">
      Select a recipe to autofill brew parameters
    </p>
    <div class="alert-warning px-3 py-2 mb-2 text-xs">
      Recipes are in early alpha, the format may change. Your brew data won't be
      affected.
    </div>
    {@render ComboControl("recipe")}
  </div>

  {#if activeRecipe}
    <div class="section-box">
      <div class="flex items-center justify-between gap-2">
        <p class="text-sm text-emphasis flex-1">{recipeSummary()}</p>
        <button
          type="button"
          onclick={() => (recipeExpanded = !recipeExpanded)}
          class="text-sm btn-secondary"
        >
          {recipeExpanded ? "Collapse" : "Edit"}
        </button>
      </div>
    </div>
  {/if}

  <fieldset class="space-y-6 border border-brown-200 rounded-lg p-4 min-w-0">
    <legend class="text-sm font-semibold text-secondary px-2">Coffee</legend>
    <div class="combo-select">
      <label class="form-label"
        >Coffee Bean <span class="text-red-500">*</span></label
      >
      {@render ComboControl("bean")}
    </div>
    {#if showRecipeOverrides()}
      <Field
        label="Coffee Amount (grams)"
        helper="Amount of ground coffee used"
      >
        <input
          type="number"
          name="coffee_amount"
          bind:value={coffeeAmount}
          placeholder="e.g. 18"
          step="1"
          class="w-full form-input-lg"
        />
      </Field>
    {:else}
      <input type="hidden" name="coffee_amount" value={coffeeAmount} />
    {/if}
    <div class="combo-select">
      <label class="form-label">Grinder</label>
      {@render ComboControl("grinder")}
    </div>
    <Field
      label="Grind Size"
      helper={'Enter a number (grinder setting) or description (e.g. "Medium", "Fine")'}
    >
      <input
        type="text"
        name="grind_size"
        bind:value={grindSize}
        placeholder="e.g. 18, Medium, 3.5, Fine"
        class="w-full form-input-lg"
      />
    </Field>
  </fieldset>

  <fieldset class="space-y-6 border border-brown-200 rounded-lg p-4 min-w-0">
    <legend class="text-sm font-semibold text-secondary px-2">Brewing</legend>
    {#if showRecipeOverrides()}
      <div class="combo-select">
        <label class="form-label">Brew Method</label>
        {@render ComboControl("brewer")}
      </div>
      <Field
        label="Water Amount (grams)"
        helper={pours.length > 0
          ? "Total water (pours tracked separately below)"
          : "Total water used"}
      >
        <input
          type="number"
          name="water_amount"
          bind:value={waterAmount}
          placeholder="e.g. 250"
          step="1"
          class="w-full form-input-lg"
        />
      </Field>
      {@render PoursEditor()}
    {:else}
      <input type="hidden" name="brewer_rkey" value={combos.brewer.rkey} />
      <input type="hidden" name="water_amount" value={waterAmount} />
      {#each pours as pour, index}
        <input type="hidden" name={`pour_water_${index}`} value={pour.water} />
        <input type="hidden" name={`pour_time_${index}`} value={pour.time} />
      {/each}
    {/if}
    <Field label="Temperature (°F/°C)">
      <input
        type="number"
        name="temperature"
        bind:value={temperature}
        placeholder="e.g. 93.5"
        step="0.1"
        class="w-full form-input-lg"
      />
    </Field>
    <Field label="Brew Time (seconds)">
      <input
        type="number"
        name="time_seconds"
        bind:value={timeSeconds}
        placeholder="e.g. 180"
        class="w-full form-input-lg"
      />
    </Field>
  </fieldset>

  {#if brewerCategory === "espresso"}
    <fieldset class="space-y-6 border border-brown-200 rounded-lg p-4 min-w-0">
      <legend class="text-sm font-semibold text-secondary px-2">Espresso</legend
      >
      <Field label="Yield Weight (grams)" helper="Weight of espresso output">
        <input
          type="number"
          name="espresso_yield_weight"
          bind:value={espressoYieldWeight}
          placeholder="e.g. 36"
          step="0.1"
          class="w-full form-input-lg"
        />
      </Field>
      <Field label="Pressure (bar)" helper="Brewing pressure">
        <input
          type="number"
          name="espresso_pressure"
          bind:value={espressoPressure}
          placeholder="e.g. 9"
          step="0.1"
          class="w-full form-input-lg"
        />
      </Field>
      <Field label="Pre-infusion Time (seconds)">
        <input
          type="number"
          name="espresso_pre_infusion_seconds"
          bind:value={espressoPreInfusionSeconds}
          placeholder="e.g. 5"
          class="w-full form-input-lg"
        />
      </Field>
    </fieldset>
  {/if}

  {#if brewerCategory === "pourover"}
    <fieldset class="space-y-6 border border-brown-200 rounded-lg p-4 min-w-0">
      <legend class="text-sm font-semibold text-secondary px-2"
        >Pour-over Details</legend
      >
      <div class="grid grid-cols-2 gap-4">
        <Field label="Bloom Water (grams)" helper="Water for bloom">
          <input
            type="number"
            name="pourover_bloom_water"
            bind:value={pouroverBloomWater}
            placeholder="e.g. 50"
            class="w-full form-input-lg"
          />
        </Field>
        <Field label="Bloom Time (seconds)" helper="Bloom wait time">
          <input
            type="number"
            name="pourover_bloom_seconds"
            bind:value={pouroverBloomSeconds}
            placeholder="e.g. 45"
            class="w-full form-input-lg"
          />
        </Field>
      </div>
      <Field
        label="Drawdown Time (seconds)"
        helper="Time after last pour until bed is dry"
      >
        <input
          type="number"
          name="pourover_drawdown_seconds"
          bind:value={pouroverDrawdownSeconds}
          placeholder="e.g. 30"
          class="w-full form-input-lg"
        />
      </Field>
      <Field label="Bypass Water (grams)" helper="Water added after brewing">
        <input
          type="number"
          name="pourover_bypass_water"
          bind:value={pouroverBypassWater}
          placeholder="e.g. 100"
          class="w-full form-input-lg"
        />
      </Field>
      <Field label="Filter" helper="Type of filter used">
        <input
          type="text"
          name="pourover_filter"
          bind:value={pouroverFilter}
          placeholder="e.g. paper, metal, cloth"
          class="w-full form-input-lg"
        />
      </Field>
    </fieldset>
  {/if}

  <fieldset class="space-y-6 border border-brown-200 rounded-lg p-4 min-w-0">
    <legend class="text-sm font-semibold text-secondary px-2">Results</legend>
    <Field label="Tasting Notes">
      <textarea
        name="tasting_notes"
        bind:value={tastingNotes}
        placeholder="Describe the flavors, aroma, and your thoughts..."
        rows="4"
        class="w-full form-input-lg"
      ></textarea>
    </Field>
    <div>
      <label class="form-label" for="brew-rating">Rating</label>
      <input
        id="brew-rating"
        type="range"
        name="rating"
        min="1"
        max="10"
        bind:value={rating}
        class="w-full accent-brown-700"
      />
      <div class="text-center text-2xl font-bold text-secondary">
        {rating}/10
      </div>
    </div>
  </fieldset>

  <button
    type="submit"
    class="w-full btn-primary py-3 px-6 rounded-xl font-semibold text-lg shadow-lg hover:shadow-xl"
  >
    {submitLabel}
  </button>
</fieldset>

{#snippet ComboControl(type: ComboType)}
  <input
    type="hidden"
    name={comboMeta[type].inputName}
    value={combos[type].rkey}
    required={comboMeta[type].required}
  />
  <div class="relative">
    <input
      type="text"
      bind:value={combos[type].query}
      oninput={() => search(type, true)}
      onfocus={() => {
        focusedCombo = type;
        search(type, true);
      }}
      onblur={() =>
        setTimeout(
          () =>
            setCombo(type, {
              open: false,
              query: combos[type].rkey
                ? combos[type].label
                : combos[type].query,
            }),
          150,
        )}
      onkeydown={(event) => {
        if (event.key === "Escape") setCombo(type, { open: false });
        if (event.key === "ArrowDown") {
          event.preventDefault();
          moveHighlight(type, 1);
        }
        if (event.key === "ArrowUp") {
          event.preventDefault();
          moveHighlight(type, -1);
        }
        if (event.key === "Enter") {
          event.preventDefault();
          selectHighlighted(type);
        }
      }}
      placeholder={comboMeta[type].placeholder}
      class="w-full form-input-lg"
      autocomplete="off"
      role="combobox"
      aria-expanded={combos[type].open ? "true" : "false"}
      aria-label={comboMeta[type].label}
    />
    {#if combos[type].rkey}
      <button
        type="button"
        onclick={() => clearCombo(type)}
        class="absolute right-2 top-1/2 -translate-y-1/2 text-placeholder hover:text-muted"
        aria-label="Clear selection">×</button
      >
    {/if}
  </div>

  {#if combos[type].open && (comboItems(type).length > 0 || combos[type].query.trim())}
    <div
      role="listbox"
      tabindex="-1"
      class="combo-dropdown"
      onmousedown={(event) => event.preventDefault()}
    >
      {#if combos[type].creating}
        <div class="combo-creating">Creating...</div>
      {:else}
        {#if combos[type].results.length > 0}
          <div class="combo-section-label">{comboMeta[type].sectionLabel}</div>
          {#each combos[type].results as entity, index}
            <button
              type="button"
              class="combo-item"
              role="option"
              aria-selected={combos[type].highlight === index}
              data-highlighted={combos[type].highlight === index}
              onmouseenter={() => setCombo(type, { highlight: index })}
              onclick={() => selectEntity(type, entity)}
            >
              {formatLabel(type, entity)}
            </button>
          {/each}
        {/if}
        {#if combos[type].closedResults.length > 0}
          <div class="combo-section-label">Closed bags</div>
          {#each combos[type].closedResults as entity, index}
            <button
              type="button"
              class="combo-item opacity-60"
              role="option"
              onclick={() => selectEntity(type, entity)}
              >{formatLabel(type, entity)}</button
            >
          {/each}
        {/if}
        {#if combos[type].suggestions.length > 0}
          <div class="combo-section-label">Community</div>
          {#each combos[type].suggestions as suggestion}
            <button
              type="button"
              class="combo-item"
              role="option"
              onclick={() => selectSuggestion(type, suggestion)}
            >
              <div>{suggestion.name}</div>
              <div class="combo-item-sub">
                {#if suggestion.fields?.origin}{suggestion.fields.origin}{/if}
                {#if suggestion.fields?.origin && suggestion.fields?.roastLevel}
                  ·
                {/if}
                {#if suggestion.fields?.roastLevel}{suggestion.fields
                    .roastLevel}{/if}
                {#if suggestion.fields?.location}{suggestion.fields
                    .location}{/if}
                {#if (suggestion.count || 0) > 1}
                  · {suggestion.count} users{/if}
              </div>
            </button>
          {/each}
        {/if}
        {#if comboMeta[type].allowCreate !== false && combos[type].query.trim() && !exactMatch(type)}
          <button
            type="button"
            class="combo-item-create"
            role="option"
            onclick={() => startCreate(type)}
            >Create "{combos[type].query.trim()}"</button
          >
        {/if}
        {#if comboItems(type).length === 0 && combos[type].query.trim()}<div
            class="combo-creating"
          >
            No matches found
          </div>{/if}
      {/if}
    </div>
  {/if}

  {#if combos[type].showCreate}
    <div
      class="mt-2 p-3 rounded-lg"
      style="background: var(--surface-bg); border: 1px solid var(--surface-border);"
    >
      <p class="text-sm font-medium text-primary mb-2">
        Creating: <span class="font-semibold"
          >{combos[type].createData.name}</span
        >
      </p>
      <div class="space-y-2">
        {#each config(type).extraFields || [] as field}
          {#if field.type === "select"}
            <select
              bind:value={combos[type].createData[field.name]}
              class="w-full form-input text-sm"
            >
              <option value="">{field.label} (optional)</option>
              {#each field.options || [] as option}<option value={option}
                  >{option}</option
                >{/each}
            </select>
          {:else}
            <input
              type={field.type === "url" ? "url" : "text"}
              bind:value={combos[type].createData[field.name]}
              placeholder={field.placeholder || `${field.label} (optional)`}
              class="w-full form-input text-sm"
            />
          {/if}
        {/each}
      </div>
      <div class="flex gap-2 mt-3">
        <button
          type="button"
          class="btn-primary text-sm"
          disabled={combos[type].creating}
          onclick={() => createEntity(type, { ...combos[type].createData })}
          >Create</button
        >
        <button
          type="button"
          class="btn-secondary text-sm"
          onclick={() => setCombo(type, { showCreate: false, createData: {} })}
          >Cancel</button
        >
      </div>
    </div>
  {/if}
{/snippet}

{#snippet PoursEditor()}
  {#if pours.length === 0}
    <button
      type="button"
      class="text-sm text-muted hover:text-secondary font-medium"
      onclick={addPour}>+ Add pours</button
    >
  {:else}
    <div>
      <div class="flex items-center justify-between mb-2">
        <span class="block text-sm font-medium text-primary">Pours</span>
        <button type="button" onclick={addPour} class="text-sm btn-secondary"
          >+ Add Pour</button
        >
      </div>
      <p class="text-sm text-emphasis mb-3">
        Track individual pours for bloom and subsequent additions
      </p>
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
{/snippet}
