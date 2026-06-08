<script lang="ts">
  type Theme = "system" | "light" | "dark";

  interface Props {
    target: HTMLElement;
  }

  let { target }: Props = $props();
  let theme = $state<Theme>("system");
  let devMode = $state(false);

  function readTheme(): Theme {
    try {
      const value = localStorage.getItem("arabica-theme");
      return value === "light" || value === "dark" ? value : "system";
    } catch {
      return "system";
    }
  }

  function readDevMode() {
    try {
      return localStorage.getItem("devMode") === "true";
    } catch {
      return false;
    }
  }

  function applyButtonState() {
    target
      .querySelectorAll<HTMLButtonElement>("[data-theme-choice]")
      .forEach((button) => {
        const selected = button.dataset.themeChoice === theme;
        button.className = selected ? "filter-pill-active" : "filter-pill";
        button.setAttribute("aria-pressed", selected ? "true" : "false");
      });

    const checkbox = target.querySelector<HTMLInputElement>(
      "[data-dev-mode-toggle]",
    );
    if (checkbox) {
      checkbox.checked = devMode;
    }
  }

  function setTheme(value: Theme) {
    theme = value;
    try {
      if (value === "system") {
        localStorage.removeItem("arabica-theme");
      } else {
        localStorage.setItem("arabica-theme", value);
      }
    } catch {
      // Local storage can be unavailable in strict privacy modes.
    }
    window.applyTheme?.();
    applyButtonState();
  }

  function setDevMode(value: boolean) {
    devMode = value;
    try {
      localStorage.setItem("devMode", String(value));
    } catch {
      // Local storage can be unavailable in strict privacy modes.
    }
    window.dispatchEvent(new CustomEvent("arabica:dev-mode-change"));
    applyButtonState();
  }

  function handleClick(event: MouseEvent) {
    const eventTarget = event.target;
    if (!(eventTarget instanceof Element)) {
      return;
    }
    const button = eventTarget.closest<HTMLButtonElement>(
      "[data-theme-choice]",
    );
    if (!button || !target.contains(button)) {
      return;
    }
    const nextTheme = button.dataset.themeChoice;
    if (
      nextTheme === "system" ||
      nextTheme === "light" ||
      nextTheme === "dark"
    ) {
      setTheme(nextTheme);
    }
  }

  function handleChange(event: Event) {
    const eventTarget = event.target;
    if (
      !(eventTarget instanceof HTMLInputElement) ||
      !target.contains(eventTarget)
    ) {
      return;
    }
    if (eventTarget.matches("[data-dev-mode-toggle]")) {
      setDevMode(eventTarget.checked);
    }
  }

  function handleStorage(event: StorageEvent) {
    if (event.key && event.key !== "arabica-theme" && event.key !== "devMode") {
      return;
    }
    theme = readTheme();
    devMode = readDevMode();
    applyButtonState();
  }

  $effect(() => {
    theme = readTheme();
    devMode = readDevMode();
    target.addEventListener("click", handleClick);
    target.addEventListener("change", handleChange);
    window.addEventListener("storage", handleStorage);
    applyButtonState();

    return () => {
      target.removeEventListener("click", handleClick);
      target.removeEventListener("change", handleChange);
      window.removeEventListener("storage", handleStorage);
    };
  });
</script>
