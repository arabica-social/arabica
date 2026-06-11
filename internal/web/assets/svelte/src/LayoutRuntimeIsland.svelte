<script lang="ts">
  import { onMount } from "svelte";

  type SavedForm = {
    path?: string;
    data?: Record<string, unknown>;
  };

  function sessionModal() {
    return document.getElementById(
      "session-expired-modal",
    ) as HTMLDialogElement | null;
  }

  function saveFormBeforeReauth() {
    const form = document.querySelector<HTMLFormElement>("main form");
    if (!form) return;
    const data: Record<string, unknown> = {};
    form
      .querySelectorAll<
        HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement
      >("input, select, textarea")
      .forEach((el) => {
        if (
          !el.name ||
          el.type === "hidden" ||
          el.type === "submit" ||
          el.type === "button"
        )
          return;
        if (
          el instanceof HTMLInputElement &&
          (el.type === "checkbox" || el.type === "radio")
        ) {
          data[el.name] = el.checked;
        } else {
          data[el.name] = el.value;
        }
      });
    if (Object.keys(data).length === 0) return;
    sessionStorage.setItem(
      "arabica_form_restore",
      JSON.stringify({
        path: window.location.pathname,
        data,
      }),
    );
  }

  function showSessionExpiredModal() {
    const modal = sessionModal();
    if (!modal || modal.open) return;
    const returnInput = document.getElementById(
      "reauth-return-to",
    ) as HTMLInputElement | null;
    if (returnInput) returnInput.value = window.location.pathname;
    modal.showModal();
  }

  function restoreSavedForm() {
    const saved = sessionStorage.getItem("arabica_form_restore");
    if (!saved) return;
    try {
      const parsed = JSON.parse(saved) as SavedForm;
      if (parsed.path !== window.location.pathname) {
        sessionStorage.removeItem("arabica_form_restore");
        return;
      }
      let attempts = 0;
      const maxAttempts = 30;
      const interval = setInterval(() => {
        attempts += 1;
        const form = document.querySelector<HTMLFormElement>("main form");
        const formData = parsed.data || {};
        if (!form) {
          if (attempts >= maxAttempts) clearInterval(interval);
          return;
        }
        const selects = Array.from(form.querySelectorAll("select"));
        const ready = selects.every((select) => select.options.length > 1);
        if (!ready && attempts < maxAttempts) return;
        clearInterval(interval);
        for (const [key, value] of Object.entries(formData)) {
          const el = form.querySelector<
            HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement
          >(`[name="${CSS.escape(key)}"]`);
          if (!el) continue;
          if (
            el instanceof HTMLInputElement &&
            (el.type === "checkbox" || el.type === "radio")
          ) {
            el.checked = Boolean(value);
          } else {
            el.value = String(value ?? "");
            el.dispatchEvent(new Event("change", { bubbles: true }));
          }
        }
        sessionStorage.removeItem("arabica_form_restore");
      }, 500);
    } catch {
      sessionStorage.removeItem("arabica_form_restore");
    }
  }

  function formatLocalTimes(root: ParentNode = document) {
    root.querySelectorAll<HTMLTimeElement>("time[data-local]").forEach((el) => {
      const datetime = el.getAttribute("datetime");
      if (!datetime) return;
      const dt = new Date(datetime);
      if (Number.isNaN(dt.getTime())) return;
      const fmt = el.getAttribute("data-local");
      let text = "";
      try {
        if (fmt === "date") {
          text = dt.toLocaleDateString(undefined, {
            month: "short",
            day: "numeric",
          });
        } else if (fmt === "year") {
          text = String(dt.getFullYear());
        } else if (fmt === "long") {
          text =
            dt.toLocaleDateString(undefined, {
              month: "long",
              day: "numeric",
              year: "numeric",
            }) +
            " at " +
            dt.toLocaleTimeString(undefined, {
              hour: "numeric",
              minute: "2-digit",
            });
        } else if (fmt === "short") {
          text =
            dt.toLocaleDateString(undefined, {
              month: "short",
              day: "numeric",
              year: "numeric",
            }) +
            " " +
            dt.toLocaleTimeString(undefined, {
              hour: "2-digit",
              minute: "2-digit",
            });
        }
        if (text) el.textContent = text;
      } catch {
        // Keep the server-rendered fallback if browser locale formatting fails.
      }
    });
  }

  function cleanHistorySnapshot() {
    document
      .querySelectorAll<HTMLElement>("main, main *, body")
      .forEach((el) => {
        el.classList.remove(
          "htmx-swapping",
          "htmx-transitioning",
          "htmx-settling",
          "htmx-added",
          "transitioning",
        );
        if (el.style.opacity || el.style.transform || el.style.visibility) {
          el.style.opacity = "";
          el.style.transform = "";
          el.style.visibility = "";
        }
      });
  }

  function showToast(message: string) {
    if (!message) return;
    const region = document.getElementById("toast-region");
    if (!region) return;
    const el = document.createElement("div");
    el.className = "toast";
    el.setAttribute("role", "status");
    el.textContent = message;
    region.appendChild(el);
    setTimeout(() => el.remove(), 2900);
  }

  function extractMessage(detail: any) {
    if (!detail) return "";
    if (detail.value && typeof detail.value === "object")
      return detail.value.message || "";
    if (typeof detail.value === "string") return detail.value;
    return detail.message || "";
  }

  function handleClick(event: MouseEvent) {
    const target = event.target;
    if (!(target instanceof Element)) return;
    if (target.closest("#session-expired-dismiss")) {
      sessionModal()?.close();
      return;
    }
    const close = target.closest("[data-dialog-close]");
    if (!close) return;
    close.closest<HTMLDialogElement>("dialog")?.close();
  }

  function handleSubmit(event: SubmitEvent) {
    const target = event.target;
    if (target instanceof HTMLFormElement && target.id === "reauth-form") {
      saveFormBeforeReauth();
    }
  }

  function handleHTMXConfigRequest(event: Event) {
    const detail = (event as CustomEvent<{ headers?: Record<string, string> }>)
      .detail;
    const traceparent = document.querySelector<HTMLMetaElement>(
      'meta[name="traceparent"]',
    )?.content;
    if (traceparent && detail?.headers) {
      detail.headers.traceparent = traceparent;
    }
  }

  function handleHTMXAfterRequest(event: Event) {
    const xhr = (event as CustomEvent<{ xhr?: XMLHttpRequest }>).detail?.xhr;
    if (xhr?.status === 401) showSessionExpiredModal();
  }

  function handleHTMXAfterSwap(event: Event) {
    const target = (event as CustomEvent<{ target?: EventTarget }>).detail
      ?.target;
    if (
      target instanceof Element ||
      target instanceof DocumentFragment ||
      target instanceof Document
    ) {
      formatLocalTimes(target);
    }
  }

  function handleCloseDialog(event: Event) {
    const value = (
      event as CustomEvent<{ value?: string | { id?: string; delay?: number } }>
    ).detail?.value;
    if (!value) return;
    const id = typeof value === "string" ? value : value.id;
    const delay = typeof value === "object" ? value.delay || 0 : 0;
    if (!id) return;
    setTimeout(() => {
      const dialog = document.getElementById(id) as HTMLDialogElement | null;
      dialog?.close();
    }, delay);
  }

  function handleNotify(event: Event) {
    showToast(extractMessage((event as CustomEvent).detail));
  }

  onMount(() => {
    window.__showSessionExpiredModal = showSessionExpiredModal;
    restoreSavedForm();
    formatLocalTimes();

    document.body.addEventListener("click", handleClick);
    document.body.addEventListener("submit", handleSubmit);
    document.body.addEventListener(
      "htmx:configRequest",
      handleHTMXConfigRequest,
    );
    document.body.addEventListener("htmx:afterRequest", handleHTMXAfterRequest);
    document.body.addEventListener("htmx:afterSwap", handleHTMXAfterSwap);
    document.body.addEventListener(
      "htmx:beforeHistorySave",
      cleanHistorySnapshot,
    );
    document.body.addEventListener("close-dialog", handleCloseDialog);
    window.addEventListener("notify", handleNotify);

    return () => {
      if (window.__showSessionExpiredModal === showSessionExpiredModal)
        delete window.__showSessionExpiredModal;
      document.body.removeEventListener("click", handleClick);
      document.body.removeEventListener("submit", handleSubmit);
      document.body.removeEventListener(
        "htmx:configRequest",
        handleHTMXConfigRequest,
      );
      document.body.removeEventListener(
        "htmx:afterRequest",
        handleHTMXAfterRequest,
      );
      document.body.removeEventListener("htmx:afterSwap", handleHTMXAfterSwap);
      document.body.removeEventListener(
        "htmx:beforeHistorySave",
        cleanHistorySnapshot,
      );
      document.body.removeEventListener("close-dialog", handleCloseDialog);
      window.removeEventListener("notify", handleNotify);
    };
  });
</script>
