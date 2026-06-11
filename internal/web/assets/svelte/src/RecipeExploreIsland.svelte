<script lang="ts">
  import { onMount } from "svelte";

  type ExploreRecipe = {
    rkey: string;
    name: string;
    author_did: string;
    author_handle?: string;
    author_display?: string;
    author_avatar?: string;
    coffee_amount: number;
    water_amount: number;
    ratio: number;
    brewer_type?: string;
    brewer_obj?: { name?: string };
    notes?: string;
    source_ref?: string;
    source_author_handle?: string;
    source_author_display?: string;
    brew_count?: number;
    fork_count?: number;
    forker_avatars?: string[];
    pours?: Array<{ water_amount: number; time_seconds: number }>;
  };

  let { target }: { target: HTMLElement } = $props();

  let query = $state("");
  let category = $state("");
  let brewerType = $state("");
  let minCoffee = $state("");
  let maxCoffee = $state("");
  let sortBy = $state("popular");
  let loading = $state(false);
  let recipes = $state<ExploreRecipe[]>([]);
  let selectedRecipe = $state<ExploreRecipe | null>(null);
  let isAuthenticated = $state(false);
  let userDID = $state("");
  let searchTimer: number | undefined;

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

  function debounceSearch() {
    window.clearTimeout(searchTimer);
    searchTimer = window.setTimeout(() => {
      void search();
    }, 300);
  }

  function setCategory(nextCategory: string) {
    category = nextCategory;
    void search();
  }

  function setSort(nextSort: string) {
    sortBy = nextSort;
    void search();
  }

  function searchParams() {
    const params = new URLSearchParams();
    if (query) params.set("q", query);
    if (category) params.set("category", category);
    if (brewerType) params.set("brewer_type", brewerType);
    if (minCoffee) params.set("min_coffee", minCoffee);
    if (maxCoffee) params.set("max_coffee", maxCoffee);
    if (sortBy) params.set("sort", sortBy);
    return params;
  }

  async function search() {
    loading = true;
    try {
      const response = await fetch(
        `/api/recipes/suggestions?${searchParams()}`,
        {
          credentials: "same-origin",
        },
      );
      if (!response.ok) {
        throw new Error("Failed to fetch recipes");
      }
      const data: unknown = await response.json();
      recipes = Array.isArray(data) ? (data as ExploreRecipe[]) : [];
    } catch (error) {
      console.error("Failed to search recipes:", error);
      recipes = [];
    } finally {
      loading = false;
    }
  }

  function selectRecipe(recipe: ExploreRecipe) {
    selectedRecipe = recipe;
    queueMicrotask(() => {
      document
        .getElementById("recipe-detail-panel")
        ?.scrollIntoView({ behavior: "smooth", block: "start" });
    });
  }

  function handleRecipeCardKeydown(
    event: KeyboardEvent,
    recipe: ExploreRecipe,
  ) {
    if (event.key !== "Enter" && event.key !== " ") {
      return;
    }
    event.preventDefault();
    selectRecipe(recipe);
  }

  function formatRatio(recipe: ExploreRecipe | null) {
    if (recipe && recipe.ratio > 0) {
      return `1:${recipe.ratio.toFixed(1)}`;
    }
    return "-";
  }

  function amount(value: number | undefined) {
    return value && value > 0 ? `${value.toFixed(1)}g` : "-";
  }

  function getBrewerDisplay(recipe: ExploreRecipe | null) {
    if (recipe?.brewer_obj?.name) {
      return recipe.brewer_type
        ? `${recipe.brewer_obj.name} · ${recipe.brewer_type}`
        : recipe.brewer_obj.name;
    }
    return recipe?.brewer_type || "-";
  }

  function authorName(recipe: ExploreRecipe) {
    return recipe.author_display || recipe.author_handle || recipe.author_did;
  }

  function authorInitial(recipe: ExploreRecipe) {
    return (
      (recipe.author_display ||
        recipe.author_handle ||
        "?")[0]?.toUpperCase() || "?"
    );
  }

  function ownerForURL(recipe: ExploreRecipe) {
    return encodeURIComponent(recipe.author_handle || recipe.author_did);
  }

  function isOwner(recipe: ExploreRecipe | null) {
    return !!(recipe && recipe.author_did === userDID);
  }

  function sourceRecipeURL(recipe: ExploreRecipe | null) {
    if (!recipe?.source_ref) {
      return "#";
    }
    const parts = recipe.source_ref.replace("at://", "").split("/");
    if (parts.length < 3) {
      return "#";
    }
    const owner =
      recipe.source_author_handle || recipe.source_author_display || parts[0];
    return `/recipes/${encodeURIComponent(owner)}/${parts[2]}`;
  }

  function notify(message: string) {
    window.dispatchEvent(
      new CustomEvent("notify", { detail: { message }, bubbles: true }),
    );
  }

  async function forkRecipe() {
    if (!selectedRecipe) {
      return;
    }
    const owner = selectedRecipe.author_handle || selectedRecipe.author_did;
    try {
      const response = await fetch(
        `/api/recipes/fork/${selectedRecipe.rkey}?owner=${encodeURIComponent(owner)}`,
        { method: "POST", credentials: "same-origin" },
      );
      if (!response.ok) {
        if (response.status === 401) {
          window.__showSessionExpiredModal?.();
          return;
        }
        const text = await response.text();
        throw new Error(text || "Failed to copy recipe");
      }
      notify("Recipe copied to your library!");
      selectedRecipe = null;
    } catch (error) {
      console.error("Failed to fork recipe:", error);
      notify(
        `Failed to copy recipe: ${error instanceof Error ? error.message : String(error)}`,
      );
    }
  }

  onMount(() => {
    isAuthenticated = target.dataset.isAuthenticated === "true";
    userDID = target.dataset.userDid || "";
    void search();
    return () => {
      window.clearTimeout(searchTimer);
    };
  });
