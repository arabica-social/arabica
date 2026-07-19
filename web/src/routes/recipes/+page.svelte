<script lang="ts">
  import LedgerHeader from "$lib/components/LedgerHeader.svelte";
  import Icon from "$lib/components/Icon.svelte";
  import StampTag from "$lib/components/StampTag.svelte";
  import { goto } from "$app/navigation";
  import { pushToast } from "$lib/stores/toasts";
  import { get } from "svelte/store";
  import { openLoginModal, session } from "$lib/stores/session";
  import type { PageData } from "./$types";
  import type { Recipe } from "$lib/types/entity_view";

  let { data }: { data: PageData } = $props();

  // Seed local state from the load function; re-sync when data changes
  // (e.g. after goto triggers a reload).
  // svelte-ignore state_referenced_locally
  let recipes = $state<Recipe[]>(data.recipes ?? []);
  let loading = $state(false);

  // Filter form state, initialized from URL.
  let query = $state("");
  let category = $state("");
  let brewerType = $state("");
  let minCoffee = $state("");
  let maxCoffee = $state("");
  let sortBy = $state("popular");
  let selectedRecipe = $state<Recipe | null>(null);
  let searchTimer: ReturnType<typeof setTimeout> | undefined;

  const categories = [
    { label: "All", value: "" },
    { label: "Small (<=12g)", value: "small" },
    { label: "Single cup (12-22g)", value: "single" },
    { label: "Large (22g+)", value: "large" },
    { label: "Batch brew (500g+ water)", value: "batch" },
  ];

  const sorts = [
    { label: "Popular", value: "popular" },
    { label: "Newest", value: "newest" },
    { label: "Most Forked", value: "most_forked" },
  ];

  $effect(() => {
    const u = new URL(window.location.href);
    query = u.searchParams.get("q") ?? "";
    category = u.searchParams.get("category") ?? "";
    brewerType = u.searchParams.get("brewer_type") ?? "";
    minCoffee = u.searchParams.get("min_coffee") ?? "";
    maxCoffee = u.searchParams.get("max_coffee") ?? "";
    sortBy = u.searchParams.get("sort") ?? "popular";
    recipes = data.recipes ?? [];
  });

  let isAuthenticated = $derived(data.isAuthenticated);
  let userDID = $derived(get(session).did);

  function debounceSearch() {
    clearTimeout(searchTimer);
    searchTimer = setTimeout(() => {
      void search();
    }, 300);
  }

  function setCategory(next: string) {
    category = next;
    void search();
  }

  function setSort(next: string) {
    sortBy = next;
    void search();
  }

  function searchParams(): URLSearchParams {
    const params = new URLSearchParams();
    if (query) params.set("q", query);
    if (category) params.set("category", category);
    if (brewerType) params.set("brewer_type", brewerType);
    if (minCoffee) params.set("min_coffee", minCoffee);
    if (maxCoffee) params.set("max_coffee", maxCoffee);
    if (sortBy) params.set("sort", sortBy);
    return params;
  }

  function updateURL() {
    const params = searchParams();
    const q = params.toString();
    goto(q ? `/recipes?${q}` : "/recipes", {
      replaceState: true,
      keepFocus: true,
      noScroll: true,
    });
  }

  async function search() {
    loading = true;
    updateURL();
    try {
      const res = await fetch(`/api/recipes/suggestions?${searchParams()}`, {
        credentials: "same-origin",
        headers: { Accept: "application/json" },
      });
      if (!res.ok) throw new Error("Failed to fetch recipes");
      const json = await res.json();
      recipes = Array.isArray(json) ? (json as Recipe[]) : [];
    } catch (error) {
      console.error("Failed to search recipes:", error);
      recipes = [];
      pushToast("Failed to search recipes");
    } finally {
      loading = false;
    }
  }

  function selectRecipe(recipe: Recipe) {
    selectedRecipe = recipe;
    queueMicrotask(() => {
      document
        .getElementById("recipe-detail-panel")
        ?.scrollIntoView({ behavior: "smooth", block: "start" });
    });
  }

  function handleRecipeCardKeydown(event: KeyboardEvent, recipe: Recipe) {
    if (event.key !== "Enter" && event.key !== " ") return;
    event.preventDefault();
    selectRecipe(recipe);
  }

  function formatRatio(recipe: Recipe | null): string {
    if (recipe && recipe.ratio && recipe.ratio > 0) {
      return `1:${recipe.ratio.toFixed(1)}`;
    }
    return "-";
  }

  function amount(value: number | undefined): string {
    return value && value > 0 ? `${value.toFixed(1)}g` : "-";
  }

  function getBrewerDisplay(recipe: Recipe | null): string {
    if (recipe?.brewer_obj?.name) {
      return recipe.brewer_type
        ? `${recipe.brewer_obj.name} · ${recipe.brewer_type}`
        : recipe.brewer_obj.name;
    }
    return recipe?.brewer_type || "-";
  }

  function authorName(recipe: Recipe): string {
    return (
      recipe.author_display || recipe.author_handle || recipe.author_did || ""
    );
  }

  function authorInitial(recipe: Recipe): string {
    return (
      (recipe.author_display ||
        recipe.author_handle ||
        "?")[0]?.toUpperCase() || "?"
    );
  }

  function ownerForURL(recipe: Recipe): string {
    return encodeURIComponent(recipe.author_handle || recipe.author_did || "");
  }

  function isOwner(recipe: Recipe | null): boolean {
    return !!(recipe && recipe.author_did && recipe.author_did === userDID);
  }

  function sourceRecipeURL(recipe: Recipe | null): string {
    if (!recipe?.source_ref) return "#";
    const parts = recipe.source_ref.replace("at://", "").split("/");
    if (parts.length < 3) return "#";
    const owner =
      recipe.source_author_handle || recipe.source_author_display || parts[0];
    return `/recipes/${encodeURIComponent(owner)}/${parts[2]}`;
  }

  async function forkRecipe() {
    if (!selectedRecipe) return;
    const owner = selectedRecipe.author_handle || selectedRecipe.author_did;
    try {
      const res = await fetch(
        `/api/recipes/fork/${selectedRecipe.rkey}?owner=${encodeURIComponent(owner ?? "")}`,
        { method: "POST", credentials: "same-origin" },
      );
      if (!res.ok) {
        if (res.status === 401) {
          pushToast("Your session has expired. Please log in again.");
          return;
        }
        const text = await res.text();
        throw new Error(text || "Failed to copy recipe");
      }
      pushToast("Recipe copied to your library!");
      selectedRecipe = null;
    } catch (error) {
      console.error("Failed to fork recipe:", error);
      pushToast(
        `Failed to copy recipe: ${error instanceof Error ? error.message : String(error)}`,
      );
    }
  }
