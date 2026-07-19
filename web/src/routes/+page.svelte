<script lang="ts">
  import FeedCard from "$lib/components/FeedCard.svelte";
  import FeedFilters from "$lib/components/FeedFilters.svelte";
  import FeedbackPrompt from "$lib/components/FeedbackPrompt.svelte";
  import Icon from "$lib/components/Icon.svelte";
  import { app, openLoginModal } from "$lib/stores/session";
  import { pushToast } from "$lib/stores/toasts";
  import { goto } from "$app/navigation";
  import { tick } from "svelte";
  import { applyFeedMasonry, installFeedMasonry } from "$lib/utils/feedMasonry";
  import { getCachedFeedJSON, setCachedFeedJSON } from "$lib/stores/feedCache";
  import ScrollTopButton from "$lib/components/ScrollTopButton.svelte";
  import type { PageData } from "./$types";
  import type { FeedItem, FeedResponse } from "$lib/types/feed";
  import {
    definitionFor,
    entityRouteForCollection,
  } from "$lib/app/definitions";

  let { data }: { data: PageData } = $props();

  let s = $derived(data);

  // svelte-ignore state_referenced_locally
  let typeFilter = $state(data.typeFilter ?? "");
  // svelte-ignore state_referenced_locally
  let sort = $state(data.sort ?? "recent");
  // svelte-ignore state_referenced_locally
  let items = $state<FeedItem[]>(data.feed?.items ?? []);
  // svelte-ignore state_referenced_locally
  let nextCursor = $state(data.feed?.next_cursor ?? "");
  let loading = $state(false);
  let loadingMore = $state(false);

  // Sync load function data into local state.
  // svelte-ignore state_referenced_locally
  $effect(() => {
    typeFilter = data.typeFilter ?? "";
    sort = data.sort ?? "recent";
    items = data.feed?.items ?? [];
    nextCursor = data.feed?.next_cursor ?? "";
  });

  // Feed masonry: re-pack cards into two corkboard columns whenever the
  // item set changes (filter/sort/load-more). `applyFeedMasonry` measures
  // offsetHeight, so it runs after Svelte commits the DOM (tick) and after
  // the browser paints (requestAnimationFrame).
  $effect(() => {
    // Reference items so the effect re-runs when the array reference
    // changes (filter/sort swap it for a new array; load-more appends).
    items;
    const teardown = installFeedMasonry();
    void tick().then(() => {
      requestAnimationFrame(() => {
        applyFeedMasonry();
      });
    });
    return teardown;
  });

  function buildURL(nextType: string, nextSort: string, cursor = ""): string {
    const params = new URLSearchParams();
    if (nextType) params.set("type", nextType);
    if (nextSort && nextSort !== "recent") params.set("sort", nextSort);
    if (cursor) params.set("cursor", cursor);
    const query = params.toString();
    return `/api/feed${query ? `?${query}` : ""}`;
  }

  // Update the URL query string (shallow) when filters change so the
  // state is shareable and survives back-button.
  function updateURL(nextType: string, nextSort: string) {
    const params = new URLSearchParams();
    if (nextType) params.set("type", nextType);
    if (nextSort && nextSort !== "recent") params.set("sort", nextSort);
    const query = params.toString();
    goto(query ? `/?${query}` : "/", {
      replaceState: true,
      keepFocus: true,
      noScroll: true,
    });
  }

  async function loadFeed(nextType: string, nextSort: string) {
    loading = true;
    updateURL(nextType, nextSort);
    const url = buildURL(nextType, nextSort);
    // Instant cached display, then refetch to sync.
    const cached = getCachedFeedJSON<FeedResponse>(url);
    if (cached) {
      items = cached.items ?? [];
      nextCursor = cached.next_cursor ?? "";
    }
    try {
      const res = await fetch(url, {
        headers: { Accept: "application/json" },
      });
      if (!res.ok) throw new Error(`Feed failed: ${res.status}`);
      const data = (await res.json()) as FeedResponse;
      items = data.items ?? [];
      nextCursor = data.next_cursor ?? "";
      setCachedFeedJSON(url, data);
    } catch {
      pushToast("Failed to load feed");
    } finally {
      loading = false;
    }
  }

  async function applyFilter(nextType: string) {
    if (typeFilter === nextType && !loading) return;
    typeFilter = nextType;
    await loadFeed(nextType, sort);
  }

  async function applySort(nextSort: string) {
    if ((sort || "recent") === nextSort && !loading) return;
    sort = nextSort;
    await loadFeed(typeFilter, nextSort);
  }

  async function loadMore() {
    if (!nextCursor || loadingMore) return;
    loadingMore = true;
    const url = buildURL(typeFilter, sort, nextCursor);
    try {
      const res = await fetch(url, {
        headers: { Accept: "application/json" },
      });
      if (!res.ok) throw new Error(`Load more failed: ${res.status}`);
      const data = (await res.json()) as FeedResponse;
      items = [...items, ...(data.items ?? [])];
      nextCursor = data.next_cursor ?? "";
    } catch {
      pushToast("Failed to load more");
    } finally {
      loadingMore = false;
    }
  }

  let isAuthenticated = $derived(data.isAuthenticated);
  let appName = $derived(data.appName ?? "arabica");
  let appDefinition = $derived(definitionFor(appName));
  // Filters only make sense for authenticated viewers — the unauth feed is
  // an unfiltered cached public list, so showing filter pills would imply
  // interactivity that doesn't exist. Mirrors the old templ gate.
  let showFilters = $derived(isAuthenticated);
  let onboarding = $derived(data.onboarding);
  let incompleteRecords = $derived(data.incompleteRecords);
  let popularRecipes = $derived(data.popularRecipes);

  // Brew readiness: if the user hasn't added the required entity types yet,
  // show a Get Started nudge instead of the Log session action.
  let ready = $derived(
    !onboarding ||
      appDefinition.readinessEntityTypes.every((t) => {
        const key =
          `Has${t.charAt(0).toUpperCase()}${t.slice(1)}` as keyof typeof onboarding.readiness;
        return onboarding.readiness[key];
      }),
  );
