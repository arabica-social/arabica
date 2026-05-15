// Apply saved theme to document
// Called on initial load (from head script), HTMX navigations, and history restores
function applyTheme() {
  var t = localStorage.getItem("arabica-theme");
  if (t === "dark" || t === "light") {
    document.documentElement.setAttribute("data-theme", t);
  } else {
    document.documentElement.removeAttribute("data-theme");
  }
}

// Re-apply theme after HTMX swaps and history restores
document.addEventListener("htmx:afterSettle", applyTheme);
document.addEventListener("htmx:historyRestore", applyTheme);

// Theme settings Alpine.js component
function themeSettings() {
  return {
    theme: localStorage.getItem("arabica-theme") || "system",
    setTheme(value) {
      this.theme = value;
      if (value === "system") {
        localStorage.removeItem("arabica-theme");
      } else {
        localStorage.setItem("arabica-theme", value);
      }
      applyTheme();
    },
  };
}
