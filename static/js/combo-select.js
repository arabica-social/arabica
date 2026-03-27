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
          return dm.brewers || [];
        case "grinder":
          return dm.grinders || [];
        case "recipe":
          return dm.recipes || [];
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
      await this._doCreate({ ...this.createFormData });
      this.showCreateForm = false;
      this.createFormData = {};
    },

    // Skip details — create with just the name (and any suggestion data)
    async skipCreateDetails() {
      const data = { name: this.createFormData.name };
      if (this.createFormData.source_ref) {
        data.source_ref = this.createFormData.source_ref;
      }
      this.showCreateForm = false;
      this.createFormData = {};
      await this._doCreate(data);
    },

    cancelCreateForm() {
      this.showCreateForm = false;
      this.createFormData = {};
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
        if (!resp.ok) throw new Error(`Create failed: ${resp.status}`);
        const created = await resp.json();
        const rkey = created.rkey || created.RKey;

        this.selectedRKey = rkey;
        this.selectedLabel = data.name;
        this.query = data.name;
        this.isOpen = false;

        if (window.ArabicaCache) {
          window.ArabicaCache.invalidateCache();
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
