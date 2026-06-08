<script lang="ts">
  import { onMount } from "svelte";

  type Suggestion = {
    name: string;
    source_uri: string;
    fields?: Record<string, string>;
    count?: number;
  };

  let {
    target,
    endpoint,
    entityType,
    placeholder,
  }: {
    target: HTMLElement;
    endpoint: string;
    entityType: string;
    placeholder: string;
  } = $props();

  let query = $state("");
  let sourceRef = $state("");
  let originalName = $state("");
  let suggestions = $state<Suggestion[]>([]);
  let showSuggestions = $state(false);
  let searchTimer: number | undefined;
  let blurTimer: number | undefined;

  function form() {
    return target.closest("form");
  }

  function setInput(name: string, value: string | undefined) {
    if (!value) {
      return;
    }
    const input = form()?.querySelector<HTMLInputElement | HTMLTextAreaElement>(
      `[name="${name}"]`,
    );
    if (!input) {
      return;
    }
    input.value = value;
    input.dispatchEvent(new Event("input", { bubbles: true }));
  }

  function setSelect(name: string, value: string | undefined) {
    if (!value) {
      return;
    }
    const select = form()?.querySelector<HTMLSelectElement>(
      `select[name="${name}"]`,
    );
    if (!select) {
      return;
    }
    const match = Array.from(select.options).find(
      (option) =>
        option.value === value ||
        option.value.toLowerCase() === value.toLowerCase(),
    );
    if (!match) {
      return;
    }
    select.value = match.value;
    select.dispatchEvent(new Event("change", { bubbles: true }));
  }

  function resetSourceIfEdited() {
    if (!originalName || query.toLowerCase() === originalName.toLowerCase()) {
      return;
    }
    sourceRef = "";
    originalName = "";
  }

  async function search() {
    resetSourceIfEdited();
    if (query.length < 2) {
      suggestions = [];
      showSuggestions = false;
      return;
    }
    try {
      const response = await fetch(
        `${endpoint}?q=${encodeURIComponent(query)}&limit=10`,
        {
          credentials: "same-origin",
        },
      );
      if (!response.ok) {
        return;
      }
      const data: unknown = await response.json();
      suggestions = Array.isArray(data) ? (data as Suggestion[]) : [];
      showSuggestions = suggestions.length > 0;
    } catch {
      // Suggestions are optional.
    }
  }

  function onInput() {
    window.clearTimeout(searchTimer);
    searchTimer = window.setTimeout(() => {
      void search();
    }, 300);
  }

  function onFocus() {
    if (suggestions.length > 0) {
      showSuggestions = true;
    }
  }

  function onBlur() {
    window.clearTimeout(blurTimer);
    blurTimer = window.setTimeout(() => {
      showSuggestions = false;
    }, 200);
  }

  function selectSuggestion(suggestion: Suggestion) {
    const fields = suggestion.fields || {};
    query = suggestion.name;
    sourceRef = suggestion.source_uri;
    originalName = suggestion.name;
    showSuggestions = false;

    if (entityType === "roaster") {
      setInput("location", fields.location);
      setInput("website", fields.website);
      return;
    }
    if (entityType === "grinder") {
      setSelect("grinder_type", fields.grinderType);
      setSelect("burr_type", fields.burrType);
      setInput("link", fields.link);
      return;
    }
    if (entityType === "brewer") {
      setInput("brewer_type", fields.brewerType);
      setInput("link", fields.link);
      return;
    }
    if (entityType === "bean") {
      setInput("origin", fields.origin);
      setSelect("roast_level", fields.roastLevel);
      setInput("process", fields.process);
      setInput("link", fields.link);
      return;
    }

    for (const [name, value] of Object.entries(fields)) {
      setInput(name, value);
      setSelect(name, value);
    }
  }

  function secondaryText(suggestion: Suggestion) {
    const fields = suggestion.fields || {};
    if (entityType === "brewer") return fields.brewerType || "";
    if (entityType === "grinder") return fields.grinderType || "";
    if (entityType === "roaster") return fields.location || "";
    if (entityType === "bean") return fields.origin || "";
    return "";
  }

  onMount(() => {
    return () => {
      window.clearTimeout(searchTimer);
      window.clearTimeout(blurTimer);
    };
  });
</script>

<input
  type="text"
  name="name"
  {placeholder}
  required
  class="w-full form-input"
  bind:value={query}
  oninput={onInput}
  onblur={onBlur}
  onfocus={onFocus}
  autocomplete="off"
/>
<input type="hidden" name="source_ref" value={sourceRef} />

{#if showSuggestions && suggestions.length > 0}
  <div class="suggestions-dropdown">
    {#each suggestions as suggestion}
      <button
        type="button"
        class="suggestions-item"
        onmousedown={(event) => {
          event.preventDefault();
          selectSuggestion(suggestion);
        }}
      >
        <span class="font-medium">{suggestion.name}</span>
        {#if secondaryText(suggestion)}
          <span class="text-xs text-faint">{secondaryText(suggestion)}</span>
        {/if}
        {#if (suggestion.count || 0) > 1}
          <span class="text-xs text-placeholder">{suggestion.count} users</span>
        {/if}
      </button>
    {/each}
  </div>
{/if}
