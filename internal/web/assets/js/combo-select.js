/**
 * Reusable combo select component for entity selection + inline creation.
 * Replaces the select + "+ New" modal pattern with a typeahead that can
 * search user entities, show community suggestions, and create new entities inline.
 */

document.addEventListener("alpine:init", () => {
  Alpine.data("comboSelect", (config) => ({
    // Config
    entityType: config.entityType || "",
    apiEndpoint: config.apiEndpoint || "",
    suggestEndpoint: config.suggestEndpoint || "",
    inputName: config.inputName || "",
    placeholder: config.placeholder || "Search...",
    formatLabel: config.formatLabel || ((e) => e.name || e.Name || ""),
    formatCreateData: config.formatCreateData || ((name) => ({ name })),
    required: config.required || false,
    passthrough: config.passthrough || false,
    extraFields: config.extraFields || [],

    // State
    query: "",
    selectedRKey: "",
    selectedLabel: "",
    isOpen: false,
    highlightIndex: -1,
    isCreating: false,

    // Inline create form state
    showCreateForm: false,
    createFormData: {},

    // Roaster picker state (for bean inline creation)
    roasterQuery: "",
    roasterResults: [],
    roasterSuggestions: [],
    roasterDropdownOpen: false,
    selectedRoasterRKey: "",
    selectedRoasterLabel: "",
    creatingNewRoaster: false,
    newRoasterName: "",
    newRoasterLocation: "",
    newRoasterWebsite: "",
    _roasterSuggestTimer: null,

    // Results
    userResults: [],
    closedResults: [], // Closed beans (only for bean entity type)
    communityResults: [],

    // All items for flat indexing (for keyboard nav)
    get allItems() {
      const items = [];
      for (const r of this.userResults) {
        items.push({ type: "user", entity: r });
      }
      for (const r of this.closedResults) {
        items.push({ type: "closed", entity: r });
      }
      for (const r of this.communityResults) {
        items.push({ type: "community", suggestion: r });
      }
      if (this.query.trim() && !this.exactMatch && !this.passthrough) {
        items.push({ type: "create", name: this.query.trim() });
      }
      return items;
    },

    // Whether query exactly matches an existing entity
    get exactMatch() {
      const q = this.query.trim().toLowerCase();
      const nameMatch = (e) => (e.name || e.Name || "").toLowerCase() === q;
      return this.userResults.some(nameMatch) || this.closedResults.some(nameMatch) || this.communityResults.some(
        (s) => (s.name || "").toLowerCase() === q,
      );
    },

    init() {
      // Listen for external set events (e.g., from recipe autofill)
      this.$el.addEventListener("combo-set", (e) => {
        if (e.detail.rkey) {
          this.selectedRKey = e.detail.rkey;
          this.selectedLabel = e.detail.label || "";
          this.query = this.selectedLabel;
        } else {
          this.clear();
        }
      });

      // Ensure the user's entities are loaded so typeahead can match them.
      // Some pages (e.g. the oolong steep form) don't otherwise prime the
      // cache, leaving getUserEntities() empty until a refresh happens.
      if (window.ArabicaCache) {
        window.ArabicaCache.getData().catch((err) => {
          console.warn("comboSelect: failed to load user data cache:", err);
        });
      }
    },

    open() {
      this.isOpen = true;
      this.highlightIndex = -1;
      this.search();
    },

    close() {
      // Delay to allow click events on dropdown items
      setTimeout(() => {
        this.isOpen = false;
        // Restore label if user didn't complete selection
        if (this.selectedRKey && this.query !== this.selectedLabel) {
          this.query = this.selectedLabel;
        }
      }, 150);
    },

    // Search: local filtering is instant, remote suggestions are debounced
    _suggestTimer: null,

    async search() {
      const q = this.query.trim().toLowerCase();

      // Instant: filter user's entities from cache
      const entities = this.getUserEntities();
      this.closedResults = [];
      if (q) {
        const matches = entities.filter((e) => {
          const label = this.formatLabel(e).toLowerCase();
          return label.includes(q);
        });
        if (this.entityType === "bean") {
          this.userResults = matches.filter((b) => !b.closed && !b.Closed);
          this.closedResults = matches.filter((b) => b.closed || b.Closed);
        } else {
          this.userResults = matches;
        }
      } else {
        const filtered =
          this.entityType === "bean"
            ? entities.filter((b) => !b.closed && !b.Closed)
            : entities;
        this.userResults = filtered.slice(0, 10);
      }

      this.highlightIndex = -1;
      if (!this.isOpen && this.query) {
        this.isOpen = true;
      }

      // Debounced: fetch community suggestions (400ms after last keystroke)
      clearTimeout(this._suggestTimer);
      if (q.length >= 2 && this.suggestEndpoint) {
        this._suggestTimer = setTimeout(() => {
          this.fetchSuggestions(q);
        }, 400);
      } else {
        this.communityResults = [];
      }
    },

    async fetchSuggestions(q) {
      try {
        const resp = await fetch(
          `${this.suggestEndpoint}?q=${encodeURIComponent(q)}&limit=5`,
          { credentials: "same-origin" },
        );
        if (resp.ok) {
          const data = await resp.json();
          // Re-read entities fresh from cache (may have loaded since search() ran)
          const freshEntities = this.getUserEntities();
          const allNames = new Set(
            freshEntities.map((e) => (e.name || e.Name || "").toLowerCase()),
          );
          this.communityResults = (data || []).filter(
            (s) => !allNames.has((s.name || "").toLowerCase()),
          );
        }
      } catch (e) {
        console.error("Suggestion fetch failed:", e);
      }
    },

    getUserEntities() {
      const dm = window.ArabicaCache?.getCachedData?.() || {};
      switch (this.entityType) {
        case "bean":
          return dm.beans || [];
        case "brewer":
        case "oolongBrewer":
          return dm.brewers || [];
        case "grinder":
          return dm.grinders || [];
        case "recipe":
        case "oolongRecipe":
          return dm.recipes || [];
        case "roaster":
          return dm.roasters || [];
        case "tea":
          return dm.teas || [];
        case "vendor":
          return dm.vendors || [];
        case "cafe":
          return dm.cafes || [];
        case "oolongVessel":
          return dm.vessels || [];
        case "oolongInfuser":
          return dm.infusers || [];
        default:
          return [];
      }
    },

    // Select an existing user entity
    selectEntity(entity) {
      const rkey = entity.rkey || entity.RKey;
      this.selectedRKey = rkey;
      this.selectedLabel = this.formatLabel(entity);
      this.query = this.selectedLabel;
      this.isOpen = false;

      // Dispatch change event for other listeners (e.g., onBrewerChange)
      this.$nextTick(() => {
        this.$dispatch("combo-change", {
          entityType: this.entityType,
          rkey,
          entity,
        });
      });
    },

    // Select a community suggestion — creates the entity first (or passthrough)
    async selectSuggestion(suggestion) {
      // Passthrough mode: use the source record directly without creating a copy
      if (this.passthrough) {
        const parts = (suggestion.source_uri || "").split("/");
        // AT-URI format: at://did/collection/rkey
        const rkey = parts.length >= 5 ? parts[4] : "";
        this.selectedRKey = rkey;
        this.selectedLabel = this.formatLabel(suggestion);
        this.query = this.selectedLabel;
        this.isOpen = false;

        this.$nextTick(() => {
          this.$dispatch("combo-change", {
            entityType: this.entityType,
            rkey,
            suggestion,
          });
        });
        return;
      }

      // Build data from suggestion fields
      const data = this.formatCreateData(suggestion.name, suggestion);
      if (suggestion.source_uri) {
        data.source_ref = suggestion.source_uri;
      }

      // If extraFields configured, show form pre-filled with suggestion data
      if (this.extraFields.length > 0) {
        this.createFormData = { ...data };
        // Ensure all extra fields have a value (even if empty)
        for (const field of this.extraFields) {
          if (!(field.name in this.createFormData)) {
            this.createFormData[field.name] = "";
          }
        }
        this.showCreateForm = true;
        this.isOpen = false;
        return;
      }

      await this._doCreate(data);
    },

    // Create a brand new entity — show detail form if extraFields configured
    createNew() {
      const name = this.query.trim();
      if (!name) return;

      if (this.extraFields.length > 0) {
        this.createFormData = { name };
        for (const field of this.extraFields) {
          this.createFormData[field.name] = "";
        }
        this.showCreateForm = true;
        this.isOpen = false;
        return;
      }

      this._doCreate({ name });
    },

    // Submit the inline create form with all details
    async submitCreateForm() {
      const data = { ...this.createFormData };

      // For beans: handle roaster creation/selection
      if (this.entityType === "bean") {
        if (this.selectedRoasterRKey) {
          data.roaster_rkey = this.selectedRoasterRKey;
        } else if (this.creatingNewRoaster && this.newRoasterName) {
          try {
            const resp = await fetch("/api/roasters", {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              credentials: "same-origin",
              body: JSON.stringify({
                name: this.newRoasterName,
                location: this.newRoasterLocation,
                website: this.newRoasterWebsite,
              }),
            });
            if (!resp.ok) {
              if (resp.status === 401) {
                window.__showSessionExpiredModal();
                return;
              }
              throw new Error("Failed to create roaster");
            }
            const roaster = await resp.json();
            data.roaster_rkey = roaster.rkey || roaster.RKey;
          } catch (e) {
            console.error("Roaster creation failed:", e);
            return;
          }
        }
      }

      // For beans: clear source_ref if the roaster doesn't match the source
      if (this.entityType === "bean" && data.source_ref && data._source_roaster_name) {
        const selected = (this.selectedRoasterLabel || this.newRoasterName || "").toLowerCase().trim();
        const source = data._source_roaster_name.toLowerCase().trim();
        if (selected !== source) {
          delete data.source_ref;
        }
      }
      delete data._source_roaster_name;

      await this._doCreate(data);
      this.showCreateForm = false;
      this.createFormData = {};
      this.resetRoasterPicker();
    },

    // Skip details — create with just the name (and any suggestion data)
    async skipCreateDetails() {
      const data = { name: this.createFormData.name };
      if (this.createFormData.source_ref) {
        data.source_ref = this.createFormData.source_ref;
      }
      // For beans: skip details means no roaster selected, so clear source_ref
      // if the source had a roaster
      if (this.entityType === "bean" && data.source_ref && this.createFormData._source_roaster_name) {
        delete data.source_ref;
      }
      this.showCreateForm = false;
      this.createFormData = {};
      this.resetRoasterPicker();
      await this._doCreate(data);
    },

    cancelCreateForm() {
      this.showCreateForm = false;
      this.createFormData = {};
      this.resetRoasterPicker();
    },

    // Roaster picker methods (for bean inline creation)
    searchRoasters() {
      const q = this.roasterQuery.trim().toLowerCase();
      const roasters =
        (window.ArabicaCache?.getCachedData?.() || {}).roasters || [];
      if (!q) {
        this.roasterResults = roasters.slice(0, 8);
      } else {
        this.roasterResults = roasters.filter((r) =>
          (r.name || r.Name || "").toLowerCase().includes(q),
        );
      }
      this.selectedRoasterRKey = "";
      this.selectedRoasterLabel = "";
      this.creatingNewRoaster = false;
      this.newRoasterName = "";

      // Debounced community suggestions
      clearTimeout(this._roasterSuggestTimer);
      if (q.length >= 2) {
        this._roasterSuggestTimer = setTimeout(() => {
          this.fetchRoasterSuggestions(q);
        }, 400);
      } else {
        this.roasterSuggestions = [];
      }
    },

    async fetchRoasterSuggestions(q) {
      try {
        const resp = await fetch(
          `/api/suggestions/roasters?q=${encodeURIComponent(q)}&limit=5`,
          { credentials: "same-origin" },
        );
        if (resp.ok) {
          const data = await resp.json();
          const roasters =
            (window.ArabicaCache?.getCachedData?.() || {}).roasters || [];
          const ownNames = new Set(
            roasters.map((r) => (r.name || r.Name || "").toLowerCase()),
          );
          this.roasterSuggestions = (data || []).filter(
            (s) => !ownNames.has((s.name || "").toLowerCase()),
          );
        }
      } catch (e) {
        console.error("Roaster suggestion fetch failed:", e);
      }
    },

    selectRoaster(roaster) {
      this.selectedRoasterRKey = roaster.rkey || roaster.RKey;
      this.selectedRoasterLabel = roaster.name || roaster.Name || "";
      this.roasterQuery = this.selectedRoasterLabel;
      this.roasterDropdownOpen = false;
      this.roasterSuggestions = [];
      this.creatingNewRoaster = false;
    },

    selectRoasterSuggestion(suggestion) {
      // Pre-fill new roaster from community suggestion
      this.newRoasterName = suggestion.name || "";
      this.newRoasterLocation =
        (suggestion.fields && suggestion.fields.location) || "";
      this.newRoasterWebsite =
        (suggestion.fields && suggestion.fields.website) || "";
      this.selectedRoasterRKey = "";
      this.roasterQuery = suggestion.name || "";
      this.creatingNewRoaster = true;
      this.roasterDropdownOpen = false;
      this.roasterSuggestions = [];
    },

    startCreateRoaster() {
      this.newRoasterName = this.roasterQuery.trim();
      this.selectedRoasterRKey = "";
      this.creatingNewRoaster = true;
      this.roasterDropdownOpen = false;
    },

    clearRoaster() {
      this.roasterQuery = "";
      this.roasterResults = [];
      this.roasterSuggestions = [];
      this.selectedRoasterRKey = "";
      this.selectedRoasterLabel = "";
      this.creatingNewRoaster = false;
      this.newRoasterName = "";
      this.newRoasterLocation = "";
      this.newRoasterWebsite = "";
      this.roasterDropdownOpen = false;
    },

    resetRoasterPicker() {
      this.clearRoaster();
    },

    // Internal: perform the actual POST to create the entity
    async _doCreate(data) {
      this.isCreating = true;
      try {
        const resp = await fetch(this.apiEndpoint, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "same-origin",
          body: JSON.stringify(data),
        });
        if (!resp.ok) {
          if (resp.status === 401) {
            window.__showSessionExpiredModal();
            return;
          }
          throw new Error(`Create failed: ${resp.status}`);
        }
        const created = await resp.json();
        const rkey = created.rkey || created.RKey;

        this.selectedRKey = rkey;
        this.selectedLabel = data.name;
        this.query = data.name;
        this.isOpen = false;

        if (window.ArabicaCache) {
          window.ArabicaCache.invalidateAndRefresh();
        }

        this.$nextTick(() => {
          this.$dispatch("combo-change", {
            entityType: this.entityType,
            rkey,
          });
        });
      } catch (e) {
        console.error("Failed to create entity:", e);
      } finally {
        this.isCreating = false;
      }
    },

    // Keyboard navigation
    moveDown() {
      if (this.highlightIndex < this.allItems.length - 1) {
        this.highlightIndex++;
      }
    },

    moveUp() {
      if (this.highlightIndex > 0) {
        this.highlightIndex--;
      }
    },

    selectHighlighted() {
      const item = this.allItems[this.highlightIndex];
      if (!item) return;
      if (item.type === "user") this.selectEntity(item.entity);
      else if (item.type === "community")
        this.selectSuggestion(item.suggestion);
      else if (item.type === "create") this.createNew();
    },

    // Clear selection
    clear() {
      this.selectedRKey = "";
      this.selectedLabel = "";
      this.query = "";
      this.showCreateForm = false;
      this.createFormData = {};
      this.$dispatch("combo-change", {
        entityType: this.entityType,
        rkey: "",
      });
    },
  }));
});
