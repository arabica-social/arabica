<script lang="ts">
  import { tick } from "svelte";
  import type { Draft, DraftPatch, FieldPath } from "#lib/ai/types.js";
  import {
    normalizeBrewerCategory,
    preparationLayout,
    type BrewFormMode,
  } from "#lib/utils/brewFormPolicy.js";
  import { fields, brewGhostText, type EditorField } from "./fields";
  type LibraryItem = {
    rkey: string;
    name: string;
    brewer_type?: string;
    brewer_rkey?: string;
    coffee_amount?: number;
    water_amount?: number;
    pours?: { water_amount: number; time_seconds: number }[];
    notes?: string;
  };
  // /api/data supplies only the authenticated user's local collection.
  export type Library = Record<string, LibraryItem[]>;
  let {
    draft,
    brewerType = "",
    library = {},
    disabled = false,
    presentation = "standard",
    ratingInEditor = true,
    onchange,
  }: {
    draft: Draft;
    brewerType?: string;
    library?: Library;
    disabled?: boolean;
    presentation?: "standard" | "chat";
    ratingInEditor?: boolean;
    onchange: (patch: DraftPatch, brewerType?: string) => void;
  } = $props();
  let values = $derived(draft.fields as DraftPatch);
  const value = (key: FieldPath) => values[key] ?? "";
  async function select(key: FieldPath, input: HTMLSelectElement) {
    if (key === "recipe_rkey") selectRecipe(input.value);
    else change(key, input);
    await tick();
    input.value = String(value(key));
  }
  function change(
    key: FieldPath,
    input: HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement,
    numeric = false,
  ) {
    const next =
      input.value === "" ? null : numeric ? Number(input.value) : input.value;
    const brewer =
      key === "brewer_rkey"
        ? library.brewers?.find((b) => b.rkey === next)?.brewer_type
        : undefined;
    onchange({ [key]: next }, brewer);
  }
  let selectedRecipe = $derived(
    !values.recipe_owner_did
      ? library.recipes?.find((r) => r.rkey === values.recipe_rkey)
      : undefined,
  );
  let recipeNotice = $state("");
  function selectRecipe(rkey: string) {
    recipeNotice = "";
    const patch: DraftPatch = {
      recipe_rkey: rkey || null,
      recipe_owner_did: null,
    };
    const recipe = library.recipes?.find((r) => r.rkey === rkey);
    if (!recipe) {
      onchange(patch);
      return;
    }
    const notices: string[] = [];
    for (const key of ["coffee_amount", "water_amount"] as const) {
      const amount = recipe[key];
      const max = key === "coffee_amount" ? 1000 : 10000;
      if (amount && amount > 0) {
        if (Number.isInteger(amount) && amount <= max) patch[key] = amount;
        else
          notices.push(
            `${key === "coffee_amount" ? "Coffee" : "Water"} is ${amount} g in this recipe. Enter a whole-gram amount in the brew; your existing value was kept.`,
          );
      }
    }
    // /api/data loses the owner from brewerRef. A colliding rkey is not proof
    // that a recipe links to an own brewer. Keep the explicit local selection.
    const brewer = library.brewers?.find((b) => b.rkey === values.brewer_rkey);
    const nextType = brewer?.brewer_type;
    const recipeType = recipe.brewer_type;
    const compatiblePours =
      normalizeBrewerCategory(nextType) === "pourover" &&
      !!recipeType &&
      normalizeBrewerCategory(recipeType) === "pourover";
    if (recipe.pours?.length) {
      if (
        compatiblePours &&
        recipe.pours.length <= 100 &&
        recipe.pours.every(
          (p) =>
            Number.isSafeInteger(p.water_amount) &&
            p.water_amount >= 0 &&
            Number.isSafeInteger(p.time_seconds) &&
            p.time_seconds >= 0,
        )
      ) {
        patch.pours = recipe.pours.map((p) => ({ ...p }));
      } else
        notices.push(
          "Recipe pours were not copied. Choose a compatible saved pour-over brewer and reselect the recipe.",
        );
    }
    if (!brewer)
      notices.push(
        "Choose one of your saved brewers. Recipes do not change this selection.",
      );
    recipeNotice = notices.join(" ");
    onchange(patch);
  }
  let pours = $derived(values.pours ?? []);
  let category = $derived(normalizeBrewerCategory(brewerType));
  const hasValue = (v: unknown) =>
    v !== null &&
    v !== undefined &&
    v !== "" &&
    (!Array.isArray(v) || v.length > 0);
  const preparationField = (key: FieldPath) =>
    [
      "coffee_amount",
      "water_amount",
      "temperature",
      "time_seconds",
      "grinder_rkey",
      "grind_size",
      "pours",
    ].includes(key) || key.includes(".");
  let layout = $derived(preparationLayout(brewerType));
  let hasPreparation = $derived(
    Object.entries(values).some(
      ([key, v]) => preparationField(key as FieldPath) && hasValue(v),
    ),
  );
  let chosenMode = $state<BrewFormMode>();
  let detailsRevealed = $state(false);
  $effect(() => {
    if (hasPreparation) detailsRevealed = true;
  });
  let mode = $derived(
    presentation === "chat"
      ? "detailed"
      : (chosenMode ??
          (hasPreparation || detailsRevealed ? "detailed" : "quick")),
  );
  let ratingExpanded = $state(false);
  let showRating = $derived(
    ratingInEditor &&
      (presentation === "chat" || ratingExpanded || hasValue(values.rating)),
  );
  let baseFields = $derived(
    fields[draft.kind].filter(
      (f) =>
        draft.kind !== "brew" ||
        (["recipe_rkey", "bean_rkey", "brewer_rkey"].includes(f.key) &&
          (presentation !== "chat" || f.key !== "recipe_rkey")),
    ),
  );
  // Chat is a compact correction surface, not the full preparation form.
  // Hidden advanced values remain in the canonical draft and read-only preview.
  let chatFields = $derived(
    fields.brew.filter((f) =>
      [
        "coffee_amount",
        layout.mainWater ? "water_amount" : "espresso_params.yield_weight",
        "grinder_rkey",
        "grind_size",
        "temperature",
        "time_seconds",
      ].includes(f.key),
    ),
  );
  const showGrind = () =>
    hasValue(values.grinder_rkey) || hasValue(values.grind_size);
  let primaryFields = $derived(
    fields.brew.filter(
      (f) =>
        f.key === "coffee_amount" ||
        (f.key === "water_amount" && layout.mainWater) ||
        (f.key === "time_seconds" && layout.mainTime) ||
        (f.key === "espresso_params.yield_weight" && category === "espresso") ||
        (layout.mainGrinder &&
          (f.key === "grinder_rkey" ||
            (f.key === "grind_size" && showGrind()))),
    ),
  );
  let moreFields = $derived(
    fields.brew.filter(
      (f) =>
        f.key === "temperature" ||
        (f.key === "time_seconds" && !layout.mainTime) ||
        (f.key === "water_amount" &&
          !layout.mainWater &&
          hasValue(value(f.key))) ||
        (!layout.mainGrinder &&
          (f.key === "grinder_rkey" ||
            (f.key === "grind_size" && showGrind()))) ||
        (f.key.startsWith("espresso_params.") &&
          (category === "espresso" || hasValue(value(f.key))) &&
          !(
            f.key === "espresso_params.yield_weight" && category === "espresso"
          )) ||
        (f.key.startsWith("pourover_params.") &&
          (category === "pourover" || hasValue(value(f.key)))),
    ),
  );
  let hasMore = $derived(
    moreFields.some((f) => hasValue(value(f.key))) || pours.length > 0,
  );
  let moreChoice = $state<boolean>();
  let moreExpanded = $derived(moreChoice ?? hasMore);
  let resultsFields = $derived(
    fields.brew.filter(
      (f) => f.key === "tasting_notes" || (f.key === "rating" && showRating),
    ),
  );
  const labelFor = (field: EditorField) =>
    field.key === "time_seconds" ? layout.timeLabel : field.label;
  let detailsSummary = $derived(
    [
      hasValue(values.coffee_amount) && `${values.coffee_amount}g coffee`,
      hasValue(values.water_amount) && `${values.water_amount}g water`,
      hasValue(values.time_seconds) && `${values.time_seconds}s`,
      hasValue(values.temperature) && `${values.temperature}° (recorded unit)`,
      hasValue(values.grinder_rkey) &&
        (library.grinders?.find((g) => g.rkey === values.grinder_rkey)?.name ??
          "Grinder selected"),
      hasValue(values.grind_size) && `Grind setting: ${values.grind_size}`,
      Object.keys(values).some(
        (k) =>
          k.startsWith("espresso_params.") && hasValue(values[k as FieldPath]),
      ) && "Espresso details",
      Object.keys(values).some(
        (k) =>
          k.startsWith("pourover_params.") && hasValue(values[k as FieldPath]),
      ) && "Pour-over details",
      pours.length > 0 && `${pours.length} pours`,
    ]
      .filter(Boolean)
      .join(" · "),
  );

  function pourChange(
    index: number,
    key: "water_amount" | "time_seconds",
    input: HTMLInputElement,
  ) {
    onchange({
      pours: pours.map((p, i) =>
        i === index ? { ...p, [key]: Number(input.value) } : p,
      ),
    });
  }
