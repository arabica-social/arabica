// @ts-check
// Typeahead suggestions for entity create modals.
// Petite-vue factory: `v-scope="entitySuggest('/api/suggestions/X')"`.

/**
 * @typedef {object} Suggestion
 * @property {string} name
 * @property {string} source_uri
 * @property {Record<string, string>} fields
 * @property {number} count
 *
 * @typedef {{
 *   query: string,
 *   suggestions: Suggestion[],
 *   showSuggestions: boolean,
 *   sourceRef: string,
 *   originalName: string,
 *   $root: HTMLElement | null,
 *   setup: ($el: HTMLElement) => void,
 *   onInput: () => void,
 *   onBlur: () => void,
 *   onFocus: () => void,
 *   search: () => Promise<void>,
 *   selectRoasterSuggestion: (s: Suggestion) => void,
 *   selectGrinderSuggestion: (s: Suggestion) => void,
 *   selectBrewerSuggestion: (s: Suggestion) => void,
 *   selectBeanSuggestion: (s: Suggestion) => void,
 * }} EntitySuggestScope
 */

/**
 * @param {string} endpoint
 * @returns {EntitySuggestScope}
 */
function entitySuggest(endpoint) {
  /** @type {number | undefined} */
  let searchTimer;
  /** @type {number | undefined} */
  let blurTimer;

  return {
    query: "",
    suggestions: [],
    showSuggestions: false,
    sourceRef: "",
    originalName: "",
    $root: null,

    setup($el) {
      this.$root = $el;
    },

    onInput() {
      window.clearTimeout(searchTimer);
      searchTimer = window.setTimeout(() => this.search(), 300);
    },
    onBlur() {
      window.clearTimeout(blurTimer);
      blurTimer = window.setTimeout(() => {
        this.showSuggestions = false;
      }, 200);
    },
    onFocus() {
      if (this.suggestions.length > 0) this.showSuggestions = true;
    },

    async search() {
      if (this.query.length < 2) {
        this.suggestions = [];
        this.showSuggestions = false;
        return;
      }
      // Clear sourceRef if name changed from the selected suggestion.
      if (
        this.originalName &&
        this.query.toLowerCase() !== this.originalName.toLowerCase()
      ) {
        this.sourceRef = "";
        this.originalName = "";
      }
      try {
        const resp = await fetch(
          endpoint + "?q=" + encodeURIComponent(this.query) + "&limit=10",
        );
        if (resp.ok) {
          this.suggestions = await resp.json();
          this.showSuggestions = this.suggestions.length > 0;
        }
      } catch (e) {
        // suggestions are optional
      }
    },

    selectRoasterSuggestion(s) {
      _applyBase(this, s);
      const form = _form(this);
      if (!form) return;
      if (s.fields.location) _setInput(form, "location", s.fields.location);
      if (s.fields.website) _setInput(form, "website", s.fields.website);
    },

    selectGrinderSuggestion(s) {
      _applyBase(this, s);
      const form = _form(this);
      if (!form) return;
      if (s.fields.grinderType)
        _setSelect(form, "grinder_type", s.fields.grinderType);
      if (s.fields.burrType) _setSelect(form, "burr_type", s.fields.burrType);
      if (s.fields.link) _setInput(form, "link", s.fields.link);
    },

    selectBrewerSuggestion(s) {
      _applyBase(this, s);
      const form = _form(this);
      if (!form) return;
      if (s.fields.brewerType)
        _setInput(form, "brewer_type", s.fields.brewerType);
      if (s.fields.link) _setInput(form, "link", s.fields.link);
    },

    selectBeanSuggestion(s) {
      _applyBase(this, s);
      const form = _form(this);
      if (!form) return;
      if (s.fields.origin) _setInput(form, "origin", s.fields.origin);
      if (s.fields.roastLevel)
        _setSelect(form, "roast_level", s.fields.roastLevel);
      if (s.fields.process) _setInput(form, "process", s.fields.process);
      if (s.fields.link) _setInput(form, "link", s.fields.link);
    },
  };
}

/**
 * @param {EntitySuggestScope} scope
 * @param {Suggestion} s
 */
function _applyBase(scope, s) {
  scope.query = s.name;
  scope.sourceRef = s.source_uri;
  scope.originalName = s.name;
  scope.showSuggestions = false;
}

/**
 * @param {EntitySuggestScope} scope
 * @returns {HTMLFormElement | null}
 */
function _form(scope) {
  return scope.$root ? scope.$root.closest("form") : null;
}

/**
 * @param {HTMLFormElement} form
 * @param {string} name
 * @param {string} value
 */
function _setInput(form, name, value) {
  const el = /** @type {HTMLInputElement | null} */ (
    form.querySelector('[name="' + name + '"]')
  );
  if (el) {
    el.value = value;
    el.dispatchEvent(new Event("input", { bubbles: true }));
  }
}

/**
 * @param {HTMLFormElement} form
 * @param {string} name
 * @param {string} value
 */
function _setSelect(form, name, value) {
  const el = /** @type {HTMLSelectElement | null} */ (
    form.querySelector('[name="' + name + '"]')
  );
  if (!el) return;
  for (const opt of Array.from(el.options)) {
    if (
      opt.value === value ||
      opt.value.toLowerCase() === value.toLowerCase()
    ) {
      el.value = opt.value;
      el.dispatchEvent(new Event("change", { bubbles: true }));
      return;
    }
  }
}

/** @type {any} */ (window).entitySuggest = entitySuggest;
