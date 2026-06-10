<script lang="ts">
  import { onMount } from "svelte";

  let { target }: { target: HTMLFormElement } = $props();

  function setError(message: string) {
    const error = target.querySelector<HTMLElement>("[data-modal-shell-error]");
    if (!error) {
      return;
    }
    error.textContent = message;
    error.hidden = message === "";
  }

  function closeDialog() {
    target.closest("dialog")?.close();
  }

  function discardDialog() {
    const dialog = target.closest("dialog");
    if (!dialog) {
      return;
    }
    dialog.close();
    dialog.remove();
  }

  function modalActionPath() {
    return (
      target.getAttribute("hx-post") || target.getAttribute("hx-put") || ""
    );
  }

  function selectNameForAction(actionPath: string) {
    if (actionPath.includes("/api/beans")) return "bean_rkey";
    if (actionPath.includes("/api/grinders")) return "grinder_rkey";
    if (actionPath.includes("/api/brewers")) return "brewer_rkey";
    if (actionPath.includes("/api/roasters")) return "roaster_rkey";
    return "";
  }

  function responseRKey(xhr: XMLHttpRequest | undefined) {
    if (!xhr?.responseText) return "";
    try {
      const data = JSON.parse(xhr.responseText);
      return data.rkey || data.RKey || "";
    } catch (error) {
      console.warn("Failed to parse entity response:", error);
      return "";
    }
  }

  async function refreshAfterSave(xhr: XMLHttpRequest | undefined) {
    const manageLoader = document.querySelector('[hx-get="/api/manage"]');
    if (manageLoader) {
      window.htmx?.trigger?.("body", "refreshManage");
      return;
    }

    document.body.dispatchEvent(
      new CustomEvent("refresh-dropdowns", { bubbles: true }),
    );
    await window.AppCache?.invalidateAndRefresh?.();

    const selectName = selectNameForAction(modalActionPath());
    const newRKey = responseRKey(xhr);
    const select = selectName
      ? document.querySelector<HTMLSelectElement>(
          `select[name="${selectName}"]`,
        )
      : null;
    if (newRKey && select) {
      window.setTimeout(() => {
        select.value = newRKey;
        select.dispatchEvent(new Event("change", { bubbles: true }));
      }, 50);
    }
  }

  onMount(() => {
    const handleAfterRequest = (event: Event) => {
      const detail = (
        event as CustomEvent<{ successful?: boolean; xhr?: XMLHttpRequest }>
      ).detail;
      if (detail?.successful) {
        setError("");
        discardDialog();
        void refreshAfterSave(detail.xhr);
        return;
      }
      if (detail?.xhr?.status === 401) {
        closeDialog();
        window.__showSessionExpiredModal?.();
        return;
      }
      setError(
        detail?.xhr
          ? "Something went wrong. Please try again."
          : "Connection error. Check your network.",
      );
    };

    target.addEventListener("htmx:afterRequest", handleAfterRequest);
    setError("");

    return () => {
      target.removeEventListener("htmx:afterRequest", handleAfterRequest);
    };
  });
</script>
