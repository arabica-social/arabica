<script lang="ts">
  import { onMount } from 'svelte';

  let { initialShowPours = false }: { initialShowPours?: boolean } = $props();

  let showPours = $state(false);

  onMount(() => {
    showPours = initialShowPours;

    const handleVisibility = (event: Event) => {
      showPours = !!(event as CustomEvent<{ visible: boolean }>).detail?.visible;
    };

    document.addEventListener('brew-pours-visibility-change', handleVisibility);
    return () => {
      document.removeEventListener('brew-pours-visibility-change', handleVisibility);
    };
  });
</script>

<p class="text-helper">
  {showPours ? 'Total water (pours tracked separately below)' : 'Total water used'}
</p>
