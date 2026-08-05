<script lang="ts">
  import Icon from "$lib/components/Icon.svelte";
  import LedgerHeader from "$lib/components/LedgerHeader.svelte";
  import {
    pluralS,
    formatAvgRating,
    formatTemp,
    formatTime,
  } from "$lib/utils/format";
  import { appCache } from "$lib/stores/appCache";
  import { openLoginModal } from "$lib/stores/session";
  import { pushToast } from "$lib/stores/toasts";
  import type { PageData } from "./$types";
  import type { ManageResponseJSON, BrewListResponse } from "$lib/types/manage";
  import type {
    Bean,
    Brewer,
    Grinder,
    Recipe,
    Roaster,
    Brew,
  } from "$lib/types/entity_view";

  let { data }: { data: PageData } = $props();

  type Tab =
    | "brews"
    | "beans"
    | "roasters"
    | "grinders"
    | "brewers"
    | "recipes";

  let activeTab = $state<Tab>("brews");

  const tabs: { id: Tab; label: string }[] = [
    { id: "brews", label: "Brews" },
    { id: "beans", label: "Beans" },
    { id: "roasters", label: "Roasters" },
    { id: "grinders", label: "Grinders" },
    { id: "brewers", label: "Brewers" },
    { id: "recipes", label: "Recipes" },
  ];

  let manage = $state<ManageResponseJSON | null>(null);
  let brewsData = $state<BrewListResponse | null>(null);
  let loadingMore = $state(false);

  // Sync load function data into local state (so refresh can update it).
  $effect(() => {
    manage = data.manage;
    brewsData = data.brews;
  });

  // Restore the last selected tab from localStorage.
  $effect(() => {
    try {
      const saved = localStorage.getItem("manageTab") as Tab | null;
      if (saved && tabs.some((t) => t.id === saved)) {
        activeTab = saved;
      }
    } catch {
      // Ignore storage errors.
    }
  });

  function switchTab(tab: Tab) {
    activeTab = tab;
    try {
      localStorage.setItem("manageTab", tab);
    } catch {
      // Ignore.
    }
  }

  async function refresh() {
    try {
      const res = await fetch("/api/manage/refresh", {
        method: "POST",
        credentials: "same-origin",
      });
      if (res.ok) {
        appCache.invalidateCache();
        // Refetch manage + brews.
        const [m, b] = await Promise.all([
          fetch("/api/manage", { headers: { Accept: "application/json" } }),
          fetch("/api/brews?limit=25", {
            headers: { Accept: "application/json" },
          }),
        ]);
        if (m.ok) manage = (await m.json()) as ManageResponseJSON;
        if (b.ok) brewsData = (await b.json()) as BrewListResponse;
        pushToast("Refreshed");
      }
    } catch {
      pushToast("Refresh failed");
    }
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
      if (!res.ok) return;
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

  function entityURI(did: string, collection: string, rkey: string): string {
    return `at://${did}/${collection}/${rkey}`;
  }

  // Stats helpers — look up counts by AT-URI.
  function beanBrewCount(did: string, rkey: string): number {
    if (!manage) return 0;
    return (
      manage.stats.bean_brew_counts[
        entityURI(did, "social.arabica.alpha.bean", rkey)
      ] ?? 0
    );
  }
  function beanAvgRating(did: string, rkey: string): number {
    if (!manage) return 0;
    return (
      manage.stats.bean_avg_brew_ratings[
        entityURI(did, "social.arabica.alpha.bean", rkey)
      ] ?? 0
    );
  }
  function roasterBeanCount(did: string, rkey: string): number {
    if (!manage) return 0;
    return (
      manage.stats.roaster_bean_counts[
        entityURI(did, "social.arabica.alpha.roaster", rkey)
      ] ?? 0
    );
  }
  function roasterAvgRating(did: string, rkey: string): number {
    if (!manage) return 0;
    return (
      manage.stats.roaster_avg_brew_ratings[
        entityURI(did, "social.arabica.alpha.roaster", rkey)
      ] ?? 0
    );
  }
  function grinderBrewCount(did: string, rkey: string): number {
    if (!manage) return 0;
    return (
      manage.stats.grinder_brew_counts[
        entityURI(did, "social.arabica.alpha.grinder", rkey)
      ] ?? 0
    );
  }
  function brewerBrewCount(did: string, rkey: string): number {
    if (!manage) return 0;
    return (
      manage.stats.brewer_brew_counts[
        entityURI(did, "social.arabica.alpha.brewer", rkey)
      ] ?? 0
    );
  }

  let did = $derived(manage?.did ?? "");
  let error = $derived(data.error);

  // Closed bags remain separate from the active bean list.
  let openBeans = $derived(
    (manage?.beans ?? []).filter((bean) => !bean.closed),
  );
  let closedBeans = $derived(
    (manage?.beans ?? []).filter((bean) => bean.closed),
  );

  function collectionCount(tab: Tab): number {
    if (tab === "brews") return brewsData?.brews.length ?? 0;
    return manage?.[tab]?.length ?? 0;
  }
</script>

<svelte:head>
  <title>My Coffee - Arabica</title>
  <meta
    name="description"
    content="Your coffee records — brews, beans, roasters, grinders, brewers, and recipes."
  />
</svelte:head>

<div class="coffee-library">
  <LedgerHeader title="My Coffee" eyebrow="Personal collection" description="Your brews, beans, roasters, grinders, brewers, and recipes.">
    {#snippet actions()}
      <div class="coffee-library-actions">
        <a href="/add" class="btn-secondary">Add records</a>
        <a href="/brews/new" class="btn-primary">Log brew</a>
      </div>
    {/snippet}
  </LedgerHeader>

  {#if error}
    <div class="card card-inner text-center py-8">
      <p class="text-secondary mb-4">{error}</p>
      <button type="button" onclick={openLoginModal} class="btn-primary"
        >Log in</button
      >
    </div>
  {:else if manage}
    <div class="coffee-library-grid">
      <aside class="collection-index" aria-label="Coffee collections">
        <p class="collection-label">Collections</p>
        <nav>
        {#each tabs as tab (tab.id)}
          <button
            type="button"
            onclick={() => switchTab(tab.id)}
            class:active={activeTab === tab.id}
            aria-current={activeTab === tab.id ? "page" : undefined}
            data-count={collectionCount(tab.id)}
          >
            {tab.label}
          </button>
        {/each}
        </nav>
      </aside>
      <main class="coffee-collection">

    <!-- Brews tab -->
    {#if activeTab === "brews"}
      <div class="space-y-3">
        {#if !brewsData || brewsData.brews.length === 0}
          <div class="card card-inner text-center py-8">
            <p class="text-secondary text-lg mb-2">
              No brews recorded yet.
            </p>
            <p class="text-sm text-muted mb-4">
              Add the cup you're drinking now.
            </p>
            <a href="/brews/new" class="btn-primary px-6 py-3"
              >Log your first brew</a
            >
          </div>
        {:else}
          {#each brewsData.brews as brew (brew.rkey)}
            <a
              href={`/brews/${did}/${brew.rkey}`}
              class="feed-card feed-card-brew block"
            >
              <div class="flex items-center justify-between mb-2">
                <div class="text-sm text-muted">
                  {new Date(brew.created_at).toLocaleDateString(undefined, {
                    month: "short",
                    day: "numeric",
                    year: "numeric",
                  })}
                </div>
                {#if brew.rating > 0}
                  <span class="badge-rating flex items-center gap-1">
                    <Icon name="star" class="w-3 h-3 text-amber-500" />
                    {brew.rating}/10
                  </span>
                {/if}
              </div>
              {#if brew.bean}
                <div class="font-bold text-primary">
                  {brew.bean.name || brew.bean.origin}
                </div>
                {#if brew.bean.roaster?.name}
                  <div class="text-sm text-muted flex items-center gap-1">
                    <Icon name="store" class="w-3 h-3" />
                    {brew.bean.roaster.name}
                  </div>
                {/if}
                <div
                  class="text-xs text-faint mt-1 flex flex-wrap gap-x-2 gap-y-1"
                >
                  {#if brew.bean.origin}<span
                      class="inline-flex items-center gap-1"
                      ><Icon name="mapPin" class="w-3 h-3" />{brew.bean
                        .origin}</span
                    >{/if}
                  {#if brew.bean.roast_level}<span
                      class="inline-flex items-center gap-1"
                      ><Icon name="flame" class="w-3 h-3" />{brew.bean
                        .roast_level}</span
                    >{/if}
                  {#if brew.coffee_amount > 0}<span
                      class="inline-flex items-center gap-1"
                      ><Icon
                        name="scale"
                        class="w-3 h-3"
                      />{brew.coffee_amount}g</span
                    >{/if}
                </div>
              {:else}
                <div class="font-bold text-primary">Brew</div>
              {/if}
              {#if brew.brewer_obj?.name || brew.method}
                <div class="mb-2">
                  <span class="text-meta">Brewer:</span>
                  <span class="text-sm font-semibold text-primary"
                    >{brew.brewer_obj?.name ?? brew.method}</span
                  >
                </div>
              {/if}
              <div
                class="grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-emphasis"
              >
                {#if brew.grinder_obj?.name}
                  <div>
                    <span class="text-label">Grinder:</span>
                    {brew.grinder_obj.name}{#if brew.grind_size}
                      ({brew.grind_size}){/if}
                  </div>
                {:else if brew.grind_size}
                  <div>
                    <span class="text-label">Grind:</span>
                    {brew.grind_size}
                  </div>
                {/if}
                {#if brew.water_amount > 0}<div>
                    <span class="text-label">Water:</span>
                    {brew.water_amount}g
                  </div>{/if}
                {#if brew.temperature > 0}<div>
                    <span class="text-label">Temp:</span>
                    {formatTemp(brew.temperature)}
                  </div>{/if}
                {#if brew.time_seconds > 0}<div>
                    <span class="text-label">Time:</span>
                    {formatTime(brew.time_seconds)}
                  </div>{/if}
              </div>
              {#if brew.tasting_notes}
                <div class="text-sm text-secondary mt-1 line-clamp-2">
                  {brew.tasting_notes}
                </div>
              {/if}
            </a>
          {/each}
          {#if brewsData.has_more}
            <div class="text-center py-4">
              <button
                type="button"
                class="btn-secondary px-6 py-2"
                onclick={loadMore}
                disabled={loadingMore}
              >
                {loadingMore ? "Loading..." : "Load more"}
              </button>
            </div>
          {/if}
        {/if}
      </div>
    {/if}

    <!-- Beans tab -->
    {#if activeTab === "beans"}
      <div class="space-y-6">
        <!-- Open bags -->
        <div>
          <h4 class="section-heading">Open bags</h4>
          {#if openBeans.length === 0}
            <div class="card card-inner text-center py-8">
              <p class="text-secondary">
                No open bags. <a href="/beans/new" class="link-bold">Add the bag you're drinking now</a
                >.
              </p>
            </div>
          {:else}
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {#each openBeans as bean (bean.rkey)}
                <div class="feed-card feed-card-bean">
                  <div class="flex items-center justify-between mb-2">
                    <a
                      href={`/beans/${did}/${bean.rkey}`}
                      class="font-bold text-primary hover:underline"
                      >{bean.name}</a
                    >
                    {#if bean.rating}
                      <span class="badge-rating flex items-center gap-1">
                        <Icon name="star" class="w-3 h-3 text-amber-500" />
                        {bean.rating}/10
                      </span>
                    {/if}
                  </div>
                  {#if bean.roaster?.name}
                    <div class="text-sm text-muted flex items-center gap-1">
                      <Icon name="store" class="w-3 h-3" />
                      {bean.roaster.name}
                    </div>
                  {/if}
                  <div
                    class="text-xs text-faint mt-1 flex flex-wrap gap-x-2 gap-y-1"
                  >
                    {#if bean.origin}<span
                        class="inline-flex items-center gap-1"
                        ><Icon
                          name="mapPin"
                          class="w-3 h-3"
                        />{bean.origin}</span
                      >{/if}
                    {#if bean.roast_level}<span
                        class="inline-flex items-center gap-1"
                        ><Icon
                          name="flame"
                          class="w-3 h-3"
                        />{bean.roast_level}</span
                      >{/if}
                    {#if bean.variety}<span
                        class="inline-flex items-center gap-1"
                        ><Icon
                          name="leaf"
                          class="w-3 h-3"
                        />{bean.variety}</span
                      >{/if}
                    {#if bean.process}<span
                        class="inline-flex items-center gap-1"
                        ><Icon
                          name="sprout"
                          class="w-3 h-3"
                        />{bean.process}</span
                      >{/if}
                  </div>
                  {#if beanBrewCount(did, bean.rkey) > 0 || beanAvgRating(did, bean.rkey) > 0}
                    <div
                      class="flex items-center gap-3 pt-2 mt-2 border-t border-brown-200/60 text-xs text-faint"
                    >
                      {#if beanBrewCount(did, bean.rkey) > 0}
                        <span class="inline-flex items-center gap-1"
                          ><Icon name="coffee" class="w-3 h-3" />{beanBrewCount(
                            did,
                            bean.rkey,
                          )} brew{pluralS(beanBrewCount(did, bean.rkey))}</span
                        >
                      {/if}
                      {#if beanAvgRating(did, bean.rkey) > 0}
                        <span class="inline-flex items-center gap-1"
                          ><Icon
                            name="star"
                            class="w-3 h-3 text-amber-500"
                          />{formatAvgRating(beanAvgRating(did, bean.rkey))} avg</span
                        >
                      {/if}
                    </div>
                  {/if}
                </div>
              {/each}
            </div>
          {/if}
        </div>

        <!-- Closed bags -->
        {#if closedBeans.length > 0}
          <div>
            <h4 class="section-heading">Finished bags</h4>
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {#each closedBeans as bean (bean.rkey)}
                <div class="feed-card feed-card-bean opacity-75">
                  <div class="flex items-center justify-between mb-2">
                    <a
                      href={`/beans/${did}/${bean.rkey}`}
                      class="font-bold text-primary hover:underline"
                      >{bean.name}</a
                    >
                    <div class="flex items-center gap-2">
                      {#if bean.rating}
                        <span class="badge-rating flex items-center gap-1">
                          <Icon name="star" class="w-3 h-3 text-amber-500" />
                          {bean.rating}/10
                        </span>
                      {/if}
                    </div>
                  </div>
                  {#if bean.roaster?.name}
                    <div class="text-sm text-muted flex items-center gap-1">
                      <Icon name="store" class="w-3 h-3" />
                      {bean.roaster.name}
                    </div>
                  {/if}
                  <div
                    class="text-xs text-faint mt-1 flex flex-wrap gap-x-2 gap-y-1"
                  >
                    {#if bean.origin}<span
                        class="inline-flex items-center gap-1"
                        ><Icon
                          name="mapPin"
                          class="w-3 h-3"
                        />{bean.origin}</span
                      >{/if}
                    {#if bean.roast_level}<span
                        class="inline-flex items-center gap-1"
                        ><Icon
                          name="flame"
                          class="w-3 h-3"
                        />{bean.roast_level}</span
                      >{/if}
                    {#if bean.variety}<span
                        class="inline-flex items-center gap-1"
                        ><Icon
                          name="leaf"
                          class="w-3 h-3"
                        />{bean.variety}</span
                      >{/if}
                    {#if bean.process}<span
                        class="inline-flex items-center gap-1"
                        ><Icon
                          name="sprout"
                          class="w-3 h-3"
                        />{bean.process}</span
                      >{/if}
                  </div>
                  {#if beanBrewCount(did, bean.rkey) > 0 || beanAvgRating(did, bean.rkey) > 0}
                    <div
                      class="flex items-center gap-3 pt-2 mt-2 border-t border-brown-200/60 text-xs text-faint"
                    >
                      {#if beanBrewCount(did, bean.rkey) > 0}
                        <span class="inline-flex items-center gap-1"
                          ><Icon name="coffee" class="w-3 h-3" />{beanBrewCount(
                            did,
                            bean.rkey,
                          )} brew{pluralS(beanBrewCount(did, bean.rkey))}</span
                        >
                      {/if}
                      {#if beanAvgRating(did, bean.rkey) > 0}
                        <span class="inline-flex items-center gap-1"
                          ><Icon
                            name="star"
                            class="w-3 h-3 text-amber-500"
                          />{formatAvgRating(beanAvgRating(did, bean.rkey))} avg</span
                        >
                      {/if}
                    </div>
                  {/if}
                </div>
              {/each}
            </div>
          </div>
        {/if}
      </div>
    {/if}

    <!-- Roasters tab -->
    {#if activeTab === "roasters"}
      {#if manage.roasters.length === 0}
        <div class="card card-inner text-center py-8">
          <p class="text-secondary">No roasters yet. <a href="/roasters/new" class="link-bold">Add a roaster</a>.</p>
        </div>
      {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {#each manage.roasters as roaster (roaster.rkey)}
            <div class="feed-card feed-card-roaster">
              <a
                href={`/roasters/${did}/${roaster.rkey}`}
                class="font-bold text-primary hover:underline">{roaster.name}</a
              >
              {#if roaster.location}
                <div
                  class="text-xs text-muted mt-1 flex flex-wrap gap-x-2 gap-y-1"
                >
                  <span class="inline-flex items-center gap-1"
                    ><Icon
                      name="mapPin"
                      class="w-3 h-3"
                    />{roaster.location}</span
                  >
                </div>
              {/if}
              {#if roasterBeanCount(did, roaster.rkey) > 0 || roasterAvgRating(did, roaster.rkey) > 0}
                <div
                  class="flex items-center gap-3 pt-2 mt-2 border-t border-brown-200/60 text-xs text-faint"
                >
                  {#if roasterBeanCount(did, roaster.rkey) > 0}
                    <span class="inline-flex items-center gap-1"
                      ><Icon name="leaf" class="w-3 h-3" />{roasterBeanCount(
                        did,
                        roaster.rkey,
                      )} bean{pluralS(
                        roasterBeanCount(did, roaster.rkey),
                      )}</span
                    >
                  {/if}
                  {#if roasterAvgRating(did, roaster.rkey) > 0}
                    <span class="inline-flex items-center gap-1"
                      ><Icon
                        name="star"
                        class="w-3 h-3 text-amber-500"
                      />{formatAvgRating(roasterAvgRating(did, roaster.rkey))} avg</span
                    >
                  {/if}
                </div>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    {/if}

    <!-- Grinders tab -->
    {#if activeTab === "grinders"}
      {#if manage.grinders.length === 0}
        <div class="card card-inner text-center py-8">
          <p class="text-secondary">No grinders yet. <a href="/grinders/new" class="link-bold">Add a grinder</a>.</p>
        </div>
      {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {#each manage.grinders as grinder (grinder.rkey)}
            <div class="feed-card feed-card-grinder">
              <a
                href={`/grinders/${did}/${grinder.rkey}`}
                class="font-bold text-primary hover:underline">{grinder.name}</a
              >
              <div
                class="text-xs text-muted mt-1 flex flex-wrap gap-x-2 gap-y-1"
              >
                {#if grinder.grinder_type}<span
                    class="inline-flex items-center gap-1"
                    ><Icon
                      name="tag"
                      class="w-3 h-3"
                    />{grinder.grinder_type}</span
                  >{/if}
                {#if grinder.burr_type}<span
                    class="inline-flex items-center gap-1"
                    ><Icon
                      name="disc"
                      class="w-3 h-3"
                    />{grinder.burr_type}</span
                  >{/if}
              </div>
              {#if grinderBrewCount(did, grinder.rkey) > 0}
                <div
                  class="text-xs text-faint pt-2 mt-2 border-t border-brown-200/60"
                >
                  <span class="inline-flex items-center gap-1"
                    ><Icon name="coffee" class="w-3 h-3" />{grinderBrewCount(
                      did,
                      grinder.rkey,
                    )} brew{pluralS(grinderBrewCount(did, grinder.rkey))}</span
                  >
                </div>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    {/if}

    <!-- Brewers tab -->
    {#if activeTab === "brewers"}
      {#if manage.brewers.length === 0}
        <div class="card card-inner text-center py-8">
          <p class="text-secondary">No brewers yet. <a href="/brewers/new" class="link-bold">Add a brewer</a>.</p>
        </div>
      {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {#each manage.brewers as brewer (brewer.rkey)}
            <div class="feed-card feed-card-brewer">
              <a
                href={`/brewers/${did}/${brewer.rkey}`}
                class="font-bold text-primary hover:underline">{brewer.name}</a
              >
              {#if brewer.brewer_type}
                <div
                  class="text-xs text-muted mt-1 flex flex-wrap gap-x-2 gap-y-1"
                >
                  <span class="inline-flex items-center gap-1"
                    ><Icon
                      name="brewer"
                      class="w-3 h-3"
                    />{brewer.brewer_type}</span
                  >
                </div>
              {/if}
              {#if brewerBrewCount(did, brewer.rkey) > 0}
                <div
                  class="text-xs text-faint pt-2 mt-2 border-t border-brown-200/60"
                >
                  <span class="inline-flex items-center gap-1"
                    ><Icon name="coffee" class="w-3 h-3" />{brewerBrewCount(
                      did,
                      brewer.rkey,
                    )} brew{pluralS(brewerBrewCount(did, brewer.rkey))}</span
                  >
                </div>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    {/if}

    <!-- Recipes tab -->
    {#if activeTab === "recipes"}
      <div class="alert-warning mb-2">
        <span class="text-sm font-bold">⚠️ Recipes are in early alpha.</span>
        <span class="text-sm">
          The recipe format may change, but your brew records will still work.</span
        >
      </div>
      {#if manage.recipes.length === 0}
        <div class="card card-inner text-center py-8">
          <p class="text-secondary">No recipes yet. <a href="/recipes/new" class="link-bold">Add a recipe</a>.</p>
        </div>
      {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {#each manage.recipes as recipe (recipe.rkey)}
            <div class="feed-card feed-card-recipe">
              <a
                href={`/recipes/${did}/${recipe.rkey}`}
                class="font-bold text-primary hover:underline">{recipe.name}</a
              >
              {#if recipe.brewer_obj?.name || recipe.brewer_type}
                <div class="text-sm text-muted">
                  <span class="inline-flex items-center gap-1"
                    ><Icon name="brewer" class="w-3 h-3" />{recipe.brewer_obj
                      ?.name ?? recipe.brewer_type}</span
                  >
                </div>
              {/if}
              {#if recipe.coffee_amount > 0 || recipe.water_amount > 0}
                <div
                  class="text-xs text-faint mt-1 flex flex-wrap gap-x-2 gap-y-1"
                >
                  {#if recipe.coffee_amount > 0}<span
                      class="inline-flex items-center gap-1"
                      ><Icon
                        name="scale"
                        class="w-3 h-3"
                      />{recipe.coffee_amount}g coffee</span
                    >{/if}
                  {#if recipe.water_amount > 0}<span
                      class="inline-flex items-center gap-1"
                      ><Icon
                        name="droplet"
                        class="w-3 h-3"
                      />{recipe.water_amount}g water</span
                    >{/if}
                </div>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    {/if}
      </main>
      <aside class="coffee-library-rail" aria-label="Collection tools">
        <section class="library-rail-section lead">
          <p class="collection-label">On the counter</p>
          <h2>{openBeans.length} open {openBeans.length === 1 ? "bag" : "bags"}</h2>
          <p>{manage.brewers.length} brewers and {manage.grinders.length} grinders are ready for your next entry.</p>
        </section>
        <section class="library-rail-section">
          <p class="collection-label">Changes missing?</p>
          <p>Made changes on another device or app? Refresh to see them here.</p>
          <button type="button" class="library-refresh" onclick={refresh}><Icon name="coffee" />Refresh</button>
        </section>
      </aside>
    </div>
  {/if}
</div>

<style>
  .coffee-library { width: 100%; max-width: 88rem; margin-inline: auto; padding-inline: clamp(.5rem, 2.2vw, 2rem); }
  .coffee-library-actions { display: flex; gap: .6rem; }
  .coffee-library-grid { display: grid; grid-template-columns: 1fr; gap: 1.5rem; padding-top: 1.5rem; }
  .coffee-collection { min-width: 0; order: 1; }
  .collection-index { order: 0; min-width: 0; }
  .coffee-library-rail { order: 2; min-width: 0; }
  .collection-label { margin: 0 0 .45rem; color: var(--text-muted); font-size: .65rem; font-weight: 700; letter-spacing: .14em; text-transform: uppercase; }
  .collection-index nav { display: flex; overflow-x: auto; border-block: 1px solid var(--card-border); }
  .collection-index button { display: flex; min-width: max-content; min-height: 2.75rem; align-items: center; gap: .5rem; padding: .55rem .8rem; color: var(--text-muted); border-bottom: 2px solid transparent; font-size: .78rem; }
  .collection-index button::after { content: attr(data-count); color: var(--text-faint); font-size: .7rem; font-weight: 400; font-variant-numeric: tabular-nums; }
  .collection-index button.active { color: var(--text-primary); border-bottom-color: var(--text-secondary); font-weight: 600; }
  .library-rail-section { padding: 1rem 0 1.2rem; border-top: 1px solid var(--card-border); }
  .library-rail-section.lead { border-top: 2px solid var(--text-secondary); }
  .library-rail-section h2 { margin: 0 0 .45rem; color: var(--text-primary); font-family: var(--font-display); font-size: 1.05rem; font-weight: 600; }
  .library-rail-section p:not(.collection-label) { margin: 0 0 .7rem; color: var(--text-muted); font-size: .75rem; line-height: 1.55; }
  .library-refresh { display: inline-flex; min-height: 2.75rem; align-items: center; gap: .45rem; color: var(--text-emphasis); font-size: .75rem; font-weight: 600; text-decoration: underline; text-underline-offset: 4px; }
  @media (min-width: 1040px) {
    .coffee-library-grid { grid-template-columns: minmax(10rem,.65fr) minmax(34rem,2.5fr) minmax(12rem,.8fr); gap: clamp(1.25rem,2.2vw,2.5rem); align-items: start; }
    .coffee-collection, .collection-index, .coffee-library-rail { order: initial; }
    .collection-index, .coffee-library-rail { position: sticky; top: 5rem; }
    .collection-index nav { display: block; overflow: visible; border-block: 0; border-top: 2px solid var(--text-secondary); }
    .collection-index button { width: 100%; justify-content: space-between; border-bottom: 1px dotted var(--card-border); border-left: 2px solid transparent; padding-inline: .65rem; }
    .collection-index button.active { border-bottom-color: var(--card-border); border-left-color: var(--text-secondary); }
  }
  @media (max-width: 640px) { .coffee-library-actions { width: 100%; } .coffee-library-actions a { flex: 1; text-align: center; } }
</style>
