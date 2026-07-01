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

  function handleDocumentClick(event: MouseEvent) {
    const clicked = event.target;
    if (!(clicked instanceof Node)) return;
    if (!input.contains(clicked) && !target.contains(clicked)) {
      open = false;
    }
  }

  onMount(() => {
    input.addEventListener("input", scheduleSearch);
    input.addEventListener("focus", () => {
      if (searched && input.value.trim().length >= 3) open = true;
    });
    document.addEventListener("click", handleDocumentClick);

    return () => {
      input.removeEventListener("input", scheduleSearch);
      document.removeEventListener("click", handleDocumentClick);
      window.clearTimeout(debounceTimer);
      abortController?.abort();
    };
  });
</script>

{#if open}
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
{/if}
