<script>
  import { onMount } from "svelte";
  import { authStore } from "../stores/auth.js";
  import { navigate } from "../lib/router.js";
  import { api } from "../lib/api.js";
  import FeedCard from "../components/FeedCard.svelte";

  let feedItems = [];
  let loading = true;
  let error = null;

  $: isAuthenticated = $authStore.isAuthenticated;
  $: user = $authStore.user;

  onMount(async () => {
    try {
      const data = await api.get("/api/feed-json");
      feedItems = data.items || [];
    } catch (err) {
      // Feed might return 401 for unauthenticated users - that's okay
      // Just log it and show empty feed
      console.error("Failed to load feed:", err);
      if (err.status !== 401 && err.status !== 403) {
        error = err.message;
      }
    } finally {
      loading = false;
    }
  });
</script>

<svelte:head>
  <title>Arabica - Coffee Brew Tracker</title>
</svelte:head>

<div class="max-w-4xl mx-auto">
  <div
    class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-xl p-5 md:p-8 mb-8 border border-brown-300"
  >
    <div class="flex items-center gap-3 mb-4">
      <h2 class="text-3xl font-bold text-brown-900">Welcome to Arabica</h2>
      <span
        class="text-xs bg-amber-400 text-brown-900 px-2 py-1 rounded-md font-semibold shadow-sm"
        >ALPHA</span
      >
    </div>
    <p class="text-brown-800 mb-2 text-lg">
      Track your coffee brewing journey with detailed logs of every cup.
    </p>
    <p class="text-sm text-brown-700 italic mb-6">
      Note: Arabica is currently in alpha. Features and data structures may
      change.
    </p>

    {#if isAuthenticated}
      <!-- Authenticated: Show app actions -->
      <div class="mb-6">
        <p class="text-sm text-brown-700">
          Logged in as: <span class="font-mono text-brown-900 font-semibold"
            >{user?.did}</span
          >
        </p>
      </div>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <a
          href="/brews/new"
          on:click|preventDefault={() => navigate("/brews/new")}
          class="block bg-gradient-to-br from-brown-700 to-brown-800 text-white text-center py-4 px-6 rounded-xl hover:from-brown-800 hover:to-brown-900 transition-all shadow-lg hover:shadow-xl transform"
        >
          <span class="text-xl font-semibold">☕ Add New Brew</span>
        </a>
        <a
          href="/brews"
          on:click|preventDefault={() => navigate("/brews")}
          class="block bg-gradient-to-br from-brown-500 to-brown-600 text-white text-center py-4 px-6 rounded-xl hover:from-brown-600 hover:to-brown-700 transition-all shadow-lg hover:shadow-xl"
        >
          <span class="text-xl font-semibold">📋 View All Brews</span>
        </a>
      </div>
    {:else}
      <!-- Not authenticated: Show login button -->
      <div class="text-center">
        <button
          on:click={() => navigate("/login")}
          class="bg-gradient-to-r from-brown-700 to-brown-800 text-white py-3 px-8 rounded-lg hover:from-brown-800 hover:to-brown-900 transition-all text-lg font-semibold shadow-lg hover:shadow-xl inline-block"
        >
          Log In to Start Tracking
        </button>
      </div>
    {/if}
  </div>

  <!-- Community Feed -->
  <div
    class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-xl p-4 md:p-6 mb-8 border border-brown-300 -mx-3 md:mx-0"
  >
    <h3 class="text-xl font-bold text-brown-900 mb-4">☕ Community Feed</h3>

    {#if loading}
      <!-- Loading state -->
      <div class="space-y-4">
        {#each Array(3) as _}
          <div class="animate-pulse">
            <div class="bg-brown-50 rounded-lg p-4 border border-brown-200">
              <div class="flex items-center gap-3 mb-3">
                <div class="w-10 h-10 rounded-full bg-brown-300"></div>
                <div class="flex-1">
                  <div class="h-4 bg-brown-300 rounded w-1/4 mb-2"></div>
                  <div class="h-3 bg-brown-200 rounded w-1/6"></div>
                </div>
              </div>
              <div class="bg-brown-200 rounded-lg p-3">
                <div class="h-4 bg-brown-300 rounded w-3/4 mb-2"></div>
                <div class="h-3 bg-brown-200 rounded w-1/2"></div>
              </div>
            </div>
          </div>
        {/each}
      </div>
    {:else if error}
      <div class="text-center py-8 text-brown-600">
        Failed to load feed: {error}
      </div>
    {:else if feedItems.length === 0}
      <div class="text-center py-8 text-brown-600">
        No activity yet. {#if isAuthenticated}Start by adding your first brew!{:else}Log
          in to see your feed.{/if}
      </div>
    {:else}
      <div class="space-y-4">
        {#each feedItems as item (item.Timestamp)}
          <FeedCard {item} />
        {/each}
      </div>
    {/if}
  </div>

  <div
    class="bg-gradient-to-br from-amber-50 to-brown-100 rounded-xl p-4 md:p-6 border-2 border-brown-300 shadow-lg"
  >
    <h3 class="text-lg font-bold text-brown-900 mb-3">✨ About Arabica</h3>
    <ul class="text-brown-800 space-y-2 leading-relaxed">
      <li class="flex items-start">
        <span class="mr-2">🔒</span><span
          ><strong>Decentralized:</strong> Your data lives in your Personal Data
          Server (PDS)</span
        >
      </li>
      <li class="flex items-start">
        <span class="mr-2">🚀</span><span
          ><strong>Portable:</strong> Own your coffee brewing history</span
        >
      </li>
      <li class="flex items-start">
        <span class="mr-2">📊</span><span
          >Track brewing variables like temperature, time, and grind size</span
        >
      </li>
      <li class="flex items-start">
        <span class="mr-2">🌍</span><span
          >Organize beans by origin and roaster</span
        >
      </li>
      <li class="flex items-start">
        <span class="mr-2">📝</span><span
          >Add tasting notes and ratings to each brew</span
        >
      </li>
    </ul>
  </div>
</div>
