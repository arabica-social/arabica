<script>
  import { onMount } from "svelte";
  import { api } from "../lib/api.js";
  import { navigate } from "../lib/router.js";
  import Modal from "../components/Modal.svelte";

  export let actor;

  let profile = null;
  let brews = [];
  let beans = [];
  let roasters = [];
  let grinders = [];
  let brewers = [];
  let isOwnProfile = false;
  let loading = true;
  let error = null;

  let activeTab = "brews";

  // Modal states
  let showBeanModal = false;
  let showRoasterModal = false;
  let showGrinderModal = false;
  let showBrewerModal = false;

  // Forms
  let beanForm = {
    name: "",
    origin: "",
    roast_level: "",
    process: "",
    description: "",
    roaster_rkey: "",
  };
  let roasterForm = { name: "", location: "", website: "", description: "" };
  let grinderForm = { name: "", grinder_type: "", burr_type: "", notes: "" };
  let brewerForm = { name: "", brewer_type: "", description: "" };

  onMount(async () => {
    await loadProfile();
  });

  async function loadProfile() {
    try {
      const data = await api.get(`/api/profile-json/${actor}`);
      profile = data.profile;
      brews = (data.brews || []).sort(
        (a, b) => new Date(b.created_at) - new Date(a.created_at),
      );
      beans = data.beans || [];
      roasters = data.roasters || [];
      grinders = data.grinders || [];
      brewers = data.brewers || [];
      isOwnProfile = data.isOwnProfile || false;
    } catch (err) {
      console.error("Failed to load profile:", err);
      error = err.message;
    } finally {
      loading = false;
    }
  }

  function formatDate(dateStr) {
    const date = new Date(dateStr);
    return date.toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  }

  // Bean handlers
  function addBean() {
    beanForm = {
      name: "",
      origin: "",
      roast_level: "",
      process: "",
      description: "",
      roaster_rkey: "",
    };
    showBeanModal = true;
  }

  async function saveBean() {
    try {
      await api.post("/api/beans", beanForm);
      await loadProfile();
      showBeanModal = false;
    } catch (err) {
      console.error("Bean save error:", err);
      alert("Failed to save bean: " + err.message);
    }
  }

  // Roaster handlers
  function addRoaster() {
    roasterForm = { name: "", location: "", website: "", description: "" };
    showRoasterModal = true;
  }

  async function saveRoaster() {
    try {
      await api.post("/api/roasters", roasterForm);
      await loadProfile();
      showRoasterModal = false;
    } catch (err) {
      alert("Failed to save roaster: " + err.message);
    }
  }

  // Grinder handlers
  function addGrinder() {
    grinderForm = { name: "", grinder_type: "", burr_type: "", notes: "" };
    showGrinderModal = true;
  }

  async function saveGrinder() {
    try {
      await api.post("/api/grinders", grinderForm);
      await loadProfile();
      showGrinderModal = false;
    } catch (err) {
      alert("Failed to save grinder: " + err.message);
    }
  }

  // Brewer handlers
  function addBrewer() {
    brewerForm = { name: "", brewer_type: "", description: "" };
    showBrewerModal = true;
  }

  async function saveBrewer() {
    try {
      await api.post("/api/brewers", brewerForm);
      await loadProfile();
      showBrewerModal = false;
    } catch (err) {
      alert("Failed to save brewer: " + err.message);
    }
  }
</script>

<svelte:head>
  <title>{profile?.displayName || profile?.handle || "Profile"} - Arabica</title
  >
</svelte:head>

