<script lang="ts">
  import type { Draft, DraftPatch, FieldPath } from "#lib/ai/types.js";
  import type { Library } from "./DraftEditor.svelte";
  import { fields, type EditorField } from "./fields";
  import { preparationLayout } from "#lib/utils/brewFormPolicy.js";

  let {
    draft,
    library = {},
    brewerType = "",
    showRating = true,
  }: {
    draft: Draft;
    library?: Library;
    brewerType?: string;
    showRating?: boolean;
  } = $props();
  let values = $derived(draft.fields as DraftPatch);
  const supplied = (value: unknown) =>
    value !== null && value !== undefined && value !== "";
  const kinds = {
    brew: "Brew",
    bean: "Bean",
    grinder: "Grinder",
    brewer: "Brewer",
  };
  const types: Record<string, string> = {
    pourover: "Pour-over",
    espresso: "Espresso",
    immersion: "Immersion",
    mokapot: "Moka pot",
    coldbrew: "Cold brew",
    cupping: "Cupping",
    other: "Other",
  };
  let bean = $derived(
    library.beans?.find((item) => item.rkey === values.bean_rkey),
  );
  let title = $derived(
    (draft.kind === "brew" ? bean?.name : values.name) ||
      values.name ||
      `${kinds[draft.kind]} draft`,
  );
  let layout = $derived(preparationLayout(brewerType));
  const primaryKeys: FieldPath[] = [
    "coffee_amount",
    "water_amount",
    "time_seconds",
    "temperature",
  ];
  let primary = $derived(
    draft.kind === "brew"
      ? fields.brew.filter(
          (field) =>
            primaryKeys.includes(field.key) && supplied(values[field.key]),
        )
      : [],
  );
  let remaining = $derived(
    fields[draft.kind].filter(
      (field) =>
        field.key !== "name" &&
        (showRating || field.key !== "rating") &&
        !(draft.kind === "brew" && primaryKeys.includes(field.key)) &&
        // The bean is already the title when resolved; unresolved selections still need context.
        !(draft.kind === "brew" && field.key === "bean_rkey" && bean?.name) &&
        supplied(values[field.key]),
    ),
  );
  let ratio = $derived(
    typeof values.coffee_amount === "number" &&
      values.coffee_amount > 0 &&
      typeof values.water_amount === "number" &&
      values.water_amount > 0
      ? Number((values.water_amount / values.coffee_amount).toPrecision(4))
      : undefined,
  );
  let pours = $derived(values.pours ?? []);
  function label(field: EditorField) {
    if (field.key === "temperature") return "Temperature";
    if (field.key === "time_seconds")
      return layout.timeLabel.replace(" (s)", "");
    return field.label;
  }
  function shown(field: EditorField) {
    const value = values[field.key];
    if (field.library) {
      // An external recipe owner cannot be joined by a coincidentally equal local key.
      const item =
        field.key === "recipe_rkey" && values.recipe_owner_did
          ? undefined
          : library[field.library]?.find((item) => item.rkey === value);
      return item?.name || `${field.label} selected (unavailable)`;
    }
    if (field.key === "temperature") return `${value}°`;
    if (field.key === "coffee_amount" || field.key === "water_amount")
      return `${value} g`;
    if (field.key === "time_seconds") return `${value} s`;
    if (field.key === "brewer_type")
      return types[String(value)] ?? String(value);
    return String(value);
  }
</script>

