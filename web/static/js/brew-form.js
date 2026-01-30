/**
 * Alpine.js component for the brew form
 * Manages pours, new entity modals, and form state
 * Uses shared entity-manager and dropdown-manager modules
 */

// Wait for Alpine to be available and register the component
document.addEventListener("alpine:init", () => {
  Alpine.data("brewForm", () => ({
    // Brew form specific
    rating: 5,
    pours: [],

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
      await this.dropdownManager.loadDropdownData();
      this.dropdownManager.populateDropdowns();
    },

    initEntityManagers() {
      // Bean entity manager
      this.beanManager = window.createEntityManager({
        entityType: "bean",
        apiEndpoint: "/api/beans",
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

          // Select the new bean
          const beanSelect = document.querySelector(
            'form select[name="bean_rkey"]',
          );
          if (beanSelect && newBean.rkey) {
            beanSelect.value = newBean.rkey;
          }
        },
      });

      // Grinder entity manager
      this.grinderManager = window.createEntityManager({
        entityType: "grinder",
        apiEndpoint: "/api/grinders",
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

          // Select the new grinder
          const grinderSelect = document.querySelector(
            'form select[name="grinder_rkey"]',
          );
          if (grinderSelect && newGrinder.rkey) {
            grinderSelect.value = newGrinder.rkey;
          }
        },
      });

      // Brewer entity manager
      this.brewerManager = window.createEntityManager({
        entityType: "brewer",
        apiEndpoint: "/api/brewers",
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

          // Select the new brewer
          const brewerSelect = document.querySelector(
            'form select[name="brewer_rkey"]',
          );
          if (brewerSelect && newBrewer.rkey) {
            brewerSelect.value = newBrewer.rkey;
          }
        },
      });
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
  }));
});
