<script>
  import { onMount } from "svelte";
  import { authStore } from "../stores/auth.js";
  import { cacheStore } from "../stores/cache.js";
  import { navigate, back } from "../lib/router.js";
  import { api } from "../lib/api.js";

  export let id = null; // RKey from route (for own brews)
  export let did = null; // DID from route (for other users' brews)
  export let rkey = null; // RKey from route (for other users' brews)

  let brew = null;
  let loading = true;
  let error = null;
  let isOwnProfile = false;
  let brewOwnerHandle = null;
  let brewOwnerDID = null;

  $: isAuthenticated = $authStore.isAuthenticated;
  $: currentUserDID = $authStore.user?.did;

  // Calculate total water from pours if water_amount is 0
  $: totalWater =
    brew &&
    (brew.water_amount || 0) === 0 &&
    brew.pours &&
    brew.pours.length > 0
      ? brew.pours.reduce((sum, pour) => sum + (pour.water_amount || 0), 0)
      : brew?.water_amount || 0;

  onMount(async () => {
    if (!isAuthenticated) {
      navigate("/login");
      return;
    }

    // Determine if viewing own brew or someone else's
    if (did && rkey) {
      // Viewing another user's brew
      isOwnProfile = did === currentUserDID;
      await loadBrewFromAPI(did, rkey);
    } else if (id) {
      // Viewing own brew (legacy route)
      isOwnProfile = true;
      await loadBrewFromCache(id);
    }

    loading = false;
  });

  async function loadBrewFromCache(brewRKey) {
    await cacheStore.load();
    const brews = $cacheStore.brews || [];
    brew = brews.find((b) => b.rkey === brewRKey);
    if (!brew) {
      error = "Brew not found";
    } else {
      // Set owner to current user for own brews
      brewOwnerDID = currentUserDID;
      brewOwnerHandle = $authStore.user?.handle;
    }
  }

  async function loadBrewFromAPI(userDID, brewRKey) {
    try {
      // Fetch brew from API using AT-URI
      const atURI = `at://${userDID}/social.arabica.alpha.brew/${brewRKey}`;
      brew = await api.get(`/api/brew?uri=${encodeURIComponent(atURI)}`);
      brewOwnerDID = userDID;
      
      // Fetch the profile to get the handle
      const profileData = await api.get(`/api/profile-json/${userDID}`);
      brewOwnerHandle = profileData.profile?.handle;
    } catch (err) {
      console.error("Failed to load brew:", err);
      error = err.message || "Failed to load brew";
    }
  }

  async function deleteBrew() {
    if (!confirm("Are you sure you want to delete this brew?")) {
      return;
    }

    const brewRKey = rkey || id;
    if (!brewRKey) {
      alert("Cannot delete brew: missing ID");
      return;
    }

    try {
      await api.delete(`/brews/${brewRKey}`);
      await cacheStore.invalidate();
      navigate("/brews");
    } catch (err) {
      alert("Failed to delete brew: " + err.message);
    }
  }

  function hasValue(val) {
    return val !== null && val !== undefined && val !== "";
  }

  function formatTemperature(temp) {
    if (!hasValue(temp)) return null;
    const unit = temp <= 100 ? "C" : "F";
    return `${temp}°${unit}`;
  }

  function formatDate(dateStr) {
    if (!dateStr) return "";
    const date = new Date(dateStr);
    return date.toLocaleDateString("en-US", {
      year: "numeric",
      month: "long",
      day: "numeric",
      hour: "numeric",
      minute: "2-digit",
    });
  }
</script>

<svelte:head>
  <title>Brew Details - Arabica</title>
</svelte:head>

