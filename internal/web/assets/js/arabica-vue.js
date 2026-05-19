// @ts-check
// Petite-vue scopes and bootstrap for arabica.

/**
 * Globally-available helpers from sibling scripts.
 * Declared so // @ts-check doesn't complain about unresolved names.
 * @typedef {object} PetiteVueAppCtor
 * @property {(globals: Record<string, unknown>) => { mount: (el?: Element | Document) => void }} createApp
 *
 * @typedef {object} ArabicaWindow
 * @property {PetiteVueAppCtor} [PetiteVue]
 * @property {() => void} [applyTheme]
 */

/** @typedef {{ open: boolean, toggle: () => void, close: () => void }} DisclosureScope */
/** @typedef {{ copied: boolean, share: () => Promise<void> }} ShareButtonScope */
/** @typedef {{ moreOpen: boolean, openUp: boolean, toggle: (e?: MouseEvent) => void, closeMenu: () => void, notify: (msg: string) => void, copyURI: (uri: string) => void, isDevMode: () => boolean }} MoreMenuScope */
/** @typedef {{ showReplyForm: boolean, toggleReply: () => void, closeReply: () => void }} ReplyToggleScope */
/** @typedef {{ visible: boolean, scrollTop: () => void, setup: () => void }} ScrollTopScope */
/** @typedef {{ devMode: boolean, toggle: () => void }} DevModeToggleScope */
/** @typedef {{ theme: string, setTheme: (v: string) => void }} ThemeSettingsScope */
/** @typedef {{ typeFilter: string, sort: string, pillClass: (tab: string) => string, feedURL: (t: string, s: string) => string, changeFilter: (t: string) => void, changeSort: (s: string) => void }} FeedFiltersScope */
/** @typedef {{ serverError: string, setup: ($el: HTMLFormElement) => void }} ModalShellScope */
/** @typedef {{ water: number | string, time: number | string }} RecipePour */
/** @typedef {{ pours: RecipePour[], addPour: () => void, removePour: (i: number) => void }} RecipePoursScope */
/** @typedef {{ rkey: string, name: string }} RoasterRow */
/** @typedef {{ allRoasters: RoasterRow[], filtered: RoasterRow[], query: string, selectedRKey: string, newRoasterName: string, newRoasterLocation: string, newRoasterWebsite: string, showDropdown: boolean, showDetails: boolean, exactMatch: boolean, filter: () => void, selectRoaster: (r: RoasterRow) => void, startCreate: () => void, cancelCreate: () => void, clear: () => void }} RoasterPickerScope */

/**
 * Self-contained disclosure (open/close + click-outside).
 * @returns {DisclosureScope & { _outside: ((e: MouseEvent) => void) | null, setup: ($el: Element) => void, teardown: () => void }}
 */
function Disclosure() {
  return {
    open: false,
    _outside: null,
    _pagehide: null,
    // $el is passed explicitly because petite-vue exposes it as a `with`-scope
    // variable inside the directive expression, not as a property on `this`.
    setup($el) {
      this._outside = (e) => {
        if (this.open && !$el.contains(/** @type {Node} */ (e.target))) {
          this.open = false;
        }
      };
      document.addEventListener("click", this._outside);
      // bfcache restores the page (and reactive state) on history back —
      // make sure the menu is closed before it's snapshotted.
      this._pagehide = () => {
        this.open = false;
      };
      window.addEventListener("pagehide", this._pagehide);
    },
    teardown() {
      if (this._outside) document.removeEventListener("click", this._outside);
      if (this._pagehide)
        window.removeEventListener("pagehide", this._pagehide);
    },
    toggle() {
      this.open = !this.open;
    },
    close() {
      this.open = false;
    },
  };
}

/**
 * Web Share API button with clipboard fallback.
 * @param {string} url - relative URL to share; origin is prepended.
 * @param {string} title
 * @param {string} text
 * @returns {ShareButtonScope}
 */
