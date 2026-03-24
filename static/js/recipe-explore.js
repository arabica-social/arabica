/**
 * Alpine.js component for the recipe explore page
 * Handles search, filtering, recipe detail display, and actions
 */
document.addEventListener("alpine:init", () => {
  Alpine.data("recipeExplore", (isAuthenticated = false, userDID = "") => ({
    query: "",
    category: "",
    brewerType: "",
    minCoffee: "",
    maxCoffee: "",
    loading: false,
    recipes: [],
    selectedRecipe: null,
    isAuthenticated: isAuthenticated,
    userDID: userDID,

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
        if (recipe.brewer_type) {
          return recipe.brewer_obj.name + ' · ' + recipe.brewer_type;
        }
        return recipe.brewer_obj.name;
      }
      return recipe.brewer_type || "-";
    },

    isOwner(recipe) {
      return recipe && recipe.author_did === this.userDID;
    },

    getRecipeURI(recipe) {
      if (!recipe) return "";
      return `at://${recipe.author_did}/social.arabica.alpha.recipe/${recipe.rkey}`;
    },

    getRecipeShareURL(recipe) {
      if (!recipe) return "";
      const owner = recipe.author_handle || recipe.author_did;
      return `/recipes/${recipe.rkey}?owner=${encodeURIComponent(owner)}`;
    },

    shareRecipe() {
      if (!this.selectedRecipe) return;
      const fullUrl =
        window.location.origin + this.getRecipeShareURL(this.selectedRecipe);
      const title = this.selectedRecipe.name || "Recipe";
      const author =
        this.selectedRecipe.author_display ||
        this.selectedRecipe.author_handle ||
        "";
      const text = `Check out this recipe by ${author} on Arabica`;

      if (navigator.share) {
        navigator.share({ title, text, url: fullUrl }).catch(() => {});
      } else {
        navigator.clipboard.writeText(fullUrl).then(() => {
          this.$dispatch("notify", { message: "Link copied!" });
        });
      }
    },

    getSourceRecipeURL(recipe) {
      if (!recipe || !recipe.source_ref) return "#";
      // source_ref is an AT-URI like at://did/collection/rkey
      const parts = recipe.source_ref.replace("at://", "").split("/");
      if (parts.length < 3) return "#";
      const rkey = parts[2];
      const owner =
        recipe.source_author_handle || recipe.source_author_display || parts[0];
      return `/recipes/${rkey}?owner=${encodeURIComponent(owner)}`;
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
            window.__showSessionExpiredModal();
            return;
          }
          const text = await resp.text();
          throw new Error(text || "Failed to fork recipe");
        }
        this.$dispatch("notify", { message: "Recipe copied to your library!" });
        this.selectedRecipe = null;
      } catch (e) {
        console.error("Failed to fork recipe:", e);
        this.$dispatch("notify", {
          message: "Failed to copy recipe: " + e.message,
        });
      }
    },

    openReport() {
      const dialog = document.getElementById("recipe-report-modal");
      if (dialog) {
        dialog.showModal();
      }
    },
  }));
});
