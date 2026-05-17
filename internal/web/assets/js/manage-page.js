// @ts-check
// Petite-vue factory for the manage / my-coffee / profile pages.
// Owns the active tab + the four entity managers (bean/roaster/grinder/brewer)
// that profile.templ's modals bind to.

function managePage() {
  return {
    // Tab state — persisted to localStorage via the setter so we don't need
    // a reactive watcher.
    _tab: (function () {
      try {
        return localStorage.getItem("manageTab") || "brews";
      } catch (e) {
        return "brews";
      }
    })(),
    get tab() {
      return this._tab;
    },
    set tab(value) {
      this._tab = value;
      try {
        localStorage.setItem("manageTab", value);
      } catch (e) {}
    },
    activeTab: "brews", // Always default to brews tab on profile

    /** @type {any} */ beanManager: null,
    /** @type {any} */ roasterManager: null,
    /** @type {any} */ grinderManager: null,
    /** @type {any} */ brewerManager: null,

    setup() {
      const cache = /** @type {any} */ (window).ArabicaCache;
      if (cache) cache.init();
      this.initEntityManagers();
      this.showIncompleteNudge();
    },

    showIncompleteNudge() {
      try {
        const raw = sessionStorage.getItem("incompleteNudge");
        if (!raw) return;
        sessionStorage.removeItem("incompleteNudge");
        const nudge = JSON.parse(raw);
        if (!nudge.name || !nudge.missing) return;

        const toast = document.createElement("div");
        toast.className = "nudge-toast";

        const body = document.createElement("div");
        body.className = "flex-1 text-sm";
        const strong = document.createElement("strong");
        strong.textContent = nudge.name;
        body.appendChild(strong);
        body.appendChild(document.createTextNode(" is missing " + nudge.missing));

        const complete = document.createElement("button");
        complete.className =
          "text-sm font-medium hover:opacity-80 whitespace-nowrap";
        complete.style.color = "var(--accent-primary, #5d4037)";
        complete.textContent = "Complete";
        complete.addEventListener("click", () => {
          toast.remove();
          const slot = document.querySelector("#modal-container");
          const htmx = /** @type {any} */ (window).htmx;
          if (slot && htmx) {
            htmx.ajax(
              "GET",
              `/api/modals/${nudge.entity_type}/${nudge.rkey}`,
              { target: "#modal-container", swap: "innerHTML" },
            );
          }
        });

        const dismiss = document.createElement("button");
        dismiss.className = "text-brown-400 hover:text-brown-600";
        dismiss.innerHTML =
          '<svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"/></svg>';
        dismiss.addEventListener("click", () => toast.remove());

        toast.append(body, complete, dismiss);
        document.body.appendChild(toast);
        setTimeout(() => toast.remove(), 10000);
      } catch (_) {
        // ignore
      }
    },

    initEntityManagers() {
      const w = /** @type {any} */ (window);
      const refresh = () =>
        document.body.dispatchEvent(new CustomEvent("refreshManage"));

      this.beanManager = w.createEntityManager({
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
        validate: (data) =>
          !data.name || !data.origin ? "Name and Origin are required" : null,
        onSuccess: refresh,
      });

      this.roasterManager = w.createEntityManager({
        entityType: "roaster",
        apiEndpoint: "/api/roasters",
        dialogId: "roaster-modal",
        defaultFormData: { name: "", location: "", website: "" },
        validate: (data) => (!data.name ? "Name is required" : null),
        onSuccess: refresh,
      });

      this.grinderManager = w.createEntityManager({
        entityType: "grinder",
        apiEndpoint: "/api/grinders",
        dialogId: "grinder-modal",
        defaultFormData: {
          name: "",
          grinder_type: "",
          burr_type: "",
          notes: "",
        },
        validate: (data) =>
          !data.name || !data.grinder_type
            ? "Name and Grinder Type are required"
            : null,
        onSuccess: refresh,
      });

      this.brewerManager = w.createEntityManager({
        entityType: "brewer",
        apiEndpoint: "/api/brewers",
        dialogId: "brewer-modal",
        defaultFormData: { name: "", brewer_type: "", description: "" },
        validate: (data) => (!data.name ? "Name is required" : null),
        onSuccess: refresh,
      });
    },

    // Bridges between the entity-manager state and template bindings.
    // Each manager has a `formData` object whose mutations need to be
    // reactive in petite-vue. petite-vue's deep reactivity wraps nested
    // accessed values, so reading via these getters gives a reactive proxy.
    get showBeanForm() {
      return this.beanManager?.showForm || false;
    },
    set showBeanForm(v) {
      if (this.beanManager) this.beanManager.showForm = v;
    },
    get showRoasterForm() {
      return this.roasterManager?.showForm || false;
    },
    set showRoasterForm(v) {
      if (this.roasterManager) this.roasterManager.showForm = v;
    },
    get showGrinderForm() {
      return this.grinderManager?.showForm || false;
    },
    set showGrinderForm(v) {
      if (this.grinderManager) this.grinderManager.showForm = v;
    },
    get showBrewerForm() {
      return this.brewerManager?.showForm || false;
    },
    set showBrewerForm(v) {
      if (this.brewerManager) this.brewerManager.showForm = v;
    },

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

    get beanForm() {
      return this.beanManager?.formData || {};
    },
    set beanForm(v) {
      if (this.beanManager) this.beanManager.formData = v;
    },
    get roasterForm() {
      return this.roasterManager?.formData || {};
    },
    set roasterForm(v) {
      if (this.roasterManager) this.roasterManager.formData = v;
    },
    get grinderForm() {
      return this.grinderManager?.formData || {};
    },
    set grinderForm(v) {
      if (this.grinderManager) this.grinderManager.formData = v;
    },
    get brewerForm() {
      return this.brewerManager?.formData || {};
    },
    set brewerForm(v) {
      if (this.brewerManager) this.brewerManager.formData = v;
    },

    editBean(rkey, name, origin, roast_level, process, description, roaster_rkey, closed) {
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
      this.editBrewer(
        row.dataset.rkey,
        row.dataset.name,
        row.dataset.brewerType || "",
        row.dataset.description || "",
      );
    },

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

    async deleteBean(rkey) {
      const ok = await this.beanManager.delete(rkey);
      if (ok) document.body.dispatchEvent(new CustomEvent("refreshManage"));
    },
    async deleteRoaster(rkey) {
      const ok = await this.roasterManager.delete(rkey);
      if (ok) document.body.dispatchEvent(new CustomEvent("refreshManage"));
    },
    async deleteGrinder(rkey) {
      const ok = await this.grinderManager.delete(rkey);
      if (ok) document.body.dispatchEvent(new CustomEvent("refreshManage"));
    },
    async deleteBrewer(rkey) {
      const ok = await this.brewerManager.delete(rkey);
      if (ok) document.body.dispatchEvent(new CustomEvent("refreshManage"));
    },
  };
}

/** @type {any} */ (window).managePage = managePage;
