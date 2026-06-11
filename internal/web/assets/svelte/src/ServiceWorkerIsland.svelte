<script lang="ts">
  import { onMount } from "svelte";

  onMount(() => {
    if (!("serviceWorker" in navigator)) return;

    navigator.serviceWorker
      .register("/static/service-worker.js")
      .then(() => console.log("Service Worker registered"))
      .catch((error) =>
        console.log("Service Worker registration failed:", error),
      );
  });
</script>
