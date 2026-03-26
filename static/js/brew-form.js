/**
 * Alpine.js component for the brew form
 * Manages pours, new entity modals, form mode, and form state
 * Uses shared entity-manager and dropdown-manager modules
 */

// Wait for Alpine to be available and register the component
document.addEventListener("alpine:init", () => {
  Alpine.data("brewForm", () => ({
    // Brew form specific
    rating: 5,
    pours: [],
    brewerCategory: "", // 'pourover' | 'espresso' | 'immersion' | ''

    // Mode state
    formMode: "recipe",
    recipeSummaryExpanded: false,
    activeRecipe: null,
    showPours: false,
    isEditing: false,

    // Recipe filter state
    searchQuery: "",
    activeCategory: "",
    filteredCount: 0,
    totalCount: 0,
    recipes: [],

    // Recipe owner DID (for cross-user recipe references)
    recipeOwnerDID: "",

    // Dropdown manager instance
    dropdownManager: null,

    async init() {
      // Initialize dropdown manager
      this.dropdownManager = window.createDropdownManager();

      // Detect state from DOM
      const root = this.$root || this.$el;
      const formEl = root.querySelector("form");

      this.isEditing = formEl?.hasAttribute("data-editing") || false;
      const recipeRKey = formEl?.getAttribute("data-recipe-rkey") || "";
      this.recipeOwnerDID = formEl?.getAttribute("data-recipe-owner") || "";

      // Load existing pours if editing
      const poursData = formEl?.getAttribute("data-pours");
      if (poursData) {
        try {
          this.pours = JSON.parse(poursData);
        } catch (e) {
          console.error("Failed to parse pours data:", e);
          this.pours = [];
        }
      }

      // Always use recipe mode (recipe selection is optional)
      this.formMode = "recipe";

      // Populate dropdowns from cache using stale-while-revalidate pattern
      await this.dropdownManager.loadDropdownData();
      this.dropdownManager.populateDropdowns();

      // Initialize recipe filter state from loaded data
      this.recipes = this.dropdownManager.recipes || [];
      this.totalCount = this.recipes.length;
      this.filteredCount = this.recipes.length;

      // Re-sync recipes when cache refreshes
      if (window.ArabicaCache) {
        window.ArabicaCache.addListener((data) => {
          this.recipes = data.recipes || [];
          this.filterRecipes();
        });
      }

      // Auto-apply recipe if rkey present
      if (recipeRKey) {
        const recipeSelect = root.querySelector(
          'form select[name="recipe_rkey"]',
        );
        if (recipeSelect) {
          recipeSelect.value = recipeRKey;
        }
        await this.applyRecipe(recipeRKey);
      }

      // Update pours visibility after setup
      this.updatePoursVisibility();

      // Listen for combo-select changes (brewer type drives form sections)
      root.addEventListener("combo-change", (e) => {
        if (e.detail.entityType === "brewer") {
          const bt =
            e.detail.entity?.brewer_type ||
            e.detail.entity?.BrewerType ||
            "";
          this.brewerCategory = this.normalizeBrewerCategory(bt);
          if (this.brewerCategory === "pourover") {
            this.showPours = true;
          }
        }
      });
    },

    // Mode switching

    // Pours visibility
    updatePoursVisibility() {
      if (this.pours.length > 0) {
        this.showPours = true;
        return;
      }
      if (this.activeRecipe?.pours?.length > 0) {
        this.showPours = true;
        return;
      }
    },

    togglePours() {
      this.showPours = !this.showPours;
      if (this.showPours && this.pours.length === 0) {
        this.addPour();
      }
    },

    onBrewerChange(rkey) {
      const brewerType = this.dropdownManager?.getBrewerType(rkey) || "";
      this.brewerCategory = this.normalizeBrewerCategory(brewerType);

      // Auto-show pours for pour-over brewers
      if (this.brewerCategory === "pourover") {
        this.showPours = true;
      }
    },

    // Map brewer type strings to canonical categories
    normalizeBrewerCategory(raw) {
      if (!raw) return "";
      const lower = raw.toLowerCase().trim();

      // Direct match on canonical values
      if (
        [
          "pourover",
          "espresso",
          "immersion",
          "mokapot",
          "coldbrew",
          "cupping",
          "other",
        ].includes(lower)
      )
        return lower;

      // Legacy freeform mappings
      if (
        ["pour-over", "pour over", "dripper"].includes(lower)
      )
        return "pourover";
      if (
        [
          "espresso machine",
          "lever espresso",
          "lever espresso machine",
        ].includes(lower)
      )
        return "espresso";
      if (
        [
          "french press",
          "aeropress",
          "siphon",
          "clever",
          "clever dripper",
        ].includes(lower)
      )
        return "immersion";

      // TODO: future method types
      // if (['moka pot', 'moka', 'bialetti'].includes(lower)) return 'mokapot';
      // if (['cold brew', 'cold drip'].includes(lower)) return 'coldbrew';

      return "";
    },

    // Recipe summary text
    get recipeSummaryText() {
      if (!this.activeRecipe) return "";
      const parts = [];
      if (this.activeRecipe.coffee_amount > 0) {
        parts.push(Math.round(this.activeRecipe.coffee_amount) + "g coffee");
      }
      if (this.activeRecipe.water_amount > 0) {
        parts.push(Math.round(this.activeRecipe.water_amount) + "g water");
      }
      if (this.activeRecipe.brewer_rkey) {
        const brewer = (this.dropdownManager?.brewers || []).find(
          (b) => (b.rkey || b.RKey) === this.activeRecipe.brewer_rkey,
        );
        if (brewer) {
          parts.push(brewer.Name || brewer.name);
        }
      }
      if (this.activeRecipe.pours && this.activeRecipe.pours.length > 0) {
        parts.push(this.activeRecipe.pours.length + " pours");
      }
      return parts.join(" \u00b7 ");
    },

    // Recipe autofill
    async applyRecipe(rkey) {
      const root = this.$root || this.$el;
      const form = root.querySelector("form") || root.closest("form");
      if (!form) return;

      // If no recipe selected, clear all recipe-populated fields
      if (!rkey) {
        this.clearRecipeFields(form);
        this.activeRecipe = null;
        this.recipeOwnerDID = "";
        this.recipeSummaryExpanded = false;
        this.updatePoursVisibility();
        return;
      }

      // Look up owner DID from cached recipes (for dropdown selections)
      const cachedRecipe = this.recipes.find(
        (r) => (r.rkey || r.RKey) === rkey,
      );
      if (cachedRecipe && cachedRecipe.author_did) {
        this.recipeOwnerDID = cachedRecipe.author_did;
      }

      try {
        let url = `/api/recipes/${rkey}`;
        if (this.recipeOwnerDID) {
          url += `?owner=${encodeURIComponent(this.recipeOwnerDID)}`;
        }
        const resp = await fetch(url, {
          credentials: "same-origin",
        });
        if (!resp.ok) return;
        const recipe = await resp.json();

        // Store recipe data for summary display
        this.activeRecipe = recipe;
        this.recipeSummaryExpanded = false;

        // Track owner DID from API response
        if (recipe.author_did) {
          this.recipeOwnerDID = recipe.author_did;
        }

        // Set or clear each field based on recipe data
        this.setFormField(
          form,
          "coffee_amount",
          recipe.coffee_amount > 0 ? Math.round(recipe.coffee_amount) : "",
        );
        this.setFormField(
          form,
          "water_amount",
          recipe.water_amount > 0 ? Math.round(recipe.water_amount) : "",
        );
        // Update brewer combo-select via event
        const brewerCombo = form.querySelector(
          '[x-data*="entityType: \'brewer\'"]',
        );
        if (brewerCombo) {
          const brewerName =
            (this.dropdownManager?.brewers || []).find(
              (b) =>
                (b.rkey || b.RKey) === recipe.brewer_rkey,
            )?.name || "";
          brewerCombo.dispatchEvent(
            new CustomEvent("combo-set", {
              detail: {
                rkey: recipe.brewer_rkey || "",
                label: brewerName,
              },
              bubbles: false,
            }),
          );
        }
        // Also update brewer category
        if (recipe.brewer_rkey) {
          this.onBrewerChange(recipe.brewer_rkey);
        }

        // Always reset pours, then apply recipe pours if present
        this.pours =
          recipe.pours && recipe.pours.length > 0
            ? recipe.pours.map((p) => ({
                water: p.water_amount || "",
                time: p.time_seconds || "",
              }))
            : [];

        this.updatePoursVisibility();
      } catch (e) {
        console.error("Failed to apply recipe:", e);
      }
    },

    setFormField(form, name, value) {
      // Set all matching fields (both mode sections have their own inputs)
      form.querySelectorAll(`[name="${name}"]`).forEach((el) => {
        el.value = value;
        el.dispatchEvent(new Event("input", { bubbles: true }));
      });
    },

    clearRecipeFields(form) {
      this.setFormField(form, "coffee_amount", "");
      this.setFormField(form, "water_amount", "");
      // Clear brewer combo-select
      const brewerCombo = form.querySelector(
        '[x-data*="entityType: \'brewer\'"]',
      );
      if (brewerCombo) {
        brewerCombo.dispatchEvent(
          new CustomEvent("combo-set", {
            detail: { rkey: "", label: "" },
            bubbles: false,
          }),
        );
      }
      this.pours = [];
    },

    // Pours management (brew-specific logic)
    addPour() {
      this.pours.push({ water: "", time: "" });
    },

    removePour(index) {
      this.pours.splice(index, 1);
    },

    // Expose dropdown data to Alpine (still needed for recipe filtering)
    get beans() {
      return this.dropdownManager?.beans || [];
    },

    get brewers() {
      return this.dropdownManager?.brewers || [];
    },

    get roasters() {
      return this.dropdownManager?.roasters || [];
    },

    get dataLoaded() {
      return this.dropdownManager?.dataLoaded || false;
    },

    // Recipe filter methods
    recipeCategories: {
      small: { maxCoffee: 12 },
      single: { minCoffee: 12, maxCoffee: 22, maxWater: 400 },
      large: { minCoffee: 22 },
      batch: { minWater: 500 },
    },

    setCategory(cat) {
      this.activeCategory = cat;
      this.filterRecipes();
    },

    filterRecipes() {
      const root = this.$root || this.$el;
      const select = root.querySelector('form select[name="recipe_rkey"]');
      if (!select) return;

      const query = this.searchQuery.toLowerCase().trim();
      const cat = this.recipeCategories[this.activeCategory];

      let total = 0;
      let shown = 0;

      // Rebuild options: keep placeholder, filter the rest
      const selectedValue = select.value;
      select.innerHTML = "";

      const placeholder = document.createElement("option");
      placeholder.value = "";
      placeholder.textContent = "No recipe";
      select.appendChild(placeholder);

      for (const recipe of this.recipes) {
        total++;
        const name = (recipe.name || recipe.Name || "").toLowerCase();
        const coffee = recipe.coffee_amount || 0;
        // Interpolate water from pours if not set
        let water = recipe.water_amount || 0;
        if (water === 0 && recipe.pours && recipe.pours.length > 0) {
          water = recipe.pours.reduce(
            (sum, p) => sum + (p.water_amount || 0),
            0,
          );
        }

        // Text filter
        if (query && !name.includes(query)) continue;

        // Category filter
        if (cat) {
          if (cat.maxCoffee && coffee > cat.maxCoffee) continue;
          if (cat.minCoffee && coffee < cat.minCoffee) continue;
          if (cat.maxWater && water > cat.maxWater) continue;
          if (cat.minWater && water < cat.minWater) continue;
          // Skip recipes with no amount data when filtering by category
          if ((cat.maxCoffee || cat.minCoffee) && coffee === 0) continue;
          if ((cat.maxWater || cat.minWater) && water === 0) continue;
        }

        shown++;
        const option = document.createElement("option");
        option.value = recipe.rkey || recipe.RKey;
        option.textContent = recipe.name || recipe.Name;
        option.className = "truncate";
        if ((recipe.rkey || recipe.RKey) === selectedValue) {
          option.selected = true;
        }
        select.appendChild(option);
      }

      this.totalCount = total;
      this.filteredCount = shown;
    },
  }));
});
