<script lang="ts">
  import { onMount } from "svelte";

  let { target }: { target: HTMLElement } = $props();

  let query = $state("");
  let sort = $state("recent");
  let visibleCount = $state(0);
  let totalCount = $state(0);

  const collator = new Intl.Collator(undefined, {
    sensitivity: "base",
    numeric: true,
  });

  function contentRoot() {
    const selector = target.dataset.contentSelector || "#manage-content";
    return document.querySelector<HTMLElement>(selector);
  }

  function activePanel(root: HTMLElement) {
    const activeTab =
      target.closest<HTMLElement>("[data-svelte-manage-tabs]")?.dataset
        .activeTab || "";
    if (activeTab) {
      const panel = root.querySelector<HTMLElement>(
        `[data-tab-panel="${CSS.escape(activeTab)}"]`,
      );
      if (panel) {
        return panel;
      }
    }
    return root.querySelector<HTMLElement>("[data-tab-panel]") || root;
  }

  function cardText(card: HTMLElement) {
    return (card.dataset.manageSearch || card.textContent || "").toLowerCase();
  }

  function cardName(card: HTMLElement) {
    return (card.dataset.manageName || card.textContent || "").trim();
  }

  function cardCreated(card: HTMLElement) {
    const parsed = Number(card.dataset.manageCreated || "0");
    return Number.isFinite(parsed) ? parsed : 0;
  }

  function compareCards(a: HTMLElement, b: HTMLElement) {
    if (sort === "name") {
      return collator.compare(cardName(a), cardName(b));
    }
    const diff = cardCreated(a) - cardCreated(b);
    if (diff === 0) {
      return collator.compare(cardName(a), cardName(b));
    }
    return sort === "oldest" ? diff : -diff;
  }

  function applyControls() {
    const root = contentRoot();
    if (!root) {
      visibleCount = 0;
      totalCount = 0;
      return;
    }

    const panel = activePanel(root);
    const cards = Array.from(
      panel.querySelectorAll<HTMLElement>("[data-manage-card]"),
    );
    const normalizedQuery = query.trim().toLowerCase();
    let nextVisible = 0;

    cards.forEach((card) => {
      const matches =
        normalizedQuery === "" || cardText(card).includes(normalizedQuery);
      card.toggleAttribute("hidden", !matches);
      if (matches) {
        nextVisible += 1;
      }
    });

    panel
      .querySelectorAll<HTMLElement>("[data-manage-collection]")
      .forEach((collection) => {
        const collectionCards = Array.from(
          collection.querySelectorAll<HTMLElement>(
            ":scope > [data-manage-card]",
          ),
        );
        collectionCards.sort(compareCards).forEach((card) => {
          collection.appendChild(card);
        });
      });

    panel
      .querySelectorAll<HTMLElement>("[data-manage-section]")
      .forEach((section) => {
        const hasVisibleCard =
          section.querySelector<HTMLElement>(
            "[data-manage-card]:not([hidden])",
          ) !== null;
        section.toggleAttribute(
          "hidden",
          !hasVisibleCard && normalizedQuery !== "",
        );
      });

    visibleCount = nextVisible;
    totalCount = cards.length;
  }

  onMount(() => {
    applyControls();
    const rerender = () => requestAnimationFrame(applyControls);
    document.addEventListener("htmx:afterSwap", rerender);
    target
      .closest<HTMLElement>("[data-svelte-manage-tabs]")
      ?.addEventListener("click", rerender);

    return () => {
      document.removeEventListener("htmx:afterSwap", rerender);
      target
        .closest<HTMLElement>("[data-svelte-manage-tabs]")
        ?.removeEventListener("click", rerender);
    };
  });

  $effect(() => {
    query;
    sort;
    applyControls();
  });
</script>

<div
  class="mb-5 rounded-lg border border-brown-200 bg-surface/70 p-3 shadow-sm"
>
  <div class="flex flex-col gap-3 sm:flex-row sm:items-end">
    <label class="flex-1 text-sm font-medium text-secondary">
      <span class="mb-1 block">Search current collection</span>
      <input
        class="form-input w-full"
        type="search"
        bind:value={query}
        placeholder="Search by name or details…"
        aria-label="Search current manage collection"
      />
    </label>
    <label class="text-sm font-medium text-secondary sm:w-44">
      <span class="mb-1 block">Sort</span>
      <select
        class="form-select w-full"
        bind:value={sort}
        aria-label="Sort current manage collection"
      >
        <option value="recent">Newest first</option>
        <option value="oldest">Oldest first</option>
        <option value="name">Name A–Z</option>
      </select>
    </label>
  </div>
  {#if totalCount > 0}
    <p class="mt-2 text-xs text-muted" aria-live="polite">
      Showing {visibleCount} of {totalCount} records in this tab.
    </p>
  {/if}
</div>
