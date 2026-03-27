/**
 * Alpine.js component for the manage page
 * Handles CRUD operations for beans, roasters, grinders, and brewers
 * Uses shared entity-manager module to eliminate duplication
 */
function managePage() {
  return {
    tab: localStorage.getItem("manageTab") || "brews",
    activeTab: "brews", // Always default to brews tab on profile

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

      // Initialize cache in background
      if (window.ArabicaCache) {
        window.ArabicaCache.init();
      }

      // Initialize entity managers
      this.initEntityManagers();

      // Check for incomplete entity nudge from brew save
      this.showIncompleteNudge();
    },

    showIncompleteNudge() {
      try {
        const raw = sessionStorage.getItem("incompleteNudge");
        if (!raw) return;
        sessionStorage.removeItem("incompleteNudge");
        const nudge = JSON.parse(raw);
        if (!nudge.name || !nudge.missing) return;

        // Create toast element
        const toast = document.createElement("div");
        toast.className =
          "fixed bottom-6 left-1/2 -translate-x-1/2 z-50 max-w-md w-full mx-4 p-4 rounded-lg shadow-lg flex items-center gap-3";
        toast.style.cssText =
          "background: var(--card-bg, #fff); border: 1px solid var(--surface-border, #d4c4a8); color: var(--text-primary, #3e2723);";
        toast.innerHTML = `
          <div class="flex-1 text-sm">
            <strong>${nudge.name}</strong> is missing ${nudge.missing}
          </div>
          <button class="text-sm font-medium hover:opacity-80 whitespace-nowrap" style="color: var(--accent-primary, #5d4037);"
            onclick="this.closest('div').remove(); document.querySelector('#modal-container') && htmx.ajax('GET', '/api/modals/${nudge.entity_type}/${nudge.rkey}', {target: '#modal-container', swap: 'innerHTML'});">
            Complete
          </button>
          <button class="text-brown-400 hover:text-brown-600" onclick="this.closest('div').remove();">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"/></svg>
          </button>
        `;
        document.body.appendChild(toast);

        // Auto-dismiss after 10 seconds
        setTimeout(() => toast.remove(), 10000);
      } catch (_) {
        // Ignore errors
      }
    },

    initEntityManagers() {
      // Bean entity manager
      this.beanManager = window.createEntityManager({
        entityType: "bean",
        apiEndpoint: "/api/beans",
        dialogId: "bean-modal",
        defaultFormData: {
          name: "",
          origin: "",
          roast_level: "",
          process: "",
          description: "",
          roaster_rkey: "",
          closed: false,
        },
        validate: (data) => {
          if (!data.name || !data.origin) {
            return "Name and Origin are required";
          }
          return null;
        },
        onSuccess: () => {
          // Reload the manage partial via HTMX by dispatching a custom event
          document.body.dispatchEvent(new CustomEvent('refreshManage'));
        },
      });

      // Roaster entity manager
      this.roasterManager = window.createEntityManager({
        entityType: "roaster",
        apiEndpoint: "/api/roasters",
        dialogId: "roaster-modal",
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
        onSuccess: () => {
          // Reload the manage partial via HTMX by dispatching a custom event
          document.body.dispatchEvent(new CustomEvent('refreshManage'));
        },
      });

      // Grinder entity manager
      this.grinderManager = window.createEntityManager({
        entityType: "grinder",
        apiEndpoint: "/api/grinders",
        dialogId: "grinder-modal",
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
        onSuccess: () => {
          // Reload the manage partial via HTMX by dispatching a custom event
          document.body.dispatchEvent(new CustomEvent('refreshManage'));
        },
      });

      // Brewer entity manager
      this.brewerManager = window.createEntityManager({
        entityType: "brewer",
        apiEndpoint: "/api/brewers",
        dialogId: "brewer-modal",
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
        onSuccess: () => {
          // Reload the manage partial via HTMX by dispatching a custom event
          document.body.dispatchEvent(new CustomEvent('refreshManage'));
        },
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
    editBean(
      rkey,
      name,
      origin,
      roast_level,
      process,
      description,
      roaster_rkey,
      closed,
    ) {
      this.beanManager.openEdit(rkey, {
        name,
        origin,
        roast_level,
        process,
        description,
        roaster_rkey: roaster_rkey || "",
        closed: closed || false,
      });
    },

    editRoaster(rkey, name, location, website) {
      this.roasterManager.openEdit(rkey, { name, location, website });
    },

    editGrinder(rkey, name, grinder_type, burr_type, notes) {
      this.grinderManager.openEdit(rkey, {
        name,
        grinder_type,
        burr_type,
        notes,
      });
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
      const success = await this.beanManager.delete(rkey);
      // Explicitly trigger refresh after successful delete
      if (success) {
        document.body.dispatchEvent(new CustomEvent('refreshManage'));
      }
    },

    async deleteRoaster(rkey) {
      const success = await this.roasterManager.delete(rkey);
      // Explicitly trigger refresh after successful delete
      if (success) {
        document.body.dispatchEvent(new CustomEvent('refreshManage'));
      }
    },

    async deleteGrinder(rkey) {
      const success = await this.grinderManager.delete(rkey);
      // Explicitly trigger refresh after successful delete
      if (success) {
        document.body.dispatchEvent(new CustomEvent('refreshManage'));
      }
    },

    async deleteBrewer(rkey) {
      const success = await this.brewerManager.delete(rkey);
      // Explicitly trigger refresh after successful delete
      if (success) {
        document.body.dispatchEvent(new CustomEvent('refreshManage'));
      }
    },
  };
}