</script>

<svelte:head>
  <title>Explore Recipes - Arabica</title>
  <meta
    name="description"
    content="Browse and fork brewing recipes from the Arabica community."
  />
</svelte:head>

{#if data.error === "Authentication required"}
  <div class="page-container-sm">
    <div class="card card-inner text-center py-8">
      <p class="text-secondary text-lg mb-2">Authentication required</p>
      <button type="button" onclick={openLoginModal} class="btn-primary"
        >Log In</button
      >
    </div>
  </div>
{:else}
  <div class="page-container-xl">
    <LedgerHeader
      title="Explore Recipes"
      eyebrow="Community catalog"
      description="Browse, compare, and fork brewing recipes shared by the Arabica community."
    >
      {#snippet actions()}
        <a href="/recipes/new" class="btn-primary shadow-lg hover:shadow-xl"
          >+ New Recipe</a
        >
      {/snippet}
    </LedgerHeader>

    <details class="recipe-alpha-notice" aria-labelledby="recipe-alpha-summary">
      <summary id="recipe-alpha-summary" class="recipe-alpha-notice__summary">
        <Icon name="fileText" />
        <span
          ><strong>Alpha format</strong><small
            >Recipes are still evolving. Existing brews remain intact.</small
          ></span
        >
      </summary>
      <div class="recipe-alpha-notice__body text-sm">
        <p>
          The recipe format may change significantly as we figure out what works
          best. If that happens, your brews won’t break because brew fields are
          copied from the recipe at creation time, so they stand on their own.
          Only the recipe record itself would need to be recreated.
        </p>
        <p>
          Feedback is very welcome! Arabica has had a pourover bias so far, so
          input from espresso folks is especially appreciated. Reach out on
          Bluesky or open an issue on Tangled or GitHub.
        </p>
      </div>
    </details>

    <div class="recipe-catalog-workspace">
      <aside
        class="recipe-catalog-index"
        aria-labelledby="recipe-catalog-index-title"
      >
        <div class="recipe-catalog-index__heading">
          <p>Catalog index</p>
          <h2 id="recipe-catalog-index-title">Find a recipe</h2>
        </div>
        <div class="recipe-filter-group">
          <label for="recipe-search">Search</label>
          <input
            id="recipe-search"
            type="text"
            bind:value={query}
            oninput={debounceSearch}
            placeholder="Recipe name"
            aria-label="Search recipes"
            class="form-input"
          />
        </div>

        <div class="recipe-filter-group">
          <span class="recipe-filter-label">Brew size</span>
          <div
            class="recipe-stamp-list"
            role="group"
            aria-label="Recipe size filters"
          >
            {#each categories as option}
              <StampTag
                label={option.label}
                active={category === option.value}
                tone="recipe"
                onclick={() => setCategory(option.value)}
              />
            {/each}
          </div>
        </div>

        <div class="recipe-filter-group">
          <label for="recipe-explore-brewer-type">Brewer type</label>
          <input
            id="recipe-explore-brewer-type"
            type="text"
            bind:value={brewerType}
            oninput={debounceSearch}
            placeholder="Pour-over, French press..."
            class="form-input"
          />
        </div>
        <div class="recipe-amount-range">
          <div class="recipe-filter-group">
            <label for="recipe-explore-min-coffee">Min coffee (g)</label>
            <input
              id="recipe-explore-min-coffee"
              type="number"
              bind:value={minCoffee}
              oninput={debounceSearch}
              placeholder="0"
              step="1"
              class="form-input"
            />
          </div>
          <div class="recipe-filter-group">
            <label for="recipe-explore-max-coffee">Max coffee (g)</label>
            <input
              id="recipe-explore-max-coffee"
              type="number"
              bind:value={maxCoffee}
              oninput={debounceSearch}
              placeholder="Any"
              step="1"
              class="form-input"
            />
          </div>
        </div>
      </aside>

      <main class="recipe-catalog-results">
        <div class="catalog-results-bar">
          {#if !loading}
            <p
              class="catalog-results-bar__count"
              aria-live="polite"
              aria-atomic="true"
            >
              {#if recipes.length > 0}
                {recipes.length} recipe{recipes.length === 1 ? "" : "s"} found
              {:else}
                No recipes found
              {/if}
            </p>
          {:else}
            <span class="catalog-results-bar__count" aria-hidden="true"
              >Loading recipes…</span
            >
          {/if}
          <div
            class="flex items-center gap-1 text-sm"
            role="group"
            aria-label="Sort recipes"
          >
            <span class="stamp-sort-label">Sort</span>
            {#each sorts as option}
              <StampTag
                label={option.label}
                active={sortBy === option.value}
                onclick={() => setSort(option.value)}
              />
            {/each}
          </div>
        </div>

        {#if selectedRecipe}
          <div class="recipe-detail-panel" id="recipe-detail-panel">
            <div class="recipe-detail-sheet">
              <div class="flex justify-between items-start mb-4">
                <div class="min-w-0 flex-1">
                  <h3 class="text-xl font-bold text-primary wrap-break-word">
                    {selectedRecipe.name}
                  </h3>
                  <a
                    href={`/profile/${selectedRecipe.author_did}`}
                    class="flex items-center gap-2 mt-1 group/author"
                  >
                    {#if selectedRecipe.author_avatar}
                      <img
                        src={selectedRecipe.author_avatar}
                        class="w-6 h-6 rounded-full object-cover"
                        alt={selectedRecipe.author_display || ""}
                        loading="lazy"
                        width="24"
                        height="24"
                      />
                    {:else}
                      <div
                        class="w-6 h-6 rounded-full bg-brown-200 flex items-center justify-center text-muted text-xs font-bold"
                      >
                        {authorInitial(selectedRecipe)}
                      </div>
                    {/if}
                    <div>
                      {#if selectedRecipe.author_display}
                        <span
                          class="block text-sm font-medium text-emphasis group-hover/author:text-primary group-hover/author:underline transition-colors"
                        >
                          {selectedRecipe.author_display}
                        </span>
                      {/if}
                      <span
                        class="block text-xs text-muted group-hover/author:text-secondary transition-colors"
                      >
                        {selectedRecipe.author_handle || ""}
                      </span>
                    </div>
                  </a>
                </div>
                <button
                  type="button"
                  onclick={() => (selectedRecipe = null)}
                  class="text-faint hover:text-emphasis text-lg font-bold"
                  aria-label="Close recipe details"
                >
                  &times;
                </button>
              </div>

              {#if getBrewerDisplay(selectedRecipe) !== "-"}
                <div class="mb-4">
                  <span class="text-xs text-muted uppercase">Brewer</span>
                  <p class="font-medium text-primary">
                    {getBrewerDisplay(selectedRecipe)}
                  </p>
                </div>
              {/if}

              <div class="grid grid-cols-3 gap-4 mb-4">
                <div>
                  <span class="text-xs text-muted uppercase">Coffee</span>
                  <p class="font-medium text-primary">
                    {amount(selectedRecipe.coffee_amount)}
                  </p>
                </div>
                <div>
                  <span class="text-xs text-muted uppercase">Water</span>
                  <p class="font-medium text-primary">
                    {amount(selectedRecipe.water_amount)}
                  </p>
                </div>
                <div>
                  <span class="text-xs text-muted uppercase">Ratio</span>
                  <p class="font-medium text-primary">
                    {formatRatio(selectedRecipe)}
                  </p>
                </div>
              </div>

              {#if selectedRecipe.pours && selectedRecipe.pours.length > 0}
                <div class="mb-4">
                  <span class="text-xs text-muted uppercase">Pours</span>
                  <div class="flex flex-wrap gap-2 mt-1">
                    {#each selectedRecipe.pours as pour, index}
                      <span
                        class="inline-flex items-center gap-1.5 text-xs bg-brown-50 px-2.5 py-1 rounded-full border border-brown-200"
                      >
                        <span class="font-medium text-secondary"
                          >{index + 1}</span
                        >
                        <span class="text-emphasis">{pour.water_amount}g</span>
                        <span class="text-placeholder">&middot;</span>
                        <span class="text-muted">{pour.time_seconds}s</span>
                      </span>
                    {/each}
                  </div>
                </div>
              {/if}

              {#if selectedRecipe.notes}
                <div class="mb-4">
                  <span class="text-xs text-muted uppercase">Notes</span>
                  <p class="text-sm text-secondary mt-1 whitespace-pre-wrap">
                    {selectedRecipe.notes}
                  </p>
                </div>
              {/if}

              {#if (selectedRecipe.brew_count || 0) > 0 || (selectedRecipe.fork_count || 0) > 0}
                <div class="flex items-center gap-4 mb-4 text-sm text-muted">
                  {#if (selectedRecipe.brew_count || 0) > 0}
                    <span class="flex items-center gap-1.5">
                      <svg
                        class="w-4 h-4"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="1.5"
                        viewBox="0 0 24 24"
                        aria-hidden="true"
                      >
                        <path
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          d="M15.362 5.214A8.252 8.252 0 0 1 12 21 8.25 8.25 0 0 1 6.038 7.047 8.287 8.287 0 0 0 9 9.601a8.983 8.983 0 0 1 3.361-6.867 8.21 8.21 0 0 0 3 2.48Z"
                        ></path>
                      </svg>
                      {selectedRecipe.brew_count} brew{selectedRecipe.brew_count ===
                      1
                        ? ""
                        : "s"}
                    </span>
                  {/if}
                  {#if (selectedRecipe.fork_count || 0) > 0}
                    <span class="flex items-center gap-1.5">
                      <svg
                        class="w-4 h-4"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="1.5"
                        viewBox="0 0 24 24"
                        aria-hidden="true"
                      >
                        <path
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          d="M7.5 21 3 16.5m0 0L7.5 12M3 16.5h13.5m0-13.5L21 7.5m0 0L16.5 12M21 7.5H7.5"
                        ></path>
                      </svg>
                      {selectedRecipe.fork_count} fork{selectedRecipe.fork_count ===
                      1
                        ? ""
                        : "s"}
                      {#if selectedRecipe.forker_avatars && selectedRecipe.forker_avatars.length > 0}
                        <span class="flex -space-x-1.5 ml-1">
                          {#each selectedRecipe.forker_avatars.slice(0, 5) as avatar}
                            {#if avatar}
                              <img
                                src={avatar}
                                alt=""
                                class="w-5 h-5 rounded-full object-cover border border-white"
                                loading="lazy"
                                width="20"
                                height="20"
                              />
                            {/if}
                          {/each}
                        </span>
                      {/if}
                    </span>
                  {/if}
                </div>
              {/if}

              {#if selectedRecipe.source_ref}
                <p class="text-sm text-faint mb-3">
                  Forked from
                  {#if selectedRecipe.source_author_display || selectedRecipe.source_author_handle}
                    <a
                      href={sourceRecipeURL(selectedRecipe)}
                      class="text-emphasis underline hover:text-primary"
                    >
                      {selectedRecipe.source_author_display ||
                        selectedRecipe.source_author_handle}'s recipe
                    </a>
                  {:else}
                    <span>another recipe</span>
                  {/if}
                </p>
              {/if}

              <div class="flex flex-col sm:flex-row gap-2 sm:gap-3">
                <a
                  href={`/brews/new?recipe=${selectedRecipe.rkey}&recipe_owner=${selectedRecipe.author_did || ""}`}
                  class="btn-primary text-sm text-center">Use in Brew</a
                >
                {#if isAuthenticated && !isOwner(selectedRecipe)}
                  <button
                    type="button"
                    onclick={forkRecipe}
                    class="btn-secondary text-sm">Copy Recipe</button
                  >
                {/if}
                <a
                  href={`/recipes/${ownerForURL(selectedRecipe)}/${selectedRecipe.rkey}`}
                  class="btn-secondary text-sm text-center">View Recipe</a
                >
              </div>
            </div>
          </div>
        {/if}

        <div>
          {#if loading}
            <div class="recipe-ledger-grid">
              {#each [0, 1, 2] as index}
                <div
                  class={`recipe-ledger-card recipe-ledger-card--loading animate-pulse ${index === 1 ? "hidden sm:block" : ""} ${index === 2 ? "hidden lg:block" : ""}`}
                >
                  <div class="h-4 bg-brown-200 rounded-sm w-1/3 mb-3"></div>
                  <div class="h-5 bg-brown-200 rounded-sm w-2/3 mb-2"></div>
                  <div class="h-4 bg-brown-200 rounded-sm w-1/2 mb-3"></div>
                  <div class="grid grid-cols-3 gap-2">
                    <div class="h-12 bg-brown-100 rounded-sm"></div>
                    <div class="h-12 bg-brown-100 rounded-sm"></div>
                    <div class="h-12 bg-brown-100 rounded-sm"></div>
                  </div>
                </div>
              {/each}
            </div>
          {:else if recipes.length === 0}
            <div class="recipe-empty-state">
              <p class="text-emphasis text-lg font-medium">
                No recipes match these filters
              </p>
              <p class="text-sm text-muted mt-2">
                Try widening the search, clearing filters, or changing the sort
                order.
              </p>
            </div>
          {:else}
            <div class="recipe-ledger-grid">
              {#each recipes as recipe (recipe.rkey)}
                <div
                  role="button"
                  tabindex="0"
                  class="recipe-ledger-card"
                  class:recipe-ledger-card--selected={selectedRecipe?.rkey ===
                    recipe.rkey}
                  onclick={() => selectRecipe(recipe)}
                  onkeydown={(event) => handleRecipeCardKeydown(event, recipe)}
                >
                  <a
                    href={`/profile/${recipe.author_did}`}
                    class="flex items-center gap-2 mb-3 group/author"
                    onclick={(event) => event.stopPropagation()}
                  >
                    {#if recipe.author_avatar}
                      <img
                        src={recipe.author_avatar}
                        class="w-7 h-7 rounded-full object-cover"
                        alt={authorName(recipe)}
                        loading="lazy"
                        width="28"
                        height="28"
                      />
                    {:else}
                      <span
                        class="w-7 h-7 rounded-full bg-brown-200 flex items-center justify-center text-muted text-xs font-bold"
                      >
                        {authorInitial(recipe)}
                      </span>
                    {/if}
                    <span class="min-w-0 flex-1">
                      {#if recipe.author_display}
                        <span
                          class="block truncate text-sm font-medium text-emphasis group-hover/author:text-primary group-hover/author:underline transition-colors"
                        >
                          {recipe.author_display}
                        </span>
                      {/if}
                      <span
                        class="block truncate text-xs text-muted group-hover/author:text-secondary transition-colors"
                      >
                        {recipe.author_handle || ""}
                      </span>
                    </span>
                  </a>

                  <span class="block font-semibold text-primary mb-2 truncate"
                    >{recipe.name}</span
                  >
                  {#if getBrewerDisplay(recipe) !== "-"}
                    <span class="block text-sm text-muted mb-3"
                      >{getBrewerDisplay(recipe)}</span
                    >
                  {/if}

                  <span class="grid grid-cols-3 gap-2 mb-3">
                    <span class="text-center bg-brown-50/60 rounded-md py-1.5">
                      <span class="stat-label-micro">Coffee</span>
                      <span class="block text-sm font-medium text-primary"
                        >{amount(recipe.coffee_amount)}</span
                      >
                    </span>
                    <span class="text-center bg-brown-50/60 rounded-md py-1.5">
                      <span class="stat-label-micro">Water</span>
                      <span class="block text-sm font-medium text-primary"
                        >{amount(recipe.water_amount)}</span
                      >
                    </span>
                    <span class="text-center bg-brown-50/60 rounded-md py-1.5">
                      <span class="stat-label-micro">Ratio</span>
                      <span class="block text-sm font-medium text-primary"
                        >{formatRatio(recipe)}</span
                      >
                    </span>
                  </span>

                  {#if (recipe.brew_count || 0) > 0 || (recipe.fork_count || 0) > 0}
                    <span
                      class="flex items-center gap-3 pt-2 border-t border-brown-200/60 text-xs text-faint"
                    >
                      {#if (recipe.brew_count || 0) > 0}
                        <span
                          class="flex items-center gap-1"
                          title={`${recipe.brew_count} brews`}
                        >
                          <svg
                            class="w-3.5 h-3.5"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="1.5"
                            viewBox="0 0 24 24"
                            aria-hidden="true"
                          >
                            <path
                              stroke-linecap="round"
                              stroke-linejoin="round"
                              d="M15.362 5.214A8.252 8.252 0 0 1 12 21 8.25 8.25 0 0 1 6.038 7.047 8.287 8.287 0 0 0 9 9.601a8.983 8.983 0 0 1 3.361-6.867 8.21 8.21 0 0 0 3 2.48Z"
                            ></path>
                          </svg>
                          {recipe.brew_count} brew{recipe.brew_count === 1
                            ? ""
                            : "s"}
                        </span>
                      {/if}
                      {#if (recipe.fork_count || 0) > 0}
                        <span
                          class="flex items-center gap-1"
                          title={`${recipe.fork_count} forks`}
                        >
                          <svg
                            class="w-3.5 h-3.5"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="1.5"
                            viewBox="0 0 24 24"
                            aria-hidden="true"
                          >
                            <path
                              stroke-linecap="round"
                              stroke-linejoin="round"
                              d="M7.5 21 3 16.5m0 0L7.5 12M3 16.5h13.5m0-13.5L21 7.5m0 0L16.5 12M21 7.5H7.5"
                            ></path>
                          </svg>
                          {recipe.fork_count} fork{recipe.fork_count === 1
                            ? ""
                            : "s"}
                        </span>
                      {/if}
                      {#if recipe.forker_avatars && recipe.forker_avatars.length > 0}
                        <span class="flex -space-x-1.5 ml-auto">
                          {#each recipe.forker_avatars.slice(0, 3) as avatar}
                            <img
                              src={avatar}
                              alt=""
                              class="w-5 h-5 rounded-full object-cover border border-white"
                              loading="lazy"
                              width="20"
                              height="20"
                            />
                          {/each}
                        </span>
                      {/if}
                    </span>
                  {/if}
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </main>
    </div>
  </div>
{/if}

<style>
  .recipe-alpha-notice {
    margin: 1rem 0 1.5rem;
    border: 0;
    border-radius: 0.25rem;
    background-color: var(--feed-card-bg);
    background-image: linear-gradient(
      color-mix(in oklch, var(--brand-amber-50) 46%, transparent),
      color-mix(in oklch, var(--brand-amber-50) 46%, transparent)
    );
    box-shadow:
      1px 2px 4px rgba(0, 0, 0, 0.1),
      0 1px 2px rgba(0, 0, 0, 0.06);
  }
  .recipe-alpha-notice__summary {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    min-height: 3.5rem;
    padding: 0.65rem 0.75rem;
    cursor: pointer;
  }
  .recipe-alpha-notice__summary::-webkit-details-marker {
    display: none;
  }
  .recipe-alpha-notice__summary :global(svg) {
    width: 1rem;
    height: 1rem;
    color: var(--brand-amber-700);
  }
  .recipe-alpha-notice__summary span,
  .recipe-alpha-notice__summary strong,
  .recipe-alpha-notice__summary small {
    display: block;
  }
  .recipe-alpha-notice__summary strong {
    color: var(--text-primary);
    font-size: 0.76rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .recipe-alpha-notice__summary small {
    margin-top: 0.1rem;
    color: var(--text-muted);
    font-size: 0.73rem;
    font-weight: 400;
  }
  .recipe-alpha-notice__body {
    max-width: 72ch;
    padding: 0 1rem 1rem 2.5rem;
    color: var(--text-muted);
    line-height: 1.55;
  }
  .recipe-alpha-notice__body > p + p {
    margin-top: 0.55rem;
  }
  .recipe-catalog-workspace {
    display: grid;
    grid-template-columns: 1fr;
    gap: 1.5rem;
    align-items: start;
  }
  .recipe-catalog-index {
    padding: 1rem;
    border: 0;
    border-radius: 0.25rem;
    background-color: var(--feed-card-bg);
    background-image: linear-gradient(
      var(--type-recipe-tint),
      var(--type-recipe-tint)
    );
    box-shadow:
      1px 2px 4px rgba(0, 0, 0, 0.1),
      0 1px 2px rgba(0, 0, 0, 0.06);
  }
  .recipe-catalog-index__heading {
    padding-bottom: 0.85rem;
  }
  .recipe-catalog-index__heading p {
    margin: 0;
    color: var(--text-faint);
    font-size: 0.62rem;
    font-weight: 600;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }
  .recipe-catalog-index__heading h2 {
    margin: 0.25rem 0 0;
    color: var(--text-primary);
    font-family: var(--font-display);
    font-size: 1.25rem;
    font-weight: 600;
  }
  .recipe-filter-group {
    margin-top: 1rem;
  }
  .recipe-filter-group > label,
  .recipe-filter-label {
    display: block;
    margin-bottom: 0.4rem;
    color: var(--text-muted);
    font-size: 0.66rem;
    font-weight: 600;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  .recipe-filter-group .form-input {
    width: 100%;
    font-size: 0.8rem;
  }
  .recipe-stamp-list {
    display: flex;
    flex-wrap: wrap;
    gap: 0.35rem;
  }
  .recipe-amount-range {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.65rem;
  }
  .recipe-catalog-results {
    min-width: 0;
  }
  .catalog-results-bar {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    margin-bottom: 1rem;
    padding: 0.25rem 0;
  }
  .catalog-results-bar__count {
    margin: 0;
    color: var(--text-muted);
    font-size: 0.78rem;
    font-variant-numeric: tabular-nums;
  }
  .stamp-sort-label {
    margin-right: 0.2rem;
    color: var(--text-faint);
    font-size: 0.62rem;
    font-weight: 600;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .recipe-detail-panel {
    scroll-margin-top: 6rem;
    margin-bottom: 1rem;
  }
  .recipe-detail-sheet {
    padding: 1.25rem;
    border: 0;
    border-radius: 0.25rem;
    background-color: var(--feed-card-bg);
    background-image: linear-gradient(
      var(--type-recipe-tint),
      var(--type-recipe-tint)
    );
    box-shadow:
      1px 2px 4px rgba(0, 0, 0, 0.1),
      0 1px 2px rgba(0, 0, 0, 0.06);
  }
  .recipe-ledger-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: 0.85rem;
  }
  .recipe-ledger-card {
    position: relative;
    min-width: 0;
    padding: 1rem;
    border: 0;
    border-radius: 0.25rem;
    color: var(--text-primary);
    background-color: var(--feed-card-bg);
    background-image: linear-gradient(
      var(--type-recipe-tint),
      var(--type-recipe-tint)
    );
    box-shadow:
      1px 2px 4px rgba(0, 0, 0, 0.1),
      0 1px 2px rgba(0, 0, 0, 0.06);
    cursor: pointer;
    text-align: left;
    transition:
      transform 150ms cubic-bezier(0.16, 1, 0.3, 1),
      box-shadow 150ms ease-out;
  }
  .recipe-ledger-card:hover {
    transform: translateY(-2px);
    box-shadow: var(--shadow-md);
  }
  .recipe-ledger-card:focus-visible {
    outline: 2px solid var(--type-recipe);
    outline-offset: 2px;
  }
  .recipe-ledger-card--selected {
    outline: 2px solid var(--type-recipe);
    outline-offset: 2px;
    box-shadow:
      var(--shadow-md);
  }
  .recipe-ledger-card--loading {
    min-height: 12rem;
    cursor: default;
  }
  .recipe-empty-state {
    padding: 3rem 1rem;
    border-radius: 0.25rem;
    background: color-mix(
      in oklch,
      var(--type-recipe-tint) 55%,
      var(--feed-card-bg)
    );
    text-align: center;
  }
  @media (min-width: 640px) {
    .catalog-results-bar {
      flex-direction: row;
      align-items: center;
      justify-content: space-between;
    }
    .recipe-ledger-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }
  @media (min-width: 960px) {
    .recipe-catalog-workspace {
      grid-template-columns: minmax(13rem, 0.72fr) minmax(0, 2.25fr);
      gap: clamp(1.5rem, 3vw, 2.5rem);
    }
    .recipe-catalog-index {
      position: sticky;
      top: 6rem;
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .recipe-ledger-card {
      transition: none;
    }
  }
</style>
