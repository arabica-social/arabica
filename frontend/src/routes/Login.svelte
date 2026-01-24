<script>
  import { onMount } from 'svelte';
  import { authStore } from '../stores/auth.js';
  import { navigate } from '../lib/router.js';
  
  let handle = '';
  let autocompleteResults = [];
  let showAutocomplete = false;
  let loading = false;
  let error = '';
  let debounceTimeout;
  let abortController;
  
  // Redirect if already authenticated
  $: if ($authStore.isAuthenticated && !$authStore.loading) {
    navigate('/');
  }
  
  async function searchActors(query) {
    // Need at least 3 characters to search
    if (query.length < 3) {
      autocompleteResults = [];
      showAutocomplete = false;
      return;
    }
    
    // Cancel previous request
    if (abortController) {
      abortController.abort();
    }
    abortController = new AbortController();
    
    try {
      const response = await fetch(
        `/api/search-actors?q=${encodeURIComponent(query)}`,
        { signal: abortController.signal }
      );
      
      if (!response.ok) {
        autocompleteResults = [];
        showAutocomplete = false;
        return;
      }
      
      const data = await response.json();
      autocompleteResults = data.actors || [];
      showAutocomplete = autocompleteResults.length > 0 || query.length >= 3;
    } catch (err) {
      if (err.name !== 'AbortError') {
        console.error('Error searching actors:', err);
      }
    }
  }
  
  function debounce(func, wait) {
    return (...args) => {
      clearTimeout(debounceTimeout);
      debounceTimeout = setTimeout(() => func(...args), wait);
    };
  }
  
  const debouncedSearch = debounce(searchActors, 300);
  
  function handleInput(e) {
    handle = e.target.value;
    debouncedSearch(handle);
  }
  
  function selectActor(actor) {
    handle = actor.handle;
    autocompleteResults = [];
    showAutocomplete = false;
  }
  
  function handleClickOutside(e) {
    if (!e.target.closest('.autocomplete-container')) {
      showAutocomplete = false;
    }
  }
  
  async function handleSubmit(e) {
    e.preventDefault();
    
    if (!handle) {
      error = 'Please enter your handle';
      return;
    }
    
    loading = true;
    error = '';
    
    // Submit form to Go backend for OAuth flow
    const form = e.target;
    form.submit();
  }
  
  onMount(() => {
    document.addEventListener('click', handleClickOutside);
    return () => {
      document.removeEventListener('click', handleClickOutside);
      if (abortController) {
        abortController.abort();
      }
    };
  });
</script>

<svelte:head>
  <title>Login - Arabica</title>
</svelte:head>

<div class="max-w-4xl mx-auto">
  <div class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-xl p-8 mb-8 border border-brown-300">
    <div class="flex items-center gap-3 mb-4">
      <h2 class="text-3xl font-bold text-brown-900">Welcome to Arabica</h2>
      <span class="text-xs bg-amber-400 text-brown-900 px-2 py-1 rounded-md font-semibold shadow-sm">ALPHA</span>
    </div>
    <p class="text-brown-800 mb-2 text-lg">Track your coffee brewing journey with detailed logs of every cup.</p>
    <p class="text-sm text-brown-700 italic mb-6">Note: Arabica is currently in alpha. Features and data structures may change.</p>
    
    <div>
      <p class="text-brown-800 mb-6 text-center text-lg">Please log in with your AT Protocol handle to start tracking your brews.</p>
      
      <form method="POST" action="/auth/login" on:submit={handleSubmit} class="max-w-md mx-auto">
        <div class="relative autocomplete-container">
          <label for="handle" class="block text-sm font-medium text-brown-900 mb-2">Your Handle</label>
          <input
            type="text"
            id="handle"
            name="handle"
            bind:value={handle}
            on:input={handleInput}
            on:focus={() => { if (autocompleteResults.length > 0 && handle.length >= 3) showAutocomplete = true; }}
            placeholder="alice.bsky.social"
            autocomplete="off"
            required
            disabled={loading}
            class="w-full px-4 py-3 border-2 border-brown-300 rounded-lg focus:ring-2 focus:ring-brown-600 focus:border-brown-600 bg-white disabled:opacity-50"
          />
          
          {#if showAutocomplete}
            <div class="absolute z-10 w-full mt-1 bg-brown-50 border-2 border-brown-300 rounded-lg shadow-lg max-h-60 overflow-y-auto">
              {#if autocompleteResults.length === 0}
                <div class="px-4 py-3 text-sm text-brown-600">No accounts found</div>
              {:else}
                {#each autocompleteResults as actor}
                  <button
                    type="button"
                    on:click={() => selectActor(actor)}
                    class="w-full px-3 py-2 hover:bg-brown-100 cursor-pointer flex items-center gap-2 text-left"
                  >
                    <img
                      src={actor.avatar || '/static/icon-placeholder.svg'}
                      alt=""
                      class="w-6 h-6 rounded-full object-cover flex-shrink-0"
                      on:error={(e) => { e.target.src = '/static/icon-placeholder.svg'; }}
                    />
                    <div class="flex-1 min-w-0">
                      <div class="font-medium text-sm text-brown-900 truncate">
                        {actor.displayName || actor.handle}
                      </div>
                      <div class="text-xs text-brown-600 truncate">
                        @{actor.handle}
                      </div>
                    </div>
                  </button>
                {/each}
              {/if}
            </div>
          {/if}
        </div>
        
        {#if error}
          <div class="mt-3 text-red-600 text-sm">{error}</div>
        {/if}
        
        <button
          type="submit"
          disabled={loading}
          class="w-full mt-4 bg-gradient-to-r from-brown-700 to-brown-800 text-white py-3 px-8 rounded-lg hover:from-brown-800 hover:to-brown-900 transition-all text-lg font-semibold shadow-lg hover:shadow-xl disabled:opacity-50"
        >
          {loading ? 'Logging in...' : 'Log In'}
        </button>
      </form>
    </div>
  </div>
  
  <div class="bg-gradient-to-br from-amber-50 to-brown-100 rounded-xl p-6 border-2 border-brown-300 shadow-lg">
    <h3 class="text-lg font-bold text-brown-900 mb-3">✨ About Arabica</h3>
    <ul class="text-brown-800 space-y-2 leading-relaxed">
      <li class="flex items-start"><span class="mr-2">🔒</span><span><strong>Decentralized:</strong> Your data lives in your Personal Data Server (PDS)</span></li>
      <li class="flex items-start"><span class="mr-2">🚀</span><span><strong>Portable:</strong> Own your coffee brewing history</span></li>
      <li class="flex items-start"><span class="mr-2">📊</span><span>Track brewing variables like temperature, time, and grind size</span></li>
      <li class="flex items-start"><span class="mr-2">🌍</span><span>Organize beans by origin and roaster</span></li>
      <li class="flex items-start"><span class="mr-2">📝</span><span>Add tasting notes and ratings to each brew</span></li>
    </ul>
  </div>
</div>
