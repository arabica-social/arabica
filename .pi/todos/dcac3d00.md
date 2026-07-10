{
  "id": "dcac3d00",
  "title": "Wire Settings theme toggle in the SPA",
  "tags": [],
  "status": "completed",
  "created_at": "2026-07-10T03:29:55.182Z"
}

Settings page theme buttons (data-theme-choice="system|light|dark") have no click handler in the SPA — the legacy SettingsControlsIsland that wired them isn't loaded. Port the theme logic into the SPA: wire onclick handlers in settings/+page.svelte that write/remove localStorage['arabica-theme'] and call applyTheme(). Also the layout's applyTheme() only handles light/dark, not system (remove attribute). Expose window.applyTheme from +layout.svelte so settings can call it, and make applyTheme handle 'system' (remove data-theme attribute). Add active-state styling on the selected theme button (filter-pill-active class). Also wire the Developer "Copy AT URI" checkbox (data-dev-mode-toggle / devMode localStorage) if low-effort, or leave it as a known minor gap. Add E2E test for theme toggle.
