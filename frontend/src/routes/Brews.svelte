<script>
  import { onMount } from 'svelte';
  import { authStore } from '../stores/auth.js';
  import { cacheStore } from '../stores/cache.js';
  import { navigate } from '../lib/router.js';
  import { api } from '../lib/api.js';
  
  let brews = [];
  let loading = true;
  let deleting = null; // Track which brew is being deleted
  
  $: isAuthenticated = $authStore.isAuthenticated;
  
  onMount(async () => {
    if (!isAuthenticated) {
      navigate('/login');
      return;
    }
    
    await cacheStore.load();
    brews = $cacheStore.brews || [];
    loading = false;
  });
  
  function formatDate(dateStr) {
    if (!dateStr) return '';
    const date = new Date(dateStr);
    return date.toLocaleDateString('en-US', { 
      year: 'numeric', 
      month: 'short', 
      day: 'numeric' 
    });
  }
  
  function hasValue(val) {
    return val !== null && val !== undefined && val !== '';
  }
  
  function getWaterDisplay(brew) {
    if (hasValue(brew.water_amount) && brew.water_amount > 0) {
      return `💧 ${brew.water_amount}ml water`;
    }
    
    // If water_amount is 0 or not set, sum from pours
    if (brew.pours && brew.pours.length > 0) {
      const totalWater = brew.pours.reduce((sum, pour) => sum + (pour.water_amount || 0), 0);
      const pourCount = brew.pours.length;
      return `💧 ${totalWater}ml water (${pourCount} pour${pourCount !== 1 ? 's' : ''})`;
    }
    
    return null;
  }
  
  async function deleteBrew(rkey) {
    if (!confirm('Are you sure you want to delete this brew?')) {
      return;
    }
    
    deleting = rkey;
    try {
      await api.delete(`/brews/${rkey}`);
      await cacheStore.invalidate();
      brews = $cacheStore.brews || [];
    } catch (err) {
      alert('Failed to delete brew: ' + err.message);
    } finally {
      deleting = null;
    }
  }
</script>

<svelte:head>
  <title>My Brews - Arabica</title>
</svelte:head>

<div class="max-w-6xl mx-auto">
  <div class="flex items-center justify-between mb-6">
    <h1 class="text-3xl font-bold text-brown-900">My Brews</h1>
    <a
      href="/brews/new"
      on:click|preventDefault={() => navigate('/brews/new')}
      class="bg-gradient-to-r from-brown-700 to-brown-800 text-white px-6 py-3 rounded-lg hover:from-brown-800 hover:to-brown-900 transition-all font-semibold shadow-lg"
    >
      ☕ Add New Brew
    </a>
  </div>

  {#if loading}
    <div class="text-center py-12">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-brown-800 mx-auto"></div>
      <p class="mt-4 text-brown-700">Loading brews...</p>
    </div>
  {:else if brews.length === 0}
    <div class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl p-12 text-center border border-brown-300">
      <div class="text-6xl mb-4">☕</div>
      <h2 class="text-2xl font-bold text-brown-900 mb-2">No Brews Yet</h2>
      <p class="text-brown-700 mb-6">Start tracking your coffee journey by adding your first brew!</p>
      <button
        on:click={() => navigate('/brews/new')}
        class="bg-gradient-to-r from-brown-700 to-brown-800 text-white px-6 py-3 rounded-lg hover:from-brown-800 hover:to-brown-900 transition-all font-semibold shadow-lg inline-block"
      >
        Add Your First Brew
      </button>
    </div>
  {:else}
    <div class="space-y-4">
      {#each brews as brew}
        <div class="bg-gradient-to-br from-brown-50 to-brown-100 rounded-lg shadow-md border border-brown-200 p-5 hover:shadow-lg transition-shadow">
          <div class="flex items-start justify-between gap-4">
            <div class="flex-1 min-w-0">
              <!-- Bean info -->
              {#if brew.bean}
                <h3 class="text-xl font-bold text-brown-900 mb-1">
                  {brew.bean.name || brew.bean.origin || 'Unknown Bean'}
                </h3>
                {#if brew.bean.Roaster?.Name}
                  <p class="text-sm text-brown-700 mb-2">🏭 {brew.bean.roaster.name}</p>
                {/if}
              {:else}
                <h3 class="text-xl font-bold text-brown-900 mb-1">Unknown Bean</h3>
              {/if}
              
              <!-- Brew details -->
              <div class="flex flex-wrap gap-x-4 gap-y-1 text-sm text-brown-600 mb-2">
                {#if brew.brewer_obj}
                  <span>☕ {brew.brewer_obj.name}</span>
                {:else if brew.method}
                  <span>☕ {brew.method}</span>
                {/if}
                {#if hasValue(brew.temperature)}
                  <span>🌡️ {brew.temperature}°C</span>
                {/if}
                {#if hasValue(brew.coffee_amount)}
                  <span>⚖️ {brew.coffee_amount}g coffee</span>
                {/if}
                {#if getWaterDisplay(brew)}
                  <span>{getWaterDisplay(brew)}</span>
                {/if}
              </div>
              
              <!-- Notes preview -->
              {#if brew.tasting_notes}
                <p class="text-sm text-brown-700 italic line-clamp-2">"{brew.tasting_notes}"</p>
              {/if}
              
              <!-- Date -->
              <p class="text-xs text-brown-500 mt-2">
                {formatDate(brew.created_at || brew.created_at)}
              </p>
            </div>
            
            <div class="flex flex-col items-end gap-2">
              {#if hasValue(brew.rating)}
                <span class="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-amber-100 text-amber-900">
                  ⭐ {brew.rating}/10
                </span>
              {/if}
              
              <div class="flex gap-2 items-center">
                <a
                  href="/brews/{brew.rkey}"
                  on:click|preventDefault={() => navigate(`/brews/${brew.rkey}`)}
                  class="text-brown-700 hover:text-brown-900 text-sm font-medium hover:underline"
                >
                  View
                </a>
                <span class="text-brown-400">|</span>
                <a
                  href="/brews/{brew.rkey}/edit"
                  on:click|preventDefault={() => navigate(`/brews/${brew.rkey}/edit`)}
                  class="text-brown-700 hover:text-brown-900 text-sm font-medium hover:underline"
                >
                  Edit
                </a>
                <span class="text-brown-400">|</span>
                <button
                  on:click={() => deleteBrew(brew.rkey)}
                  disabled={deleting === brew.rkey}
                  class="text-red-600 hover:text-red-800 text-sm font-medium hover:underline disabled:opacity-50"
                >
                  {deleting === brew.rkey ? 'Deleting...' : 'Delete'}
                </button>
              </div>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .line-clamp-2 {
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
</style>
