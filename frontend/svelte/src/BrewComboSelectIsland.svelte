<script lang="ts">
  import { onMount } from 'svelte';
  import type { AppCacheAPI } from './appCache';
  import {
    comboSelectEntities,
    type EntityConfig,
    type EntityRecord,
    type ExtraField,
    type Suggestion
  } from './comboSelectRegistry';

  type ComboItem =
    | { type: 'user'; entity: EntityRecord }
    | { type: 'closed'; entity: EntityRecord }
    | { type: 'community'; suggestion: Suggestion }
    | { type: 'create'; name: string };

  let {
    target,
    entityType,
    apiEndpoint,
    suggestEndpoint,
    inputName,
    placeholder = 'Search...',
    sectionLabel,
    required = false,
    passthrough = false,
    allowCreate = true,
    initialRKey = '',
    initialLabel = ''
  }: {
    target: HTMLElement;
    entityType: string;
    apiEndpoint: string;
    suggestEndpoint: string;
    inputName: string;
    placeholder?: string;
    sectionLabel: string;
    required?: boolean;
    passthrough?: boolean;
    allowCreate?: boolean;
    initialRKey?: string;
    initialLabel?: string;
  } = $props();

  let query = $state('');
  let selectedRKey = $state('');
  let selectedLabel = $state('');
  let isOpen = $state(false);
  let highlightIndex = $state(-1);
  let isCreating = $state(false);
  let userResults = $state<EntityRecord[]>([]);
  let closedResults = $state<EntityRecord[]>([]);
  let communityResults = $state<Suggestion[]>([]);
  let showCreateForm = $state(false);
  let createFormData = $state<EntityRecord>({});
  let cachedData = $state<Record<string, any> | null>(null);
  let suggestTimer: ReturnType<typeof setTimeout> | undefined;

  function getAppCache(): AppCacheAPI | undefined {
    return window.AppCache;
  }

  function getEntityConfig() {
    return (comboSelectEntities[entityType] || {}) as EntityConfig;
  }

  let exactMatch = $derived.by(() => {
    const q = query.trim().toLowerCase();
    if (!q) return false;
    const matchesName = (entity: EntityRecord) => formatLabel(entity).toLowerCase() === q;
    return (
      userResults.some(matchesName) ||
      closedResults.some(matchesName) ||
      communityResults.some((suggestion) => (suggestion.name || '').toLowerCase() === q)
    );
  });
  let extraFields = $derived(getEntityConfig().extraFields || []);

  let allItems: ComboItem[] = $derived([
    ...userResults.map((entity): ComboItem => ({ type: 'user', entity })),
    ...closedResults.map((entity): ComboItem => ({ type: 'closed', entity })),
    ...communityResults.map((suggestion): ComboItem => ({ type: 'community', suggestion })),
    ...(allowCreate && !passthrough && query.trim() && !exactMatch
      ? ([{ type: 'create', name: query.trim() }] as ComboItem[])
      : [])
  ]);

  function formatLabel(entity: EntityRecord | Suggestion) {
    return getEntityConfig().formatLabel?.(entity) || entity.name || (entity as EntityRecord).Name || '';
  }

  function getRKey(entity: EntityRecord) {
    return entity.rkey || entity.RKey || '';
  }

  function getCachedEntities(cache: Record<string, any> = {}) {
    switch (entityType) {
      case 'bean':
        return cache.beans || [];
      case 'brewer':
      case 'oolongBrewer':
        return cache.brewers || [];
      case 'grinder':
        return cache.grinders || [];
      case 'recipe':
      case 'oolongRecipe':
        return cache.recipes || [];
      case 'roaster':
        return cache.roasters || [];
      case 'tea':
        return cache.teas || [];
      case 'vendor':
        return cache.vendors || [];
      case 'cafe':
        return cache.cafes || [];
      case 'oolongVessel':
        return cache.vessels || [];
      case 'oolongInfuser':
        return cache.infusers || [];
      default:
        return [];
    }
  }

  function refreshCachedData() {
    const appCache = getAppCache();
    if (!appCache) return;
    const cached = appCache.getCachedData?.();
    if (cached) {
      cachedData = cached;
    }
  }

  async function primeCache() {
    const appCache = getAppCache();
    if (!appCache) {
      search(false);
      return;
    }

    refreshCachedData();
    search(false);
    try {
      const freshData = await appCache.getData?.();
      if (freshData) {
        cachedData = freshData;
      }
      search(false);
    } catch (error) {
      console.warn('svelte combo-select: failed to load user data cache:', error);
    }
  }

  function search(openOnMatch = false) {
    const q = query.trim().toLowerCase();
    const entities = getCachedEntities(cachedData || {});
    const matches = q
      ? entities.filter((entity: EntityRecord) => formatLabel(entity).toLowerCase().includes(q))
      : entities.slice(0, 10);

    if (entityType === 'bean') {
      userResults = matches.filter((bean: EntityRecord) => !bean.closed && !bean.Closed);
      closedResults = q ? matches.filter((bean: EntityRecord) => bean.closed || bean.Closed) : [];
    } else {
      userResults = matches;
      closedResults = [];
    }

    highlightIndex = -1;
    if (openOnMatch && !isOpen && query) isOpen = true;

    clearTimeout(suggestTimer);
    if (q.length >= 2 && suggestEndpoint) {
      suggestTimer = setTimeout(() => fetchSuggestions(q), 400);
    } else {
      communityResults = [];
    }
  }

  async function fetchSuggestions(q: string) {
    try {
      const response = await fetch(`${suggestEndpoint}?q=${encodeURIComponent(q)}&limit=5`, {
        credentials: 'same-origin'
      });
      if (!response.ok) return;
      const data = await response.json();
      const ownNames = new Set(
        getCachedEntities(cachedData || {}).map((entity: EntityRecord) =>
          (entity.name || entity.Name || '').toLowerCase()
        )
      );
      communityResults = (data || []).filter(
        (suggestion: Suggestion) => !ownNames.has((suggestion.name || '').toLowerCase())
      );
    } catch (error) {
      console.error('Suggestion fetch failed:', error);
    }
  }

  function dispatchChange(detail: Record<string, any>) {
    target.dispatchEvent(new CustomEvent('combo-change', { detail, bubbles: true }));
  }

  function selectEntity(entity: EntityRecord) {
    selectedRKey = getRKey(entity);
    selectedLabel = formatLabel(entity);
    query = selectedLabel;
    isOpen = false;
    dispatchChange({ entityType, rkey: selectedRKey, entity });
  }

  async function selectSuggestion(suggestion: Suggestion) {
    if (passthrough) {
      const parts = (suggestion.source_uri || '').split('/');
      selectedRKey = parts.length >= 5 ? parts[4] : '';
      selectedLabel = formatLabel(suggestion);
      query = selectedLabel;
      isOpen = false;
      dispatchChange({ entityType, rkey: selectedRKey, suggestion });
      return;
    }

    const data = getEntityConfig().formatCreateData?.(suggestion.name || '', suggestion) || {
      name: suggestion.name || ''
    };
    if (suggestion.source_uri) data.source_ref = suggestion.source_uri;

    if (extraFields.length > 0) {
      createFormData = { ...data };
      for (const field of extraFields) {
        if (!(field.name in createFormData)) createFormData[field.name] = '';
      }
      showCreateForm = true;
      isOpen = false;
      return;
    }
    await createEntity(data);
  }

  function startCreate() {
    if (!allowCreate) return;
    const name = query.trim();
    if (!name) return;
    if (extraFields.length > 0) {
      createFormData = { name };
      for (const field of extraFields) createFormData[field.name] = '';
      showCreateForm = true;
      isOpen = false;
      return;
    }
    void createEntity({ name });
  }

  async function createEntity(data: EntityRecord) {
    if (!apiEndpoint) return;
    isCreating = true;
    try {
      const appCache = getAppCache();
      const response = await fetch(apiEndpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'same-origin',
        body: JSON.stringify(data)
      });
      if (!response.ok) {
        if (response.status === 401) window.__showSessionExpiredModal?.();
        throw new Error(`Create failed: ${response.status}`);
      }
      const created = await response.json();
      selectedRKey = created.rkey || created.RKey || '';
      selectedLabel = data.name || formatLabel(created);
      query = selectedLabel;
      isOpen = false;
      showCreateForm = false;
      createFormData = {};
      await appCache?.invalidateAndRefresh?.();
      dispatchChange({ entityType, rkey: selectedRKey, entity: created });
    } catch (error) {
      console.error('Failed to create entity:', error);
    } finally {
      isCreating = false;
    }
  }

  function submitCreateForm() {
    void createEntity({ ...createFormData });
  }

  function cancelCreateForm() {
    showCreateForm = false;
    createFormData = {};
  }

  function clear() {
    selectedRKey = '';
    selectedLabel = '';
    query = '';
    userResults = [];
    closedResults = [];
    communityResults = [];
    dispatchChange({ entityType, rkey: '', entity: null });
  }

  function closeSoon() {
    setTimeout(() => {
      isOpen = false;
      if (selectedRKey && query !== selectedLabel) query = selectedLabel;
    }, 150);
  }

  function moveDown() {
    if (!isOpen) isOpen = true;
    if (allItems.length === 0) return;
    highlightIndex = (highlightIndex + 1) % allItems.length;
  }

  function moveUp() {
    if (!isOpen) isOpen = true;
    if (allItems.length === 0) return;
    highlightIndex = highlightIndex <= 0 ? allItems.length - 1 : highlightIndex - 1;
  }

  function selectHighlighted() {
    const item = allItems[highlightIndex];
    if (!item) return;
    if (item.type === 'user' || item.type === 'closed') selectEntity(item.entity);
    if (item.type === 'community') void selectSuggestion(item.suggestion);
    if (item.type === 'create') startCreate();
  }

  onMount(() => {
    query = initialLabel;
    selectedRKey = initialRKey;
    selectedLabel = initialLabel;

    const handleSet = (event: Event) => {
      const detail = (event as CustomEvent<{ rkey?: string; label?: string }>).detail;
      if (detail?.rkey) {
        selectedRKey = detail.rkey;
        selectedLabel = detail.label || '';
        query = selectedLabel;
      } else {
        clear();
      }
    };
    target.addEventListener('combo-set', handleSet);
    const cacheListener = (data: Record<string, any>) => {
      cachedData = data;
      search(false);
    };
    window.AppCache?.addListener?.(cacheListener);
    void primeCache();
    return () => {
      target.removeEventListener('combo-set', handleSet);
      window.AppCache?.removeListener?.(cacheListener);
      clearTimeout(suggestTimer);
    };
  });
