/**
 * Alpine.js component for the brew form
 * Manages pours, new entity modals, and form state
 * Populates dropdowns from client-side cache for faster UX
 */
function brewForm() {
  return {
    // Modal state (matching manage page)
    showBeanForm: false,
    showGrinderForm: false,
    showBrewerForm: false,
    editingBean: null,
    editingGrinder: null,
    editingBrewer: null,
    
    // Form data (matching manage page with snake_case)
    beanForm: {
      name: "",
      origin: "",
      roast_level: "",
      process: "",
      description: "",
      roaster_rkey: "",
    },
    grinderForm: { name: "", grinder_type: "", burr_type: "", notes: "" },
    brewerForm: { name: "", brewer_type: "", description: "" },
    
    // Brew form specific
    rating: 5,
    pours: [],

    // Dropdown data
    beans: [],
    grinders: [],
    brewers: [],
    roasters: [],
    dataLoaded: false,

    async init() {
      // Load existing pours if editing
      // $el is now the parent div, so find the form element
      const formEl = this.$el.querySelector("form");
      const poursData = formEl?.getAttribute("data-pours");
      if (poursData) {
        try {
          this.pours = JSON.parse(poursData);
        } catch (e) {
          console.error("Failed to parse pours data:", e);
          this.pours = [];
        }
      }

      // Populate dropdowns from cache using stale-while-revalidate pattern
      await this.loadDropdownData();
    },

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

    applyData(data) {
      this.beans = data.beans || [];
      this.grinders = data.grinders || [];
      this.brewers = data.brewers || [];
      this.roasters = data.roasters || [];
      this.dataLoaded = true;

      // Populate the select elements
      this.populateDropdowns();
    },

    populateDropdowns() {
      // Get the current selected values (from server-rendered form when editing)
      // Use document.querySelector to ensure we find the form selects, not modal selects
      const beanSelect = document.querySelector('form select[name="bean_rkey"]');
      const grinderSelect = document.querySelector('form select[name="grinder_rkey"]');
      const brewerSelect = document.querySelector('form select[name="brewer_rkey"]');

      const selectedBean = beanSelect?.value || "";
      const selectedGrinder = grinderSelect?.value || "";
      const selectedBrewer = brewerSelect?.value || "";

      // Populate beans - using DOM methods to prevent XSS
      if (beanSelect && this.beans.length > 0) {
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
      }

      // Populate grinders - using DOM methods to prevent XSS
      if (grinderSelect && this.grinders.length > 0) {
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
      }

      // Populate brewers - using DOM methods to prevent XSS
      if (brewerSelect && this.brewers.length > 0) {
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
      }

      // Populate roasters in new bean modal - using DOM methods to prevent XSS
      const roasterSelect = document.querySelector('select[name="roaster_rkey_modal"]');
      if (roasterSelect && this.roasters.length > 0) {
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
          roasterSelect.appendChild(option);
        });
      }
    },

    addPour() {
      this.pours.push({ water: "", time: "" });
    },

    removePour(index) {
      this.pours.splice(index, 1);
    },

    async saveBean() {
      if (!this.beanForm.name || !this.beanForm.origin) {
        alert("Bean name and origin are required");
        return;
      }
      
      const response = await fetch("/api/beans", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(this.beanForm),
      });
      
      if (response.ok) {
        const newBean = await response.json();
        
        // Invalidate cache and refresh data in one call
        let freshData = null;
        if (window.ArabicaCache) {
          freshData = await window.ArabicaCache.invalidateAndRefresh();
        }
        
        // Apply the fresh data to update dropdowns
        if (freshData) {
          this.applyData(freshData);
        }
        
        // Select the new bean
        const beanSelect = document.querySelector('form select[name="bean_rkey"]');
        if (beanSelect && newBean.rkey) {
          beanSelect.value = newBean.rkey;
        }
        
        // Close modal and reset form
        this.showBeanForm = false;
        this.beanForm = {
          name: "",
          origin: "",
          roast_level: "",
          process: "",
          description: "",
          roaster_rkey: "",
        };
      } else {
        const errorText = await response.text();
        alert("Failed to add bean: " + errorText);
      }
    },

    async saveGrinder() {
      if (!this.grinderForm.name) {
        alert("Grinder name is required");
        return;
      }
      
      const response = await fetch("/api/grinders", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(this.grinderForm),
      });
      
      if (response.ok) {
        const newGrinder = await response.json();
        
        // Invalidate cache and refresh data in one call
        let freshData = null;
        if (window.ArabicaCache) {
          freshData = await window.ArabicaCache.invalidateAndRefresh();
        }
        
        // Apply the fresh data to update dropdowns
        if (freshData) {
          this.applyData(freshData);
        }
        
        // Select the new grinder
        const grinderSelect = document.querySelector('form select[name="grinder_rkey"]');
        if (grinderSelect && newGrinder.rkey) {
          grinderSelect.value = newGrinder.rkey;
        }
        
        // Close modal and reset form
        this.showGrinderForm = false;
        this.grinderForm = {
          name: "",
          grinder_type: "",
          burr_type: "",
          notes: "",
        };
      } else {
        const errorText = await response.text();
        alert("Failed to add grinder: " + errorText);
      }
    },

    async saveBrewer() {
      if (!this.brewerForm.name) {
        alert("Brewer name is required");
        return;
      }
      
      const response = await fetch("/api/brewers", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(this.brewerForm),
      });
      
      if (response.ok) {
        const newBrewer = await response.json();
        
        // Invalidate cache and refresh data in one call
        let freshData = null;
        if (window.ArabicaCache) {
          freshData = await window.ArabicaCache.invalidateAndRefresh();
        }
        
        // Apply the fresh data to update dropdowns
        if (freshData) {
          this.applyData(freshData);
        }
        
        // Select the new brewer
        const brewerSelect = document.querySelector('form select[name="brewer_rkey"]');
        if (brewerSelect && newBrewer.rkey) {
          brewerSelect.value = newBrewer.rkey;
        }
        
        // Close modal and reset form
        this.showBrewerForm = false;
        this.brewerForm = { name: "", brewer_type: "", description: "" };
      } else {
        const errorText = await response.text();
        alert("Failed to add brewer: " + errorText);
      }
    },
  };
}