function ShareButton(url, title, text) {
  return {
    copied: false,
    async share() {
      const fullUrl = window.location.origin + url;
      if (navigator.share) {
        try {
          await navigator.share({ title, text, url: fullUrl });
        } catch (e) {}
      } else {
        try {
          await navigator.clipboard.writeText(fullUrl);
          this.copied = true;
          setTimeout(() => (this.copied = false), 2000);
        } catch (e) {}
      }
    },
  };
}

/**
 * Action-bar overflow menu. Picks open-direction based on viewport position
 * and dispatches `notify` events for toast UI.
 * @returns {MoreMenuScope & { _outside: ((e: MouseEvent) => void) | null, setup: ($el: Element) => void, teardown: () => void }}
 */
function MoreMenu() {
  return {
    moreOpen: false,
    openUp: true,
    _outside: null,
    _pagehide: null,
    setup($el) {
      this._outside = (e) => {
        if (this.moreOpen && !$el.contains(/** @type {Node} */ (e.target))) {
          this.moreOpen = false;
        }
      };
      document.addEventListener("click", this._outside);
      // bfcache restores reactive state on history back — close before snapshot.
      this._pagehide = () => {
        this.moreOpen = false;
      };
      window.addEventListener("pagehide", this._pagehide);
    },
    teardown() {
      if (this._outside) document.removeEventListener("click", this._outside);
      if (this._pagehide)
        window.removeEventListener("pagehide", this._pagehide);
    },
    toggle(e) {
      if (!this.moreOpen && e && e.currentTarget) {
        const rect = /** @type {Element} */ (
          e.currentTarget
        ).getBoundingClientRect();
        this.openUp = rect.top > window.innerHeight * 0.25;
      }
      this.moreOpen = !this.moreOpen;
    },
    closeMenu() {
      this.moreOpen = false;
    },
    notify(message) {
      window.dispatchEvent(
        new CustomEvent("notify", { detail: { message }, bubbles: true }),
      );
    },
    copyURI(uri) {
      navigator.clipboard.writeText(uri).catch(() => {});
      this.moreOpen = false;
      this.notify("AT URI copied");
    },
    isDevMode() {
      try {
        return localStorage.getItem("devMode") === "true";
      } catch (e) {
        return false;
      }
    },
  };
}

/**
 * Inline reply form toggle for comment items.
 * @returns {ReplyToggleScope}
 */
function ReplyToggle() {
  return {
    showReplyForm: false,
    toggleReply() {
      this.showReplyForm = !this.showReplyForm;
    },
    closeReply() {
      this.showReplyForm = false;
    },
  };
}

/**
 * Back-to-top button visible after scrolling past a threshold.
 * @returns {ScrollTopScope & { _handler: (() => void) | null }}
 */
function ScrollTop() {
  return {
    visible: false,
    _handler: null,
    setup() {
      this._handler = () => {
        this.visible = window.scrollY > 400;
      };
      window.addEventListener("scroll", this._handler, { passive: true });
    },
    scrollTop() {
      window.scrollTo({ top: 0, behavior: "smooth" });
    },
  };
}

/**
 * Dev-mode preference toggle stored in localStorage. The action-bar
 * `MoreMenu.isDevMode()` reads from the same key.
 * @returns {DevModeToggleScope}
 */
function DevModeToggle() {
  return {
    devMode: (() => {
      try {
        return localStorage.getItem("devMode") === "true";
      } catch (e) {
        return false;
      }
    })(),
    toggle() {
      this.devMode = !this.devMode;
      try {
        localStorage.setItem("devMode", String(this.devMode));
      } catch (e) {}
    },
  };
}

/**
 * Theme picker for the settings page. Mirrors the head-script that applies
 * `data-theme` on initial load.
 * @returns {ThemeSettingsScope}
 */
function themeSettings() {
  return {
    theme: (() => {
      try {
        return localStorage.getItem("arabica-theme") || "system";
      } catch (e) {
        return "system";
      }
    })(),
    setTheme(value) {
      this.theme = value;
      try {
        if (value === "system") localStorage.removeItem("arabica-theme");
        else localStorage.setItem("arabica-theme", value);
      } catch (e) {}
      const w = /** @type {Window & ArabicaWindow} */ (window);
      if (typeof w.applyTheme === "function") w.applyTheme();
    },
  };
}

