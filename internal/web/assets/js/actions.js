// @ts-check
// Event delegation for `data-action` attributes. Lets templates request
// common DOM actions without inline `onclick` handlers (which strict CSP
// blocks under script-src-attr).
//
// Supported actions:
//   data-action="history-back"          history.back()
//   data-action="close-dialog"          closest <dialog>.close()
//   data-action="open-modal"            document.getElementById(data-target).showModal()
//   data-action="dispatch-event"        window.dispatchEvent(new CustomEvent(data-event))
//   data-action="close-drawer"          closest [data-drawer].remove()

// Only one onboarding station drawer may be open at a time. When any "Add"
// button is clicked, clear all drawer slots before htmx swaps the new drawer
// into its target.
document.addEventListener("click", (e) => {
  const target = /** @type {Element | null} */ (e.target);
  if (!target) return;
  const addBtn = target.closest(".station-add[hx-target^='#station-drawer-slot-']");
  if (!addBtn) return;
  document
    .querySelectorAll(".station-drawer-row")
    .forEach((slot) => {
      slot.innerHTML = "";
    });
});

document.addEventListener("click", (e) => {
  const target = /** @type {Element | null} */ (e.target);
  if (!target) return;
  const el = target.closest("[data-action]");
  if (!el) return;
  const action = el.getAttribute("data-action");
  switch (action) {
    case "history-back": {
      history.back();
      break;
    }
    case "close-dialog": {
      const dialog = el.closest("dialog");
      if (dialog) /** @type {HTMLDialogElement} */ (dialog).close();
      break;
    }
    case "open-modal": {
      const id = el.getAttribute("data-target");
      if (!id) break;
      const dlg = /** @type {HTMLDialogElement | null} */ (
        document.getElementById(id)
      );
      if (dlg) dlg.showModal();
      break;
    }
    case "dispatch-event": {
      const name = el.getAttribute("data-event");
      if (name) window.dispatchEvent(new CustomEvent(name));
      break;
    }
    case "close-drawer": {
      const drawer = el.closest("[data-drawer]");
      if (drawer) drawer.remove();
      break;
    }
  }
});
