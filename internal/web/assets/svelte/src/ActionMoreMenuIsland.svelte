<script lang="ts">
  interface Props {
    target: HTMLElement;
  }

  interface HXNotify {
    message?: string;
  }

  interface HXCloseDialog {
    id?: string;
    delay?: number;
  }

  let { target }: Props = $props();
  let open = false;
  let openUp = true;

  let button: HTMLButtonElement | null = null;
  let menu: HTMLElement | null = null;

  function isDevMode() {
    try {
      return localStorage.getItem("devMode") === "true";
    } catch {
      return false;
    }
  }

  function notify(message: string) {
    window.dispatchEvent(
      new CustomEvent("notify", { detail: { message }, bubbles: true }),
    );
  }

  function applyState() {
    button?.setAttribute("aria-expanded", open ? "true" : "false");
    menu?.classList.toggle("is-open", open);
    menu?.classList.toggle("bottom-full", openUp);
    menu?.classList.toggle("mb-1", openUp);
    menu?.classList.toggle("top-full", !openUp);
    menu?.classList.toggle("mt-1", !openUp);

    target
      .querySelectorAll<HTMLElement>("[data-more-menu-dev-only]")
      .forEach((node) => {
        node.hidden = !isDevMode();
      });
  }

  function close() {
    open = false;
    applyState();
  }

  function toggle() {
    if (!open && button) {
      const rect = button.getBoundingClientRect();
      openUp = rect.top > window.innerHeight * 0.25;
    }
    open = !open;
    applyState();
  }

  async function copyURI(uri: string) {
    try {
      await navigator.clipboard.writeText(uri);
    } catch {
      // Clipboard can fail on non-secure origins; the menu action should still close.
    }
    close();
    notify("AT URI copied");
  }

  async function performAction(
    url: string,
    options: {
      method?: string;
      body?: Record<string, string>;
      confirm?: string;
      redirect?: string;
      deleteTarget?: string;
      targetNode?: Element;
    },
  ) {
    if (options.confirm && !window.confirm(options.confirm)) {
      return;
    }

    close();
    try {
      const response = await fetch(url, {
        method: (options.method || "POST").toUpperCase(),
        credentials: "same-origin",
        headers: {
          "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8",
        },
        body: options.body
          ? new URLSearchParams(options.body).toString()
          : undefined,
      });

      if (!response.ok) {
        if (response.status === 401) {
          window.__showSessionExpiredModal?.();
          return;
        }
        const errorText = await response.text();
        notify(errorText || "Action failed");
        return;
      }

      const redirectURL = options.redirect || "";
      if (redirectURL) {
        window.location.href = redirectURL;
        return;
      }

      const deleteTarget = options.deleteTarget || "";
      if (deleteTarget && options.targetNode) {
        if (deleteTarget.startsWith("closest ")) {
          options.targetNode
            .closest(deleteTarget.replace("closest ", ""))
            ?.remove();
        } else {
          document.querySelector<HTMLElement>(deleteTarget)?.remove();
        }
      }
    } catch {
      notify("Action failed");
    }
  }

  async function openEditModal(editURL: string) {
    if (!editURL) {
      notify("Edit link missing");
      return;
    }

    close();

    try {
      const response = await fetch(editURL, {
        method: "GET",
        credentials: "same-origin",
      });
      const responseText = await response.text();

      if (!response.ok) {
        if (response.status === 401) {
          window.__showSessionExpiredModal?.();
          return;
        }
        notify(responseText || "Failed to load editor");
        return;
      }

      const container = document.getElementById("modal-container");
      if (!container) {
        notify("Modal container not found");
        return;
      }

      container.innerHTML = responseText;
      const dialog = container.querySelector<HTMLDialogElement>("dialog");
      if (!dialog || typeof dialog.showModal !== "function") {
        notify("Failed to open editor");
        return;
      }

      const refreshManage = () => window.location.reload();
      document.body.addEventListener("refreshManage", refreshManage, {
        once: true,
      });
      dialog.addEventListener(
        "close",
        () => {
          document.body.removeEventListener("refreshManage", refreshManage);
        },
        { once: true },
      );

      window.setTimeout(() => {
        if (!dialog.open) {
          dialog.showModal();
        }
      }, 0);
    } catch {
      notify("Failed to load editor");
    }
  }

  function parsePayload(rawPayload: string | null): Record<string, string> {
    if (!rawPayload) {
      return {};
    }

    try {
      const parsed = JSON.parse(rawPayload);
      if (typeof parsed === "object" && parsed && !Array.isArray(parsed)) {
        return parsed as Record<string, string>;
      }
    } catch {
      // Ignore malformed payloads and proceed with an empty body.
    }

    return {};
  }

  function applyHXTrigger(headerValue: string | null) {
    if (!headerValue) {
      return false;
    }

    try {
      const parsed = JSON.parse(headerValue) as Record<string, unknown>;
      if (!parsed || typeof parsed !== "object") {
        return false;
      }

      const notifyPayload = parsed.notify;
      if (typeof notifyPayload === "string") {
        notify(notifyPayload);
      } else if (
        typeof notifyPayload === "object" &&
        notifyPayload &&
        typeof (notifyPayload as HXNotify).message === "string"
      ) {
        notify((notifyPayload as HXNotify).message || "Done");
      }

      const closePayload = parsed["close-dialog"];
      if (typeof closePayload === "string") {
        document.body.dispatchEvent(
          new CustomEvent("close-dialog", {
            detail: {
              value: {
                id: closePayload,
                delay: 0,
              },
            },
            bubbles: true,
          }),
        );
      } else if (
        closePayload &&
        typeof closePayload === "object" &&
        typeof (closePayload as HXCloseDialog).id === "string"
      ) {
        const payload = closePayload as HXCloseDialog;
        document.body.dispatchEvent(
          new CustomEvent("close-dialog", {
            detail: {
              value: {
                id: payload.id,
                delay: payload.delay ?? 0,
              },
            },
            bubbles: true,
          }),
        );
      }
      return true;
    } catch {
      return false;
    }
  }

  function handleReportSubmit(event: SubmitEvent) {
    const eventTarget = event.target;
    if (!(eventTarget instanceof HTMLFormElement)) {
      return;
    }

    const form = eventTarget.closest<HTMLFormElement>(
      "form[data-svelte-report-form]",
    );
    if (!form || !target.contains(form)) {
      return;
    }

    event.preventDefault();

    const submitButtons = [
      ...form.querySelectorAll<HTMLButtonElement>('button[type="submit"]'),
    ];
    const dialog = form.closest<HTMLDialogElement>("dialog");
    const bodyId = dialog?.id ? `${dialog.id}-body` : "";
    const body =
      (bodyId ? document.getElementById(bodyId) : null) ??
      dialog?.querySelector<HTMLElement>(".modal-content") ??
      null;

    submitButtons.forEach((button) => {
      button.disabled = true;
    });

    void (async () => {
      try {
        const response = await fetch(form.action || "/api/report", {
          method: (form.method || "POST").toUpperCase(),
          credentials: "same-origin",
          body: new FormData(form),
        });
        const responseText = await response.text();

        if (body) {
          body.innerHTML = responseText;
        }

        if (!response.ok) {
          if (response.status === 401) {
            window.__showSessionExpiredModal?.();
            return;
          }
          if (!applyHXTrigger(response.headers.get("HX-Trigger"))) {
            const fallbackMessage = responseText.trim() || "Report failed";
            notify(fallbackMessage);
          }
          return;
        }

        if (!applyHXTrigger(response.headers.get("HX-Trigger"))) {
          notify("Report submitted");
        }
      } catch {
        notify("Failed to submit report");
      } finally {
        submitButtons.forEach((button) => {
          button.disabled = false;
        });
      }
    })();
  }

  function handleTargetClick(event: MouseEvent) {
    const eventTarget = event.target;
    if (!(eventTarget instanceof Element)) {
      return;
    }

    const toggleButton = eventTarget.closest<HTMLButtonElement>(
      "[data-more-menu-button]",
    );
    if (toggleButton && target.contains(toggleButton)) {
      toggle();
      return;
    }

    const copyButton = eventTarget.closest<HTMLElement>(
      "[data-more-menu-copy-uri]",
    );
    if (copyButton && target.contains(copyButton)) {
      void copyURI(copyButton.dataset.moreMenuCopyUri || "");
      return;
    }

    const reportButton = eventTarget.closest<HTMLElement>(
      "[data-more-menu-report-dialog]",
    );
    if (reportButton && target.contains(reportButton)) {
      close();
      const dialog = document.getElementById(
        reportButton.dataset.moreMenuReportDialog || "",
      );
      if (dialog instanceof HTMLDialogElement) {
        dialog.showModal();
      }
      return;
    }

    const editButton = eventTarget.closest<HTMLElement>(
      "[data-more-menu-edit-url]",
    );
    if (editButton && target.contains(editButton)) {
      event.preventDefault();
      void openEditModal(editButton.dataset.moreMenuEditUrl || "");
      return;
    }

    const actionButton = eventTarget.closest<HTMLElement>(
      "[data-more-menu-action]",
    );
    if (actionButton && target.contains(actionButton)) {
      const actionURL = actionButton.dataset.moreMenuActionUrl || "";
      if (!actionURL) {
        return;
      }
      const actionMethod = actionButton.dataset.moreMenuActionMethod || "POST";
      const actionPayload = parsePayload(
        actionButton.dataset.moreMenuActionPayload || "",
      );
      const actionConfirm = actionButton.dataset.moreMenuActionConfirm;
      const actionRedirect = actionButton.dataset.moreMenuActionRedirect || "";
      const actionTarget = actionButton.dataset.moreMenuActionTarget || "";

      event.preventDefault();
      event.stopPropagation();
      const targetNode =
        actionButton.closest<HTMLElement>(
          "li, button, [data-svelte-action-more-menu]",
        ) ?? actionButton;
      void performAction(actionURL, {
        method: actionMethod,
        body: actionPayload,
        confirm: actionConfirm || "",
        redirect: actionRedirect,
        deleteTarget: actionTarget,
        targetNode,
      });
      return;
    }

    const closeTrigger = eventTarget.closest("[data-more-menu-close]");
    if (closeTrigger && target.contains(closeTrigger)) {
      close();
    }
  }

  function handleOutsideClick(event: MouseEvent) {
    if (
      open &&
      event.target instanceof Node &&
      !target.contains(event.target)
    ) {
      close();
    }
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === "Escape") {
      close();
    }
  }

  $effect(() => {
    button = target.querySelector<HTMLButtonElement>("[data-more-menu-button]");
    menu = target.querySelector<HTMLElement>("[data-more-menu]");

    target.addEventListener("click", handleTargetClick);
    target.addEventListener("submit", handleReportSubmit);
    document.addEventListener("click", handleOutsideClick);
    document.addEventListener("keydown", handleKeydown);
    window.addEventListener("pagehide", close);
    window.addEventListener("storage", applyState);
    window.addEventListener("arabica:dev-mode-change", applyState);
    applyState();

    return () => {
      target.removeEventListener("click", handleTargetClick);
      target.removeEventListener("submit", handleReportSubmit);
      document.removeEventListener("click", handleOutsideClick);
      document.removeEventListener("keydown", handleKeydown);
      window.removeEventListener("pagehide", close);
      window.removeEventListener("storage", applyState);
      window.removeEventListener("arabica:dev-mode-change", applyState);
    };
  });
</script>