/**
 * Feed filter pills + sort selector. Reissues `htmx.ajax` calls when state
 * flips.
 * @param {string} initialType
 * @param {string} initialSort
 * @returns {FeedFiltersScope}
 */
function FeedFilters(initialType, initialSort) {
  return {
    typeFilter: initialType,
    sort: initialSort,
    pillClass(tab) {
      if (this.typeFilter !== tab) return "filter-pill";
      if (!tab) return "filter-pill-active";
      return "filter-pill-" + tab;
    },
    feedURL(t, s) {
      let u = "/api/feed";
      let sep = "?";
      if (t) {
        if (t === "equipment") u += sep + "type=grinder&type=brewer";
        else u += sep + "type=" + t;
        sep = "&";
      }
      if (s && s !== "recent") u += sep + "sort=" + s;
      return u;
    },
    changeFilter(t) {
      this.typeFilter = t;
      /** @type {any} */ (window).htmx.ajax("GET", this.feedURL(t, this.sort), {
        target: "#feed-items",
        swap: "outerHTML",
        select: "#feed-items",
      });
    },
    changeSort(s) {
      this.sort = s;
      /** @type {any} */ (window).htmx.ajax(
        "GET",
        this.feedURL(this.typeFilter, s),
        { target: "#feed-items", swap: "outerHTML", select: "#feed-items" },
      );
    },
  };
}

/**
 * Wraps the entity create/edit form. Listens for HTMX's `afterRequest` to
 * either close the dialog, trigger the manage-page refresh, prompt re-auth,
 * or surface a server error. Replaces the old `_x_dataStack[0].serverError`
 * DOM-poke from `hx-on::after-request`.
 * @returns {ModalShellScope}
 */
function ModalShell() {
  return {
    serverError: "",
    setup($el) {
      $el.addEventListener(
        "htmx:afterRequest",
        /** @param {Event} evt */ (evt) => {
          const detail = /** @type {any} */ (evt).detail;
          const dialog = $el.closest("dialog");
          if (detail && detail.successful) {
            if (dialog) dialog.close();
            /** @type {any} */ (window).htmx.trigger("body", "refreshManage");
            return;
          }
          if (detail && detail.xhr && detail.xhr.status === 401) {
            if (dialog) dialog.close();
            const showExpired = /** @type {any} */ (window)
              .__showSessionExpiredModal;
            if (typeof showExpired === "function") showExpired();
            return;
          }
          this.serverError =
            detail && detail.xhr
              ? "Something went wrong. Please try again."
              : "Connection error. Check your network.";
        },
      );
    },
  };
}

/**
 * Inline roaster picker for the bean create/edit modal. Lets you search an
 * existing roaster or start a "new roaster" flow with location/website
 * fields that get submitted as hidden form inputs.
 * @param {{ allRoasters: RoasterRow[], initialRKey?: string, initialName?: string }} opts
 * @returns {RoasterPickerScope}
 */
