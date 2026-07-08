<script lang="ts">
  import { onMount } from "svelte";

  let { target }: { target: HTMLButtonElement } = $props();
  let copied = false;
  let copiedTimer: number | undefined;

  function setCopiedState(isCopied: boolean) {
    target
      .querySelectorAll<HTMLElement>("[data-share-status]")
      .forEach((node) => {
        node.hidden = !isCopied;
      });
    copied = isCopied;
  }

  async function share() {
    const url = window.location.origin + (target.dataset.shareUrl || "");
    const title = target.dataset.shareTitle || "";
    const text = target.dataset.shareText || "";
    if (navigator.share) {
      try {
        await navigator.share({ title, text, url });
      } catch {
        // User-cancelled shares are not actionable.
      }
      return;
    }
    try {
      await navigator.clipboard.writeText(url);
      setCopiedState(true);
      window.clearTimeout(copiedTimer);
      copiedTimer = window.setTimeout(() => {
        setCopiedState(false);
      }, 2000);
    } catch {
      // Clipboard failures are silent, matching the previous behavior.
    }
  }

  onMount(() => {
    const handleClick = () => {
      void share();
    };
    target.addEventListener("click", handleClick);

    return () => {
      target.removeEventListener("click", handleClick);
      window.clearTimeout(copiedTimer);
      if (copied) {
        setCopiedState(false);
      }
    };
  });
</script>