</script>

<input type="hidden" name={inputName} value={selectedRKey} required={required} />
<div class="relative">
  <input
    type="text"
    bind:value={query}
    oninput={() => search(true)}
    onfocus={() => {
      isOpen = true;
      search(true);
    }}
    onblur={closeSoon}
    onkeydown={(event) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        isOpen = false;
      }
      if (event.key === 'ArrowDown') {
        event.preventDefault();
        moveDown();
      }
      if (event.key === 'ArrowUp') {
        event.preventDefault();
        moveUp();
      }
      if (event.key === 'Enter') {
        event.preventDefault();
        selectHighlighted();
      }
    }}
    {placeholder}
    class="w-full form-input-lg"
    autocomplete="off"
    role="combobox"
    aria-autocomplete="list"
    aria-controls={`${entityType}-combo-listbox`}
    aria-expanded={isOpen && (allItems.length > 0 || query.trim()) ? 'true' : 'false'}
    aria-label="Search and select"
  />
  {#if selectedRKey}
    <button
      type="button"
      onclick={clear}
      class="absolute right-2 top-1/2 -translate-y-1/2 text-placeholder hover:text-muted"
      aria-label="Clear selection"
    >
      ×
    </button>
  {/if}
</div>

{#if isOpen && (allItems.length > 0 || query.trim())}
  <div id={`${entityType}-combo-listbox`} role="listbox" tabindex="-1" class="combo-dropdown" onmousedown={(event) => event.preventDefault()}>
    {#if isCreating}
      <div class="combo-creating">Creating...</div>
    {:else}
      {#if userResults.length > 0}
        <div class="combo-section-label">{sectionLabel}</div>
        {#each userResults as entity, index}
          <button
            type="button"
            class="combo-item"
            role="option"
            aria-selected={highlightIndex === index}
            data-highlighted={highlightIndex === index}
            onmouseenter={() => (highlightIndex = index)}
            onclick={() => selectEntity(entity)}
          >
            {formatLabel(entity)}
          </button>
        {/each}
      {/if}
      {#if closedResults.length > 0}
        <div class="combo-section-label">Closed bags</div>
        {#each closedResults as entity, index}
          <button
            type="button"
            class="combo-item opacity-60"
            role="option"
            aria-selected={highlightIndex === userResults.length + index}
            data-highlighted={highlightIndex === userResults.length + index}
            onmouseenter={() => (highlightIndex = userResults.length + index)}
            onclick={() => selectEntity(entity)}
          >
            {formatLabel(entity)}
          </button>
        {/each}
      {/if}
      {#if communityResults.length > 0}
        <div class="combo-section-label">Community</div>
        {#each communityResults as suggestion, index}
          <button
            type="button"
            class="combo-item"
            role="option"
            aria-selected={highlightIndex === userResults.length + closedResults.length + index}
            data-highlighted={highlightIndex === userResults.length + closedResults.length + index}
            onmouseenter={() => (highlightIndex = userResults.length + closedResults.length + index)}
            onclick={() => selectSuggestion(suggestion)}
          >
            <div>{suggestion.name}</div>
            <div class="combo-item-sub">
              {#if suggestion.fields?.origin}{suggestion.fields.origin}{/if}
              {#if suggestion.fields?.origin && suggestion.fields?.roastLevel} · {/if}
              {#if suggestion.fields?.roastLevel}{suggestion.fields.roastLevel}{/if}
              {#if suggestion.fields?.location}{suggestion.fields.location}{/if}
              {#if (suggestion.count || 0) > 1} · {suggestion.count} users{/if}
            </div>
          </button>
        {/each}
      {/if}
      {#if allowCreate && query.trim() && !exactMatch && !passthrough}
        <button
          type="button"
          class="combo-item-create"
          role="option"
          aria-selected={highlightIndex === userResults.length + closedResults.length + communityResults.length}
          data-highlighted={highlightIndex === userResults.length + closedResults.length + communityResults.length}
          onmouseenter={() => (highlightIndex = userResults.length + closedResults.length + communityResults.length)}
          onclick={startCreate}
        >
          Create "{query.trim()}"
        </button>
      {/if}
      {#if allItems.length === 0 && query.trim()}
        <div class="combo-creating">No matches found</div>
      {/if}
    {/if}
  </div>
{/if}

<div class="sr-only" aria-live="polite">{allItems.length} results available</div>

{#if showCreateForm}
  <div class="mt-2 p-3 rounded-lg" style="background: var(--surface-bg); border: 1px solid var(--surface-border);">
    <p class="text-sm font-medium text-primary mb-2">
      Creating: <span class="font-semibold">{createFormData.name}</span>
    </p>
    <div class="space-y-2">
      {#each extraFields as field}
        <div>
          {#if field.type === 'select'}
            <select bind:value={createFormData[field.name]} class="w-full form-input text-sm">
              <option value="">{field.label} (optional)</option>
              {#each field.options || [] as option}
                <option value={option}>{option}</option>
              {/each}
            </select>
          {:else}
            <input
              type={field.type === 'url' ? 'url' : 'text'}
              bind:value={createFormData[field.name]}
              placeholder={field.placeholder || `${field.label} (optional)`}
              class="w-full form-input text-sm"
            />
          {/if}
        </div>
      {/each}
    </div>
    <div class="flex gap-2 mt-3">
      <button type="button" class="btn-primary text-sm" disabled={isCreating} onclick={submitCreateForm}>
        Create
      </button>
      <button type="button" class="btn-secondary text-sm" onclick={cancelCreateForm}>Cancel</button>
    </div>
  </div>
{/if}