<div class="draft-preview">
  <header>
    <p class="kind">{kinds[draft.kind]}</p>
    <h3>{title}</h3>
  </header>
  {#if primary.length || ratio !== undefined}
    <dl class="metrics">
      {#each primary as field (field.key)}
        <div>
          <dt>{label(field)}</dt>
          <dd>
            {shown(field)}{#if field.key === "temperature"}<span
                class="sr-only"
              >
                (recorded unit; °F or °C not specified)</span
              >{/if}
          </dd>
        </div>
      {/each}
      {#if ratio !== undefined}<div>
          <dt>Coffee : water</dt>
          <dd>1 : {ratio}</dd>
        </div>{/if}
    </dl>
  {/if}
  {#if remaining.length || (draft.kind === "bean" && supplied(values.closed))}
    <dl class="facts">
      {#each remaining as field (field.key)}
        <div class:prose={field.type === "textarea"}>
          <dt>{label(field)}</dt>
          <dd>{shown(field)}</dd>
        </div>
      {/each}
      {#if draft.kind === "bean" && supplied(values.closed)}<div>
          <dt>Bag</dt>
          <dd>{values.closed ? "Finished" : "Open"}</dd>
        </div>{/if}
    </dl>
  {/if}
  {#if pours.length}
    {#if pours.length > 6}
      <details class="pour-details">
        <summary>{pours.length} pours</summary>{@render pourList()}
      </details>
    {:else}
      <section aria-label="Pours">
        <h4>Pours</h4>
        {@render pourList()}
      </section>
    {/if}
  {/if}
</div>

{#snippet pourList()}
  <ol class="pour-list" aria-label="Ordered pours">
    {#each pours as pour}<li>
        <span>{pour.water_amount} g</span><span>at {pour.time_seconds} s</span>
      </li>{/each}
  </ol>
  <p class="relative-note">Times relative to the start of brewing.</p>
{/snippet}

<style>
  .draft-preview {
    color: var(--text-primary);
    min-width: 0;
  }
  header {
    margin-bottom: 1.25rem;
  }
  .kind {
    margin: 0 0 0.35rem;
    color: var(--text-muted);
    font-size: 0.8rem;
  }
  h3 {
    margin: 0;
    font-family: var(--font-display);
    font-size: clamp(1.4rem, 3vw, 1.85rem);
    line-height: 1.2;
    font-weight: 600;
    overflow-wrap: anywhere;
  }
  dl {
    margin: 0;
  }
  dt {
    color: var(--text-muted);
    font-size: 0.75rem;
    line-height: 1.5;
  }
  dd {
    margin: 0.2rem 0 0;
    font-size: 0.9rem;
    line-height: 1.6;
    overflow-wrap: anywhere;
    white-space: pre-wrap;
  }
  .metrics {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(5.5rem, 1fr));
    gap: 1rem 0.75rem;
    margin-bottom: 1.25rem;
  }
  .metrics dd {
    font-size: 1.05rem;
    font-variant-numeric: tabular-nums;
  }
  .facts {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem 1.5rem;
  }
  .prose {
    grid-column: 1 / -1;
  }
  section,
  details {
    margin-top: 1.25rem;
  }
  h4 {
    margin: 0;
    font-size: 0.8rem;
    font-weight: 400;
    color: var(--text-muted);
  }
  summary {
    cursor: pointer;
    font-size: 0.8rem;
    color: var(--text-muted);
  }
  summary:focus-visible {
    outline: 2px solid var(--brand-amber-400);
    outline-offset: 3px;
  }
  .pour-list {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    margin: 0.65rem 0 0;
    padding: 0;
    list-style: none;
    font-size: 0.875rem;
    font-variant-numeric: tabular-nums;
  }
  .pour-list li {
    position: relative;
    flex: 1 1 4.5rem;
    padding: 0.5rem 0 0.5rem 1rem;
    border-top: 1px solid var(--surface-border);
  }
  .pour-list li::before {
    content: counter(list-item);
    position: absolute;
    left: 0;
    color: var(--text-muted);
    font-size: 0.65rem;
  }
  .pour-list li span {
    display: block;
  }
  .pour-list li span + span {
    margin-top: 0.25rem;
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .relative-note {
    font-size: 0.75rem;
    color: var(--text-muted);
    margin: 0.5rem 0 0;
  }
  @media (max-width: 480px) {
    .facts {
      grid-template-columns: 1fr;
    }
  }
</style>