</script>

{#snippet renderField(field: EditorField)}
  <div class:wide={field.type === "textarea" || field.key === "recipe_rkey"}>
    <label for={`draft-${field.key}`}
      >{labelFor(field)}{field.required ? " *" : ""}</label
    >
    {#if field.library}
      <select
        id={`draft-${field.key}`}
        name={field.key}
        class="form-input-lg"
        value={value(field.key)}
        required={field.required}
        onchange={(e) => select(field.key, e.currentTarget)}
      >
        <option value="">Select {field.label.toLowerCase()}</option>
        {#if value(field.key) && !(library[field.library] ?? []).some((item) => item.rkey === value(field.key))}
          <option value={String(value(field.key))}
            >{String(value(field.key))} (unavailable)</option
          >
        {/if}
        {#each library[field.library] ?? [] as item (item.rkey)}<option
            value={item.rkey}>{item.name}</option
          >{/each}
      </select>
    {:else if field.options}
      <select
        id={`draft-${field.key}`}
        name={field.key}
        class="form-input-lg"
        value={value(field.key)}
        onchange={(e) => change(field.key, e.currentTarget)}
      >
        <option value="">Not specified</option>
        {#if value(field.key) && !field.options.includes(String(value(field.key)))}<option
            value={String(value(field.key))}>{String(value(field.key))}</option
          >{/if}
        {#each field.options as option}<option value={option}>{option}</option
          >{/each}
      </select>
    {:else if field.type === "textarea"}
      <textarea
        id={`draft-${field.key}`}
        name={field.key}
        class="form-textarea"
        rows="3"
        placeholder={draft.kind === "brew"
          ? brewGhostText[field.key]
          : undefined}
        value={String(value(field.key))}
        oninput={(e) => change(field.key, e.currentTarget)}
      ></textarea>
    {:else}
      <input
        id={`draft-${field.key}`}
        name={field.key}
        class="form-input-lg"
        type={field.type ?? "text"}
        placeholder={draft.kind === "brew"
          ? brewGhostText[field.key]
          : undefined}
        value={String(value(field.key))}
        required={field.required}
        min={field.min}
        max={field.max}
        step={field.type === "number" ? (field.step ?? "1") : undefined}
        oninput={(e) =>
          change(field.key, e.currentTarget, field.type === "number")}
      />
    {/if}
  </div>
{/snippet}

<fieldset {disabled} class="draft-fields" class:chat={presentation === "chat"}>
  {#if draft.kind === "brew" && presentation !== "chat"}
    <div class="wide mode-switch" role="group" aria-label="Brew logging mode">
      <button
        type="button"
        class="btn-secondary"
        aria-pressed={mode === "quick"}
        onclick={() => (chosenMode = "quick")}>Quick</button
      >
      <button
        type="button"
        class="btn-secondary"
        aria-pressed={mode === "detailed"}
        onclick={() => (chosenMode = "detailed")}>Detailed</button
      >
    </div>
  {/if}
  {#each baseFields as field (field.key)}{@render renderField(field)}{/each}
  {#if draft.kind === "brew" && presentation !== "chat" && selectedRecipe}
    <details class="wide recipe-details">
      <summary>{selectedRecipe.name} — recipe details</summary>
      <p>
        {selectedRecipe.coffee_amount ?? "—"} g coffee · {selectedRecipe.water_amount ??
          "—"} g water{selectedRecipe.brewer_type
          ? ` · ${selectedRecipe.brewer_type}`
          : ""}
      </p>
      {#if selectedRecipe.pours?.length}
        <ol>
          {#each selectedRecipe.pours as pour}<li>
              {pour.water_amount} g at {pour.time_seconds} s
            </li>{/each}
        </ol>
      {/if}
      {#if selectedRecipe.notes}<p class="recipe-notes">
          {selectedRecipe.notes}
        </p>{/if}
    </details>
  {/if}
  {#if draft.kind === "brew" && recipeNotice}<p
      class="wide recipe-notice"
      role="status"
    >
      {recipeNotice}
    </p>{/if}
  {#if draft.kind === "bean"}
    <label class="wide checkbox"
      ><input
        type="checkbox"
        checked={values.closed ?? false}
        onchange={(e) => onchange({ closed: e.currentTarget.checked })}
      /> Finished bag</label
    >
  {/if}
  {#if draft.kind === "brew"}
    {#if presentation === "chat"}
      {#each chatFields as field (field.key)}{@render renderField(field)}{/each}
    {:else if mode === "quick" && hasPreparation}
      <div class="wide preparation-summary" aria-label="Brewing details added">
        <p>{detailsSummary}</p>
        <button type="button" onclick={() => (chosenMode = "detailed")}
          >Edit brewing details</button
        >
      </div>
    {/if}
    {#if presentation !== "chat" && mode === "detailed"}
      <div class="wide preparation-fields">
        <div class="preparation-grid">
          {#each primaryFields as field (field.key)}{@render renderField(
              field,
            )}{/each}
        </div>
        <details
          class="wide more-details"
          open={moreExpanded}
          ontoggle={(e) => {
            if (e.currentTarget.open !== moreExpanded)
              moreChoice = e.currentTarget.open;
          }}
        >
          <summary>More brewing details</summary>
          <div class="detail-fields">
            {#each moreFields as field (field.key)}{@render renderField(
                field,
              )}{/each}
          </div>
          {#if category === "pourover" || pours.length}
            <div class="wide pours">
              {#each pours as pour, index}
                <div class="pour">
                  <label
                    >Pour {index + 1} water (g)<input
                      class="form-input-lg"
                      type="number"
                      min="0"
                      step="1"
                      placeholder="e.g. 50"
                      value={pour.water_amount}
                      oninput={(e) =>
                        pourChange(index, "water_amount", e.currentTarget)}
                    /></label
                  >
                  <label
                    >Time (s)<input
                      class="form-input-lg"
                      type="number"
                      min="0"
                      step="1"
                      placeholder="e.g. 0"
                      value={pour.time_seconds}
                      oninput={(e) =>
                        pourChange(index, "time_seconds", e.currentTarget)}
                    /></label
                  >
                  <button
                    type="button"
                    class="btn-secondary"
                    aria-label={`Remove pour ${index + 1}`}
                    onclick={() =>
                      onchange({ pours: pours.filter((_, i) => i !== index) })}
                    >Remove</button
                  >
                </div>
              {/each}
              <button
                type="button"
                class="btn-secondary"
                disabled={pours.length >= 100}
                onclick={() =>
                  onchange({
                    pours: [...pours, { water_amount: 0, time_seconds: 0 }],
                  })}>Add pour</button
              >
            </div>
          {/if}
        </details>
      </div>
    {/if}
    {#each resultsFields as field (field.key)}{@render renderField(
        field,
      )}{/each}
    {#if ratingInEditor && presentation !== "chat"}<div
        class="wide rating-action"
      >
        <button
          type="button"
          aria-expanded={showRating}
          aria-controls="draft-rating"
          onclick={() => {
            if (showRating) {
              ratingExpanded = false;
              onchange({ rating: null });
            } else {
              ratingExpanded = true;
              void tick().then(() =>
                document.getElementById("draft-rating")?.focus(),
              );
            }
          }}>{showRating ? "Remove rating" : "Add rating"}</button
        >
      </div>{/if}
  {/if}
</fieldset>

<style>
  .chat {
    gap: 1rem;
  }
  .chat label {
    font-size: 0.8rem;
    color: var(--text-muted);
  }
  .chat input:not([type="checkbox"]),
  .chat textarea,
  .chat select {
    padding: 0.65rem 0.75rem;
    font-size: 0.875rem;
    border: 1px solid var(--input-border);
    border-radius: 0.5rem;
    background: var(--surface-bg);
  }
  .mode-switch {
    display: flex;
    gap: 0.5rem;
  }
  .mode-switch button {
    padding: 0.45rem 0.85rem;
    font-size: 0.8rem;
  }
  .mode-switch button[aria-pressed="true"] {
    background: var(--surface-bg);
    border-color: var(--text-muted);
    color: var(--text-primary);
  }
  .preparation-summary p {
    margin: 0 0 0.5rem;
    font-size: 0.85rem;
    line-height: 1.6;
  }
  .preparation-summary button,
  .rating-action button {
    color: var(--text-muted);
    font-size: 0.8rem;
  }
  .preparation-summary button:hover,
  .rating-action button:hover {
    color: var(--text-primary);
  }
  .more-details {
    padding-block: 0.5rem;
  }
  input::placeholder,
  textarea::placeholder {
    color: var(--text-muted);
  }
  button:focus-visible,
  summary:focus-visible {
    outline: 2px solid var(--brand-amber-400);
    outline-offset: 3px;
  }
  .recipe-details p,
  .recipe-details ol,
  .recipe-notice {
    margin: 0.65rem 0 0;
    font-size: 0.875rem;
  }
  .recipe-notes {
    white-space: pre-wrap;
  }
  .preparation-grid,
  .detail-fields {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
    padding: 1rem 0;
  }
  summary {
    cursor: pointer;
  }
  .draft-fields {
    border: 0;
    padding: 0;
    margin: 0;
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1.25rem;
  }
  label {
    display: block;
    margin-bottom: 0.35rem;
    font-size: 0.875rem;
  }
  input:not([type="checkbox"]),
  textarea,
  select {
    width: 100%;
  }
  .wide {
    grid-column: 1/-1;
  }
  .checkbox {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }
  .pour {
    display: flex;
    gap: 0.5rem;
    align-items: end;
    margin-bottom: 0.75rem;
  }
  .pour label {
    flex: 1;
    min-width: 0;
  }
  @media (max-width: 600px) {
    .draft-fields,
    .preparation-grid,
    .detail-fields {
      grid-template-columns: 1fr;
    }
    .pour {
      flex-wrap: wrap;
    }
  }
</style>
