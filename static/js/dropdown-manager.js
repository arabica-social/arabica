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
     * Populates bean dropdown
     * @param {string} selectSelector - CSS selector for the select element (optional)
     */
    populateBeans(selectSelector = 'form select[name="bean_rkey"]') {
      const beanSelect = document.querySelector(selectSelector);
      if (!beanSelect || this.beans.length === 0) return;

      const selectedBean = beanSelect.value || "";

      // Clear existing options
      beanSelect.innerHTML = "";

      // Add placeholder
      const placeholderOption = document.createElement("option");
      placeholderOption.value = "";
      placeholderOption.textContent = "Select a bean...";
      beanSelect.appendChild(placeholderOption);

      // Add bean options
      this.beans.forEach((bean) => {
        const option = document.createElement("option");
        option.value = bean.rkey || bean.RKey;
        const roasterName = bean.Roaster?.Name || bean.roaster?.name || "";
        const roasterSuffix = roasterName ? ` - ${roasterName}` : "";
        // Using textContent ensures all user input is safely escaped
        option.textContent = `${bean.Name || bean.name} (${bean.Origin || bean.origin} - ${bean.RoastLevel || bean.roast_level})${roasterSuffix}`;
        option.className = "truncate";
        if ((bean.rkey || bean.RKey) === selectedBean) {
          option.selected = true;
        }
        beanSelect.appendChild(option);
      });
    },

    /**
     * Populates grinder dropdown
     * @param {string} selectSelector - CSS selector for the select element (optional)
     */
    populateGrinders(selectSelector = 'form select[name="grinder_rkey"]') {
      const grinderSelect = document.querySelector(selectSelector);
      if (!grinderSelect || this.grinders.length === 0) return;

      const selectedGrinder = grinderSelect.value || "";

      // Clear existing options
      grinderSelect.innerHTML = "";

      // Add placeholder
      const placeholderOption = document.createElement("option");
      placeholderOption.value = "";
      placeholderOption.textContent = "Select a grinder...";
      grinderSelect.appendChild(placeholderOption);

      // Add grinder options
      this.grinders.forEach((grinder) => {
        const option = document.createElement("option");
        option.value = grinder.rkey || grinder.RKey;
        // Using textContent ensures all user input is safely escaped
        option.textContent = grinder.Name || grinder.name;
        option.className = "truncate";
        if ((grinder.rkey || grinder.RKey) === selectedGrinder) {
          option.selected = true;
        }
        grinderSelect.appendChild(option);
      });
    },

    /**
     * Populates brewer dropdown
     * @param {string} selectSelector - CSS selector for the select element (optional)
     */
    populateBrewers(selectSelector = 'form select[name="brewer_rkey"]') {
      const brewerSelect = document.querySelector(selectSelector);
      if (!brewerSelect || this.brewers.length === 0) return;

      const selectedBrewer = brewerSelect.value || "";

      // Clear existing options
      brewerSelect.innerHTML = "";

      // Add placeholder
      const placeholderOption = document.createElement("option");
      placeholderOption.value = "";
      placeholderOption.textContent = "Select brew method...";
      brewerSelect.appendChild(placeholderOption);

      // Add brewer options
      this.brewers.forEach((brewer) => {
        const option = document.createElement("option");
        option.value = brewer.rkey || brewer.RKey;
        // Using textContent ensures all user input is safely escaped
        option.textContent = brewer.Name || brewer.name;
        option.className = "truncate";
        if ((brewer.rkey || brewer.RKey) === selectedBrewer) {
          option.selected = true;
        }
        brewerSelect.appendChild(option);
      });
    },

    /**
     * Populates roaster dropdown (used in new bean modal)
     * @param {string} selectSelector - CSS selector for the select element (optional)
     */
    populateRoasters(selectSelector = 'select[name="roaster_rkey_modal"]') {
      const roasterSelect = document.querySelector(selectSelector);
      if (!roasterSelect || this.roasters.length === 0) return;

      const selectedRoaster = roasterSelect.value || "";

      // Clear existing options
      roasterSelect.innerHTML = "";

      // Add placeholder
      const placeholderOption = document.createElement("option");
      placeholderOption.value = "";
      placeholderOption.textContent = "No roaster";
      roasterSelect.appendChild(placeholderOption);

      // Add roaster options
      this.roasters.forEach((roaster) => {
        const option = document.createElement("option");
        option.value = roaster.rkey || roaster.RKey;
        // Using textContent ensures all user input is safely escaped
        option.textContent = roaster.Name || roaster.name;
        if ((roaster.rkey || roaster.RKey) === selectedRoaster) {
          option.selected = true;
        }
        roasterSelect.appendChild(option);
      });
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
