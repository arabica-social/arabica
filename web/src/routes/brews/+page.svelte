<script lang="ts">
  import Icon from "$lib/components/Icon.svelte";
  import LedgerHeader from "$lib/components/LedgerHeader.svelte";
  import { formatTempForUnit, formatTime } from "$lib/utils/format";
  import { pushToast } from "$lib/stores/toasts";
  import { session, openLoginModal } from "$lib/stores/session";
  import { get } from "svelte/store";
  import type { PageData } from "./$types";
  import type { BrewListResponse } from "$lib/types/manage";
  import type { Brew } from "$lib/types/entity_view";

  let { data }: { data: PageData } = $props();

  let brewsData = $state<BrewListResponse | null>(null);
  let loadingMore = $state(false);
  let deletingRKey = $state<string | null>(null);

  // Sync load function data into local state so load-more can extend it.
  $effect(() => {
    brewsData = data.brews;
  });

  let error = $derived(data.error);
  let temperatureUnit = $derived(get(session).temperatureUnit ?? "recorded");

  // The brew list API returns the viewer's own brews but not their DID.
  // Resolve the owner DID from the session store to build view links
  // (/brews/{did}/{rkey}), matching the profile and my-coffee pages.
  let ownerDID = $derived(get(session).did ?? "");

  type MonthGroup = {
    label: string;
    brews: Brew[];
  };

  // Group brews into chronological month buckets. The API returns newest
  // first; we preserve that order within each group.
  function groupBrewsByMonth(list: Brew[]): MonthGroup[] {
    const map = new Map<string, Brew[]>();
    for (const brew of list) {
      const label = new Date(brew.created_at).toLocaleDateString(undefined, {
        month: "long",
        year: "numeric",
      });
      const existing = map.get(label);
      if (existing) {
        existing.push(brew);
      } else {
        map.set(label, [brew]);
      }
    }
    return Array.from(map.entries()).map(([label, brews]) => ({
      label,
      brews,
    }));
  }

  let groupedBrews = $derived<MonthGroup[]>(
    groupBrewsByMonth(brewsData?.brews ?? []),
  );

  function brewDate(brew: Brew): string {
    return new Date(brew.created_at).toLocaleDateString(undefined, {
      weekday: "short",
      month: "short",
      day: "numeric",
    });
  }

  function brewerLabel(brew: Brew): string {
    return brew.brewer_obj?.name || brew.method || "";
  }

  function grinderLabel(brew: Brew): string {
    return brew.grinder_obj?.name || "";
  }

  function beanName(brew: Brew): string {
    return brew.bean?.name || brew.bean?.origin || "Coffee Brew";
  }

  function roasterName(brew: Brew): string {
    return brew.bean?.roaster?.name || "";
  }

  async function loadMore() {
    if (!brewsData?.has_more) return;
    loadingMore = true;
    try {
      const res = await fetch(
        `/api/brews?offset=${brewsData.next_offset}&limit=25`,
        {
          headers: { Accept: "application/json" },
        },
      );
      if (!res.ok) {
        pushToast("Failed to load more");
        return;
      }
      const next = (await res.json()) as BrewListResponse;
      brewsData = {
        brews: [...(brewsData.brews ?? []), ...next.brews],
        has_more: next.has_more,
        next_offset: next.next_offset,
      };
    } catch {
      pushToast("Failed to load more");
    } finally {
      loadingMore = false;
    }
  }

  async function deleteBrew(brew: Brew) {
    if (!window.confirm("Are you sure you want to delete this brew?")) return;
    deletingRKey = brew.rkey;
    try {
      const res = await fetch(`/api/brews/${brew.rkey}`, {
        method: "DELETE",
        credentials: "same-origin",
      });
      if (!res.ok) {
        pushToast("Failed to delete brew");
        return;
      }
      if (brewsData) {
        brewsData = {
          ...brewsData,
          brews: brewsData.brews.filter((b) => b.rkey !== brew.rkey),
        };
      }
      pushToast("Brew deleted");
    } catch {
      pushToast("Failed to delete brew");
    } finally {
      deletingRKey = null;
    }
  }
</script>

<svelte:head>
  <title>Brew Logbook - Arabica</title>
  <meta
    name="description"
    content="Your chronological coffee brew journal, every cup you've logged."
  />
</svelte:head>