</script>

<svelte:head>
  <title>{appDefinition.displayName}</title>
  <meta name="description" content={appDefinition.metaDescription} />
</svelte:head>

{#snippet feedSection()}
  <section class="cafe-feed" aria-labelledby="community-activity-title">
    <div class="cafe-feed-heading">
      <div>
        <p class="cafe-label">Around the café</p>
        <h2 id="community-activity-title">Community Activity</h2>
      </div>
      {#if showFilters}
        <FeedFilters
          {typeFilter}
          {sort}
          {loading}
          onType={applyFilter}
          onSort={applySort}
          tabs={data.feed?.tabs ?? []}
        />
      {/if}
    </div>
    <div id="feed-board" class="feed-board cafe-feed-board">
      <div id="feed-items" class="feed-grid" data-feed-masonry>
        {#if data.error && items.length === 0}
          <div class="cafe-feed-state" style="grid-column: 1 / -1;">
            <Icon name="coffee" class="w-8 h-8 text-muted mb-3" />
            <p class="text-emphasis font-medium mb-1">
              The feed is quiet today
            </p>
            <p class="text-sm text-faint">{data.error}</p>
          </div>
        {:else if loading && items.length === 0}
          <div class="space-y-4 animate-pulse" style="grid-column: 1 / -1;">
            <div class="section-box">
              <div class="flex items-center gap-3 mb-3">
                <div class="w-10 h-10 rounded-full cafe-skeleton"></div>
                <div class="flex-1">
                  <div class="h-4 cafe-skeleton rounded-sm w-1/4 mb-2"></div>
                  <div class="h-3 cafe-skeleton rounded-sm w-1/6"></div>
                </div>
              </div>
              <div class="cafe-skeleton-soft p-3">
                <div class="h-4 cafe-skeleton rounded-sm w-3/4 mb-2"></div>
                <div class="h-3 cafe-skeleton rounded-sm w-1/2"></div>
              </div>
            </div>
          </div>
        {:else if items.length === 0}
          <div class="cafe-feed-state" style="grid-column: 1 / -1;">
            <Icon name="coffee" class="w-8 h-8 text-muted mb-3" />
            <p class="text-emphasis font-medium mb-1">
              The feed is quiet today
            </p>
            <p class="text-sm text-faint">
              Follow people or add your first record to get started.
            </p>
          </div>
        {:else}
          {#each items as item (item.subject_uri)}
            <FeedCard {item} {isAuthenticated} />
          {/each}
          {#if nextCursor}
            <div class="text-center pt-2" style="grid-column: 1 / -1;">
              <button
                class="btn-secondary text-sm load-more-btn"
                disabled={loadingMore}
                onclick={loadMore}
              >
                {loadingMore ? "Loading..." : "Load more"}
              </button>
            </div>
          {/if}
        {/if}
      </div>
    </div>
  </section>
{/snippet}

<div class="cafe-counter">
    <aside class="cafe-rail cafe-rail-left" aria-label="Your coffee journal">
      <section class="cafe-rail-section cafe-rail-lead">
        {#if isAuthenticated}
          <p class="cafe-label">Your coffee</p>
          <h2>Ready for the next cup.</h2>
          <p>Your beans, gear, recipes, and brew notes.</p>
          <nav class="cafe-shortcuts" aria-label="Coffee journal shortcuts">
            <a href={appDefinition.libraryPath}
              ><Icon name="bean" /><span>{appDefinition.libraryLabel}</span></a
            >
            <a href="/explore"><Icon name="brewer" /><span>Explore</span></a>
            <a href="/recipes"><Icon name="fileText" /><span>Recipes</span></a>
            <a href="/profile/{data.userDID}"
              ><Icon name="coffee" /><span>Your profile</span></a
            >
          </nav>
        {:else}
          <p class="cafe-label">Arabica</p>
          <h2>Keep the notes you will want next time.</h2>
          <p>
            Save beans, gear, recipes, and the brews that taught you something.
          </p>
          <button
            type="button"
            onclick={openLoginModal}
            class="cafe-text-link cafe-rail-action"
            >Sign in</button
          >
        {/if}
      </section>

      {#if isAuthenticated && incompleteRecords?.records?.length}
        <section class="cafe-rail-section">
          <p class="cafe-label">To finish</p>
          <h2>
            {incompleteRecords.records.length}
            {incompleteRecords.records.length === 1 ? "record" : "records"} to complete.
          </h2>
          <div class="cafe-record-list">
            {#each incompleteRecords.records as rec (rec.RKey)}
              <a
                href={`/${entityRouteForCollection($app, rec.EntityType)}/${rec.RKey}/edit`}
              >
                <span
                  ><strong>{rec.Name}</strong
                  >{#if rec.MissingFields?.length}<small
                      >Missing {rec.MissingFields.join(", ")}</small
                    >{/if}</span
                >
                <span aria-hidden="true">→</span>
              </a>
            {/each}
          </div>
        </section>
      {/if}
    </aside>

    <main class="cafe-main">
      <header class="cafe-journal-head">
        <div>
          <p class="cafe-label">
            {isAuthenticated
              ? "Your home coffee journal"
              : "A coffee journal on the Atmosphere"}
          </p>
          <h1>
            {isAuthenticated
              ? "Good coffee deserves a record."
              : appDefinition.heroHeading}
          </h1>
          <p class="cafe-deck">
            {isAuthenticated
              ? "Keep the details that helped a cup click, then compare notes with people brewing around the Atmosphere."
              : appDefinition.heroDescription}
          </p>
          {#if !isAuthenticated}
            <p class="cafe-ownership">
              <a href="/atproto">Built on AT Protocol</a>. Your journal stays
              yours.
            </p>
          {/if}
        </div>
        <div class="cafe-actions">
          {#if isAuthenticated}
            <a
              href={ready ? "/brews/new" : "/onboarding"}
              class="home-action-primary"
            >
              {ready ? `${appDefinition.sessionAction} →` : "Finish setup →"}
            </a>
            <a href={appDefinition.libraryPath} class="cafe-text-link"
              >Browse your journal</a
            >
          {:else}
            <button
              type="button"
              onclick={openLoginModal}
              class="home-action-primary">Log In</button
            >
            <a href="/join/create" class="cafe-text-link">Create an account</a>
          {/if}
        </div>
        {#if isAuthenticated && !ready}
          <div class="cafe-setup-nudge">
            <span
              ><strong>First brew</strong>{appDefinition.readinessNudge}</span
            >
            <a href="/onboarding">Open setup</a>
          </div>
        {/if}
      </header>
      {@render feedSection()}
    </main>

    <aside class="cafe-rail cafe-rail-right" aria-label="Around the café">
      <FeedbackPrompt />
      {#if isAuthenticated && popularRecipes?.length}
        <section class="cafe-rail-section cafe-rail-lead">
          <p class="cafe-label">Popular recipes</p>
          <h2>Worth a closer look.</h2>
          <div class="cafe-recipe-list">
            {#each popularRecipes.slice(0, 3) as recipe (recipe.rkey)}
              <a
                href={recipe.author_handle || recipe.author_did
                  ? `/recipes/${encodeURIComponent(recipe.author_handle || recipe.author_did || "")}/${encodeURIComponent(recipe.rkey)}`
                  : "/recipes"}
              >
                <strong>{recipe.name}</strong>
                <span
                  >{recipe.brewer_obj?.name ||
                    recipe.brewer_type ||
                    "Coffee recipe"}</span
                >
                {#if recipe.brew_count || recipe.fork_count}
                  <small
                    >{recipe.brew_count || 0} brews {#if recipe.fork_count}
                      · {#if recipe.fork_count == 1}
                        fork
                      {:else}
                        {recipe.fork_count} forks
                      {/if}
                    {/if}</small
                  >
                {/if}
              </a>
            {/each}
          </div>
          <a href="/recipes" class="cafe-text-link">Explore all recipes</a>
        </section>
      {/if}
      {#if !isAuthenticated}
        <section class="cafe-rail-section cafe-rail-lead">
          <p class="cafe-label">Community</p>
          <h2>See what others are making.</h2>
          <p>
            Browse the public feed before you start keeping notes of your own.
          </p>
          <a href="/about" class="cafe-text-link">Learn more</a>
        </section>
      {/if}
      <section class="cafe-rail-section">
        <p class="cafe-label">About Arabica</p>
        <p>
          A federated coffee journal for people who enjoy keeping the useful
          details.
        </p>
        <a href="/about" class="cafe-text-link">Read the story</a>
      </section>
    </aside>
    {#if isAuthenticated}<ScrollTopButton />{/if}
  </div>

<style>
  .cafe-counter {
    width: 100%;
    max-width: 90rem;
    margin-inline: auto;
    padding-inline: clamp(0.5rem, 2.2vw, 2rem);
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }
  .cafe-main {
    order: 1;
    min-width: 0;
  }
  .cafe-rail-left {
    order: 2;
  }
  .cafe-rail-right {
    order: 3;
  }
  .cafe-journal-head {
    position: relative;
    overflow: hidden;
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    align-items: end;
    gap: clamp(1.25rem, 3vw, 2.5rem);
    margin-bottom: 1.75rem;
    padding: 0.5rem clamp(0.25rem, 2vw, 1.25rem) 1.4rem 0;
    border-bottom: 1px solid var(--card-border);
  }
  .cafe-journal-head::before {
    content: "";
    position: absolute;
    right: clamp(-1rem, -1vw, -0.25rem);
    bottom: -7.5rem;
    width: clamp(17rem, 30vw, 25rem);
    height: clamp(17rem, 30vw, 25rem);
    background-color: var(--text-emphasis);
    opacity: 0.045;
    -webkit-mask: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='1.5'%3E%3Cellipse cx='12' cy='12' rx='5.5' ry='9' transform='rotate(25 12 12)'/%3E%3Cpath d='M12 3c-1.5 3 1.5 6 0 9s1.5 6 0 9' transform='rotate(25 12 12)'/%3E%3C/svg%3E")
      center / contain no-repeat;
    mask: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='1.5'%3E%3Cellipse cx='12' cy='12' rx='5.5' ry='9' transform='rotate(25 12 12)'/%3E%3Cpath d='M12 3c-1.5 3 1.5 6 0 9s1.5 6 0 9' transform='rotate(25 12 12)'/%3E%3C/svg%3E")
      center / contain no-repeat;
    pointer-events: none;
  }
  .cafe-journal-head > * {
    position: relative;
    z-index: 1;
  }
  .cafe-journal-head h1 {
    max-width: 13ch;
    margin: 0.35rem 0 0.8rem;
    color: var(--text-primary);
    font-family: var(--font-display);
    font-size: clamp(2.35rem, 5vw, 3.9rem);
    font-weight: 600;
    line-height: 0.96;
    letter-spacing: -0.025em;
    font-variation-settings:
      "opsz" 96,
      "SOFT" 50,
      "WONK" 0;
  }
  .cafe-deck {
    max-width: 60ch;
    margin: 0;
    color: var(--text-secondary);
    font-size: 0.95rem;
    line-height: 1.65;
  }
  .cafe-ownership {
    margin: 0.8rem 0 0;
    color: var(--text-faint);
    font-size: 0.72rem;
  }
  .cafe-ownership a {
    color: var(--text-muted);
    text-decoration: underline;
    text-underline-offset: 3px;
  }
  .cafe-label {
    margin: 0 0 0.45rem;
    color: var(--text-muted);
    font-size: 0.65rem;
    font-weight: 700;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }
  .cafe-actions {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 0.55rem;
  }
  .cafe-text-link {
    display: inline-flex;
    min-height: 2.75rem;
    align-items: center;
    color: var(--text-emphasis);
    font-size: 0.75rem;
    font-weight: 600;
    text-decoration: underline;
    text-decoration-color: color-mix(
      in oklch,
      var(--text-faint) 45%,
      transparent
    );
    text-underline-offset: 4px;
  }
  .cafe-text-link:hover {
    color: var(--text-primary);
    text-decoration-color: currentColor;
  }
  .cafe-rail-action {
    padding: 0;
    border: 0;
    background: transparent;
    cursor: pointer;
  }
  .cafe-setup-nudge {
    grid-column: 1/-1;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 0.8rem 0;
    border-top: 1px dashed var(--card-border);
    color: var(--text-secondary);
    font-size: 0.78rem;
  }
  .cafe-setup-nudge span {
    display: flex;
    gap: 0.65rem;
    align-items: baseline;
  }
  .cafe-setup-nudge strong {
    color: var(--text-muted);
    font-size: 0.65rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .cafe-setup-nudge a {
    min-height: 2.75rem;
    display: inline-flex;
    align-items: center;
    color: var(--text-emphasis);
    font-weight: 600;
  }
  .cafe-feed {
    min-width: 0;
    margin-bottom: 2rem;
  }
  .cafe-feed-heading {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 1rem;
    padding-top: 0.2rem;
  }
  .cafe-feed-heading h2 {
    margin: 0;
    color: var(--text-primary);
    font-family: var(--font-display);
    font-size: 1.45rem;
    font-weight: 600;
    letter-spacing: -0.012em;
  }
  .cafe-feed-board {
    border-radius: 4px;
  }
  .cafe-feed-state {
    display: grid;
    justify-items: center;
    padding: 2rem;
    text-align: center;
    background: var(--surface-bg);
    border: 1px dashed var(--card-border);
    border-radius: 4px;
  }
  .cafe-skeleton {
    background: var(--surface-border);
  }
  .cafe-skeleton-soft {
    background: var(--surface-bg);
    border-radius: 4px;
  }
  .cafe-rail {
    min-width: 0;
  }
  .cafe-rail-section {
    padding: 1rem 0 1.2rem;
    border-top: 1px solid var(--card-border);
  }
  .cafe-rail-lead {
    border-top: 2px solid var(--text-secondary);
  }
  .cafe-rail-section h2 {
    margin: 0 0 0.5rem;
    color: var(--text-primary);
    font-family: var(--font-display);
    font-size: 1.05rem;
    font-weight: 600;
    line-height: 1.25;
    letter-spacing: -0.01em;
  }
  .cafe-rail-section p:not(.cafe-label) {
    margin: 0 0 0.75rem;
    color: var(--text-muted);
    font-size: 0.76rem;
    line-height: 1.55;
  }
  .cafe-shortcuts {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    border-top: 1px dotted var(--card-border);
  }
  .cafe-shortcuts a {
    display: flex;
    min-height: 2.75rem;
    align-items: center;
    gap: 0.5rem;
    color: var(--text-secondary);
    border-bottom: 1px dotted var(--card-border);
    font-size: 0.75rem;
  }
  .cafe-shortcuts a:hover {
    color: var(--text-primary);
  }
  .cafe-record-list,
  .cafe-recipe-list {
    border-top: 1px dotted var(--card-border);
  }
  .cafe-record-list a {
    display: flex;
    min-height: 3.2rem;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
    color: var(--text-secondary);
    border-bottom: 1px dotted var(--card-border);
    font-size: 0.72rem;
  }
  .cafe-record-list strong,
  .cafe-record-list small {
    display: block;
  }
  .cafe-record-list small {
    margin-top: 0.15rem;
    color: var(--text-faint);
    font-weight: 400;
  }
  .cafe-recipe-list a {
    display: block;
    padding: 0.8rem 0;
    color: var(--text-secondary);
    border-bottom: 1px dotted var(--card-border);
  }
  .cafe-recipe-list strong,
  .cafe-recipe-list span,
  .cafe-recipe-list small {
    display: block;
  }
  .cafe-recipe-list strong {
    color: var(--text-primary);
    font-size: 0.78rem;
  }
  .cafe-recipe-list span,
  .cafe-recipe-list small {
    margin-top: 0.2rem;
    color: var(--text-muted);
    font-size: 0.66rem;
  }
  @media (min-width: 1040px) {
    .cafe-counter {
      display: grid;
      grid-template-columns:
        minmax(10.5rem, 0.72fr) minmax(34rem, 2.5fr)
        minmax(12rem, 0.85fr);
      align-items: start;
      gap: clamp(1.25rem, 2.2vw, 2.5rem);
    }
    .cafe-main,
    .cafe-rail-left,
    .cafe-rail-right {
      order: initial;
    }
    .cafe-rail {
      position: sticky;
      top: 4rem;
      padding-top: 1.5rem;
    }
    .cafe-shortcuts {
      grid-template-columns: 1fr;
    }
  }
  @media (max-width: 719px) {
    .cafe-journal-head {
      grid-template-columns: 1fr;
      align-items: start;
      padding-top: 0;
    }
    .cafe-actions {
      align-items: stretch;
    }
    .cafe-actions :global(.home-action-primary) {
      width: 100%;
      justify-content: center;
    }
    .cafe-text-link {
      justify-content: center;
    }
    .cafe-setup-nudge {
      align-items: flex-start;
      flex-direction: column;
    }
    .cafe-feed-heading {
      align-items: stretch;
      flex-direction: column;
    }
    .cafe-shortcuts {
      grid-template-columns: 1fr;
    }
  }
  @media (prefers-reduced-transparency: reduce) {
    .cafe-journal-head::before {
      display: none;
    }
  }
</style>
