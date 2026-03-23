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

    // Mode state
    formMode: "choose", // 'choose' | 'recipe' | 'freeform'
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

    // Dropdown manager instance
    dropdownManager: null,

    // Entity managers for each entity type
    beanManager: null,
    grinderManager: null,
    brewerManager: null,

    async init() {
      // Initialize dropdown manager
      this.dropdownManager = window.createDropdownManager();

      // Initialize entity managers
      this.initEntityManagers();

      // Detect state from DOM
      const root = this.$root || this.$el;
      const formEl = root.querySelector("form");

      this.isEditing = formEl?.hasAttribute("data-editing") || false;
      const recipeRKey = formEl?.getAttribute("data-recipe-rkey") || "";

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

      // Determine initial mode
      if (this.isEditing) {
        this.formMode = recipeRKey ? "recipe" : "freeform";
      } else if (recipeRKey) {
        this.formMode = "recipe";
      } else {
        this.formMode = "choose";
      }

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

      // Also check brewer type after DOM is settled
      this.$nextTick(() => {
        const selects =
          formEl?.querySelectorAll('select[name="brewer_rkey"]') || [];
        for (const sel of selects) {
          if (sel.value && !sel.disabled) {
            this.onBrewerChange(sel.value);
            break;
          }
        }
      });
    },

    // Mode switching
    chooseRecipeMode() {
      this.formMode = "recipe";
    },

    chooseFreeformMode() {
      this.formMode = "freeform";
      this.updatePoursVisibility();
    },

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
      if (brewerType.toLowerCase().includes("pour")) {
        this.showPours = true;
      }
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

    initEntityManagers() {
      // Bean entity manager
      this.beanManager = window.createEntityManager({
        entityType: "bean",
        apiEndpoint: "/api/beans",
        dialogId: "entity-modal",
        defaultFormData: {
          name: "",
          origin: "",
          roast_level: "",
          process: "",
          description: "",
          roaster_rkey: "",
        },
        validate: (data) => {
          if (!data.name || !data.origin) {
            return "Bean name and origin are required";
          }
          return null;
        },
        onSuccess: async (newBean) => {
          // Refresh dropdown data and repopulate
          await this.dropdownManager.invalidateAndRefresh();

          // Select the new bean in all matching selects
          document
            .querySelectorAll('form select[name="bean_rkey"]')
            .forEach((sel) => {
              if (newBean.rkey) sel.value = newBean.rkey;
            });
        },
      });

      // Grinder entity manager
      this.grinderManager = window.createEntityManager({
        entityType: "grinder",
        apiEndpoint: "/api/grinders",
        dialogId: "entity-modal",
        defaultFormData: {
          name: "",
          grinder_type: "",
          burr_type: "",
          notes: "",
        },
        validate: (data) => {
          if (!data.name) {
            return "Grinder name is required";
          }
          return null;
        },
        onSuccess: async (newGrinder) => {
          // Refresh dropdown data and repopulate
          await this.dropdownManager.invalidateAndRefresh();

          // Select the new grinder in all matching selects
          document
            .querySelectorAll('form select[name="grinder_rkey"]')
            .forEach((sel) => {
              if (newGrinder.rkey) sel.value = newGrinder.rkey;
            });
        },
      });

      // Brewer entity manager
      this.brewerManager = window.createEntityManager({
        entityType: "brewer",
        apiEndpoint: "/api/brewers",
        dialogId: "entity-modal",
        defaultFormData: {
          name: "",
          brewer_type: "",
          description: "",
        },
        validate: (data) => {
          if (!data.name) {
            return "Brewer name is required";
          }
          return null;
        },
        onSuccess: async (newBrewer) => {
          // Refresh dropdown data and repopulate
          await this.dropdownManager.invalidateAndRefresh();

          // Select the new brewer in all matching selects
          document
            .querySelectorAll('form select[name="brewer_rkey"]')
            .forEach((sel) => {
              if (newBrewer.rkey) sel.value = newBrewer.rkey;
            });
        },
      });
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
        this.recipeSummaryExpanded = false;
        this.updatePoursVisibility();
        return;
      }

      try {
        const resp = await fetch(`/api/recipes/${rkey}`, {
          credentials: "same-origin",
        });
        if (!resp.ok) return;
        const recipe = await resp.json();

        // Store recipe data for summary display
        this.activeRecipe = recipe;
        this.recipeSummaryExpanded = false;

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
        this.setFormField(form, "brewer_rkey", recipe.brewer_rkey || "");

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
      this.setFormField(form, "brewer_rkey", "");
      this.pours = [];
    },

    // Pours management (brew-specific logic)
    addPour() {
      this.pours.push({ water: "", time: "" });
    },

    removePour(index) {
      this.pours.splice(index, 1);
    },

    // Expose entity manager state to Alpine
    get showBeanForm() {
      return this.beanManager?.showForm || false;
    },
    set showBeanForm(value) {
      if (this.beanManager) this.beanManager.showForm = value;
    },

    get showGrinderForm() {
      return this.grinderManager?.showForm || false;
    },
    set showGrinderForm(value) {
      if (this.grinderManager) this.grinderManager.showForm = value;
    },

    get showBrewerForm() {
      return this.brewerManager?.showForm || false;
    },
    set showBrewerForm(value) {
      if (this.brewerManager) this.brewerManager.showForm = value;
    },

    // Expose entity manager editing state to Alpine (for modal titles)
    get editingBean() {
      return this.beanManager?.editingId !== null;
    },

    get editingGrinder() {
      return this.grinderManager?.editingId !== null;
    },

    get editingBrewer() {
      return this.brewerManager?.editingId !== null;
    },

    // Expose entity manager form data to Alpine
    get beanForm() {
      return this.beanManager?.formData || {};
    },
    set beanForm(value) {
      if (this.beanManager) this.beanManager.formData = value;
    },

    get grinderForm() {
      return this.grinderManager?.formData || {};
    },
    set grinderForm(value) {
      if (this.grinderManager) this.grinderManager.formData = value;
    },

    get brewerForm() {
      return this.brewerManager?.formData || {};
    },
    set brewerForm(value) {
      if (this.brewerManager) this.brewerManager.formData = value;
    },

    // Expose dropdown data to Alpine
    get beans() {
      return this.dropdownManager?.beans || [];
    },

    get grinders() {
      return this.dropdownManager?.grinders || [];
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

    // Delegate save methods to entity managers
    async saveBean() {
      await this.beanManager.save();
    },

    async saveGrinder() {
      await this.grinderManager.save();
    },

    async saveBrewer() {
      await this.brewerManager.save();
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
