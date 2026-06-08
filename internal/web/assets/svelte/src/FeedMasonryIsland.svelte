<script lang="ts">
  import { onMount } from "svelte";
  import { applyFeedMasonry, installFeedMasonry } from "./feedMasonry";

  onMount(() => {
    const teardown = installFeedMasonry();
    applyFeedMasonry();
    return teardown;
  });
</script>