function RoasterPicker(opts) {
  const all = (opts && opts.allRoasters) || [];
  return {
    allRoasters: all,
    filtered: all.slice(0, 10),
    query: (opts && opts.initialName) || "",
    selectedRKey: (opts && opts.initialRKey) || "",
    newRoasterName: "",
    newRoasterLocation: "",
    newRoasterWebsite: "",
    showDropdown: false,
    showDetails: false,
    get exactMatch() {
      const q = this.query.trim().toLowerCase();
      return this.allRoasters.some((r) => r.name.toLowerCase() === q);
    },
    filter() {
      const q = this.query.trim().toLowerCase();
      this.selectedRKey = "";
      this.newRoasterName = "";
      if (!q) {
        this.filtered = this.allRoasters.slice(0, 10);
        return;
      }
      this.filtered = this.allRoasters.filter((r) =>
        r.name.toLowerCase().includes(q),
      );
    },
    selectRoaster(r) {
      this.selectedRKey = r.rkey;
      this.newRoasterName = "";
      this.newRoasterLocation = "";
      this.newRoasterWebsite = "";
      this.query = r.name;
      this.showDropdown = false;
      this.showDetails = false;
    },
    startCreate() {
      this.newRoasterName = this.query.trim();
      this.selectedRKey = "";
      this.showDropdown = false;
      this.showDetails = true;
    },
    cancelCreate() {
      this.newRoasterName = "";
      this.newRoasterLocation = "";
      this.newRoasterWebsite = "";
      this.showDetails = false;
    },
    clear() {
      this.query = "";
      this.selectedRKey = "";
      this.newRoasterName = "";
      this.newRoasterLocation = "";
      this.newRoasterWebsite = "";
      this.showDetails = false;
      this.filtered = this.allRoasters.slice(0, 10);
    },
  };
}

/**
 * Generic copy-to-clipboard button. Sets `copied = true` for 2s after copy.
 * Pair with `@click="copy($refs.X.textContent)"` and a `ref="X"` on the
 * source element.
 */
function CopyText() {
  return {
    copied: false,
    async copy(text) {
      try {
        await navigator.clipboard.writeText((text || "").trim());
        this.copied = true;
        setTimeout(() => {
          this.copied = false;
        }, 2000);
      } catch (e) {}
    },
  };
}

/**
 * Add-label form on the admin page. Listens for an `open-add-label` window
 * event to reveal itself; resets and closes on successful submit.
 */
function AddLabelForm() {
  return {
    open: false,
    setup($el) {
      window.addEventListener("open-add-label", () => {
        this.open = true;
      });
      $el.addEventListener("htmx:afterRequest", (evt) => {
        const detail = /** @type {any} */ (evt).detail;
        if (detail && detail.successful) {
          this.open = false;
          const form = $el.querySelector("form");
          if (form) /** @type {HTMLFormElement} */ (form).reset();
        }
      });
    },
  };
}

/**
 * Generic full-page form scope: surfaces an htmx:afterRequest error to
 * `serverError`. Used by full-page forms (tea form, etc.) that previously
 * poked `_x_dataStack[0].serverError` from `hx-on::after-request`.
 */
function PageForm() {
  return {
    serverError: "",
    rating: 5,
    setup($el) {
      // Seed rating from form data attribute. We can't rely on the slider's
      // own `value=` attribute because petite-vue's `v-model` writes the
      // scope default to the DOM before any @vue:mounted hook can read it.
      const ratingAttr = $el.getAttribute("data-rating");
      if (ratingAttr) {
        const r = Number(ratingAttr);
        if (!Number.isNaN(r)) this.rating = r;
      }
      $el.addEventListener(
        "htmx:afterRequest",
        /** @param {Event} evt */ (evt) => {
          const detail = /** @type {any} */ (evt).detail;
          if (detail && detail.successful) return;
          if (detail && detail.xhr && detail.xhr.status === 401) {
            const showExpired = /** @type {any} */ (window)
              .__showSessionExpiredModal;
            if (typeof showExpired === "function") showExpired();
            return;
          }
          this.serverError =
            detail && detail.xhr
              ? detail.xhr.responseText ||
                "Something went wrong. Please try again."
              : "Connection error. Check your network.";
        },
      );
    },
  };
}

/**
 * Steep-form scope. Like PageForm but also tracks `infusionMethod` to toggle
 * the infuser section.
 * @param {string} initialInfusionMethod
 */