<div class="max-w-4xl mx-auto">
  {#if loading}
    <div class="text-center py-12">
      <div
        class="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-brown-900"
      ></div>
      <p class="mt-4 text-brown-700">Loading profile...</p>
    </div>
  {:else if error}
    <div
      class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded"
    >
      Error: {error}
    </div>
  {:else if profile}
    <!-- Profile Header -->
    <div
      class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-xl p-6 mb-6 border border-brown-300"
    >
      <div class="flex items-center gap-4">
        {#if profile.avatar}
          <img
            src={profile.avatar}
            alt=""
            class="w-20 h-20 rounded-full object-cover border-2 border-brown-300"
          />
        {:else}
          <div
            class="w-20 h-20 rounded-full bg-brown-300 flex items-center justify-center"
          >
            <span class="text-brown-600 text-2xl">?</span>
          </div>
        {/if}
        <div>
          {#if profile.displayName}
            <h1 class="text-2xl font-bold text-brown-900">
              {profile.displayName}
            </h1>
          {/if}
          <p class="text-brown-700">@{profile.handle}</p>
        </div>
      </div>
    </div>

    <!-- Stats -->
    <div class="grid grid-cols-2 md:grid-cols-5 gap-4 mb-6">
      <div
        class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-lg shadow-md p-4 text-center border border-brown-300"
      >
        <div class="text-2xl font-bold text-brown-800">{brews.length}</div>
        <div class="text-sm text-brown-700">Brews</div>
      </div>
      <div
        class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-lg shadow-md p-4 text-center border border-brown-300"
      >
        <div class="text-2xl font-bold text-brown-800">{beans.length}</div>
        <div class="text-sm text-brown-700">Beans</div>
      </div>
      <div
        class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-lg shadow-md p-4 text-center border border-brown-300"
      >
        <div class="text-2xl font-bold text-brown-800">{roasters.length}</div>
        <div class="text-sm text-brown-700">Roasters</div>
      </div>
      <div
        class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-lg shadow-md p-4 text-center border border-brown-300"
      >
        <div class="text-2xl font-bold text-brown-800">{grinders.length}</div>
        <div class="text-sm text-brown-700">Grinders</div>
      </div>
      <div
        class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-lg shadow-md p-4 text-center border border-brown-300"
      >
        <div class="text-2xl font-bold text-brown-800">{brewers.length}</div>
        <div class="text-sm text-brown-700">Brewers</div>
      </div>
    </div>

    <!-- Tabs -->
    <div>
      <div
        class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-md mb-4 border border-brown-300"
      >
        <div class="flex border-b border-brown-300">
          <button
            on:click={() => (activeTab = "brews")}
            class="flex-1 py-3 px-4 text-center font-medium transition-colors {activeTab ===
            'brews'
              ? 'border-b-2 border-brown-700 text-brown-900'
              : 'text-brown-600 hover:text-brown-800'}"
          >
            Brews
          </button>
          <button
            on:click={() => (activeTab = "beans")}
            class="flex-1 py-3 px-4 text-center font-medium transition-colors {activeTab ===
            'beans'
              ? 'border-b-2 border-brown-700 text-brown-900'
              : 'text-brown-600 hover:text-brown-800'}"
          >
            Beans
          </button>
          <button
            on:click={() => (activeTab = "gear")}
            class="flex-1 py-3 px-4 text-center font-medium transition-colors {activeTab ===
            'gear'
              ? 'border-b-2 border-brown-700 text-brown-900'
              : 'text-brown-600 hover:text-brown-800'}"
          >
            Gear
          </button>
        </div>
      </div>

      <!-- Tab Content -->
      {#if activeTab === "brews"}
        {#if brews.length === 0}
          <div
            class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-xl p-8 text-center border border-brown-300"
          >
            <p class="text-brown-800 text-lg font-medium">No brews yet.</p>
            {#if isOwnProfile}
              <div class="mt-4">
                <button
                  on:click={() => navigate("/brews/new")}
                  class="bg-brown-700 text-white px-6 py-2 rounded-lg hover:bg-brown-800 font-medium transition-colors"
                >
                  + Add Brew
                </button>
              </div>
            {/if}
          </div>
        {:else}
          <div
            class="overflow-x-auto bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-xl border border-brown-300"
          >
            <table class="min-w-full divide-y divide-brown-300">
              <thead class="bg-brown-200/80">
                <tr>
                  <th
                    class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                    >📅 Date</th
                  >
                  <th
                    class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                    >☕ Bean</th
                  >
                  <th
                    class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                    >🫖 Method</th
                  >
                  <th
                    class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                    >📝 Notes</th
                  >
                  <th
                    class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                    >⭐ Rating</th
                  >
                </tr>
              </thead>
              <tbody class="bg-brown-50/60 divide-y divide-brown-200">
                {#each brews as brew}
                  <tr class="hover:bg-brown-100/60 transition-colors">
                    <td class="px-4 py-3 text-sm text-brown-900"
                      >{formatDate(brew.created_at)}</td
                    >
                    <td class="px-4 py-3 text-sm font-bold text-brown-900"
                      >{brew.bean?.name || brew.bean?.origin || "Unknown"}</td
                    >
                    <td class="px-4 py-3 text-sm text-brown-900"
                      >{brew.brewer_obj?.name || "-"}</td
                    >
                    <td
                      class="px-4 py-3 text-sm text-brown-700 truncate max-w-xs"
                      >{brew.tasting_notes || "-"}</td
                    >
                    <td class="px-4 py-3 text-sm text-brown-900">
                      {#if brew.rating}
                        <span
                          class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-amber-100 text-amber-900"
                        >
                          ⭐ {brew.rating}/10
                        </span>
                      {:else}
                        <span class="text-brown-400">-</span>
                      {/if}
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
          {#if isOwnProfile}
            <div class="mt-4 text-center">
              <button
                on:click={() => navigate("/brews/new")}
                class="bg-brown-700 text-white px-6 py-2 rounded-lg hover:bg-brown-800 font-medium transition-colors"
              >
                + Add Brew
              </button>
            </div>
          {/if}
        {/if}
      {:else if activeTab === "beans"}
        <div class="space-y-6">
          {#if beans.length > 0}
            <div>
              <h3 class="text-lg font-semibold text-brown-900 mb-3">
                ☕ Coffee Beans
              </h3>
              <div
                class="overflow-x-auto bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-xl border border-brown-300"
              >
                <table class="min-w-full divide-y divide-brown-300">
                  <thead class="bg-brown-200/80">
                    <tr>
                      <th
                        class="px-6 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider whitespace-nowrap"
                        >Name</th
                      >
                      <th
                        class="px-6 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider whitespace-nowrap"
                        >☕ Roaster</th
                      >
                      <th
                        class="px-6 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider whitespace-nowrap"
                        >📍 Origin</th
                      >
                      <th
                        class="px-6 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider whitespace-nowrap"
                        >🔥 Roast</th
                      >
                      <th
                        class="px-6 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider whitespace-nowrap"
                        >🌱 Process</th
                      >
                      <th
                        class="px-6 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider whitespace-nowrap"
                        >📝 Description</th
                      >
                    </tr>
                  </thead>
                  <tbody class="bg-brown-50/60 divide-y divide-brown-200">
                    {#each beans as bean}
                      <tr class="hover:bg-brown-100/60 transition-colors">
                        <td class="px-6 py-4 text-sm font-bold text-brown-900"
                          >{bean.name || bean.origin}</td
                        >
                        <td class="px-6 py-4 text-sm text-brown-900"
                          >{bean.roaster?.name || "-"}</td
                        >
                        <td class="px-6 py-4 text-sm text-brown-900"
                          >{bean.origin || "-"}</td
                        >
                        <td class="px-6 py-4 text-sm text-brown-900"
                          >{bean.roast_level || "-"}</td
                        >
                        <td class="px-6 py-4 text-sm text-brown-900"
                          >{bean.process || "-"}</td
                        >
                        <td
                          class="px-6 py-4 text-sm text-brown-700 italic max-w-xs"
                          >{bean.description || "-"}</td
                        >
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
              {#if isOwnProfile}
                <div class="mt-4 text-center">
                  <button
                    on:click={addBean}
                    class="bg-brown-700 text-white px-6 py-2 rounded-lg hover:bg-brown-800 font-medium transition-colors"
                  >
                    + Add Bean
                  </button>
                </div>
              {/if}
            </div>
          {/if}

          {#if roasters.length > 0}
            <div>
              <h3 class="text-lg font-semibold text-brown-900 mb-3">
                🏭 Favorite Roasters
              </h3>
              <div
                class="overflow-x-auto bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-xl border border-brown-300"
              >
                <table class="min-w-full divide-y divide-brown-300">
                  <thead class="bg-brown-200/80">
                    <tr>
                      <th
                        class="px-6 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                        >Name</th
                      >
                      <th
                        class="px-6 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                        >📍 Location</th
                      >
                      <th
                        class="px-6 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                        >🌐 Website</th
                      >
                    </tr>
                  </thead>
                  <tbody class="bg-brown-50/60 divide-y divide-brown-200">
                    {#each roasters as roaster}
                      <tr class="hover:bg-brown-100/60 transition-colors">
                        <td class="px-6 py-4 text-sm font-bold text-brown-900"
                          >{roaster.name}</td
                        >
                        <td class="px-6 py-4 text-sm text-brown-900"
                          >{roaster.location || "-"}</td
                        >
                        <td class="px-6 py-4 text-sm text-brown-900">
                          {#if roaster.website}
                            <a
                              href={roaster.website}
                              target="_blank"
                              rel="noopener noreferrer"
                              class="text-brown-700 hover:underline font-medium"
                              >Visit Site</a
                            >
                          {:else}
                            -
                          {/if}
                        </td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
              {#if isOwnProfile}
                <div class="mt-4 text-center">
                  <button
                    on:click={addRoaster}
                    class="bg-brown-700 text-white px-6 py-2 rounded-lg hover:bg-brown-800 font-medium transition-colors"
                  >
                    + Add Roaster
                  </button>
                </div>
              {/if}
            </div>
          {/if}

          {#if beans.length === 0 && roasters.length === 0}
            <div
              class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-xl p-8 text-center text-brown-800 border border-brown-300"
            >
              <p class="font-medium">No beans or roasters yet.</p>
              {#if isOwnProfile}
                <div class="mt-4 flex gap-4 justify-center">
                  <button
                    on:click={addBean}
                    class="bg-brown-700 text-white px-6 py-2 rounded-lg hover:bg-brown-800 font-medium transition-colors"
                  >
                    + Add Bean
                  </button>
                  <button
                    on:click={addRoaster}
                    class="bg-brown-700 text-white px-6 py-2 rounded-lg hover:bg-brown-800 font-medium transition-colors"
                  >
                    + Add Roaster
                  </button>
                </div>
              {/if}
            </div>
          {/if}
        </div>
      {:else if activeTab === "gear"}
        <div class="space-y-6">
          {#if grinders.length > 0}
            <div>
              <h3 class="text-lg font-semibold text-brown-900 mb-3">
                ⚙️ Grinders
              </h3>
              <div
                class="overflow-x-auto bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-xl border border-brown-300"
              >
                <table class="min-w-full divide-y divide-brown-300">
                  <thead class="bg-brown-200/80">
                    <tr>
                      <th
                        class="px-6 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                        >Name</th
                      >
                      <th
                        class="px-6 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                        >🔧 Type</th
                      >
                      <th
                        class="px-6 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                        >💎 Burrs</th
                      >
                      <th
                        class="px-6 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                        >📝 Notes</th
                      >
                    </tr>
                  </thead>
                  <tbody class="bg-brown-50/60 divide-y divide-brown-200">
                    {#each grinders as grinder}
                      <tr class="hover:bg-brown-100/60 transition-colors">
                        <td class="px-6 py-4 text-sm font-bold text-brown-900"
                          >{grinder.name}</td
                        >
                        <td class="px-6 py-4 text-sm text-brown-900"
                          >{grinder.grinder_type || "-"}</td
                        >
                        <td class="px-6 py-4 text-sm text-brown-900"
                          >{grinder.burr_type || "-"}</td
                        >
                        <td
                          class="px-6 py-4 text-sm text-brown-700 italic max-w-xs"
                          >{grinder.notes || "-"}</td
                        >
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
              {#if isOwnProfile}
                <div class="mt-4 text-center">
                  <button
                    on:click={addGrinder}
                    class="bg-brown-700 text-white px-6 py-2 rounded-lg hover:bg-brown-800 font-medium transition-colors"
                  >
                    + Add Grinder
                  </button>
                </div>
              {/if}
            </div>
          {/if}

          {#if brewers.length > 0}
            <div>
              <h3 class="text-lg font-semibold text-brown-900 mb-3">
                ☕ Brewers
              </h3>
              <div
                class="overflow-x-auto bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-xl border border-brown-300"
              >
                <table class="min-w-full divide-y divide-brown-300">
                  <thead class="bg-brown-200/80">
                    <tr>
                      <th
                        class="px-6 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                        >Name</th
                      >
                      <th
                        class="px-6 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                        >🔧 Type</th
                      >
                      <th
                        class="px-6 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                        >📝 Description</th
                      >
                    </tr>
                  </thead>
                  <tbody class="bg-brown-50/60 divide-y divide-brown-200">
                    {#each brewers as brewer}
                      <tr class="hover:bg-brown-100/60 transition-colors">
                        <td class="px-6 py-4 text-sm font-bold text-brown-900"
                          >{brewer.name}</td
                        >
                        <td class="px-6 py-4 text-sm text-brown-900"
                          >{brewer.brewer_type || "-"}</td
                        >
                        <td
                          class="px-6 py-4 text-sm text-brown-700 italic max-w-xs"
                          >{brewer.description || "-"}</td
                        >
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
              {#if isOwnProfile}
                <div class="mt-4 text-center">
                  <button
                    on:click={addBrewer}
                    class="bg-brown-700 text-white px-6 py-2 rounded-lg hover:bg-brown-800 font-medium transition-colors"
                  >
                    + Add Brewer
                  </button>
                </div>
              {/if}
            </div>
          {/if}

          {#if grinders.length === 0 && brewers.length === 0}
            <div
              class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-xl p-8 text-center text-brown-800 border border-brown-300"
            >
              <p class="font-medium">No gear added yet.</p>
              {#if isOwnProfile}
                <div class="mt-4 flex gap-4 justify-center">
                  <button
                    on:click={addGrinder}
                    class="bg-brown-700 text-white px-6 py-2 rounded-lg hover:bg-brown-800 font-medium transition-colors"
                  >
                    + Add Grinder
                  </button>
                  <button
                    on:click={addBrewer}
                    class="bg-brown-700 text-white px-6 py-2 rounded-lg hover:bg-brown-800 font-medium transition-colors"
                  >
                    + Add Brewer
                  </button>
                </div>
              {/if}
            </div>
          {/if}
        </div>
      {/if}
    </div>
  {/if}
</div>

<!-- Modals -->
{#if isOwnProfile}
  <Modal
    bind:isOpen={showBeanModal}
    title="Add Bean"
    onSave={saveBean}
    onCancel={() => (showBeanModal = false)}
  >
    <input
      type="text"
      bind:value={beanForm.name}
      placeholder="Name *"
      class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
    />
    <input
      type="text"
      bind:value={beanForm.origin}
      placeholder="Origin *"
      class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
    />
    <select
      bind:value={beanForm.roaster_rkey}
      class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
    >
      <option value="">Select Roaster (Optional)</option>
      {#each roasters as roaster}
        <option value={roaster.rkey}>{roaster.name}</option>
      {/each}
    </select>
    <select
      bind:value={beanForm.roast_level}
      class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
    >
      <option value="">Select Roast Level (Optional)</option>
      <option value="Ultra-Light">Ultra-Light</option>
      <option value="Light">Light</option>
      <option value="Medium-Light">Medium-Light</option>
      <option value="Medium">Medium</option>
      <option value="Medium-Dark">Medium-Dark</option>
      <option value="Dark">Dark</option>
    </select>
    <input
      type="text"
      bind:value={beanForm.process}
      placeholder="Process (e.g. Washed, Natural, Honey)"
      class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
    />
    <textarea
      bind:value={beanForm.description}
      placeholder="Description"
      rows="3"
      class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
    ></textarea>
  </Modal>

  <Modal
    bind:isOpen={showRoasterModal}
    title="Add Roaster"
    onSave={saveRoaster}
    onCancel={() => (showRoasterModal = false)}
  >
    <input
      type="text"
      bind:value={roasterForm.name}
      placeholder="Name *"
      class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
    />
    <input
      type="text"
      bind:value={roasterForm.location}
      placeholder="Location"
      class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
    />
    <input
      type="url"
      bind:value={roasterForm.website}
      placeholder="Website"
      class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
    />
  </Modal>

  <Modal
    bind:isOpen={showGrinderModal}
    title="Add Grinder"
    onSave={saveGrinder}
    onCancel={() => (showGrinderModal = false)}
  >
    <input
      type="text"
      bind:value={grinderForm.name}
      placeholder="Name *"
      class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
    />
    <select
      bind:value={grinderForm.grinder_type}
      class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
    >
      <option value="">Select Grinder Type *</option>
      <option value="Hand">Hand</option>
      <option value="Electric">Electric</option>
      <option value="Portable Electric">Portable Electric</option>
    </select>
    <select
      bind:value={grinderForm.burr_type}
      class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
    >
      <option value="">Select Burr Type (Optional)</option>
      <option value="Conical">Conical</option>
      <option value="Flat">Flat</option>
    </select>
    <textarea
      bind:value={grinderForm.notes}
      placeholder="Notes"
      rows="3"
      class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
    ></textarea>
  </Modal>

  <Modal
    bind:isOpen={showBrewerModal}
    title="Add Brewer"
    onSave={saveBrewer}
    onCancel={() => (showBrewerModal = false)}
  >
    <input
      type="text"
      bind:value={brewerForm.name}
      placeholder="Name *"
      class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
    />
    <input
      type="text"
      bind:value={brewerForm.brewer_type}
      placeholder="Type (e.g., Pour-Over, Immersion, Espresso)"
      class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
    />
    <textarea
      bind:value={brewerForm.description}
      placeholder="Description"
      rows="3"
      class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
    ></textarea>
  </Modal>
{/if}
