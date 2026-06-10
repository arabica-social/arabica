<script lang="ts">
  import { onMount } from "svelte";
  import { getCachedFeedHTML, setCachedFeedHTML } from "./feedCache";
  type FeedTab = {
    label: string;
    value: string;
  };

  let {
    initialType = "",
    initialSort = "recent",
    initialTabs = [],
  }: {
    initialType?: string;
    initialSort?: string;
    initialTabs?: FeedTab[];
  } = $props();

  // svelte-ignore state_referenced_locally
  let typeFilter = $state(initialType);
  // svelte-ignore state_referenced_locally
  let sort = $state(initialSort || "recent");
  // svelte-ignore state_referenced_locally
  let tabs = $state<FeedTab[]>(initialTabs);
  let loading = $state(false);

  function feedURL(nextType: string, nextSort: string) {
    const params = new URLSearchParams();
    if (nextType) {
      params.set("type", nextType);
    }
    if (nextSort && nextSort !== "recent") {
      params.set("sort", nextSort);
    }
    const query = params.toString();
    return query ? `/api/feed?${query}` : "/api/feed";
  }

  function pillClass(tab: string) {
    if (typeFilter !== tab) {
      return "filter-pill";
    }
    if (!tab) {
      return "filter-pill-active";
    }
    return `filter-pill-active filter-pill-${tab}`;
  }

  function finishFeedSwap() {
    const feedItems = document.querySelector<HTMLElement>("#feed-items");
    if (feedItems) {
      window.htmx?.process?.(feedItems);
    }
    window.__arabicaSvelteIslands?.mountAll();
    window.__arabicaSvelteIslands?.applyFeedMasonry();
  }

  function replaceFeedFromCache(html: string) {
    const current = document.querySelector<HTMLElement>("#feed-items");
    if (!current) {
      return false;
    }

    const template = document.createElement("template");
    template.innerHTML = html.trim();
    const cached = template.content.querySelector<HTMLElement>("#feed-items");
    if (!cached) {
      return false;
    }

    current.replaceWith(cached);
    finishFeedSwap();
    return true;
  }

  function cacheCurrentFeed(url: string) {
    const feedItems = document.querySelector<HTMLElement>("#feed-items");
    if (!feedItems) {
      return;
    }

    // The home page starts with a server-rendered loading skeleton before HTMX
    // swaps in the real feed. Never cache that placeholder, or switching back
    // to All can restore the skeleton with no HTMX trigger attached.
    if (
      feedItems.querySelector(".animate-pulse") &&
      !feedItems.querySelector(".feed-card")
    ) {
      return;
    }

    setCachedFeedHTML(url, feedItems.outerHTML);
  }

  function applyFeed(url: string) {
    const cached = getCachedFeedHTML(url);
    if (cached && replaceFeedFromCache(cached)) {
      return;
    }

    const htmx = window.htmx;
    if (!htmx?.ajax) {
      window.location.href = url;
      return;
    }
    loading = true;
    void Promise.resolve(
      htmx.ajax("GET", url, {
        target: "#feed-items",
        swap: "outerHTML",
        select: "#feed-items",
      }),
    ).finally(() => {
      loading = false;
      cacheCurrentFeed(url);
      finishFeedSwap();
    });
  }

  function changeFilter(nextType: string) {
    if (typeFilter === nextType && !loading) {
      return;
    }
    typeFilter = nextType;
    applyFeed(feedURL(nextType, sort));
  }

  onMount(() => {
    const handleFeedSwap = (event: Event) => {
      const detail = (event as CustomEvent).detail as
        | { target?: Element; pathInfo?: { finalRequestPath?: string } }
        | undefined;
      if (
        !(detail?.target instanceof HTMLElement) ||
        detail.target.id !== "feed-items"
      ) {
        return;
      }
      const path =
        detail.pathInfo?.finalRequestPath || feedURL(typeFilter, sort);
      if (path.startsWith("/api/feed")) {
        cacheCurrentFeed(path);
      }
    };

    document.body.addEventListener("htmx:afterSwap", handleFeedSwap);
    cacheCurrentFeed(feedURL(typeFilter, sort));

    return () => {
      document.body.removeEventListener("htmx:afterSwap", handleFeedSwap);
    };
  });

  function changeSort(nextSort: string) {
    if ((sort || "recent") === nextSort && !loading) {
      return;
    }
    sort = nextSort;
    applyFeed(feedURL(typeFilter, nextSort));
  }
</script>

<div
  class="flex flex-wrap items-center justify-between gap-2"
  aria-busy={loading}
>
  <div class="flex flex-wrap gap-1" role="group" aria-label="Feed filters">
    {#each tabs as tab}
      <button
        type="button"
        class={pillClass(tab.value)}
        aria-pressed={typeFilter === tab.value ? "true" : "false"}
        data-tab={tab.value}
        disabled={loading}
        onclick={() => changeFilter(tab.value)}
      >
        {tab.label}
      </button>
    {/each}
  </div>
  <div class="flex items-center gap-1 flex-shrink-0">
    <button
      type="button"
      class={sort === "" || sort === "recent"
        ? "filter-pill-active"
        : "filter-pill"}
      aria-pressed={sort === "" || sort === "recent" ? "true" : "false"}
      disabled={loading}
      onclick={() => changeSort("recent")}
    >
      New
    </button>
    <button
      type="button"
      class={sort === "popular" ? "filter-pill-active" : "filter-pill"}
      aria-pressed={sort === "popular" ? "true" : "false"}
      disabled={loading}
      onclick={() => changeSort("popular")}
    >
      Popular
    </button>
  </div>
</div>