function SteepForm(initialInfusionMethod) {
  return {
    serverError: "",
    infusionMethod: initialInfusionMethod || "",
    rating: 5,
    setup($el) {
      const ratingAttr = $el.getAttribute("data-rating");
      if (ratingAttr) {
        const r = Number(ratingAttr);
        if (!Number.isNaN(r)) this.rating = r;
      }
      $el.addEventListener(
        "htmx:afterRequest",
        /** @param {Event} evt */ (evt) => {
          const detail = /** @type {any} */ (evt).detail;
          if (detail && detail.successful) return; // HX-Redirect handled by browser
          if (detail && detail.xhr && detail.xhr.status === 401) {
            const showExpired = /** @type {any} */ (window)
              .__showSessionExpiredModal;
            if (typeof showExpired === "function") showExpired();
            return;
          }
          this.serverError =
            detail && detail.xhr
              ? detail.xhr.responseText ||
                "Something went wrong. Please try again."
              : "Connection error. Check your network.";
        },
      );
    },
  };
}

/**
 * Pour-list editor used inside the recipe create/edit modal.
 * @param {RecipePour[]} initialPours
 * @returns {RecipePoursScope}
 */
function RecipePours(initialPours) {
  return {
    pours: Array.isArray(initialPours) ? initialPours.slice() : [],
    addPour() {
      this.pours.push({ water: "", time: "" });
    },
    removePour(i) {
      this.pours.splice(i, 1);
    },
  };
}

/**
 * @typedef {object} BeanRatingScope
 * @property {boolean} showRating
 * @property {number} rating
 */

/**
 * Rating sub-scope for the bean create/edit form.
 * @param {number} [initialRating] — 1-10 when editing, 0 on create
 * @returns {BeanRatingScope}
 */
function BeanRating(initialRating) {
  const r = Number(initialRating);
  const has = Number.isFinite(r) && r > 0;
  return {
    showRating: has,
    rating: has ? r : 0,
  };
}

(function () {
  const w = /** @type {Window & ArabicaWindow} */ (window);
  if (typeof w.PetiteVue === "undefined") return;
  const PetiteVue = w.PetiteVue;

  const globals = {
    Disclosure,
    ShareButton,
    MoreMenu,
    ReplyToggle,
    ScrollTop,
    DevModeToggle,
    themeSettings,
    FeedFilters,
    ModalShell,
    RecipePours,
    BeanRating,
    RoasterPicker,
    PageForm,
    SteepForm,
    CopyText,
    AddLabelForm,
    // Factories defined in sibling scripts and assigned to window. We probe
    // at mount time so script-load order is the only requirement.
    entitySuggest: /** @type {any} */ (window).entitySuggest,
    recipeExplore: /** @type {any} */ (window).recipeExplore,
    comboSelect: /** @type {any} */ (window).comboSelect,
    brewForm: /** @type {any} */ (window).brewForm,
    managePage: /** @type {any} */ (window).managePage,
  };

  /** @param {Element | Document | null | undefined} root */
  function mountWithin(root) {
    // petite-vue removes the v-scope attribute as it walks, so the
    // `closest("[v-scope]")` ancestor check misses already-mounted
    // parents and double-mounts their nested scopes. data-pv-root
    // survives the walk and keeps the descendant check honest.
    const scope = root && "querySelectorAll" in root ? root : document;
    const ANCESTOR_SELECTOR = "[v-scope],[data-pv-root]";
    if (
      root &&
      root instanceof Element &&
      root.matches("[v-scope]") &&
      !(/** @type {any} */ (root).__pv)
    ) {
      /** @type {any} */ (root).__pv = true;
      root.setAttribute("data-pv-root", "");
      PetiteVue.createApp(globals).mount(root);
    }
    scope.querySelectorAll("[v-scope]").forEach((el) => {
      const tagged = /** @type {any} */ (el);
      if (tagged.__pv) return;
      if (el.parentElement && el.parentElement.closest(ANCESTOR_SELECTOR))
        return;
      tagged.__pv = true;
      el.setAttribute("data-pv-root", "");
      PetiteVue.createApp(globals).mount(el);
    });
  }

  function init() {
    mountWithin(document);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }

  document.addEventListener("htmx:afterSwap", (evt) => {
    const detail = /** @type {any} */ (evt).detail;
    const target =
      /** @type {Element | null} */ (evt.target) ||
      (detail && detail.target) ||
      null;
    if (target) mountWithin(target);
  });
})();
