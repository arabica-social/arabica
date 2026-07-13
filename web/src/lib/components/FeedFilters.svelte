<script lang="ts">
  import type { FeedFilterTab } from "../types/feed";
  import StampTag from "./StampTag.svelte";

  type Props = {
    typeFilter: string;
    sort: string;
    loading: boolean;
    onType: (value: string) => void;
    onSort: (value: string) => void;
    tabs?: FeedFilterTab[];
  };

  let {
    typeFilter,
    sort,
    loading,
    onType,
    onSort,
    tabs = [],
  }: Props = $props();
  let activeTabs = $derived(
    tabs.length > 0 ? tabs : [{ label: "All", value: "" }],
  );

  function sortActive(value: string): boolean {
    if (value === "recent") return sort === "" || sort === "recent";
    return sort === value;
  }
</script>

<div
  class="mb-5 flex flex-wrap items-center justify-between gap-2"
  aria-busy={loading}
>
  <div class="flex flex-wrap gap-1" role="group" aria-label="Feed filters">
    {#each activeTabs as tab (tab.value)}
      <StampTag
        label={tab.label}
        active={typeFilter === tab.value}
        tone={tab.value}
        disabled={loading}
        onclick={() => onType(tab.value)}
      />
    {/each}
  </div>
  <div class="flex items-center gap-1 flex-shrink-0">
    <span class="sort-label">Sort</span>
    <StampTag
      label="New"
      active={sortActive("recent")}
      disabled={loading}
      onclick={() => onSort("recent")}
    />
    <StampTag
      label="Popular"
      active={sortActive("popular")}
      disabled={loading}
      onclick={() => onSort("popular")}
    />
  </div>
</div>

<style>
  .sort-label {
    margin-right: 0.2rem;
    color: var(--text-faint);
    font-size: 0.62rem;
    font-weight: 600;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
</style>
