/**
 * Alpine.js component for the recipe explore page
 * Handles search, filtering, and recipe detail display
 */
document.addEventListener("alpine:init", () => {
  Alpine.data("recipeExplore", () => ({
    query: "",
    category: "",
    brewerType: "",
    minCoffee: "",
    maxCoffee: "",
    loading: false,
    recipes: [],
    selectedRecipe: null,

    init() {
      this.search();
    },

    setCategory(cat) {
      this.category = cat;
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

        const resp = await fetch(`/api/recipes/suggestions?${params}`, {
          credentials: "same-origin",
        });
        if (!resp.ok) throw new Error("Failed to fetch");
        this.recipes = await resp.json();
        // If no results returned, ensure it's an array
        if (!Array.isArray(this.recipes)) this.recipes = [];
      } catch (e) {
        console.error("Failed to search recipes:", e);
        this.recipes = [];
      } finally {
        this.loading = false;
      }
    },

    selectRecipe(recipe) {
      this.selectedRecipe = recipe;
    },

    formatRatio(recipe) {
      if (recipe.ratio > 0) {
        return `1:${recipe.ratio.toFixed(1)}`;
      }
      return "-";
    },

    getBrewerDisplay(recipe) {
      if (recipe.brewer_obj && recipe.brewer_obj.name) {
        return recipe.brewer_obj.name;
      }
      return recipe.brewer_type || "-";
    },
  }));
});
