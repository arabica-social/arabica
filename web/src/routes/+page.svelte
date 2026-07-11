<script lang="ts">
  import FeedCard from "$lib/components/FeedCard.svelte";
  import FeedFilters from "$lib/components/FeedFilters.svelte";
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
  import { definitionFor, entityRouteForCollection } from "$lib/app/definitions";

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
        const key = `Has${t.charAt(0).toUpperCase()}${t.slice(1)}` as keyof typeof onboarding.readiness;
        return onboarding.readiness[key];
      }),
  );
</script>

<svelte:head>
  <title>{appDefinition.displayName}</title>
  <meta
    name="description"
    content={appDefinition.metaDescription}
  />
</svelte:head>

<div class="page-container-lg">
  {#if appName === "oolong"}
    <div class="alert-warning mb-4">
      <strong>Heads up:</strong> oolong is <em>very</em> new and unstable
      (initial release was May 17th). Breaking lexicon changes
      <strong>will</strong> happen frequently.
    </div>
  {/if}

  {#if isAuthenticated}
    <!-- Welcome card (authenticated) -->
    <section class="home-hero">
      <span class="home-hero-eyebrow">Your journal</span>
      <h1 class="home-hero-title">{appDefinition.heroHeading}</h1>
      {#if !ready}
        <div class="home-nudge">
          <div class="home-nudge-copy">
            <span class="home-nudge-label">Get started</span>
            <span class="home-nudge-text">{appDefinition.readinessNudge}</span>
          </div>
          <a href="/onboarding" class="btn-primary text-sm">Get Started →</a>
        </div>
      {/if}
      <div class="home-actions">
        <a
          href="/brews/new"
          class="home-action-primary"
          class:is-disabled={!ready}
          aria-disabled={!ready}
        >
          {appDefinition.sessionAction}
        </a>
        <a href="/explore" class="home-action-secondary">Explore</a>
        <a href={appDefinition.libraryPath} class="home-action-secondary"
          >{appDefinition.libraryLabel}</a
        >
        <a href="/profile/{data.userDID}" class="home-action-secondary"
          >Profile</a
        >
      </div>
    </section>
  {:else}
    <!-- Welcome hero (unauthenticated) -->
    <section class="home-hero">
      <span class="home-hero-eyebrow">Federated coffee journal</span>
      <h1 class="home-hero-title">{appDefinition.heroHeading}</h1>
      <p class="home-hero-deck">{appDefinition.heroDescription}</p>
      <p class="home-hero-foot">
        <a href="/atproto">Built on AT Protocol</a> — you own your data.
      </p>
      <div class="home-actions mt-6 justify-center">
        <button type="button" onclick={openLoginModal} class="home-action-primary">
          Log In
        </button>
        <a href="/join/create" class="home-action-secondary">Create an account</a>
        <a href="/about" class="home-action-secondary">Learn more</a>
      </div>
    </section>
  {/if}

  <!-- Community feed -->
  <div class="card p-2 sm:p-6 mb-8">
    <h3 class="feed-section-title">Community Activity</h3>
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
    <div id="feed-board" class="feed-board">
      <div id="feed-items" class="feed-grid" data-feed-masonry>
        {#if data.error && items.length === 0}
          <div class="rounded-xl p-8 text-center" style="grid-column: 1 / -1;">
            <div class="text-4xl mb-3">✦</div>
            <p class="text-emphasis font-medium mb-1">
              The feed is quiet today
            </p>
            <p class="text-sm text-faint">{data.error}</p>
          </div>
        {:else if loading && items.length === 0}
          <!-- Loading skeleton -->
          <div class="space-y-4 animate-pulse" style="grid-column: 1 / -1;">
            <div class="section-box">
              <div class="flex items-center gap-3 mb-3">
                <div class="w-10 h-10 rounded-full bg-brown-300"></div>
                <div class="flex-1">
                  <div class="h-4 bg-brown-300 rounded-sm w-1/4 mb-2"></div>
                  <div class="h-3 bg-brown-200 rounded-sm w-1/6"></div>
                </div>
              </div>
              <div class="bg-brown-200 rounded-lg p-3">
                <div class="h-4 bg-brown-300 rounded-sm w-3/4 mb-2"></div>
                <div class="h-3 bg-brown-200 rounded-sm w-1/2"></div>
              </div>
            </div>
          </div>
        {:else if items.length === 0}
          <div
            class="rounded-xl p-8 text-center"
            style="grid-column: 1 / -1; background: var(--surface-bg); border: 2px dashed var(--card-border);"
          >
            <div class="text-4xl mb-3">✦</div>
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
  </div>

  {#if isAuthenticated}
    <ScrollTopButton />
    <!-- Incomplete records nudge -->
    {#if incompleteRecords?.records?.length}
      <div class="card p-6 mb-8">
        <h3 class="text-lg font-bold text-primary mb-3">
          Complete your records
        </h3>
        <div class="space-y-2">
          {#each incompleteRecords.records as rec (rec.RKey)}
            <a
              href={`/${entityRouteForCollection($app, rec.EntityType)}/${rec.RKey}/edit`}
              class="block p-3 rounded-lg border border-brown-200 hover:bg-brown-50 transition-colors"
            >
              <div class="flex items-center justify-between">
                <span class="font-medium text-primary">{rec.Name}</span>
                <span class="text-xs text-faint capitalize"
                  >{rec.EntityType}</span
                >
              </div>
              {#if rec.MissingFields?.length}
                <p class="text-sm text-muted mt-1">
                  Missing: {rec.MissingFields.join(", ")}
                </p>
              {/if}
            </a>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Popular recipes -->
    <!-- {#if popularRecipes?.length} -->
    <!-- 	<div class="card p-6 mb-8"> -->
    <!-- 		<h3 class="text-lg font-bold text-primary mb-3">Popular recipes</h3> -->
    <!-- 		<div class="grid grid-cols-1 sm:grid-cols-3 gap-4"> -->
    <!-- 			{#each popularRecipes as recipe (recipe.rkey)} -->
    <!-- 				<a href={`/recipes/${recipe.author_did ?? ""}/${recipe.rkey}`} class="card card-inner p-4 hover:shadow-md transition"> -->
    <!-- 					<div class="font-semibold text-primary mb-1">{recipe.name}</div> -->
    <!-- 					{#if recipe.brewer_obj?.name} -->
    <!-- 						<div class="text-sm text-muted">{recipe.brewer_obj.name}</div> -->
    <!-- 					{/if} -->
    <!-- 				</a> -->
    <!-- 			{/each} -->
    <!-- 		</div> -->
    <!-- 	</div> -->
    <!-- {/if} -->

    <!-- About info card -->
    <div class="card p-6 mb-8">
      <h3 class="text-lg font-bold text-primary mb-2">{appDefinition.aboutHeading}</h3>
      <p class="text-sm text-secondary">
        {appDefinition.aboutBody}
      </p>
      <a
        href="/about"
        class="text-sm text-secondary hover:text-primary hover:underline mt-2 inline-block"
        >Learn more →</a
      >
    </div>
  {/if}
</div>