<div class="page-container-xl brew-logbook">
  <LedgerHeader
    title="Brew Logbook"
    eyebrow="Coffee Ledger"
    description="A chronological journal of every cup you've brewed. Review, revisit, and refine your routine."
  >
    {#snippet actions()}
      <a href="/brews/new" class="btn-primary shadow-lg hover:shadow-xl"
        >+ Log Brew</a
      >
    {/snippet}
  </LedgerHeader>

  {#if error}
    <div class="card card-inner text-center py-8">
      <p class="text-secondary mb-4">{error}</p>
      {#if error === "Authentication required"}
        <button type="button" onclick={openLoginModal} class="btn-primary"
          >Log In</button
        >
      {/if}
    </div>
  {:else if !brewsData || brewsData.brews.length === 0}
    <div class="card card-inner text-center py-8">
      <p class="text-secondary text-lg mb-2">Your brew journal is empty.</p>
      <p class="text-sm text-muted mb-4">
        Log your first cup and start building your coffee story.
      </p>
      <a href="/brews/new" class="btn-primary px-6 py-3">Log Your First Brew</a>
    </div>
  {:else}
    <div
      class="ledger-month-list"
      role="feed"
      aria-label="Brew logbook entries"
    >
      {#each groupedBrews as group (group.label)}
        <section class="ledger-month">
          <h3 class="ledger-month-heading">{group.label}</h3>
          <ul class="ledger-rows">
            {#each group.brews as brew (brew.rkey)}
              <li class="ledger-row">
                <div class="ledger-cell ledger-cell--identity">
                  <div class="ledger-date">{brewDate(brew)}</div>
                  <a
                    href={ownerDID
                      ? `/brews/${ownerDID}/${brew.rkey}`
                      : "/brews"}
                    class="ledger-bean"
                  >
                    {beanName(brew)}
                  </a>
                  {#if roasterName(brew)}
                    <div class="ledger-roaster">
                      <Icon name="store" class="w-3 h-3" />
                      {roasterName(brew)}
                    </div>
                  {/if}
                </div>

                <div class="ledger-cell ledger-cell--specs">
                  {#if brewerLabel(brew)}
                    <span class="ledger-spec">
                      <Icon name="coffee" class="w-3.5 h-3.5" />
                      {brewerLabel(brew)}
                    </span>
                  {/if}
                  {#if brew.coffee_amount > 0}
                    <span class="ledger-spec">
                      <Icon name="scale" class="w-3.5 h-3.5" />
                      {brew.coffee_amount}g
                    </span>
                  {/if}
                  {#if brew.water_amount > 0}
                    <span class="ledger-spec">
                      <Icon name="droplet" class="w-3.5 h-3.5" />
                      {brew.water_amount}g
                    </span>
                  {/if}
                  {#if brew.temperature > 0}
                    <span class="ledger-spec">
                      <Icon name="thermometer" class="w-3.5 h-3.5" />
                      {formatTempForUnit(brew.temperature, temperatureUnit)}
                    </span>
                  {/if}
                  {#if brew.time_seconds > 0}
                    <span class="ledger-spec">
                      <Icon name="clock" class="w-3.5 h-3.5" />
                      {formatTime(brew.time_seconds)}
                    </span>
                  {/if}
                  {#if brew.grind_size}
                    <span class="ledger-spec">
                      <Icon name="disc" class="w-3.5 h-3.5" />
                      {brew.grind_size}{#if grinderLabel(brew)}
                        · {grinderLabel(brew)}{/if}
                    </span>
                  {:else if grinderLabel(brew)}
                    <span class="ledger-spec">
                      <Icon name="gear" class="w-3.5 h-3.5" />
                      {grinderLabel(brew)}
                    </span>
                  {/if}
                </div>

                <div class="ledger-cell ledger-cell--notes">
                  {#if brew.tasting_notes}
                    <p class="ledger-tasting-notes line-clamp-2">
                      {brew.tasting_notes}
                    </p>
                  {/if}
                  {#if brew.rating > 0}
                    <span class="badge-rating ledger-rating">
                      <Icon name="star" class="w-3 h-3 text-amber-500" />
                      {brew.rating}/10
                    </span>
                  {/if}
                </div>

                <div class="ledger-cell ledger-cell--actions">
                  <div class="ledger-action-cluster">
                    {#if ownerDID}
                      <a
                        href={`/brews/${ownerDID}/${brew.rkey}`}
                        class="ledger-action"
                      >
                        View
                      </a>
                    {/if}
                    <a href={`/brews/${brew.rkey}/edit`} class="ledger-action">
                      Edit
                    </a>
                    <button
                      type="button"
                      onclick={() => deleteBrew(brew)}
                      disabled={deletingRKey === brew.rkey}
                      class="ledger-action ledger-action--danger"
                    >
                      {deletingRKey === brew.rkey ? "Deleting…" : "Delete"}
                    </button>
                  </div>
                </div>
              </li>
            {/each}
          </ul>
        </section>
      {/each}

      {#if brewsData.has_more}
        <div class="text-center py-4">
          <button
            type="button"
            class="btn-secondary px-6 py-2"
            onclick={loadMore}
            disabled={loadingMore}
          >
            {loadingMore ? "Loading..." : "Load More"}
          </button>
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .brew-logbook {
    padding-bottom: 2rem;
  }

  .ledger-month-list {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .ledger-month {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .ledger-month-heading {
    margin: 0;
    font-family: var(--font-mono);
    font-size: 0.75rem;
    font-weight: 700;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--text-muted);
    padding-bottom: 0.35rem;
    border-bottom: 1px solid var(--surface-border);
  }

  .ledger-rows {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
  }

  .ledger-row {
    display: grid;
    grid-template-columns: minmax(0, 1.25fr) minmax(0, 1.5fr) minmax(
        0,
        1fr
      ) auto;
    gap: 1rem;
    align-items: start;
    padding: 0.85rem 0;
    border-bottom: 1px solid var(--surface-border);
  }

  .ledger-row:last-child {
    border-bottom: none;
  }

  .ledger-cell {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    min-width: 0;
  }

  .ledger-cell--identity {
    justify-content: center;
  }

  .ledger-cell--specs {
    flex-direction: row;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem 1rem;
    font-variant-numeric: tabular-nums;
  }

  .ledger-cell--notes {
    gap: 0.5rem;
    align-items: flex-start;
  }

  .ledger-cell--actions {
    align-items: flex-end;
    justify-content: center;
  }

  .ledger-date {
    font-size: 0.75rem;
    color: var(--text-faint);
  }

  .ledger-bean {
    font-weight: 600;
    color: var(--text-primary);
    overflow-wrap: anywhere;
  }

  .ledger-bean:hover {
    text-decoration: underline;
  }

  .ledger-roaster {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    font-size: 0.8rem;
    color: var(--text-muted);
  }

  .ledger-spec {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    font-size: 0.8rem;
    color: var(--text-secondary);
    white-space: nowrap;
  }

  .ledger-tasting-notes {
    margin: 0;
    font-size: 0.8rem;
    line-height: 1.45;
    color: var(--text-secondary);
    overflow-wrap: anywhere;
  }

  .ledger-rating {
    align-self: flex-start;
  }

  .ledger-action-cluster {
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }

  .ledger-action {
    display: inline-flex;
    align-items: center;
    padding: 0.35rem 0.6rem;
    border-radius: 0.375rem;
    font-size: 0.8rem;
    font-weight: 500;
    color: var(--text-muted);
    background: transparent;
    border: none;
    cursor: pointer;
    text-decoration: none;
    transition:
      color 150ms ease,
      background-color 150ms ease;
  }

  .ledger-action:hover {
    color: var(--text-secondary);
    background: var(--surface-bg);
  }

  .ledger-action--danger:hover {
    color: var(--brand-red-700);
    background: var(--brand-red-50);
  }

  .ledger-action:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* Tablet and narrower: collapse to a stacked card-like row. */
  @media (max-width: 1024px) {
    .ledger-row {
      grid-template-columns: 1fr auto;
      gap: 0.75rem 1rem;
    }

    .ledger-cell--identity {
      grid-column: 1 / -1;
      flex-direction: row;
      align-items: baseline;
      gap: 0.75rem;
      flex-wrap: wrap;
    }

    .ledger-cell--specs {
      grid-column: 1 / 2;
    }

    .ledger-cell--notes {
      grid-column: 1 / -1;
      order: 1;
    }

    .ledger-cell--actions {
      grid-column: 2 / -1;
      justify-content: flex-start;
    }
  }

  /* Mobile: every cell stacks in a single column. */
  @media (max-width: 640px) {
    .ledger-row {
      grid-template-columns: 1fr;
      gap: 0.5rem;
      padding: 1rem 0;
    }

    .ledger-cell--identity,
    .ledger-cell--specs,
    .ledger-cell--notes,
    .ledger-cell--actions {
      grid-column: auto;
      order: initial;
    }

    .ledger-cell--actions {
      align-items: flex-start;
    }

    .ledger-cell--specs {
      gap: 0.5rem 0.875rem;
    }
  }
</style>
