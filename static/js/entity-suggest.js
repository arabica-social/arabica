// entitySuggest - Alpine.js component for typeahead suggestions in entity creation modals
// Usage: x-data="entitySuggest('/api/suggestions/roasters')"
function entitySuggest(endpoint) {
  return {
    query: '',
    suggestions: [],
    showSuggestions: false,
    sourceRef: '',
    originalName: '',

    async search() {
      if (this.query.length < 2) {
        this.suggestions = [];
        this.showSuggestions = false;
        return;
      }

      // Clear sourceRef if name changed from the selected suggestion
      if (this.originalName && this.query.toLowerCase() !== this.originalName.toLowerCase()) {
        this.sourceRef = '';
        this.originalName = '';
      }

      try {
        const resp = await fetch(endpoint + '?q=' + encodeURIComponent(this.query) + '&limit=10');
        if (resp.ok) {
          this.suggestions = await resp.json();
          this.showSuggestions = this.suggestions.length > 0;
        }
      } catch (e) {
        // Silently fail - suggestions are optional
      }
    },

    // Entity-specific selection methods that populate the right form fields

    selectRoasterSuggestion(s) {
      this.query = s.name;
      this.sourceRef = s.source_uri;
      this.originalName = s.name;
      this.showSuggestions = false;

      const form = this.$el.closest('form');
      if (s.fields.location) this._setInput(form, 'location', s.fields.location);
      if (s.fields.website) this._setInput(form, 'website', s.fields.website);
    },

    selectGrinderSuggestion(s) {
      this.query = s.name;
      this.sourceRef = s.source_uri;
      this.originalName = s.name;
      this.showSuggestions = false;

      const form = this.$el.closest('form');
      if (s.fields.grinderType) this._setSelect(form, 'grinder_type', s.fields.grinderType);
      if (s.fields.burrType) this._setSelect(form, 'burr_type', s.fields.burrType);
    },

    selectBrewerSuggestion(s) {
      this.query = s.name;
      this.sourceRef = s.source_uri;
      this.originalName = s.name;
      this.showSuggestions = false;

      const form = this.$el.closest('form');
      if (s.fields.brewerType) this._setInput(form, 'brewer_type', s.fields.brewerType);
    },

    selectBeanSuggestion(s) {
      this.query = s.name;
      this.sourceRef = s.source_uri;
      this.originalName = s.name;
      this.showSuggestions = false;

      const form = this.$el.closest('form');
      if (s.fields.origin) this._setInput(form, 'origin', s.fields.origin);
      if (s.fields.roastLevel) this._setSelect(form, 'roast_level', s.fields.roastLevel);
      if (s.fields.process) this._setInput(form, 'process', s.fields.process);
    },

    // Helper: set value on an input/textarea by name
    _setInput(form, name, value) {
      const el = form.querySelector('[name="' + name + '"]');
      if (el) {
        el.value = value;
        el.dispatchEvent(new Event('input', { bubbles: true }));
      }
    },

    // Helper: set value on a select by name
    _setSelect(form, name, value) {
      const el = form.querySelector('[name="' + name + '"]');
      if (!el) return;
      // Try exact match first, then case-insensitive
      for (const opt of el.options) {
        if (opt.value === value || opt.value.toLowerCase() === value.toLowerCase()) {
          el.value = opt.value;
          el.dispatchEvent(new Event('change', { bubbles: true }));
          return;
        }
      }
    },
  };
}
