<script lang="ts">
  interface Props {
    target: HTMLElement;
  }

  let { target }: Props = $props();
  let activeTab = $state("hidden");
  let copiedButton: HTMLButtonElement | null = null;
  let copiedTimer: ReturnType<typeof setTimeout> | undefined;

  function readStoredTab() {
    try {
      return sessionStorage.getItem("mod-tab") || "hidden";
    } catch {
      return "hidden";
    }
  }

  function writeStoredTab(tab: string) {
    try {
      sessionStorage.setItem("mod-tab", tab);
    } catch {
      // Session storage can be unavailable in strict privacy modes.
    }
  }

  function validTabs() {
    return Array.from(target.querySelectorAll<HTMLElement>("[data-admin-tab]"))
      .map((button) => button.dataset.adminTab || "")
      .filter(Boolean);
  }

  function applyTabs() {
    const tabs = validTabs();
    if (!tabs.includes(activeTab)) {
      activeTab = tabs[0] || "";
    }

    target
      .querySelectorAll<HTMLButtonElement>("[data-admin-tab]")
      .forEach((button) => {
        const selected = button.dataset.adminTab === activeTab;
        button.classList.toggle("tab-pill-active", selected);
        button.classList.toggle("tab-pill-inactive", !selected);
        button.setAttribute("aria-selected", selected ? "true" : "false");
      });

    target
      .querySelectorAll<HTMLElement>("[data-admin-panel]")
      .forEach((panel) => {
        panel.hidden = panel.dataset.adminPanel !== activeTab;
      });
  }

  function setTab(tab: string) {
    activeTab = tab;
    writeStoredTab(tab);
    applyTabs();
  }

  function setAddLabelOpen(open: boolean) {
    const formWrap = target.querySelector<HTMLElement>(
      "[data-admin-add-label-form]",
    );
    if (formWrap) {
      formWrap.hidden = !open;
    }
  }

  function handleAfterRequest(event: Event) {
    const detail = (event as CustomEvent<{ successful?: boolean }>).detail;
    if (!detail?.successful) {
      return;
    }
    const formWrap = target.querySelector<HTMLElement>(
      "[data-admin-add-label-form]",
    );
    if (!formWrap || event.target !== formWrap.querySelector("form")) {
      return;
    }
    setAddLabelOpen(false);
    formWrap.querySelector("form")?.reset();
  }

  function setCopied(button: HTMLButtonElement | null) {
    if (copiedButton) {
      copiedButton.title =
        copiedButton.dataset.copyTitle || "Copy to clipboard";
      copiedButton
        .querySelectorAll<HTMLElement>("[data-copy-idle]")
        .forEach((node) => {
          node.hidden = false;
        });
      copiedButton
        .querySelectorAll<HTMLElement>("[data-copy-copied]")
        .forEach((node) => {
          node.hidden = true;
        });
    }
    copiedButton = button;
    if (!button) {
      return;
    }
    button.dataset.copyTitle = button.title || "Copy to clipboard";
    button.title = "Copied!";
    button.querySelectorAll<HTMLElement>("[data-copy-idle]").forEach((node) => {
      node.hidden = true;
    });
    button
      .querySelectorAll<HTMLElement>("[data-copy-copied]")
      .forEach((node) => {
        node.hidden = false;
      });
    window.clearTimeout(copiedTimer);
    copiedTimer = window.setTimeout(() => setCopied(null), 2000);
  }

  async function copyFrom(button: HTMLButtonElement) {
    const wrapper = button.closest<HTMLElement>("[data-admin-copy]");
    const source = wrapper?.querySelector<HTMLElement>("[data-copy-source]");
    const text = source?.textContent?.trim() || "";
    if (!text) {
      return;
    }
    try {
      await navigator.clipboard.writeText(text);
      setCopied(button);
    } catch {
      // Clipboard failures should not break the admin dashboard.
    }
  }

  function handleClick(event: MouseEvent) {
    const eventTarget = event.target;
    if (!(eventTarget instanceof Element)) {
      return;
    }

    const tabButton =
      eventTarget.closest<HTMLButtonElement>("[data-admin-tab]");
    if (tabButton && target.contains(tabButton)) {
      setTab(tabButton.dataset.adminTab || "");
      return;
    }

    const openAddLabel = eventTarget.closest("[data-admin-add-label-open]");
    if (openAddLabel && target.contains(openAddLabel)) {
      setAddLabelOpen(true);
      return;
    }

    const closeAddLabel = eventTarget.closest("[data-admin-add-label-close]");
    if (closeAddLabel && target.contains(closeAddLabel)) {
      setAddLabelOpen(false);
      return;
    }

    const copyButton =
      eventTarget.closest<HTMLButtonElement>("[data-copy-button]");
    if (copyButton && target.contains(copyButton)) {
      void copyFrom(copyButton);
    }
  }

  $effect(() => {
    activeTab = readStoredTab();
    target.addEventListener("click", handleClick);
    target.addEventListener("htmx:afterRequest", handleAfterRequest);
    applyTabs();
    setAddLabelOpen(false);

    return () => {
      target.removeEventListener("click", handleClick);
      target.removeEventListener("htmx:afterRequest", handleAfterRequest);
      window.clearTimeout(copiedTimer);
    };
  });
</script>
