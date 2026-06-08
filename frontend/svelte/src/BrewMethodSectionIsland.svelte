<script lang="ts">
  import { onMount } from 'svelte';

  let { target, method }: { target: HTMLElement; method: string } = $props();

  function setCategory(category: string) {
    target.hidden = category !== method;
  }

  onMount(() => {
    const handleCategory = (event: Event) => {
      const detail = (event as CustomEvent<{ category?: string }>).detail;
      setCategory(detail?.category || '');
    };

    document.addEventListener('brew-method-category-change', handleCategory);
    return () => {
      document.removeEventListener('brew-method-category-change', handleCategory);
    };
  });
</script>
