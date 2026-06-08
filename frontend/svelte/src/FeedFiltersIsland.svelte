<script lang="ts">
  type FeedTab = {
    label: string;
    value: string;
  };

  let {
    initialType = '',
    initialSort = 'recent',
    initialTabs = []
  }: {
    initialType?: string;
    initialSort?: string;
    initialTabs?: FeedTab[];
  } = $props();

  // svelte-ignore state_referenced_locally
  let typeFilter = $state(initialType);
  // svelte-ignore state_referenced_locally
  let sort = $state(initialSort || 'recent');
  // svelte-ignore state_referenced_locally
  let tabs = $state<FeedTab[]>(initialTabs);
  let loading = $state(false);

  function feedURL(nextType: string, nextSort: string) {
    const params = new URLSearchParams();
    if (nextType) {
      params.set('type', nextType);
    }
    if (nextSort && nextSort !== 'recent') {
      params.set('sort', nextSort);
    }
    const query = params.toString();
    return query ? `/api/feed?${query}` : '/api/feed';
  }

  function pillClass(tab: string) {
    if (typeFilter !== tab) {
      return 'filter-pill';
    }
    if (!tab) {
      return 'filter-pill-active';
    }
    return `filter-pill-active filter-pill-${tab}`;
  }

  function applyFeed(url: string) {
    const htmx = window.htmx;
    if (!htmx?.ajax) {
      window.location.href = url;
      return;
    }
    loading = true;
    void Promise.resolve(
      htmx.ajax('GET', url, {
        target: '#feed-items',
        swap: 'outerHTML',
        select: '#feed-items'
      })
    ).finally(() => {
      loading = false;
      window.__arabicaSvelteIslands?.applyFeedMasonry();
    });
  }

  function changeFilter(nextType: string) {
    if (typeFilter === nextType && !loading) {
      return;
    }
    typeFilter = nextType;
    applyFeed(feedURL(nextType, sort));
  }

  function changeSort(nextSort: string) {
    if ((sort || 'recent') === nextSort && !loading) {
      return;
    }
    sort = nextSort;
    applyFeed(feedURL(typeFilter, nextSort));
  }

</script>

<div class="flex flex-wrap items-center justify-between gap-2" aria-busy={loading}>
  <div class="flex flex-wrap gap-1" role="group" aria-label="Feed filters">
    {#each tabs as tab}
      <button
        type="button"
        class={pillClass(tab.value)}
        aria-pressed={typeFilter === tab.value ? 'true' : 'false'}
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
      class={(sort === '' || sort === 'recent') ? 'filter-pill-active' : 'filter-pill'}
      aria-pressed={(sort === '' || sort === 'recent') ? 'true' : 'false'}
      disabled={loading}
      onclick={() => changeSort('recent')}
    >
      New
    </button>
    <button
      type="button"
      class={sort === 'popular' ? 'filter-pill-active' : 'filter-pill'}
      aria-pressed={sort === 'popular' ? 'true' : 'false'}
      disabled={loading}
      onclick={() => changeSort('popular')}
    >
      Popular
    </button>
  </div>
</div>
