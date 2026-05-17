// @ts-check
// Re-apply the saved theme after HTMX swaps and history restores.
// The initial application happens in a head-script in layout.templ so the
// dark-mode flash never reaches the user; the function lives on window so
// the petite-vue theme picker (arabica-vue.js) can call it after a change.

function applyTheme() {
  const t = localStorage.getItem("arabica-theme");
  if (t === "dark" || t === "light") {
    document.documentElement.setAttribute("data-theme", t);
  } else {
    document.documentElement.removeAttribute("data-theme");
  }
}

/** @type {any} */ (window).applyTheme = applyTheme;

document.addEventListener("htmx:afterSettle", applyTheme);
document.addEventListener("htmx:historyRestore", applyTheme);