<div class="max-w-2xl mx-auto">
  {#if loading}
    <div class="text-center py-12">
      <div
        class="animate-spin rounded-full h-12 w-12 border-b-2 border-brown-800 mx-auto"
      ></div>
      <p class="mt-4 text-brown-700">Loading brew...</p>
    </div>
  {:else if !brew}
    <div
      class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl p-12 text-center border border-brown-300"
    >
      <h2 class="text-2xl font-bold text-brown-900 mb-2">Brew Not Found</h2>
      <p class="text-brown-700 mb-6">
        The brew you're looking for doesn't exist.
      </p>
      <button
        on:click={() => navigate("/brews")}
        class="bg-gradient-to-r from-brown-700 to-brown-800 text-white px-6 py-3 rounded-lg hover:from-brown-800 hover:to-brown-900 transition-all font-semibold shadow-lg"
      >
        Back to Brews
      </button>
    </div>
  {:else}
    <div
      class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-xl p-8 border border-brown-300"
    >
      <!-- Header with title and actions -->
      <div class="flex justify-between items-start mb-6">
        <div>
          <h2 class="text-3xl font-bold text-brown-900">Brew Details</h2>
          <p class="text-sm text-brown-600 mt-1">
            {formatDate(brew.created_at)}
          </p>
        </div>
        {#if isOwnProfile}
          <div class="flex gap-2">
            <button
              on:click={() =>
                navigate(`/brews/${rkey || id || brew.rkey}/edit`)}
              class="inline-flex items-center bg-brown-300 text-brown-900 px-4 py-2 rounded-lg hover:bg-brown-400 font-medium transition-colors"
            >
              Edit
            </button>
            <button
              on:click={deleteBrew}
              class="inline-flex items-center bg-brown-200 text-brown-700 px-4 py-2 rounded-lg hover:bg-brown-300 font-medium transition-colors"
            >
              Delete
            </button>
          </div>
        {/if}
      </div>

      <div class="space-y-6">
        <!-- Rating (prominent at top) -->
        {#if hasValue(brew.rating)}
          <div
            class="text-center py-4 bg-brown-50 rounded-lg border border-brown-200"
          >
            <div class="text-4xl font-bold text-brown-800">
              {brew.rating}/10
            </div>
            <div class="text-sm text-brown-600 mt-1">Rating</div>
          </div>
        {/if}

        <!-- Coffee Bean -->
        <div class="bg-brown-50 rounded-lg p-4 border border-brown-200">
          <h3
            class="text-sm font-medium text-brown-600 uppercase tracking-wider mb-2"
          >
            Coffee Bean
          </h3>
          {#if brew.bean}
            <div class="font-bold text-lg text-brown-900">
              {brew.bean.name || brew.bean.origin}
            </div>
            {#if brew.bean.roaster?.Name}
              <div class="text-sm text-brown-700 mt-1">
                by {brew.bean.roaster.name}
              </div>
            {/if}
            <div class="flex flex-wrap gap-3 mt-2 text-sm text-brown-600">
              {#if brew.bean.origin}<span>Origin: {brew.bean.origin}</span>{/if}
              {#if brew.bean.roast_level}<span
                  >Roast: {brew.bean.roast_level}</span
                >{/if}
            </div>
          {:else}
            <span class="text-brown-400">Not specified</span>
          {/if}
        </div>

        <!-- Brew Parameters -->
        <div class="grid grid-cols-2 gap-4">
          <!-- Brew Method -->
          <div class="bg-brown-50 rounded-lg p-4 border border-brown-200">
            <h3
              class="text-sm font-medium text-brown-600 uppercase tracking-wider mb-2"
            >
              Brew Method
            </h3>
            {#if brew.brewer_obj}
              <div class="font-semibold text-brown-900">
                {brew.brewer_obj.name}
              </div>
            {:else if brew.method}
              <div class="font-semibold text-brown-900">{brew.method}</div>
            {:else}
              <span class="text-brown-400">Not specified</span>
            {/if}
          </div>

          <!-- Grinder -->
          <div class="bg-brown-50 rounded-lg p-4 border border-brown-200">
            <h3
              class="text-sm font-medium text-brown-600 uppercase tracking-wider mb-2"
            >
              Grinder
            </h3>
            {#if brew.grinder_obj}
              <div class="font-semibold text-brown-900">
                {brew.grinder_obj.name}
              </div>
            {:else}
              <span class="text-brown-400">Not specified</span>
            {/if}
          </div>

          <!-- Coffee Amount -->
          <div class="bg-brown-50 rounded-lg p-4 border border-brown-200">
            <h3
              class="text-sm font-medium text-brown-600 uppercase tracking-wider mb-2"
            >
              Coffee
            </h3>
            {#if hasValue(brew.coffee_amount)}
              <div class="font-semibold text-brown-900">
                {brew.coffee_amount}g
              </div>
            {:else}
              <span class="text-brown-400">Not specified</span>
            {/if}
          </div>

          <!-- Water Amount -->
          <div class="bg-brown-50 rounded-lg p-4 border border-brown-200">
            <h3
              class="text-sm font-medium text-brown-600 uppercase tracking-wider mb-2"
            >
              Water
            </h3>
            {#if hasValue(totalWater)}
              <div class="font-semibold text-brown-900">{totalWater}g</div>
            {:else}
              <span class="text-brown-400">Not specified</span>
            {/if}
          </div>

          <!-- Grind Size -->
          <div class="bg-brown-50 rounded-lg p-4 border border-brown-200">
            <h3
              class="text-sm font-medium text-brown-600 uppercase tracking-wider mb-2"
            >
              Grind Size
            </h3>
            {#if brew.grind_size}
              <div class="font-semibold text-brown-900">{brew.grind_size}</div>
            {:else}
              <span class="text-brown-400">Not specified</span>
            {/if}
          </div>

          <!-- Water Temperature -->
          <div class="bg-brown-50 rounded-lg p-4 border border-brown-200">
            <h3
              class="text-sm font-medium text-brown-600 uppercase tracking-wider mb-2"
            >
              Water Temp
            </h3>
            {#if hasValue(brew.temperature)}
              <div class="font-semibold text-brown-900">
                {formatTemperature(brew.temperature)}
              </div>
            {:else}
              <span class="text-brown-400">Not specified</span>
            {/if}
          </div>
        </div>

        <!-- Pours (if any) -->
        {#if brew.pours && brew.pours.length > 0}
          <div class="bg-brown-50 rounded-lg p-4 border border-brown-200">
            <h3
              class="text-sm font-medium text-brown-600 uppercase tracking-wider mb-3"
            >
              Pour Schedule
            </h3>
            <div class="space-y-2">
              {#each brew.pours as pour, i}
                <div class="flex justify-between text-sm">
                  <span class="text-brown-700">Pour {i + 1}:</span>
                  <span class="font-semibold text-brown-900"
                    >{pour.water_amount}g at {pour.time_seconds}s</span
                  >
                </div>
              {/each}
            </div>
          </div>
        {/if}

        <!-- Tasting Notes -->
        {#if brew.tasting_notes}
          <div class="bg-brown-50 rounded-lg p-4 border border-brown-200">
            <h3
              class="text-sm font-medium text-brown-600 uppercase tracking-wider mb-2"
            >
              Tasting Notes
            </h3>
            <p class="text-brown-900 italic">"{brew.tasting_notes}"</p>
          </div>
        {/if}
      </div>

      <!-- Back button -->
      <div class="mt-6">
        <button
          on:click={() => {
            if (isOwnProfile) {
              navigate("/brews");
            } else if (brewOwnerHandle) {
              navigate(`/profile/${brewOwnerHandle}`);
            } else if (brewOwnerDID) {
              navigate(`/profile/${brewOwnerDID}`);
            } else {
              navigate("/");
            }
          }}
          class="text-brown-700 hover:text-brown-900 font-medium hover:underline"
        >
          ← {isOwnProfile ? "Back to My Brews" : "Back to Profile"}
        </button>
      </div>
    </div>
  {/if}
</div>