</script>

<div class="card card-inner mb-6">
  <div class="space-y-4">
    <input
      type="text"
      bind:value={query}
      oninput={debounceSearch}
      placeholder="Search recipes by name..."
      aria-label="Search recipes"
      class="w-full form-input"
    />

    <div
      class="flex flex-wrap gap-2"
      role="group"
      aria-label="Recipe size filters"
    >
      {#each categories as option}
        <button
          type="button"
          class={category === option.value
            ? "filter-pill-active"
            : "filter-pill"}
          aria-pressed={category === option.value ? "true" : "false"}
          onclick={() => setCategory(option.value)}
        >
          {option.label}
        </button>
      {/each}
    </div>

    <div class="flex flex-col sm:flex-row gap-4 sm:items-end">
      <div class="flex-1">
        <label class="form-label text-sm" for="recipe-explore-brewer-type"
          >Brewer Type</label
        >
        <input
          id="recipe-explore-brewer-type"
          type="text"
          bind:value={brewerType}
          oninput={debounceSearch}
          placeholder="e.g. Pour-Over, French Press..."
          class="w-full form-input text-sm"
        />
      </div>
      <div class="sm:w-28">
        <label class="form-label text-sm" for="recipe-explore-min-coffee"
          >Min coffee (g)</label
        >
        <input
          id="recipe-explore-min-coffee"
          type="number"
          bind:value={minCoffee}
          oninput={debounceSearch}
          placeholder="0"
          step="1"
          class="w-full form-input text-sm"
        />
      </div>
      <div class="sm:w-28">
        <label class="form-label text-sm" for="recipe-explore-max-coffee"
          >Max coffee (g)</label
        >
        <input
          id="recipe-explore-max-coffee"
          type="number"
          bind:value={maxCoffee}
          oninput={debounceSearch}
          placeholder="any"
          step="1"
          class="w-full form-input text-sm"
        />
      </div>
    </div>
  </div>
</div>

<div class="flex items-center justify-between mb-3">
  {#if !loading && recipes.length > 0}
    <p class="text-sm text-muted">
      {recipes.length} recipe{recipes.length === 1 ? "" : "s"} found
    </p>
  {:else}
    <div></div>
  {/if}
  <div class="flex items-center gap-1 text-sm">
    <span class="text-faint mr-1">Sort:</span>
    {#each sorts as option}
      <button
        type="button"
        class={sortBy === option.value
          ? "filter-pill-active text-xs"
          : "filter-pill text-xs"}
        aria-pressed={sortBy === option.value ? "true" : "false"}
        onclick={() => setSort(option.value)}
      >
        {option.label}
      </button>
    {/each}
  </div>
</div>

{#if selectedRecipe}
  <div class="mb-4" id="recipe-detail-panel">
    <div class="card card-inner">
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
          <p class="font-medium text-primary">{formatRatio(selectedRecipe)}</p>
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
                <span class="font-medium text-secondary">{index + 1}</span>
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
                ><path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M15.362 5.214A8.252 8.252 0 0 1 12 21 8.25 8.25 0 0 1 6.038 7.047 8.287 8.287 0 0 0 9 9.601a8.983 8.983 0 0 1 3.361-6.867 8.21 8.21 0 0 0 3 2.48Z"
                ></path></svg
              >
              {selectedRecipe.brew_count} brew{selectedRecipe.brew_count === 1
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
                ><path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M7.5 21 3 16.5m0 0L7.5 12M3 16.5h13.5m0-13.5L21 7.5m0 0L16.5 12M21 7.5H7.5"
                ></path></svg
              >
              {selectedRecipe.fork_count} fork{selectedRecipe.fork_count === 1
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
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each [0, 1, 2] as index}
        <div
          class={`feed-card animate-pulse ${index === 1 ? "hidden sm:block" : ""} ${index === 2 ? "hidden lg:block" : ""}`}
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
    <div class="card card-inner text-center py-8">
      <p class="text-emphasis text-lg font-medium">No recipes found</p>
      <p class="text-sm text-muted mt-2">
        Try adjusting your filters or search terms
      </p>
    </div>
  {:else}
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each recipes as recipe}
        <div
          role="button"
          tabindex="0"
          class={`feed-card feed-card-recipe cursor-pointer transition-shadow text-left ${selectedRecipe?.rkey === recipe.rkey ? "ring-2 ring-brown-400" : ""}`}
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
                    ><path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      d="M15.362 5.214A8.252 8.252 0 0 1 12 21 8.25 8.25 0 0 1 6.038 7.047 8.287 8.287 0 0 0 9 9.601a8.983 8.983 0 0 1 3.361-6.867 8.21 8.21 0 0 0 3 2.48Z"
                    ></path></svg
                  >
                  {recipe.brew_count} brew{recipe.brew_count === 1 ? "" : "s"}
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
                    ><path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      d="M7.5 21 3 16.5m0 0L7.5 12M3 16.5h13.5m0-13.5L21 7.5m0 0L16.5 12M21 7.5H7.5"
                    ></path></svg
                  >
                  {recipe.fork_count} fork{recipe.fork_count === 1 ? "" : "s"}
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
