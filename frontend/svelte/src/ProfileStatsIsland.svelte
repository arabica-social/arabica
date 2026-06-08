<script lang="ts">
  import { onMount } from 'svelte';

  type Stats = {
    brews: string;
    beans: string;
    roasters: string;
    grinders: string;
    brewers: string;
  };

  const labels: Array<{ key: keyof Stats; label: string }> = [
    { key: 'brews', label: 'Brews' },
    { key: 'beans', label: 'Beans' },
    { key: 'roasters', label: 'Roasters' },
    { key: 'grinders', label: 'Grinders' },
    { key: 'brewers', label: 'Brewers' }
  ];

  let { target }: { target: HTMLElement } = $props();
  let stats = $state<Stats>({
    brews: '-',
    beans: '-',
    roasters: '-',
    grinders: '-',
    brewers: '-'
  });

  function readStatsData() {
    const statsData = document.getElementById('profile-stats-data');
    if (!statsData) return;
    stats = {
      brews: statsData.dataset.brews || '-',
      beans: statsData.dataset.beans || '-',
      roasters: statsData.dataset.roasters || '-',
      grinders: statsData.dataset.grinders || '-',
      brewers: statsData.dataset.brewers || '-'
    };
  }

  onMount(() => {
    const handleSwap = (event: Event) => {
      const detail = (event as CustomEvent<{ target?: Element }>).detail;
      if ((detail?.target as HTMLElement | undefined)?.id === 'profile-content') {
        readStatsData();
      }
    };

    document.body.addEventListener('htmx:afterSwap', handleSwap);
    readStatsData();
    return () => {
      document.body.removeEventListener('htmx:afterSwap', handleSwap);
    };
  });
</script>

{#each labels as item}
  <div class="card-sm p-4">
    <div class="text-2xl font-bold text-secondary">{stats[item.key]}</div>
    <div class="text-sm text-emphasis">{item.label}</div>
  </div>
{/each}
