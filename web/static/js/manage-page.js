/**
 * Alpine.js component for the manage page
 * Handles CRUD operations for beans, roasters, grinders, and brewers
 * Uses shared entity-manager module to eliminate duplication
 */
function managePage() {
  return {
    tab: localStorage.getItem("manageTab") || "beans",
    activeTab: localStorage.getItem("profileTab") || "brews",

    // Entity managers for each entity type
    beanManager: null,
    roasterManager: null,
    grinderManager: null,
    brewerManager: null,

    init() {
      // Watch tab changes and persist to localStorage
      this.$watch("tab", (value) => {
        localStorage.setItem("manageTab", value);
      });

      this.$watch("activeTab", (value) => {
        localStorage.setItem("profileTab", value);
      });

      // Initialize cache in background
      if (window.ArabicaCache) {
        window.ArabicaCache.init();
      }

      // Initialize entity managers
      this.initEntityManagers();
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
            return "Name and Origin are required";
          }
          return null;
        },
        reloadOnSuccess: true,
      });

      // Roaster entity manager
      this.roasterManager = window.createEntityManager({
        entityType: "roaster",
        apiEndpoint: "/api/roasters",
        defaultFormData: {
          name: "",
          location: "",
          website: "",
        },
        validate: (data) => {
          if (!data.name) {
            return "Name is required";
          }
          return null;
        },
        reloadOnSuccess: true,
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
          if (!data.name || !data.grinder_type) {
            return "Name and Grinder Type are required";
          }
          return null;
        },
        reloadOnSuccess: true,
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
            return "Name is required";
          }
          return null;
        },
        reloadOnSuccess: true,
      });
    },

    // Expose entity manager modal state to Alpine
    get showBeanForm() {
      return this.beanManager?.showForm || false;
    },
    set showBeanForm(value) {
      if (this.beanManager) this.beanManager.showForm = value;
    },

    get showRoasterForm() {
      return this.roasterManager?.showForm || false;
    },
    set showRoasterForm(value) {
      if (this.roasterManager) this.roasterManager.showForm = value;
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

    get editingRoaster() {
      return this.roasterManager?.editingId !== null;
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

    get roasterForm() {
      return this.roasterManager?.formData || {};
    },
    set roasterForm(value) {
      if (this.roasterManager) this.roasterManager.formData = value;
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

    // Edit methods - populate form and open modal
    editBean(rkey, name, origin, roast_level, process, description, roaster_rkey) {
      this.beanManager.openEdit(rkey, {
        name,
        origin,
        roast_level,
        process,
        description,
        roaster_rkey: roaster_rkey || "",
      });
    },

    editRoaster(rkey, name, location, website) {
      this.roasterManager.openEdit(rkey, { name, location, website });
    },

    editGrinder(rkey, name, grinder_type, burr_type, notes) {
      this.grinderManager.openEdit(rkey, { name, grinder_type, burr_type, notes });
    },

    editBrewer(rkey, name, brewer_type, description) {
      this.brewerManager.openEdit(rkey, { name, brewer_type, description });
    },

    editBrewerFromRow(row) {
      const rkey = row.dataset.rkey;
      const name = row.dataset.name;
      const brewer_type = row.dataset.brewerType || "";
      const description = row.dataset.description || "";
      this.editBrewer(rkey, name, brewer_type, description);
    },

    // Delegate save methods to entity managers
    async saveBean() {
      await this.beanManager.save();
    },

    async saveRoaster() {
      await this.roasterManager.save();
    },

    async saveGrinder() {
      await this.grinderManager.save();
    },

    async saveBrewer() {
      await this.brewerManager.save();
    },

    // Delegate delete methods to entity managers
    async deleteBean(rkey) {
      await this.beanManager.delete(rkey);
    },

    async deleteRoaster(rkey) {
      await this.roasterManager.delete(rkey);
    },

    async deleteGrinder(rkey) {
      await this.grinderManager.delete(rkey);
    },

    async deleteBrewer(rkey) {
      await this.brewerManager.delete(rkey);
    },
  };
}
