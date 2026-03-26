/**
 * Dropdown Manager Module
 * Handles dropdown population with stale-while-revalidate caching pattern
 * Eliminates duplication of dropdown logic in brew-form.js
 */

/**
 * Creates a dropdown manager instance
 * @returns {Object} Dropdown manager with methods for loading and populating data
 */
function createDropdownManager() {
  return {
    // Cached data
    beans: [],
    grinders: [],
    brewers: [],
    roasters: [],
    recipes: [],
    dataLoaded: false,

    /**
     * Loads dropdown data using stale-while-revalidate pattern
     * @param {boolean} forceRefresh - Force a fresh fetch from server
     */
    async loadDropdownData(forceRefresh = false) {
      if (!window.ArabicaCache) {
        console.warn("ArabicaCache not available");
        return;
      }

      // If forcing refresh, always get fresh data
      if (forceRefresh) {
        try {
          const freshData = await window.ArabicaCache.refreshCache(true);
          if (freshData) {
            this.applyData(freshData);
          }
        } catch (e) {
          console.error("Failed to refresh dropdown data:", e);
        }
        return;
      }

      // First, try to immediately populate from cached data (sync)
      // This prevents flickering by showing data instantly
      const cachedData = window.ArabicaCache.getCachedData();
      if (cachedData) {
        this.applyData(cachedData);
      }

      // Then refresh in background if cache is stale
      if (!window.ArabicaCache.isCacheValid()) {
        try {
          const freshData = await window.ArabicaCache.refreshCache();
          if (freshData) {
            this.applyData(freshData);
          }
        } catch (e) {
          console.error("Failed to refresh dropdown data:", e);
          // We already have cached data displayed, so this is non-fatal
        }
      }
    },

    /**
     * Applies loaded data to local state
     * @param {Object} data - Data object with beans, grinders, brewers, roasters
     */
    applyData(data) {
      this.beans = data.beans || [];
      this.grinders = data.grinders || [];
      this.brewers = data.brewers || [];
      this.roasters = data.roasters || [];
      this.recipes = data.recipes || [];
      this.dataLoaded = true;
    },

    /**
     * Populates all dropdowns with current data
     * Preserves currently selected values
     */
    populateDropdowns() {
      this.populateBeans();
      this.populateGrinders();
      this.populateBrewers();
      this.populateRoasters();
    },

    /**
     * Populates bean dropdown(s)
     * @param {string} selectSelector - CSS selector for the select elements (optional)
     */
    populateBeans(selectSelector = 'form select[name="bean_rkey"]') {
      const selects = document.querySelectorAll(selectSelector);
      if (selects.length === 0 || this.beans.length === 0) return;

      selects.forEach((beanSelect) => {
        const selectedBean = beanSelect.value || "";
        beanSelect.innerHTML = "";

        const placeholderOption = document.createElement("option");
        placeholderOption.value = "";
        placeholderOption.textContent = "Select a bean...";
        beanSelect.appendChild(placeholderOption);

        this.beans.forEach((bean) => {
          if (bean.Closed || bean.closed) return;
          const option = document.createElement("option");
          option.value = bean.rkey || bean.RKey;
          const roasterName = bean.Roaster?.Name || bean.roaster?.name || "";
          const roasterSuffix = roasterName ? ` - ${roasterName}` : "";
          option.textContent = `${bean.Name || bean.name} (${bean.Origin || bean.origin} - ${bean.RoastLevel || bean.roast_level})${roasterSuffix}`;
          option.className = "truncate";
          if ((bean.rkey || bean.RKey) === selectedBean) {
            option.selected = true;
          }
          beanSelect.appendChild(option);
        });
      });
    },

    /**
     * Populates grinder dropdown(s)
     * @param {string} selectSelector - CSS selector for the select elements (optional)
     */
    populateGrinders(selectSelector = 'form select[name="grinder_rkey"]') {
      const selects = document.querySelectorAll(selectSelector);
      if (selects.length === 0 || this.grinders.length === 0) return;

      selects.forEach((grinderSelect) => {
        const selectedGrinder = grinderSelect.value || "";
        grinderSelect.innerHTML = "";

        const placeholderOption = document.createElement("option");
        placeholderOption.value = "";
        placeholderOption.textContent = "Select a grinder...";
        grinderSelect.appendChild(placeholderOption);

        this.grinders.forEach((grinder) => {
          const option = document.createElement("option");
          option.value = grinder.rkey || grinder.RKey;
          option.textContent = grinder.Name || grinder.name;
          option.className = "truncate";
          if ((grinder.rkey || grinder.RKey) === selectedGrinder) {
            option.selected = true;
          }
          grinderSelect.appendChild(option);
        });
      });
    },

    /**
     * Populates brewer dropdown(s)
     * @param {string} selectSelector - CSS selector for the select elements (optional)
     */
    populateBrewers(selectSelector = 'form select[name="brewer_rkey"]') {
      const selects = document.querySelectorAll(selectSelector);
      if (selects.length === 0 || this.brewers.length === 0) return;

      selects.forEach((brewerSelect) => {
        const selectedBrewer = brewerSelect.value || "";
        brewerSelect.innerHTML = "";

        const placeholderOption = document.createElement("option");
        placeholderOption.value = "";
        placeholderOption.textContent = "Select brew method...";
        brewerSelect.appendChild(placeholderOption);

        this.brewers.forEach((brewer) => {
          const option = document.createElement("option");
          option.value = brewer.rkey || brewer.RKey;
          option.textContent = brewer.Name || brewer.name;
          option.className = "truncate";
          if ((brewer.rkey || brewer.RKey) === selectedBrewer) {
            option.selected = true;
          }
          brewerSelect.appendChild(option);
        });
      });
    },

    /**
     * Populates roaster dropdown(s) (used in new bean modal)
     * @param {string} selectSelector - CSS selector for the select elements (optional)
     */
    populateRoasters(selectSelector = 'select[name="roaster_rkey_modal"]') {
      const selects = document.querySelectorAll(selectSelector);
      if (selects.length === 0 || this.roasters.length === 0) return;

      selects.forEach((roasterSelect) => {
        const selectedRoaster = roasterSelect.value || "";
        roasterSelect.innerHTML = "";

        const placeholderOption = document.createElement("option");
        placeholderOption.value = "";
        placeholderOption.textContent = "No roaster";
        roasterSelect.appendChild(placeholderOption);

        this.roasters.forEach((roaster) => {
          const option = document.createElement("option");
          option.value = roaster.rkey || roaster.RKey;
          option.textContent = roaster.Name || roaster.name;
          if ((roaster.rkey || roaster.RKey) === selectedRoaster) {
            option.selected = true;
          }
          roasterSelect.appendChild(option);
        });
      });
    },

    /**
     * Populates recipe dropdown(s)
     * @param {string} selectSelector - CSS selector for the select elements (optional)
     */
    populateRecipes(selectSelector = 'form select[name="recipe_rkey"]') {
      const selects = document.querySelectorAll(selectSelector);
      if (selects.length === 0 || this.recipes.length === 0) return;

      selects.forEach((recipeSelect) => {
        const selectedRecipe = recipeSelect.value || "";
        recipeSelect.innerHTML = "";

        const placeholderOption = document.createElement("option");
        placeholderOption.value = "";
        placeholderOption.textContent = "No recipe";
        recipeSelect.appendChild(placeholderOption);

        this.recipes.forEach((recipe) => {
          const option = document.createElement("option");
          option.value = recipe.rkey || recipe.RKey;
          option.textContent = recipe.Name || recipe.name;
          option.className = "truncate";
          if ((recipe.rkey || recipe.RKey) === selectedRecipe) {
            option.selected = true;
          }
          recipeSelect.appendChild(option);
        });
      });
    },

    /**
     * Looks up a brewer's type from the cached brewers array
     * @param {string} rkey - The brewer's record key
     * @returns {string} The brewer_type value, or empty string if not found
     */
    getBrewerType(rkey) {
      if (!rkey) return "";
      const brewer = this.brewers.find((b) => (b.rkey || b.RKey) === rkey);
      if (!brewer) return "";
      return brewer.brewer_type || brewer.BrewerType || "";
    },

    /**
     * Invalidates cache and refreshes dropdowns
     * @returns {Promise<Object>} Fresh data from cache
     */
    async invalidateAndRefresh() {
      if (window.ArabicaCache) {
        const freshData = await window.ArabicaCache.invalidateAndRefresh();
        if (freshData) {
          this.applyData(freshData);
          this.populateDropdowns();
        }
        return freshData;
      }
      return null;
    },
  };
}

// Export for use in other modules
window.createDropdownManager = createDropdownManager;
