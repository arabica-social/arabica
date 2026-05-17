// Petite-vue scopes and bootstrap for arabica.
// Coexists with Alpine: Alpine binds inside `x-data` subtrees, petite-vue
// binds inside `v-scope` subtrees. Never put both on the same element.

function Disclosure() {
  return {
    open: false,
    _outside: null,
    // $el is passed explicitly because petite-vue exposes it as a `with`-scope
    // variable inside the directive expression, not as a property on `this`.
    setup($el) {
      this._outside = (e) => {
        if (this.open && !$el.contains(e.target)) this.open = false;
      };
      document.addEventListener("click", this._outside);
    },
    teardown() {
      if (this._outside) document.removeEventListener("click", this._outside);
    },
    toggle() {
      this.open = !this.open;
    },
    close() {
      this.open = false;
    },
  };
}

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

function MoreMenu() {
  return {
    moreOpen: false,
    openUp: true,
    _outside: null,
    setup($el) {
      this._outside = (e) => {
        if (this.moreOpen && !$el.contains(e.target)) this.moreOpen = false;
      };
      document.addEventListener("click", this._outside);
    },
    teardown() {
      if (this._outside) document.removeEventListener("click", this._outside);
    },
    toggle(e) {
      if (!this.moreOpen && e && e.currentTarget) {
        this.openUp =
          e.currentTarget.getBoundingClientRect().top >
          window.innerHeight * 0.25;
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

(function () {
  if (typeof PetiteVue === "undefined") return;
  const globals = {
    Disclosure,
    ShareButton,
    MoreMenu,
    ReplyToggle,
  };

  function mountWithin(root) {
    // Mount only top-level v-scope elements; petite-vue handles nested
    // v-scope children as part of the same app tree.
    const scope = root && root.querySelectorAll ? root : document;
    if (root && root.matches && root.matches("[v-scope]") && !root.__pv) {
      root.__pv = true;
      PetiteVue.createApp(globals).mount(root);
    }
    scope.querySelectorAll("[v-scope]").forEach((el) => {
      if (el.__pv) return;
      // Skip nested v-scope; the ancestor's mount already covers it.
      if (el.parentElement && el.parentElement.closest("[v-scope]")) return;
      el.__pv = true;
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
    const target = evt.target || (evt.detail && evt.detail.target);
    if (target) mountWithin(target);
  });
})();
