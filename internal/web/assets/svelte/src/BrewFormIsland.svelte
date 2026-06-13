<script lang="ts">
  import { onMount } from "svelte";
  import EntityCombo from "./EntityCombo.svelte";
  import Field from "./BrewFormField.svelte";
  import PoursEditor from "./PoursEditor.svelte";
  import type { AppCacheAPI } from "./appCache";
  import {
    comboSelectEntities,
    type EntityRecord,
  } from "./comboSelectRegistry";

  type ComboType = "recipe" | "bean" | "grinder" | "brewer";
  type Pour = {
    water: number | string;
    time: number | string;
    water_amount?: number;
    time_seconds?: number;
  };
  let { target }: { target: HTMLElement } = $props();

  let cachedData = $state<Record<string, any>>({});
  let recipeOwnerDID = $state("");
  let activeRecipe = $state<EntityRecord | null>(null);
  let recipeExpanded = $state(false);
  let brewerCategory = $state("");
  let recipeRKey = $state("");
  let recipeLabel = $state("");
  let beanRKey = $state("");
  let beanLabel = $state("");
  let grinderRKey = $state("");
  let grinderLabel = $state("");
  let brewerRKey = $state("");
  let brewerLabel = $state("");
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

  function appCache(): AppCacheAPI | undefined {
    return window.AppCache;
  }

  function numericValue(value: string | number | null) {
    if (value === null || value === "") return null;
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : Number.NaN;
  }

  function mustBePositive(value: string | number | null) {
    const parsed = numericValue(value);
    return parsed !== null && (!Number.isFinite(parsed) || parsed <= 0);
  }

  let coffeeAmountError = $derived(mustBePositive(coffeeAmount));
  let waterAmountError = $derived(mustBePositive(waterAmount));
  let temperatureError = $derived(mustBePositive(temperature));
  let timeSecondsError = $derived(mustBePositive(timeSeconds));

  function config(type: ComboType) {
    return comboSelectEntities[type] || {};
  }

  function formatLabel(type: ComboType, entity: EntityRecord) {
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

  function selectEntity(type: ComboType, entity: EntityRecord) {
    const label = formatLabel(type, entity);
    if (type === "recipe") {
      recipeRKey = rkey(entity);
      recipeLabel = label;
    }
    if (type === "bean") {
      beanRKey = rkey(entity);
      beanLabel = label;
    }
    if (type === "grinder") {
      grinderRKey = rkey(entity);
      grinderLabel = label;
    }
    if (type === "brewer") {
      brewerRKey = rkey(entity);
      brewerLabel = label;
    }
    if (type === "brewer")
      brewerCategory = normalizeBrewerCategory(
        entity.brewer_type || entity.BrewerType || "",
      );
    if (type === "recipe")
      void applyRecipe(rkey(entity), entity.author_did || "");
  }

  function clearCombo(type: ComboType) {
    if (type === "recipe") {
      recipeRKey = "";
      recipeLabel = "";
      activeRecipe = null;
      recipeOwnerDID = "";
      recipeExpanded = false;
    }
    if (type === "bean") {
      beanRKey = "";
      beanLabel = "";
    }
    if (type === "grinder") {
      grinderRKey = "";
      grinderLabel = "";
    }
    if (type === "brewer") {
      brewerRKey = "";
      brewerLabel = "";
      brewerCategory = "";
    }
  }

  function handleComboChange(type: ComboType, detail: Record<string, any>) {
    if (!detail.rkey) {
      clearCombo(type);
      return;
    }
    if (type === "recipe") {
      recipeRKey = detail.rkey;
      recipeLabel = detail.entity
        ? formatLabel(type, detail.entity)
        : recipeLabel;
      void applyRecipe(
        detail.rkey,
        detail.owner || detail.entity?.author_did || "",
      );
      return;
    }
    if (type === "bean") {
      beanRKey = detail.rkey;
      beanLabel = detail.entity ? formatLabel(type, detail.entity) : beanLabel;
      return;
    }
    if (type === "grinder") {
      grinderRKey = detail.rkey;
      grinderLabel = detail.entity
        ? formatLabel(type, detail.entity)
        : grinderLabel;
      return;
    }
    if (type === "brewer") {
      brewerRKey = detail.rkey;
      brewerLabel = detail.entity
        ? formatLabel(type, detail.entity)
        : brewerLabel;
      brewerCategory = normalizeBrewerCategory(
        detail.entity?.brewer_type || detail.entity?.BrewerType || "",
      );
    }
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
    if (cachedRecipe && !recipeLabel)
      recipeLabel = formatLabel("recipe", cachedRecipe);
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
    if (!recipeLabel) recipeLabel = formatLabel("recipe", recipe);
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

    recipeRKey = d.recipeRkey || "";
    recipeLabel = d.recipeLabel || "";
    beanRKey = d.beanRkey || "";
    beanLabel = d.beanLabel || "";
    grinderRKey = d.grinderRkey || "";
    grinderLabel = d.grinderLabel || "";
    brewerRKey = d.brewerRkey || "";
    brewerLabel = d.brewerLabel || "";

    try {
      pours = JSON.parse(d.pours || "[]").map((pour: Pour) => ({
        water: pour.water ?? pour.water_amount ?? "",
        time: pour.time ?? pour.time_seconds ?? "",
      }));
    } catch {
      pours = [];
    }
  }

  initializeFromDataset();

  onMount(() => {
    const cached = appCache()?.getCachedData?.();
    if (cached) cachedData = cached;
    const listener = (data: Record<string, any>) => {
      cachedData = data;
    };
    appCache()?.addListener?.(listener);
    void appCache()
      ?.getData?.()
      .then((data) => {
        if (data) cachedData = data;
        if (target.dataset.recipeRkey)
          void applyRecipe(
            target.dataset.recipeRkey,
            target.dataset.recipeOwner || "",
          );
      });

    return () => {
      appCache()?.removeListener?.(listener);
    };
  });
</script>

<fieldset class="space-y-6 border-0 p-0 m-0 min-w-0">
  <input type="hidden" name="recipe_owner_did" value={recipeOwnerDID} />

  <div class="combo-select">
    <span class="form-label">Recipe (Optional)</span>
    <p class="text-sm text-muted mb-2">
      Select a recipe to autofill brew parameters
    </p>
    <div class="alert-warning px-3 py-2 mb-2 text-xs">
      Recipes are in early alpha, the format may change. Your brew data won't be
      affected.
    </div>
    <EntityCombo
      entityType="recipe"
      inputName="recipe_rkey"
      apiEndpoint="/api/recipes"
      suggestEndpoint="/api/suggestions/recipes"
      placeholder="Search recipes..."
      sectionLabel="Your recipes"
      passthrough={true}
      allowCreate={false}
      bind:rkey={recipeRKey}
      bind:label={recipeLabel}
      ariaLabel="Search recipes"
      onChange={(detail) => handleComboChange("recipe", detail)}
    />
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
      <span class="form-label"
        >Coffee Bean <span class="text-red-500">*</span></span
      >
      <EntityCombo
        entityType="bean"
        inputName="bean_rkey"
        apiEndpoint="/api/beans"
        suggestEndpoint="/api/suggestions/beans"
        placeholder="Search beans..."
        sectionLabel="Your beans"
        required={true}
        allowCreate={false}
        bind:rkey={beanRKey}
        bind:label={beanLabel}
        ariaLabel="Search coffee beans"
        onChange={(detail) => handleComboChange("bean", detail)}
      />
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
          aria-invalid={coffeeAmountError}
        />
        {#if coffeeAmountError}
          <p class="text-xs text-red-600 mt-1">
            Coffee amount must be greater than 0.
          </p>
        {/if}
      </Field>
    {:else}
      <input type="hidden" name="coffee_amount" value={coffeeAmount} />
    {/if}
    <div class="combo-select">
      <span class="form-label">Grinder</span>
      <EntityCombo
        entityType="grinder"
        inputName="grinder_rkey"
        apiEndpoint="/api/grinders"
        suggestEndpoint="/api/suggestions/grinders"
        placeholder="Search grinders..."
        sectionLabel="Your grinders"
        bind:rkey={grinderRKey}
        bind:label={grinderLabel}
        ariaLabel="Search grinders"
        onChange={(detail) => handleComboChange("grinder", detail)}
      />
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
        <span class="form-label">Brew Method</span>
        <EntityCombo
          entityType="brewer"
          inputName="brewer_rkey"
          apiEndpoint="/api/brewers"
          suggestEndpoint="/api/suggestions/brewers"
          placeholder="Search brew methods..."
          sectionLabel="Your brewers"
          bind:rkey={brewerRKey}
          bind:label={brewerLabel}
          ariaLabel="Search brew methods"
          onChange={(detail) => handleComboChange("brewer", detail)}
        />
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
          aria-invalid={waterAmountError}
        />
        {#if waterAmountError}
          <p class="text-xs text-red-600 mt-1">
            Water amount must be greater than 0.
          </p>
        {/if}
      </Field>
      <PoursEditor bind:pours expectedWater={waterAmount} />
    {:else}
      <input type="hidden" name="brewer_rkey" value={brewerRKey} />
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
        aria-invalid={temperatureError}
      />
      {#if temperatureError}
        <p class="text-xs text-red-600 mt-1">
          Temperature must be greater than 0.
        </p>
      {/if}
    </Field>
    <Field label="Brew Time (seconds)">
      <input
        type="number"
        name="time_seconds"
        bind:value={timeSeconds}
        placeholder="e.g. 180"
        class="w-full form-input-lg"
        aria-invalid={timeSecondsError}
      />
      {#if timeSecondsError}
        <p class="text-xs text-red-600 mt-1">
          Brew time must be greater than 0.
        </p>
      {/if}
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
