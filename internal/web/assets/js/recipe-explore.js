// @ts-check
// Petite-vue factory for the recipe explore page.
// Wraps search/filter/select/fork flows. Share + Report live on the
// canonical recipe view page via the shared action bar.

/**
 * @typedef {object} ExploreRecipe
 * @property {string} rkey
 * @property {string} name
 * @property {string} author_did
 * @property {string} [author_handle]
 * @property {string} [author_display]
 * @property {string} [author_avatar]
 * @property {number} coffee_amount
 * @property {number} water_amount
 * @property {number} ratio
 * @property {string} [brewer_type]
 * @property {{ name: string }} [brewer_obj]
 * @property {string} [notes]
 * @property {string} [source_ref]
 * @property {string} [source_author_handle]
 * @property {string} [source_author_display]
 * @property {number} [brew_count]
 * @property {number} [fork_count]
 * @property {string[]} [forker_avatars]
 * @property {Array<{ water_amount: number, time_seconds: number }>} [pours]
 *
 * @typedef {{
 *   query: string,
 *   category: string,
 *   brewerType: string,
 *   minCoffee: string,
 *   maxCoffee: string,
 *   sortBy: string,
 *   loading: boolean,
 *   recipes: ExploreRecipe[],
 *   selectedRecipe: ExploreRecipe | null,
 *   isAuthenticated: boolean,
 *   userDID: string,
 *   setup: () => void,
 *   onQueryInput: () => void,
 *   setCategory: (cat: string) => void,
 *   setSort: (sort: string) => void,
 *   search: () => Promise<void>,
 *   selectRecipe: (recipe: ExploreRecipe) => void,
 *   formatRatio: (recipe: ExploreRecipe) => string,
 *   getBrewerDisplay: (recipe: ExploreRecipe) => string,
 *   isOwner: (recipe: ExploreRecipe | null) => boolean,
 *   getSourceRecipeURL: (recipe: ExploreRecipe | null) => string,
 *   forkRecipe: () => Promise<void>,
 *   notify: (message: string) => void,
 * }} RecipeExploreScope
 */

/**
 * @param {boolean} isAuthenticated
 * @param {string} userDID
 * @returns {RecipeExploreScope}
 */
function recipeExplore(isAuthenticated, userDID) {
  /** @type {number | undefined} */
  let searchTimer;

  return {
    query: "",
    category: "",
    brewerType: "",
    minCoffee: "",
    maxCoffee: "",
    sortBy: "popular",
    loading: false,
    recipes: [],
    selectedRecipe: null,
    isAuthenticated: !!isAuthenticated,
    userDID: userDID || "",

    setup() {
      this.search();
    },

    onQueryInput() {
      window.clearTimeout(searchTimer);
      searchTimer = window.setTimeout(() => this.search(), 300);
    },

    setCategory(cat) {
      this.category = cat;
      this.search();
    },

    setSort(sort) {
      this.sortBy = sort;
      this.search();
    },

    async search() {
      this.loading = true;
      try {
        const params = new URLSearchParams();
        if (this.query) params.set("q", this.query);
        if (this.category) params.set("category", this.category);
        if (this.brewerType) params.set("brewer_type", this.brewerType);
        if (this.minCoffee) params.set("min_coffee", this.minCoffee);
        if (this.maxCoffee) params.set("max_coffee", this.maxCoffee);
        if (this.sortBy) params.set("sort", this.sortBy);
        const resp = await fetch(`/api/recipes/suggestions?${params}`, {
          credentials: "same-origin",
        });
        if (!resp.ok) throw new Error("Failed to fetch");
        const data = await resp.json();
        this.recipes = Array.isArray(data) ? data : [];
      } catch (e) {
        console.error("Failed to search recipes:", e);
        this.recipes = [];
      } finally {
        this.loading = false;
      }
    },

    selectRecipe(recipe) {
      this.selectedRecipe = recipe;
      // Scroll into view after petite-vue patches the DOM. petite-vue runs
      // updates synchronously, so a microtask is enough — no $nextTick.
      queueMicrotask(() => {
        const el = document.getElementById("recipe-detail-panel");
        if (el) el.scrollIntoView({ behavior: "smooth", block: "start" });
      });
    },

    formatRatio(recipe) {
      if (recipe && recipe.ratio > 0) return `1:${recipe.ratio.toFixed(1)}`;
      return "-";
    },

    getBrewerDisplay(recipe) {
      if (recipe && recipe.brewer_obj && recipe.brewer_obj.name) {
        if (recipe.brewer_type) {
          return recipe.brewer_obj.name + " · " + recipe.brewer_type;
        }
        return recipe.brewer_obj.name;
      }
      return (recipe && recipe.brewer_type) || "-";
    },

    isOwner(recipe) {
      return !!(recipe && recipe.author_did === this.userDID);
    },

    getSourceRecipeURL(recipe) {
      if (!recipe || !recipe.source_ref) return "#";
      const parts = recipe.source_ref.replace("at://", "").split("/");
      if (parts.length < 3) return "#";
      const rkey = parts[2];
      const owner =
        recipe.source_author_handle || recipe.source_author_display || parts[0];
      return `/recipes/${encodeURIComponent(owner)}/${rkey}`;
    },

    async forkRecipe() {
      if (!this.selectedRecipe) return;
      const owner =
        this.selectedRecipe.author_handle || this.selectedRecipe.author_did;
      try {
        const resp = await fetch(
          `/api/recipes/fork/${this.selectedRecipe.rkey}?owner=${encodeURIComponent(owner)}`,
          { method: "POST", credentials: "same-origin" },
        );
        if (!resp.ok) {
          if (resp.status === 401) {
            const showExpired = /** @type {any} */ (window)
              .__showSessionExpiredModal;
            if (typeof showExpired === "function") showExpired();
            return;
          }
          const text = await resp.text();
          throw new Error(text || "Failed to fork recipe");
        }
        this.notify("Recipe copied to your library!");
        this.selectedRecipe = null;
      } catch (e) {
        console.error("Failed to fork recipe:", e);
        const msg = e instanceof Error ? e.message : String(e);
        this.notify("Failed to copy recipe: " + msg);
      }
    },

    notify(message) {
      window.dispatchEvent(
        new CustomEvent("notify", { detail: { message }, bubbles: true }),
      );
    },
  };
}

/** @type {any} */ (window).recipeExplore = recipeExplore;
