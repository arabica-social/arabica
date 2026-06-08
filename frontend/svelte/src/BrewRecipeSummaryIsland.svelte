<script lang="ts">
  import { onMount } from 'svelte';

  let { target }: { target: HTMLElement } = $props();

  let active = $state(false);
  let expanded = $state(false);
  let summary = $state('');

  function updateOverrideVisibility() {
    document.querySelectorAll<HTMLElement>('[data-svelte-brew-recipe-overrides]').forEach((section) => {
      section.hidden = active && !expanded;
    });
  }

  function toggleExpanded() {
    expanded = !expanded;
    updateOverrideVisibility();
  }

  $effect(() => {
    target.hidden = !active;
    updateOverrideVisibility();
  });

  onMount(() => {
    const handleRecipeState = (event: Event) => {
      const detail = (event as CustomEvent<{ active?: boolean; summary?: string }>).detail;
      active = !!detail?.active;
      expanded = false;
      summary = detail?.summary || '';
    };

    document.addEventListener('brew-recipe-state-change', handleRecipeState);
    updateOverrideVisibility();

    return () => {
      document.removeEventListener('brew-recipe-state-change', handleRecipeState);
    };
  });
</script>

{#if active}
  <div class="section-box">
    <div class="flex items-center justify-between gap-2">
      <p class="text-sm text-emphasis flex-1">{summary}</p>
      <button type="button" onclick={toggleExpanded} class="text-sm btn-secondary">
        {expanded ? 'Collapse' : 'Edit'}
      </button>
    </div>
  </div>
{/if}
