// @ts-check
// Petite-vue factory for the manage / my-coffee / profile pages.
// Owns tab state and the "incomplete entity" nudge toast. Modal CRUD goes
// through the HTMX /api/modals/* flow, not this store.

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

    setup() {
      const cache = /** @type {any} */ (window).AppCache;
      if (cache) cache.init();
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
        body.appendChild(
          document.createTextNode(" is missing " + nudge.missing),
        );

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
            htmx.ajax("GET", `/api/modals/${nudge.entity_type}/${nudge.rkey}`, {
              target: "#modal-container",
              swap: "innerHTML",
            });
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
  };
}

/** @type {any} */ (window).managePage = managePage;
