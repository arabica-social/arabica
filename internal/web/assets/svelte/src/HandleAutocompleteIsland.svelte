<script lang="ts">
  import { onMount } from "svelte";

  type Actor = {
    handle: string;
    displayName?: string;
    avatar?: string;
  };

  let {
    input,
    target,
  }: {
    input: HTMLInputElement;
    target: HTMLElement;
  } = $props();

  let actors = $state<Actor[]>([]);
  let open = $state(false);
  let loading = $state(false);
  let searched = $state(false);
  let debounceTimer: ReturnType<typeof setTimeout> | undefined;
  let abortController: AbortController | undefined;
  // True while we are programmatically setting the input value (e.g. after the
  // user picks a suggestion). Prevents the dispatched "input" event from
  // re-triggering our own typeahead search and reopening the dropdown.
  let suppressSearch = false;

  // The dropdown is rendered as a top-layer popover portaled into <body>.
  // Why a popover rather than a plain portaled <div>:
  //  - The login form lives inside a top-layer <dialog>. A normal element
  //    portaled into <body> would paint BEHIND the dialog's ::backdrop.
  //    showPopover() promotes the element to the browser's top layer, so it
  //    renders above the dialog and its backdrop.
  //  - Because the popover is a child of <body> (which has no transform), its
  //    position:fixed resolves against the viewport. A transform on a DOM
  //    ancestor (e.g. .modal-dialog[open]'s scale(1)) establishes the
  //    containing block for fixed descendants EVEN in the top layer, which is
  //    why we portal to <body> and not to the <dialog>.
  //  - As a separate top-layer element it is not a child of the dialog box,
  //    so the dialog cannot clip or scroll it — long lists extend past the
  //    modal's bottom edge, which is the desired behavior here.
  let popoverEl = $state<HTMLDivElement | null>(null);
  let dropdownStyle = $state("");

  function updateDropdownPosition() {
    if (!open) return;
    const rect = input.getBoundingClientRect();
    const below = window.innerHeight - rect.bottom;
    const above = rect.top;
    // Prefer opening below the input; flip above if there's more room there
    // and not enough below for even a short list.
    const minHeight = 8 * 16; // 8rem (~128px) — comfortable minimum
    const preferAbove = below < minHeight && above > below;
    const available = preferAbove ? above : below;
    // 15rem matches .handle-dropdown max-height; never exceed available space
    // minus a small gap so it never overflows the viewport.
    const maxH = Math.min(15 * 16, available - 8);
    const top = preferAbove ? rect.top - maxH - 4 : rect.bottom + 4;
    // position:fixed coords against the viewport (the popover is a child of
    // <body>, which has no transform, so fixed resolves to the viewport).
    dropdownStyle =
      `position:fixed;left:${rect.left}px;width:${rect.width}px;` +
      `top:${top}px;max-height:${maxH}px;`;
  }

  function handleScrollOrResize() {
    updateDropdownPosition();
  }

  function safeAvatar(actor: Actor) {
    const avatar = actor.avatar || "";
    if (avatar.startsWith("https://") || avatar.startsWith("/static/")) {
      return avatar;
    }
    return "/static/icon-placeholder.svg";
  }

  function displayName(actor: Actor) {
    return actor.displayName || actor.handle;
  }

  function clearResults() {
    window.clearTimeout(debounceTimer);
    actors = [];
    open = false;
    searched = false;
  }

  // Open: show the popover (promotes to top layer) and place it under the
  // input. Svelte binds popoverEl after mount, so guard against the first
  // render where it is still null.
  function openDropdown() {
    if (!popoverEl) return;
    if (typeof popoverEl.showPopover !== "function") return; // unsupported
    try {
      popoverEl.showPopover();
    } catch {
      // Already showing or not popover-capable; ignore.
    }
    updateDropdownPosition();
  }

  function closeDropdown() {
    if (popoverEl && typeof popoverEl.hidePopover === "function") {
      try {
        popoverEl.hidePopover();
      } catch {
        // Already hidden; ignore.
      }
    }
  }

  async function searchActors(query: string) {
    const trimmed = query.trim();
    if (trimmed.length < 3) {
      clearResults();
      return;
    }

    abortController?.abort();
    abortController = new AbortController();
    loading = true;
    searched = true;

    try {
      const response = await fetch(
        `/api/search-actors?q=${encodeURIComponent(trimmed)}`,
        {
          signal: abortController.signal,
        },
      );
      if (!response.ok) {
        actors = [];
        open = false;
        return;
      }

      const data = await response.json();
      actors = Array.isArray(data?.actors) ? data.actors : [];
      open = true;
    } catch (error) {
      if ((error as Error).name !== "AbortError") {
        console.error("Error searching actors:", error);
      }
    } finally {
      loading = false;
    }
  }

  function scheduleSearch() {
    if (suppressSearch) return;
    window.clearTimeout(debounceTimer);
    debounceTimer = window.setTimeout(() => {
      void searchActors(input.value);
    }, 300);
  }

  function selectActor(actor: Actor) {
    input.value = actor.handle;
    input.dispatchEvent(new Event("input", { bubbles: true }));
    clearResults();
  }

  // Drive the popover from the `open` state. An $effect (re)runs whenever
  // `open` changes so we show/hide in sync with search results.
  $effect(() => {
    if (open) {
      openDropdown();
    } else {
      closeDropdown();
    }
  });

  // Reposition whenever results arrive (the list height changes) or the
  // dropdown opens.
  $effect(() => {
    if (open) updateDropdownPosition();
  });

  function handleDocumentClick(event: MouseEvent) {
    const clicked = event.target;
    if (!(clicked instanceof Node)) return;
    // The dropdown is portaled to <body>, so treat clicks inside it as
    // inside the component (don't dismiss when clicking a suggestion).
    if (
      !input.contains(clicked) &&
      !target.contains(clicked) &&
      !(popoverEl && popoverEl.contains(clicked))
    ) {
      open = false;
    }
  }

  onMount(() => {
    input.addEventListener("input", scheduleSearch);
    input.addEventListener("focus", () => {
      if (searched && input.value.trim().length >= 3) open = true;
    });
    document.addEventListener("click", handleDocumentClick);
    window.addEventListener("resize", handleScrollOrResize);
    // Capture phase so repositioning also fires for scroll within the modal.
    window.addEventListener("scroll", handleScrollOrResize, true);

    return () => {
      input.removeEventListener("input", scheduleSearch);
      document.removeEventListener("click", handleDocumentClick);
      window.removeEventListener("resize", handleScrollOrResize);
      window.removeEventListener("scroll", handleScrollOrResize, true);
      closeDropdown();
      window.clearTimeout(debounceTimer);
      abortController?.abort();
    };
  });
</script>

<!--
  The popover lives in <body> via the `popover` attribute + showPopover().
  `popover=manual` so it isn't auto-dismissed by clicks/Escape (we manage
  dismissal ourselves via handleDocumentClick). `margin:0` zeroes the
  UA popover inset so our inline top/left are absolute from the viewport.
-->
<div
  bind:this={popoverEl}
  popover="manual"
  class="handle-dropdown"
  style={dropdownStyle}
>
  {#if loading && actors.length === 0}
    <div class="handle-no-results">Searching...</div>
  {:else if actors.length === 0}
    <div class="handle-no-results">No accounts found</div>
  {:else}
    {#each actors as actor}
      <button
        type="button"
        class="handle-result"
        data-handle={actor.handle}
        onclick={() => selectActor(actor)}
      >
        <img
          src={safeAvatar(actor)}
          alt=""
          width="32"
          height="32"
          class="handle-result-avatar"
          onerror={(event) => {
            (event.currentTarget as HTMLImageElement).src =
              "/static/icon-placeholder.svg";
          }}
        />
        <span class="handle-result-text">
          <span class="handle-name">{displayName(actor)}</span>
          <span class="handle-at">@{actor.handle}</span>
        </span>
      </button>
    {/each}
  {/if}
</div>
