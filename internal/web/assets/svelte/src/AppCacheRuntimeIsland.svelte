<script lang="ts">
  import { onMount } from "svelte";
  import { appCache } from "./appCache";
  import { clearFeedCache } from "./feedCache";

  function clearHTMXHistoryCache() {
    try {
      localStorage.removeItem("htmx-history-cache");
    } catch {
      // Ignore storage failures.
    }
  }

  onMount(() => {
    window.AppCache = appCache;

    const handleRefreshManage = () => {
      appCache.invalidateCache();
      clearFeedCache();
      clearHTMXHistoryCache();
    };
    const handleEntityDeleted = () => {
      appCache.invalidateCache();
      clearFeedCache();
    };

    const handleHTMXBeforeRequest = (event: Event) => {
      const detail = (event as CustomEvent).detail as
        | { requestConfig?: { verb?: string } }
        | undefined;
      if (detail?.requestConfig?.verb?.toLowerCase() !== "get") {
        clearFeedCache();
      }
    };

    document.body.addEventListener("refreshManage", handleRefreshManage);
    document.body.addEventListener("entityDeleted", handleEntityDeleted);
    document.body.addEventListener(
      "htmx:beforeRequest",
      handleHTMXBeforeRequest,
    );

    return () => {
      document.body.removeEventListener("refreshManage", handleRefreshManage);
      document.body.removeEventListener("entityDeleted", handleEntityDeleted);
      document.body.removeEventListener(
        "htmx:beforeRequest",
        handleHTMXBeforeRequest,
      );
      if (window.AppCache === appCache) {
        delete window.AppCache;
      }
    };
  });
</script>
