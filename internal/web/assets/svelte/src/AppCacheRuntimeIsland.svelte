<script lang="ts">
  import { onMount } from "svelte";
  import { appCache } from "./appCache";

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
      clearHTMXHistoryCache();
    };
    const handleEntityDeleted = () => {
      appCache.invalidateCache();
    };

    document.body.addEventListener("refreshManage", handleRefreshManage);
    document.body.addEventListener("entityDeleted", handleEntityDeleted);

    return () => {
      document.body.removeEventListener("refreshManage", handleRefreshManage);
      document.body.removeEventListener("entityDeleted", handleEntityDeleted);
      if (window.AppCache === appCache) {
        delete window.AppCache;
      }
    };
  });
</script>
