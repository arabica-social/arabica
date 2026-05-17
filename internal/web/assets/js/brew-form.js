// @ts-check
// Petite-vue factory for the brew form. Manages pours, recipe autofill,
// brewer category inference, and cross-component combo-select coordination.

// Capture incomplete entity nudge from brew save response before HTMX redirect.
// htmx:afterRequest fires after any HTMX request completes, including redirects.
document.addEventListener("htmx:afterRequest", (e) => {
  const xhr = /** @type {any} */ (e).detail.xhr;
  if (xhr) {
    const nudge = xhr.getResponseHeader("X-Incomplete-Nudge");
    if (nudge) {
      try {
        sessionStorage.setItem("incompleteNudge", nudge);
      } catch (_) {
        // sessionStorage may be unavailable
      }
    }
  }
});

/**
 * @typedef {{ rkey?: string, RKey?: string, name?: string, Name?: string, brewer_type?: string, BrewerType?: string, author_did?: string }} Recipe
 * @typedef {{ water?: number | string, time?: number | string, water_amount?: number, time_seconds?: number }} Pour
 */

function brewForm() {
  return {
    // Brew form specific
    rating: 5,
    /** @type {Pour[]} */ pours: [],
    brewerCategory: "", // 'pourover' | 'espresso' | 'immersion' | ''

    // Mode state
    formMode: "recipe",
    recipeSummaryExpanded: false,
    /** @type {any} */ activeRecipe: null,
    showPours: false,
    isEditing: false,

    /** @type {Recipe[]} */ recipes: [],
    recipeOwnerDID: "",

    /** @type {any} */ dropdownManager: null,
    /** @type {HTMLElement | null} */ $root: null,

    async setup($el) {
      this.$root = $el;
      this.dropdownManager = /** @type {any} */ (window).createDropdownManager();

      const formEl = $el.querySelector("form");
      this.isEditing = formEl?.hasAttribute("data-editing") || false;
      const recipeRKey = formEl?.getAttribute("data-recipe-rkey") || "";
      this.recipeOwnerDID = formEl?.getAttribute("data-recipe-owner") || "";

      // Load existing pours if editing.
      const poursData = formEl?.getAttribute("data-pours");
      if (poursData) {
        try {
          this.pours = JSON.parse(poursData);
        } catch (e) {
          console.error("Failed to parse pours data:", e);
          this.pours = [];
        }
      }

      // Seed rating from form data attribute. We can't rely on the slider's
      // own `value=` attribute because petite-vue's `v-model` writes the
      // scope default to the DOM before any @vue:mounted hook can read it.
      const ratingAttr = formEl?.getAttribute("data-rating");
      if (ratingAttr) {
        const r = Number(ratingAttr);
        if (!Number.isNaN(r)) this.rating = r;
      }

      this.formMode = "recipe";

      await this.dropdownManager.loadDropdownData();
      this.dropdownManager.populateDropdowns();

      this.recipes = this.dropdownManager.recipes || [];
      const cache = /** @type {any} */ (window).ArabicaCache;
      if (cache) {
        cache.addListener(
          /** @param {any} data */ (data) => {
            this.recipes = data.recipes || [];
          },
        );
      }

      // Refresh dropdown cache when entity-helpers signals one was created.
      document.body.addEventListener("refresh-dropdowns", async () => {
        if (this.dropdownManager?.invalidateAndRefresh) {
          await this.dropdownManager.invalidateAndRefresh();
        }
      });

      // Listen for combo-select changes (bubbled from inner v-scope combos).
      $el.addEventListener("combo-change", (e) => {
        const detail = /** @type {any} */ (e).detail;
        if (detail.entityType === "brewer") {
          const bt =
            detail.entity?.brewer_type || detail.entity?.BrewerType || "";
          this.brewerCategory = this.normalizeBrewerCategory(bt);
          if (this.brewerCategory === "pourover") this.showPours = true;
        }
        if (detail.entityType === "recipe") {
          if (detail.suggestion) {
            const parts = (detail.suggestion.source_uri || "").split("/");
            this.recipeOwnerDID = parts.length >= 3 ? parts[2] : "";
          } else {
            this.recipeOwnerDID = "";
          }
          this.applyRecipe(detail.rkey);
        }
      });

      // Auto-apply recipe if rkey present (e.g., from URL param).
      if (recipeRKey) {
        const recipeCombo = this._combo("recipe");
        if (recipeCombo) {
          const match = this.recipes.find(
            (r) => (r.rkey || r.RKey) === recipeRKey,
          );
          const recipeName = match ? match.name || match.Name || "" : "";
          recipeCombo.dispatchEvent(
            new CustomEvent("combo-set", {
              detail: { rkey: recipeRKey, label: recipeName },
              bubbles: false,
            }),
          );
        }
        await this.applyRecipe(recipeRKey);
      }

      this.updatePoursVisibility();
    },

    /**
     * Locate a combo-select wrapper by its entity type. Each wrapper carries
     * a `data-combo-entity-type` attribute set at the call site.
     * @param {string} type
     * @returns {Element | null}
     */
    _combo(type) {
      if (!this.$root) return null;
      return this.$root.querySelector(`[data-combo-entity-type="${type}"]`);
    },

    updatePoursVisibility() {
      if (this.pours.length > 0) {
        this.showPours = true;
        return;
      }
      if (this.activeRecipe?.pours?.length > 0) {
        this.showPours = true;
        return;
      }
    },

    togglePours() {
      this.showPours = !this.showPours;
      if (this.showPours && this.pours.length === 0) this.addPour();
    },

    onBrewerChange(rkey) {
      const brewerType = this.dropdownManager?.getBrewerType(rkey) || "";
      this.brewerCategory = this.normalizeBrewerCategory(brewerType);
      if (this.brewerCategory === "pourover") this.showPours = true;
    },

    normalizeBrewerCategory(raw) {
      if (!raw) return "";
      const lower = raw.toLowerCase().trim();
      if (
        [
          "pourover",
          "espresso",
          "immersion",
          "mokapot",
          "coldbrew",
          "cupping",
          "other",
        ].includes(lower)
      )
        return lower;
      if (["pour-over", "pour over", "dripper"].includes(lower))
        return "pourover";
      if (
        [
          "espresso machine",
          "lever espresso",
          "lever espresso machine",
        ].includes(lower)
      )
        return "espresso";
      if (
        [
          "french press",
          "aeropress",
          "siphon",
          "clever",
          "clever dripper",
        ].includes(lower)
      )
        return "immersion";
      return "";
    },

    get recipeSummaryText() {
      if (!this.activeRecipe) return "";
      const parts = [];
      if (this.activeRecipe.coffee_amount > 0) {
        parts.push(Math.round(this.activeRecipe.coffee_amount) + "g coffee");
      }
      if (this.activeRecipe.water_amount > 0) {
        parts.push(Math.round(this.activeRecipe.water_amount) + "g water");
      }
      if (this.activeRecipe.brewer_rkey) {
        const brewer = (this.dropdownManager?.brewers || []).find(
          /** @param {any} b */ (b) =>
            (b.rkey || b.RKey) === this.activeRecipe.brewer_rkey,
        );
        if (brewer) parts.push(brewer.Name || brewer.name);
      }
      if (this.activeRecipe.pours && this.activeRecipe.pours.length > 0) {
        parts.push(this.activeRecipe.pours.length + " pours");
      }
      return parts.join(" · ");
    },

    async applyRecipe(rkey) {
      if (!this.$root) return;
      const form = this.$root.querySelector("form");
      if (!form) return;

      if (!rkey) {
        this.clearRecipeFields(form);
        this.activeRecipe = null;
        this.recipeOwnerDID = "";
        this.recipeSummaryExpanded = false;
        this.updatePoursVisibility();
        return;
      }

      const cachedRecipe = this.recipes.find(
        (r) => (r.rkey || r.RKey) === rkey,
      );
      if (cachedRecipe && cachedRecipe.author_did) {
        this.recipeOwnerDID = cachedRecipe.author_did;
      }

      try {
        let url = `/api/recipes/${rkey}`;
        if (this.recipeOwnerDID) {
          url += `?owner=${encodeURIComponent(this.recipeOwnerDID)}`;
        }
        const resp = await fetch(url, { credentials: "same-origin" });
        if (!resp.ok) return;
        const recipe = await resp.json();

        this.activeRecipe = recipe;
        this.recipeSummaryExpanded = false;
        if (recipe.author_did) this.recipeOwnerDID = recipe.author_did;

        this.setFormField(
          form,
          "coffee_amount",
          recipe.coffee_amount > 0 ? Math.round(recipe.coffee_amount) : "",
        );
        this.setFormField(
          form,
          "water_amount",
          recipe.water_amount > 0 ? Math.round(recipe.water_amount) : "",
        );

        const brewerCombo = this._combo("brewer");
        if (brewerCombo) {
          const localBrewer = (this.dropdownManager?.brewers || []).find(
            /** @param {any} b */ (b) =>
              (b.rkey || b.RKey) === recipe.brewer_rkey,
          );
          const brewerRKey = localBrewer ? recipe.brewer_rkey : "";
          const brewerName = localBrewer
            ? localBrewer.name || localBrewer.Name || ""
            : "";
          brewerCombo.dispatchEvent(
            new CustomEvent("combo-set", {
              detail: { rkey: brewerRKey, label: brewerName },
              bubbles: false,
            }),
          );
        }
        if (recipe.brewer_rkey) this.onBrewerChange(recipe.brewer_rkey);
        if (!this.brewerCategory) {
          const recipeBrewerType =
            recipe.brewer_type || recipe.brewer_obj?.brewer_type || "";
          if (recipeBrewerType) {
            this.brewerCategory =
              this.normalizeBrewerCategory(recipeBrewerType);
          }
        }

        this.pours =
          recipe.pours && recipe.pours.length > 0
            ? recipe.pours.map(
                /** @param {any} p */ (p) => ({
                  water: p.water_amount || "",
                  time: p.time_seconds || "",
                }),
              )
            : [];

        this.updatePoursVisibility();
      } catch (e) {
        console.error("Failed to apply recipe:", e);
      }
    },

    /**
     * @param {HTMLElement} form
     * @param {string} name
     * @param {string | number} value
     */
    setFormField(form, name, value) {
      form.querySelectorAll(`[name="${name}"]`).forEach(
        /** @param {Element} el */ (el) => {
          /** @type {any} */ (el).value = value;
          el.dispatchEvent(new Event("input", { bubbles: true }));
        },
      );
    },

    /** @param {HTMLElement} form */
    clearRecipeFields(form) {
      this.setFormField(form, "coffee_amount", "");
      this.setFormField(form, "water_amount", "");
      const brewerCombo = this._combo("brewer");
      if (brewerCombo) {
        brewerCombo.dispatchEvent(
          new CustomEvent("combo-set", {
            detail: { rkey: "", label: "" },
            bubbles: false,
          }),
        );
      }
      this.pours = [];
    },

    addPour() {
      this.pours.push({ water: "", time: "" });
    },

    removePour(index) {
      this.pours.splice(index, 1);
    },

    // Exposed lists for template bindings.
    get beans() {
      return this.dropdownManager?.beans || [];
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
  };
}

/** @type {any} */ (window).brewForm = brewForm;
