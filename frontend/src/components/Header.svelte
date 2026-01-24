<script>
  import { authStore } from '../stores/auth.js';
  import { navigate } from '../lib/router.js';
  
  let dropdownOpen = false;
  
  $: user = $authStore.user;
  $: isAuthenticated = $authStore.isAuthenticated;
  
  function toggleDropdown() {
    dropdownOpen = !dropdownOpen;
  }
  
  function closeDropdown() {
    dropdownOpen = false;
  }
  
  async function handleLogout() {
    await authStore.logout();
  }
  
  // Close dropdown when clicking outside
  function handleClickOutside(event) {
    if (dropdownOpen && !event.target.closest('.user-menu')) {
      closeDropdown();
    }
  }
</script>

<svelte:window on:click={handleClickOutside} />

<nav class="sticky top-0 z-50 bg-gradient-to-br from-brown-800 to-brown-900 text-white shadow-xl border-b-2 border-brown-600">
  <div class="container mx-auto px-4 py-4">
    <div class="flex items-center justify-between">
      <!-- Logo - always visible -->
      <a href="/" on:click|preventDefault={() => navigate('/')} class="flex items-center gap-2 hover:opacity-80 transition">
        <h1 class="text-2xl font-bold">☕ Arabica</h1>
        <span class="text-xs bg-amber-400 text-brown-900 px-2 py-1 rounded-md font-semibold shadow-sm">ALPHA</span>
      </a>
      
      <!-- Navigation links -->
      <div class="flex items-center gap-4">
        {#if isAuthenticated}
          <!-- User profile dropdown -->
          <div class="relative user-menu">
            <button
              on:click|stopPropagation={toggleDropdown}
              class="flex items-center gap-2 hover:opacity-80 transition focus:outline-none"
              aria-label="User menu"
            >
              {#if user?.avatar}
                <img src={user.avatar} alt="" class="w-8 h-8 rounded-full object-cover ring-2 ring-brown-600" />
              {:else}
                <div class="w-8 h-8 rounded-full bg-brown-600 flex items-center justify-center ring-2 ring-brown-500">
                  <span class="text-sm font-medium">{user?.displayName ? user.displayName.charAt(0).toUpperCase() : '?'}</span>
                </div>
              {/if}
              <svg
                class="w-4 h-4 transition-transform {dropdownOpen ? 'rotate-180' : ''}"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
              </svg>
            </button>
            
            {#if dropdownOpen}
              <div
                class="absolute right-0 mt-2 w-48 bg-white rounded-lg shadow-lg border border-brown-200 py-1 z-50 animate-fade-in"
              >
                {#if user?.handle}
                  <div class="px-4 py-2 border-b border-brown-100">
                    <p class="text-sm font-medium text-brown-900 truncate">{user.displayName || user.handle}</p>
                    <p class="text-xs text-brown-500 truncate">@{user.handle}</p>
                  </div>
                {/if}
                <a
                  href="/profile/{user?.handle || user?.did}"
                  on:click|preventDefault={() => { navigate(`/profile/${user?.handle || user?.did}`); closeDropdown(); }}
                  class="block px-4 py-2 text-sm text-brown-700 hover:bg-brown-50 transition-colors"
                >
                  View Profile
                </a>
                <a
                  href="/brews"
                  on:click|preventDefault={() => { navigate('/brews'); closeDropdown(); }}
                  class="block px-4 py-2 text-sm text-brown-700 hover:bg-brown-50 transition-colors"
                >
                  My Brews
                </a>
                <a
                  href="/manage"
                  on:click|preventDefault={() => { navigate('/manage'); closeDropdown(); }}
                  class="block px-4 py-2 text-sm text-brown-700 hover:bg-brown-50 transition-colors"
                >
                  Manage Records
                </a>
                <a
                  href="/settings"
                  on:click|preventDefault={() => { navigate('/settings'); closeDropdown(); }}
                  class="block px-4 py-2 text-sm text-brown-400 cursor-not-allowed"
                >
                  Settings (coming soon)
                </a>
                <div class="border-t border-brown-100 mt-1 pt-1">
                  <button
                    on:click={() => { handleLogout(); closeDropdown(); }}
                    class="w-full text-left px-4 py-2 text-sm text-brown-700 hover:bg-brown-50 transition-colors"
                  >
                    Logout
                  </button>
                </div>
              </div>
            {/if}
          </div>
        {/if}
      </div>
    </div>
  </div>
</nav>

<style>
  @keyframes fade-in {
    from {
      opacity: 0;
      transform: translateY(-10px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }
  
  .animate-fade-in {
    animation: fade-in 0.2s ease-out;
  }
</style>
